package run

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/veridit/hug/muninn/internal/app"
	"github.com/veridit/hug/muninn/internal/config"
	"github.com/veridit/hug/muninn/internal/ingest"
)

// boardToml commits hug.toml on hug/tasks with the given content.
func boardToml(t *testing.T, dir, content string) {
	t.Helper()
	wt := filepath.Join(t.TempDir(), "tasks")
	env := append(os.Environ(), "GIT_TERMINAL_PROMPT=0", "GIT_AUTHOR_NAME=Synthetic", "GIT_AUTHOR_EMAIL=synthetic@example.invalid", "GIT_COMMITTER_NAME=Synthetic", "GIT_COMMITTER_EMAIL=synthetic@example.invalid")
	run := func(args ...string) {
		cmd := exec.Command("git", args...)
		cmd.Env = env
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v %s", args, err, out)
		}
	}
	run("-C", dir, "worktree", "add", "-q", "--orphan", "-b", "hug/tasks", wt)
	if err := os.WriteFile(filepath.Join(wt, "hug.toml"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	run("-C", wt, "add", "hug.toml")
	run("-C", wt, "commit", "-q", "-m", "board settings")
	run("-C", dir, "worktree", "remove", "--force", wt)
}

// rnpdnm checks 1 and 4 through hug mem sync: the first sync seeds the
// term list into the local config (names from the store's other places and
// hug.toml [people]), says so naming the file and never the terms, holds
// what the rule holds, prints one line per reason and the read-only list
// command, and the memory branch carries neither the held lines nor any
// term. --held lists them without writing; --release lets one push.
func TestCLISyncGateHoldsAndReports(t *testing.T) {
	home := memHome(t)
	p := gitProject(t, home)
	boardToml(t, home, "[memory]\nshare = true\n\n[people.zed]\nname = \"Zøe Quorra\"\n")
	native := app.NativePath()
	put := func(text, cat, scope string) {
		if _, _, err := ingest.Remember(native, text, cat, scope, nil); err != nil {
			t.Fatal(err)
		}
	}
	// Another client's memory: its name is a term the seed takes from the
	// store, not from any repo.
	put("Synthetic client fact about invoices", "fact", "client:plumtree")
	put("Synthetic project fact that pushes", "fact", p.Scope())
	put("Synthetic: Zoe's rule, confirm on the board", "decision", p.Scope())
	put("Synthetic plumtree invoices use the old API", "fact", p.Scope())
	put("Synthetic scratch file in ~/notes.md", "fact", p.Scope())
	put("Synthetic preference: short replies", "preference", p.Scope())

	out, errOut, code := runCLI(t, "", "sync")
	if code != 0 {
		t.Fatalf("sync: %d %s%s", code, out, errOut)
	}
	if !strings.Contains(out, "placement gate: seeded ") || !strings.Contains(out, config.ConfigPath()) {
		t.Fatalf("no seed line naming the config: %s", out)
	}
	for _, w := range []string{"held 1: names a home folder\n", "held 2: names another project, client or person\n", "held 1: is a preference or correction\n", "fix: hug mem sync --held\n"} {
		if !strings.Contains(out, w) {
			t.Errorf("sync output lacks %q:\n%s", w, out)
		}
	}
	if steersToRelease(out) {
		t.Errorf("sync output steers to a release: %s", out)
	}
	// The terms are in the local config only.
	cfg, err := config.Load()
	if err != nil || cfg.SyncGate == nil {
		t.Fatalf("config: %v %+v", err, cfg.SyncGate)
	}
	terms := strings.Join(cfg.SyncGate.Terms, "|")
	for _, w := range []string{"plumtree", "zed", "Zøe Quorra", "Zøe"} {
		if !strings.Contains("|"+terms+"|", "|"+w+"|") {
			t.Errorf("seed lacks %q: %s", w, terms)
		}
	}
	if strings.Contains(out, "plumtree") || strings.Contains(out, "Quorra") {
		t.Errorf("sync output prints the terms: %s", out)
	}
	// Nothing the gate holds, and no term, is on the branch: neither its
	// file nor any commit.
	log, err := exec.Command("git", "-C", home, "log", "-p", "--all", "--", ".").CombinedOutput()
	if err != nil {
		t.Fatalf("git log: %v %s", err, log)
	}
	branch, _ := exec.Command("git", "-C", home, "log", "-p", "hug/memory").CombinedOutput()
	if !strings.Contains(string(branch), "Synthetic project fact that pushes") {
		t.Fatalf("the passing line is not on hug/memory: %s", branch)
	}
	for _, w := range []string{"plumtree", "Zoe", "Zøe", "Quorra", "~/notes", "short replies", "sync_gate"} {
		if strings.Contains(string(branch), w) {
			t.Errorf("hug/memory carries %q", w)
		}
	}
	// --held: read-only, lists each with its reason and term, names the
	// release as a sentence for a person, not a fix: or next: line.
	cfgBefore, _ := os.ReadFile(config.ConfigPath())
	storeBefore, _ := os.ReadFile(native)
	out, errOut, code = runCLI(t, "", "sync", "--held")
	if code != 0 {
		t.Fatalf("--held: %d %s%s", code, out, errOut)
	}
	for _, w := range []string{"4 project memories stay on this machine", "names another project, client or person (plumtree)", "names a home folder", "A person who has read a held memory"} {
		if !strings.Contains(out, w) {
			t.Errorf("--held lacks %q:\n%s", w, out)
		}
	}
	if steersToRelease(out) || strings.Contains(out, "fix:") {
		t.Errorf("--held has a fix: line or a release next: line: %s", out)
	}
	if b, _ := os.ReadFile(config.ConfigPath()); string(b) != string(cfgBefore) {
		t.Error("--held wrote the config")
	}
	if b, _ := os.ReadFile(native); string(b) != string(storeBefore) {
		t.Error("--held wrote the store")
	}
	// --release ID: one local exception; the next sync pushes it.
	sid, _ := storedSID(t, "Synthetic plumtree invoices use the old API")
	if out, errOut, code = runCLI(t, "", "sync", "--release", sid); code != 0 || !strings.Contains(out, "released "+sid) {
		t.Fatalf("--release: %d %s%s", code, out, errOut)
	}
	if out, errOut, code = runCLI(t, "", "sync"); code != 0 || !strings.Contains(out, "held 1: names another project, client or person\n") {
		t.Fatalf("sync after release: %d %s%s", code, out, errOut)
	}
	raw, _ := os.ReadFile(p.File())
	if !strings.Contains(string(raw), "plumtree invoices") {
		t.Fatalf("released line not pushed: %s", raw)
	}
	var c map[string]any
	b, _ := os.ReadFile(config.ConfigPath())
	_ = json.Unmarshal(b, &c)
	if c["sync_gate"] == nil {
		t.Fatalf("config lost sync_gate: %s", b)
	}
}

// Crab point 3: a line pushed before the gate existed, held now, drops from
// the tip, and sync says once that the remote's history keeps it.
func TestCLISyncSaysHeldLinesStayInHistory(t *testing.T) {
	home := memHome(t)
	p := gitProject(t, home)
	boardToml(t, home, "[memory]\nshare = true\n")
	native := app.NativePath()
	text := "Synthetic: the scratch file lives in ~/notes.md"
	if _, _, err := ingest.Remember(native, text, "fact", p.Scope(), nil); err != nil {
		t.Fatal(err)
	}
	// Pushed before the gate: a sync with an empty gate config that lets
	// it through (as an older hug would).
	if err := config.Save(func() config.Config {
		c, _ := config.Load()
		c.SyncGate = &config.SyncGate{Terms: []string{}, Release: []string{mustID(t, text)}}
		return c
	}()); err != nil {
		t.Fatal(err)
	}
	if out, errOut, code := runCLI(t, "", "sync"); code != 0 {
		t.Fatalf("first sync: %d %s%s", code, out, errOut)
	}
	if raw, _ := os.ReadFile(p.File()); !strings.Contains(string(raw), "~/notes.md") {
		t.Fatalf("fixture: line not pushed: %s", raw)
	}
	// The exception is withdrawn: the gate holds it now.
	c, _ := config.Load()
	c.SyncGate.Release = nil
	if err := config.Save(c); err != nil {
		t.Fatal(err)
	}
	out, errOut, code := runCLI(t, "", "sync")
	if code != 0 {
		t.Fatalf("sync: %d %s%s", code, out, errOut)
	}
	if !strings.Contains(out, "held 1 lines that are already on the remote's history: dropping them from the tip does not unpublish them\n") {
		t.Fatalf("no history note: %s", out)
	}
	if raw, _ := os.ReadFile(p.File()); strings.Contains(string(raw), "~/notes.md") {
		t.Fatalf("held line still in the tip: %s", raw)
	}
	out, _, _ = runCLI(t, "", "sync", "--held")
	if !strings.Contains(out, "[already in the remote's history]") {
		t.Fatalf("--held does not mark it: %s", out)
	}
	// The next sync no longer counts it as in the file, but --held still
	// knows it was pushed once.
	out, _, _ = runCLI(t, "", "sync")
	if strings.Contains(out, "already on the remote's history") {
		t.Fatalf("history note repeated on every sync: %s", out)
	}
	out, _, _ = runCLI(t, "", "sync", "--held")
	if !strings.Contains(out, "[already in the remote's history]") {
		t.Fatalf("--held forgot it was pushed: %s", out)
	}
}

// steersToRelease: a fix: or next: line an agent would run that releases
// (crab, e2582a0f: an agent runs those as given, and a release publishes).
func steersToRelease(out string) bool {
	for _, l := range strings.Split(out, "\n") {
		if (strings.HasPrefix(l, "fix:") || strings.HasPrefix(l, "next:")) && strings.Contains(l, "--release") {
			return true
		}
	}
	return false
}

func mustID(t *testing.T, text string) string {
	t.Helper()
	mems, err := ingest.LoadNative(app.NativePath())
	if err != nil {
		t.Fatal(err)
	}
	for _, m := range mems {
		if m.Content == text {
			return m.ID
		}
	}
	t.Fatalf("%q not stored", text)
	return ""
}
