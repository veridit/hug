package app_test

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/veridit/hug/muninn/internal/app"
	"github.com/veridit/hug/muninn/internal/scopetest"
)

// 2026-11-09 (user): moving 1,526 memories one process at a time took ~2.6 s
// each. A batch is checked as a whole, written once and indexed once.
func TestRescopeBatchMovesAllOrNothing(t *testing.T) {
	f := scopetest.New(t)
	global := f.Bind(t, "global")
	ctx := context.Background()
	shared, jg := f.Units["Shared"], f.Units["JcodeGlobal"]
	target := "project:matrix-project"
	before, err := os.ReadFile(app.NativePath())
	if err != nil {
		t.Fatal(err)
	}

	// One bad line (client crossing) means nothing moves, and every problem is named.
	bad := []app.RescopeItem{{shared.SID, target}, {jg.SID, "client:acme"}, {"nosuch", target}}
	if _, err := global.RescopeBatch(ctx, f.App, bad, false, false); err == nil || !strings.Contains(err.Error(), "2 of 3") || !strings.Contains(err.Error(), "--cross-client") {
		t.Fatalf("want all-or-nothing refusal naming both problems, got %v", err)
	}
	// A memory listed twice is refused too.
	if _, err := global.RescopeBatch(ctx, f.App, []app.RescopeItem{{shared.SID, target}, {shared.SID, target}}, false, false); err == nil || !strings.Contains(err.Error(), "listed twice") {
		t.Fatalf("duplicate line accepted: %v", err)
	}
	// A dry run reports the moves without writing.
	res, err := global.RescopeBatch(ctx, f.App, []app.RescopeItem{{shared.SID, target}, {jg.SID, target}}, false, true)
	if err != nil || len(res.Moved) != 2 || res.Backup != "" {
		t.Fatalf("dry run: %+v %v", res, err)
	}
	after, _ := os.ReadFile(app.NativePath())
	if !bytes.Equal(before, after) {
		t.Fatal("refused or dry batch changed memories.json")
	}

	res, err = global.RescopeBatch(ctx, f.App, []app.RescopeItem{{shared.SID, target}, {jg.SID, target}}, false, false)
	if err != nil || len(res.Moved) != 2 {
		t.Fatalf("batch: %+v %v", res, err)
	}
	for _, sid := range []string{shared.SID, jg.SID} {
		u, err := f.App.DB.Resolve(sid)
		if err != nil || u.Scope != target {
			t.Fatalf("%s not moved: %+v %v", sid, u, err)
		}
	}
	// The backup restores the exact file.
	backup, err := os.ReadFile(filepath.Join(res.Backup, "1-memories.json"))
	if err != nil || !bytes.Equal(backup, before) {
		t.Fatalf("backup differs from the original: %v", err)
	}
	where, _ := os.ReadFile(filepath.Join(res.Backup, "1-memories.json.path"))
	if strings.TrimSpace(string(where)) != app.NativePath() {
		t.Fatalf("backup path note: %q", where)
	}
	// Capture is free again afterwards.
	var held int
	f.App.DB.QueryRow(`SELECT COUNT(*) FROM capture_lock`).Scan(&held)
	if held != 0 {
		t.Fatal("batch left capture held off")
	}
	// j13xs0: the refused and dry batches above recorded nothing; the real
	// one left one trail row per memory, with where it moved from and to.
	fs, err := app.Fixes(ctx, f.App, time.Time{})
	if err != nil || len(fs) != 2 {
		t.Fatalf("fix trail after one batch of 2: %+v %v", fs, err)
	}
	from := map[string]string{shared.SID: shared.Scope, jg.SID: jg.Scope}
	for _, x := range fs {
		if x.Action != "rescope" || x.Mode != "batch" || x.OldScope != from[x.SID] || x.NewScope != target || x.At.IsZero() {
			t.Fatalf("trail row %+v", x)
		}
	}
	// The single form leaves one row too.
	named := f.Units["Named"]
	if _, err := global.Rescope(ctx, f.App, named.SID, target, false); err != nil {
		t.Fatal(err)
	}
	if fs, _ = app.Fixes(ctx, f.App, time.Time{}); len(fs) != 3 || fs[2].SID != named.SID || fs[2].Mode != "single" || fs[2].OldScope != named.Scope || fs[2].NewScope != target {
		t.Fatalf("single rescope trail: %+v", fs)
	}
	scopetest.Pass(t, "rescope.batch", "global")
}
