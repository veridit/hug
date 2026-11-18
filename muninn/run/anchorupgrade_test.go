package run

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/veridit/hug/muninn/internal/anchor"
	"github.com/veridit/hug/muninn/internal/config"
	"github.com/veridit/hug/muninn/internal/store"
)

// The anchor backfill (n7jbmx) is one pass over every memory, 2.2 s on 30k.
// hug mem upgrade runs it (preview says so, --apply does it), so it rarely
// lands in a hook, and wherever it runs it leaves one hook.log line (crab
// 05:24).
func TestMemUpgradeRecordsAnchors(t *testing.T) {
	home := memHome(t)
	rememberN(t, 3)
	db, err := store.Open(config.DBPath())
	if err != nil {
		t.Fatal(err)
	}
	// A store indexed by a hug from before anchors: no recipe, no rows.
	db.Exec(`DELETE FROM unit_anchors`)
	db.SetMeta("anchor_recipe", "")
	db.Close()
	out, errOut, code := runCLI(t, "", "upgrade")
	if code != ExitUpgradePending || !strings.Contains(out, "memory: would record the paths and commands memories name (tool-call recall)") || !strings.Contains(out, "fix: hug mem upgrade --apply") {
		t.Fatalf("preview: exit %d\n%s%s", code, out, errOut)
	}
	recipe := func() string {
		db, err := store.Open(config.DBPath())
		if err != nil {
			t.Fatal(err)
		}
		defer db.Close()
		return db.GetMeta("anchor_recipe")
	}
	if r := recipe(); r != "" {
		t.Fatalf("the preview ran the backfill (recipe %q)", r)
	}
	out, errOut, code = runCLI(t, "", "upgrade", "--apply")
	if code != 0 || !strings.Contains(out, "memory: recorded the paths and commands of 3 memories (tool-call recall)") {
		t.Fatalf("apply: exit %d\n%s%s", code, out, errOut)
	}
	if r := recipe(); r != anchor.Recipe {
		t.Fatalf("recipe after --apply %q, want %q", r, anchor.Recipe)
	}
	if out, _, code = runCLI(t, "", "upgrade"); code != 0 || !strings.Contains(out, "memory: nothing to do") {
		t.Fatalf("second run: exit %d\n%s", code, out)
	}
	log, _ := os.ReadFile(filepath.Join(home, "data", "hook.log"))
	if n := strings.Count(string(log), "anchors: recorded for 3 memories"); n != 1 {
		t.Fatalf("hook.log has %d backfill lines, want 1:\n%s", n, log)
	}
}

// hug mem index does it too, and says so in its stats line.
func TestMemIndexRecordsAnchors(t *testing.T) {
	memHome(t)
	rememberN(t, 2)
	db, err := store.Open(config.DBPath())
	if err != nil {
		t.Fatal(err)
	}
	db.SetMeta("anchor_recipe", "")
	db.Close()
	out, errOut, code := runCLI(t, "", "index")
	if code != 0 || !strings.Contains(out, "anchors recorded for 2 memories (once)") {
		t.Fatalf("index: exit %d\n%s%s", code, out, errOut)
	}
	if out, _, _ = runCLI(t, "", "index"); strings.Contains(out, "anchors recorded") {
		t.Fatalf("second index recorded again:\n%s", out)
	}
}
