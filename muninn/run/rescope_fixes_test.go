package run

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/veridit/hug/muninn/internal/scopetest"
)

// j13xs0: two batch rescopes (2026-11-18, 58 and 9 memories) left no row in
// memory_fixes, so who moved what had to be traced through session journals.
// Every rescope, single or --from a batch, now leaves one row per memory with
// the old and new scope; a dry run leaves none; hug mem stats --fixes shows them.
func TestRescopeWritesFixTrail(t *testing.T) {
	f := scopetest.New(t)
	f.Bind(t, "global")
	t.Chdir(f.Cwd)
	t.Setenv("JCODE_SESSION_ID", "")
	shared, jg, named := f.Units["Shared"], f.Units["JcodeGlobal"], f.Units["Named"]
	target := "project:matrix-project"
	rescopes := func() int {
		t.Helper()
		var n int
		if err := f.App.DB.QueryRow(`SELECT COUNT(*) FROM memory_fixes WHERE action='rescope'`).Scan(&n); err != nil {
			t.Fatal(err)
		}
		return n
	}
	moves := filepath.Join(t.TempDir(), "moves.tsv")
	if err := os.WriteFile(moves, []byte("# two moves\n"+shared.SID+" "+target+"\n"+jg.SID+" "+target+"\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if out, err := cliSurface(t, rescopeCmd(), "--from", moves, "--dry-run"); err != nil {
		t.Fatalf("dry run: %s", out)
	}
	if n := rescopes(); n != 0 {
		t.Fatalf("a dry run wrote %d memory_fixes rows", n)
	}
	if out, err := cliSurface(t, rescopeCmd(), "--from", moves); err != nil {
		t.Fatalf("batch: %s", out)
	}
	if n := rescopes(); n != 2 {
		t.Fatalf("batch of 2 wrote %d rescope rows in memory_fixes, want 2", n)
	}
	if out, err := cliSurface(t, rescopeCmd(), named.SID, "--to", target); err != nil {
		t.Fatalf("single: %s", out)
	}
	if n := rescopes(); n != 3 {
		t.Fatalf("after a single rescope: %d rescope rows, want 3", n)
	}

	out, err := cliSurface(t, statsCmd(), "--fixes", "--json")
	if err != nil {
		t.Fatalf("stats --fixes: %s", out)
	}
	var s struct {
		Days   []map[string]any `json:"days"`
		Recent []map[string]any `json:"recent"`
	}
	if err := json.Unmarshal([]byte(out), &s); err != nil {
		t.Fatalf("%v: %s", err, out)
	}
	today := time.Now().UTC().Format("2006-01-02")
	if len(s.Days) != 1 || s.Days[0]["day"] != today || s.Days[0]["rescope"] != float64(3) || s.Days[0]["fixes"] != float64(3) {
		t.Fatalf("days %+v", s.Days)
	}
	want := []struct{ sid, from, mode string }{{shared.SID, shared.Scope, "batch"}, {jg.SID, jg.Scope, "batch"}, {named.SID, named.Scope, "single"}}
	if len(s.Recent) != len(want) {
		t.Fatalf("recent %+v", s.Recent)
	}
	got := map[string]map[string]any{}
	for _, r := range s.Recent {
		got[r["sid"].(string)] = r
	}
	for _, w := range want {
		r := got[w.sid]
		if r == nil || r["action"] != "rescope" || r["old_scope"] != w.from || r["new_scope"] != target || r["mode"] != w.mode || r["surface"] != "cli" || r["at"] == "" {
			t.Fatalf("row for %s: %+v", w.sid, r)
		}
	}

	out, err = cliSurface(t, statsCmd(), "--fixes")
	if err != nil {
		t.Fatalf("stats --fixes: %s", out)
	}
	for _, line := range []string{"rescope", "rescope " + shared.SID + " " + shared.Scope + " -> " + target + " (batch)", "rescope " + named.SID + " " + named.Scope + " -> " + target + " (single)"} {
		if !strings.Contains(out, line) {
			t.Fatalf("text lacks %q:\n%s", line, out)
		}
	}
}
