package run

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/veridit/hug/muninn/internal/ingest"
)

// 5sc5rw check 2: forgetting a memory a sync once copied (an imp_ twin with
// origin repo:PROJECT/ID, as every repo-project capture had on 2026-11-17)
// leaves nothing of it searchable. Before the fix, forget xgxqs6 left its
// twin 49ckwr active, and search returned the twin first.
func TestForgetLeavesNoTwinSearchable(t *testing.T) {
	home := memHome(t)
	ids := rememberN(t, 1)
	store := filepath.Join(home, "data", "memories.json")
	mems, err := ingest.LoadNative(store)
	if err != nil || len(mems) != 1 {
		t.Fatalf("%v %+v", err, mems)
	}
	m := mems[0]
	old := time.Date(2026, 11, 16, 10, 0, 0, 0, time.UTC)
	if _, err := ingest.MergeImport(store, []ingest.Imported{{Origin: "repo:rc-demo/" + m.ID, Content: m.Content, Category: m.Category, Scope: m.Scope, Active: true, Created: old, Updated: old}}, false); err != nil {
		t.Fatal(err)
	}
	search := func() []string {
		out, errOut, code := runCLI(t, "", "search", "me51r2 quartz", "--json")
		if code != 0 {
			t.Fatalf("search: %s%s", out, errOut)
		}
		var r memoriesOut
		if err := json.Unmarshal([]byte(out), &r); err != nil {
			t.Fatalf("%v %s", err, out)
		}
		var got []string
		for _, x := range r.Memories {
			if strings.Contains(x.Text, "me51r2 memory number 0") {
				got = append(got, x.ID)
			}
		}
		return got
	}
	// Search shows one of two identical texts; the twin is in the store.
	if got := search(); len(got) != 1 {
		t.Fatalf("fixture: want the memory found, got %v", got)
	}
	if mems, _ = ingest.LoadNative(store); len(mems) != 2 {
		t.Fatalf("fixture: want the memory and its twin in the store, got %d", len(mems))
	}
	if out, errOut, code := runCLI(t, "", "forget", ids[0]); code != 0 {
		t.Fatalf("forget: %s%s", out, errOut)
	}
	if got := search(); len(got) != 0 {
		t.Fatalf("after forget %s, search still finds %v", ids[0], got)
	}
}

// 5sc5rw check 3: hug mem upgrade previews the fold (exit 5, nothing
// written), --apply folds with a backup and reindexes, and a second run has
// nothing to do. The memory keeps its own id and stays found once.
func TestMemUpgradeFoldsTwins(t *testing.T) {
	home := memHome(t)
	ids := rememberN(t, 2)
	store := filepath.Join(home, "data", "memories.json")
	mems, _ := ingest.LoadNative(store)
	var items []ingest.Imported
	old := time.Date(2026, 11, 16, 10, 0, 0, 0, time.UTC)
	for _, m := range mems {
		items = append(items, ingest.Imported{Origin: "repo:rc-demo/" + m.ID, Content: m.Content, Category: m.Category, Scope: m.Scope, Active: true, Created: old, Updated: old})
	}
	if _, err := ingest.MergeImport(store, items, false); err != nil {
		t.Fatal(err)
	}
	before, _ := os.ReadFile(store)
	out, errOut, code := runCLI(t, "", "upgrade")
	if code != ExitUpgradePending || !strings.Contains(out, "memory: would fold 2 copies of this machine's own memories in "+mems[0].Scope+"\n") || !strings.Contains(out, "fix: hug mem upgrade --apply") {
		t.Fatalf("preview: exit %d\n%s%s", code, out, errOut)
	}
	if after, _ := os.ReadFile(store); string(after) != string(before) {
		t.Fatal("the preview wrote the store")
	}
	out, errOut, code = runCLI(t, "", "upgrade", "--apply")
	if code != 0 || !strings.Contains(out, "memory: folded 2 copies") || !strings.Contains(out, "memory: backup memories.json.bak-upgrade-") {
		t.Fatalf("apply: exit %d\n%s%s", code, out, errOut)
	}
	if mems, _ = ingest.LoadNative(store); len(mems) != 2 {
		t.Fatalf("%d memories after the fold, want 2", len(mems))
	}
	for _, m := range mems {
		if m.SID != ids[0] && m.SID != ids[1] {
			t.Fatalf("the fold kept %s, not the memory written here (%v)", m.SID, ids)
		}
	}
	if out, _, code = runCLI(t, "", "upgrade"); code != 0 || strings.TrimSpace(strings.SplitN(out, "\n", 2)[0]) != "memory: nothing to do" {
		t.Fatalf("second run: exit %d\n%s", code, out)
	}
	if backups, _ := filepath.Glob(store + ".bak-upgrade-*"); len(backups) != 1 {
		t.Fatalf("backups %v", backups)
	}
}
