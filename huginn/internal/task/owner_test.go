package task

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// v3k6cp check 3: a doing task whose assignee has had no live session for
// over 24 h is reported, only when the session list is readable.
func TestOwnerFindings(t *testing.T) {
	dir := t.TempDir()
	now := time.Date(2026, 11, 17, 10, 0, 0, 0, time.UTC)
	touch := func(name string, at time.Time) {
		p := filepath.Join(dir, name)
		if err := os.WriteFile(p, []byte("{}"), 0o600); err != nil {
			t.Fatal(err)
		}
		if err := os.Chtimes(p, at, at); err != nil {
			t.Fatal(err)
		}
	}
	touch("session_humpback_1793_aa.json", now.Add(-40*time.Hour))
	touch("session_humpback_1790_bb.json", now.Add(-60*time.Hour))
	touch("session_wyvern_1794_cc.json", now.Add(-time.Hour))
	touch("session_vole_1794_dd.json", now.Add(-30*time.Hour))
	touch("session_vole_1795_ee.json", now.Add(-2*time.Hour)) // a newer vole is live
	touch("session_x.bak", now.Add(-90*time.Hour))
	live := LivenessFrom(dir)
	ts := []Task{
		{ID: "44w1ev", State: "doing", Assignee: []string{"humpback"}, Path: "a.md"},
		{ID: "zhvkmj", State: "doing", Assignee: []string{"wyvern"}},
		{ID: "kx2ndm", State: "doing", Assignee: []string{"vole"}},
		{ID: "t7wetj", State: "doing", Assignee: []string{"Jørgen"}},          // not a session name: never judged
		{ID: "p39w4a", State: "doing", Assignee: []string{"claude-opus-5-5"}}, // a model
		{ID: "qawmnb", State: "done", Assignee: []string{"humpback"}},
		{ID: "e3vw30", State: "doing", Assignee: []string{"humpback", "wyvern"}}, // one live owner is enough
	}
	fs := OwnerFindings(live, ts, now)
	if len(fs) != 1 {
		t.Fatalf("findings %+v", fs)
	}
	f := fs[0]
	if f.Code != "stale-owner" || f.Fix != "hug task set 44w1ev --assignee NAME" || f.Path != "a.md" ||
		!strings.HasPrefix(f.Problem, "owner humpback of 44w1ev has had no live session since "+now.Add(-40*time.Hour).Local().Format("2006-01-02 15:04")) {
		t.Fatalf("%+v", f)
	}
	if FindingCode(f.Problem) != "stale-owner" {
		t.Fatal(FindingCode(f.Problem))
	}
	// No readable session list: nothing is said.
	if fs := OwnerFindings(LivenessFrom(filepath.Join(dir, "missing")), ts, now); fs != nil {
		t.Fatal(fs)
	}
}
