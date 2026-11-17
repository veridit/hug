package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/veridit/hug/huginn/internal/task"
)

// commit runs git commit in dir with message, returning its output and
// whether it went through.
func tryCommit(t *testing.T, dir string, args ...string) (string, bool) {
	t.Helper()
	c := exec.Command("git", append([]string{"-C", dir, "commit", "--allow-empty"}, args...)...)
	out, e := c.CombinedOutput()
	return string(out), e == nil
}

// t709he check 1: hug task install commit-msg refuses a commit without one
// Model: trailer, with a fix: line whose slot is refused as pasted.
func TestCommitMsgGate(t *testing.T) {
	t.Parallel()
	a := app(t)
	dir := realDir(t, a.Cwd)
	r := execute(t, a, "install", "commit-msg")
	hook := filepath.Join(dir, ".git", "hooks", "commit-msg")
	if r["hook"] != hook || !isOurs(hook) {
		t.Fatalf("install: %v", r)
	}

	// A real refusal leaves the staged change in place: the fix replays it.
	if e := os.WriteFile(filepath.Join(dir, "work.txt"), []byte("work\n"), 0o644); e != nil {
		t.Fatal(e)
	}
	git(t, dir, "add", "work.txt")
	out, ok := tryCommit(t, dir, "-m", "no trailer")
	if ok || !strings.Contains(out, "commit refused: no Model: trailer") || !strings.Contains(out, "\nfix: git -c trailer.ifexists=replace commit --cleanup=strip -F '") || !strings.Contains(out, "--trailer 'Model: <model-id>/<effort>'") {
		t.Fatalf("no trailer:\n%s", out)
	}
	if strings.Contains(out, "Agent:") {
		t.Fatalf("mentions Agent:\n%s", out)
	}
	// The fix line run as printed is refused: the slot was not replaced.
	fix := out[strings.Index(out, "fix: ")+5:]
	fix = strings.TrimSpace(fix[:strings.Index(fix, "\n")+1])
	sh := exec.Command("sh", "-c", fix)
	sh.Dir = dir
	pasted, e := sh.CombinedOutput()
	if e == nil || !strings.Contains(string(pasted), "names no model (replace the <...> slot)") {
		t.Fatalf("pasted fix %q: %v\n%s", fix, e, pasted)
	}
	// The same fix with the slot filled in goes through.
	sh = exec.Command("sh", "-c", strings.Replace(fix, "<model-id>/<effort>", "gpt-6.1-sol/high", 1))
	sh.Dir = dir
	if b, e := sh.CombinedOutput(); e != nil {
		t.Fatalf("filled fix: %v\n%s", e, b)
	}
	if got := git(t, dir, "log", "-1", "--format=%s|%(trailers:key=Model,valueonly)"); got != "no trailer|gpt-6.1-sol/high" {
		t.Fatalf("after the fix: %q", got)
	}

	for _, c := range []struct {
		name string
		args []string
		ok   bool
		says string
	}{
		{"model", []string{"-m", "work", "-m", "Model: claude-opus-5-5/high"}, true, ""},
		// jr55s8: without a [models] roster, any effort word goes, and a typo
		// or a missing effort is refused with a fix line.
		{"no effort", []string{"-m", "work", "-m", "Model: claude-opus-5-5"}, false, "names no effort: Model: claude-opus-5-5/<effort>"},
		{"no effort fix", []string{"-m", "work", "-m", "Model: claude-opus-5-5"}, false, "--trailer 'Model: claude-opus-5-5/<effort>'"},
		{"typo", []string{"-m", "work", "-m", "Model: claude-opus-5-5/hihg"}, false, "effort hihg is not an effort word (none, minimal, low, medium, high, xhigh, max)"},
		{"xhigh", []string{"-m", "work", "-m", "Model: gpt-6.1-sol/xhigh"}, true, ""},
		{"human", []string{"-m", "my own", "-m", "Model: human"}, true, ""},
		{"two", []string{"-m", "two", "-m", "Model: a/low\nModel: b/low"}, false, "2 Model: trailers, want exactly one"},
		{"empty", []string{"-m", "empty", "-m", "Model:"}, false, "names no model"},
		{"in body only", []string{"-m", "subject", "-m", "Model: x is mentioned here in prose\n\nbody ends"}, false, "no Model: trailer"},
		{"fixup", []string{"-m", "fixup! work"}, true, ""},
		{"no-verify", []string{"--no-verify", "-m", "emergency"}, true, ""},
	} {
		out, ok := tryCommit(t, dir, c.args...)
		if ok != c.ok || !strings.Contains(out, c.says) {
			t.Errorf("%s: ok=%v\n%s", c.name, ok, out)
		}
	}

	// hug's own branches are exempt (hug adds Model: itself when known).
	git(t, dir, "checkout", "-q", "-b", "muninn-notes")
	if out, ok := tryCommit(t, dir, "-m", "hug mem sync"); !ok {
		t.Fatalf("muninn-notes refused:\n%s", out)
	}
	git(t, dir, "checkout", "-q", "main")

	// A rebase carries commits already made.
	git(t, dir, "checkout", "-q", "-b", "side")
	tryCommit(t, dir, "--no-verify", "-m", "old commit without a trailer")
	git(t, dir, "checkout", "-q", "main")
	if _, ok := tryCommit(t, dir, "-m", "main moves", "-m", "Model: human"); !ok {
		t.Fatal("main commit refused")
	}
	git(t, dir, "checkout", "-q", "side")
	if b, e := exec.Command("git", "-C", dir, "rebase", "-q", "--force-rebase", "main").CombinedOutput(); e != nil {
		t.Fatalf("rebase refused: %v\n%s", e, b)
	}
	git(t, dir, "checkout", "-q", "main")

	// Cheap: the gate starts three git processes and one awk per commit, no
	// more. Counted, not timed (Jørgen 2026-11-17 10:00: no wall-clock
	// budgets in tests; the 50 ms limit failed under load at 50.7 ms).
	if runtime.GOOS != "windows" {
		if got := gateProcesses(t, dir); got != "awk=1 git=3" {
			t.Errorf("the gate starts %s per commit, want awk=1 git=3", got)
		}
	}

	r = execute(t, a, "uninstall", "commit-msg")
	if _, e := os.Stat(hook); !os.IsNotExist(e) {
		t.Fatalf("uninstall left %s: %v", hook, r)
	}
	if _, ok := tryCommit(t, dir, "-m", "gate gone"); !ok {
		t.Fatal("commit refused after uninstall")
	}
}

// gateProcesses runs the hook once on a message with a Model: trailer, with
// git and awk on PATH replaced by wrappers that log their name and run the
// real one, and returns how many of each it started ("awk=1 git=3"). POSIX
// sh wrappers: not on Windows, where the hook runs under git's own sh.
func gateProcesses(t *testing.T, dir string) string {
	t.Helper()
	bin, logFile := t.TempDir(), filepath.Join(t.TempDir(), "calls")
	for _, name := range []string{"git", "awk"} {
		real, e := exec.LookPath(name)
		if e != nil {
			t.Fatal(e)
		}
		script := "#!/bin/sh\necho " + name + " >> '" + logFile + "'\nexec '" + real + "' \"$@\"\n"
		if e := os.WriteFile(filepath.Join(bin, name), []byte(script), 0o755); e != nil {
			t.Fatal(e)
		}
	}
	msg := filepath.Join(t.TempDir(), "MSG")
	if e := os.WriteFile(msg, []byte("subject\n\nModel: human\n"), 0o644); e != nil {
		t.Fatal(e)
	}
	c := exec.Command(filepath.Join(dir, ".git", "hooks", "commit-msg"), msg)
	c.Dir = dir
	c.Env = append(os.Environ(), "PATH="+bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	if out, e := c.CombinedOutput(); e != nil {
		t.Fatalf("hook refused a good message: %v\n%s", e, out)
	}
	b, _ := os.ReadFile(logFile)
	n := map[string]int{}
	for _, l := range strings.Fields(string(b)) {
		n[l]++
	}
	return fmt.Sprintf("awk=%d git=%d", n["awk"], n["git"])
}

// A commit-msg hook already in .git/hooks keeps running, first; uninstall
// puts it back.
func TestCommitMsgGateChainsExistingHook(t *testing.T) {
	t.Parallel()
	a := app(t)
	dir := realDir(t, a.Cwd)
	hook := filepath.Join(dir, ".git", "hooks", "commit-msg")
	own := "#!/bin/sh\ngrep -q 'refs: ' \"$1\" || { echo 'own hook: no refs line' >&2; exit 1; }\n"
	if e := os.WriteFile(hook, []byte(own), 0o755); e != nil {
		t.Fatal(e)
	}
	r := execute(t, a, "install", "commit-msg")
	if r["kept"] != hook+".pre-hug" {
		t.Fatalf("install: %v", r)
	}
	if out, ok := tryCommit(t, dir, "-m", "x", "-m", "Model: human"); ok || !strings.Contains(out, "own hook: no refs line") {
		t.Fatalf("own hook did not run:\n%s", out)
	}
	if out, ok := tryCommit(t, dir, "-m", "x", "-m", "refs: none"); ok || !strings.Contains(out, "no Model: trailer") {
		t.Fatalf("gate did not run after the own hook:\n%s", out)
	}
	if out, ok := tryCommit(t, dir, "-m", "x", "-m", "refs: none\nModel: human"); !ok {
		t.Fatalf("both pass, still refused:\n%s", out)
	}
	execute(t, a, "install", "commit-msg") // again: idempotent, keeps the chain
	if b, _ := os.ReadFile(hook + ".pre-hug"); string(b) != own {
		t.Fatalf("second install changed the kept hook: %q", b)
	}
	execute(t, a, "uninstall", "commit-msg")
	if b, _ := os.ReadFile(hook); string(b) != own {
		t.Fatalf("uninstall did not restore the own hook: %q", b)
	}
}

// core.hooksPath (statbus, seid): the tracked hook is never edited; setup
// shows the two lines to add and lists the step open until they are there.
func TestCommitMsgGateWithHooksPath(t *testing.T) {
	t.Parallel()
	a := app(t)
	dir := realDir(t, a.Cwd)
	tracked := filepath.Join(dir, ".githooks", "commit-msg")
	if e := os.MkdirAll(filepath.Dir(tracked), 0o755); e != nil {
		t.Fatal(e)
	}
	own := "#!/bin/bash\nset -euo pipefail\nexit 0\n"
	if e := os.WriteFile(tracked, []byte(own), 0o755); e != nil {
		t.Fatal(e)
	}
	git(t, dir, "config", "core.hooksPath", ".githooks")
	r := execute(t, a, "install", "commit-msg")
	if b, _ := os.ReadFile(tracked); string(b) != own {
		t.Fatalf("tracked hook edited: %q", b)
	}
	if !strings.Contains(r["message"].(string), "add these lines to "+tracked) {
		t.Fatalf("install message: %v", r)
	}
	step := func() SetupStep {
		for _, s := range execute(t, a, "setup")["setup"].([]SetupStep) {
			if s.Step == "Model: trailer gate" {
				return s
			}
		}
		t.Fatal("no gate step in setup")
		return SetupStep{}
	}
	s := step()
	if s.Done || !strings.Contains(s.Do, tailLines) {
		t.Fatalf("setup before wiring: %+v", s)
	}
	if _, ok := tryCommit(t, dir, "-m", "not wired yet"); !ok {
		t.Fatal("refused before the lines were added")
	}
	// The project adds the lines after its first line, as the step says.
	wired := strings.Replace(own, "\n", "\n"+tailLines+"\n", 1)
	if e := os.WriteFile(tracked, []byte(wired), 0o755); e != nil {
		t.Fatal(e)
	}
	if s = step(); !s.Done {
		t.Fatalf("setup after wiring: %+v", s)
	}
	if out, ok := tryCommit(t, dir, "-m", "wired"); ok || !strings.Contains(out, "no Model: trailer") {
		t.Fatalf("wired gate did not refuse:\n%s", out)
	}
	if out, ok := tryCommit(t, dir, "-m", "wired", "-m", "Model: human"); !ok {
		t.Fatalf("wired gate refused a good commit:\n%s", out)
	}
	// In a clone without hug the lines do nothing.
	execute(t, a, "uninstall", "commit-msg")
	if out, ok := tryCommit(t, dir, "-m", "no hug here"); !ok {
		t.Fatalf("lines without hug refused:\n%s", out)
	}
}

// realDir is dir as git reports it: t.TempDir is /var/... on macOS, a
// symlink to /private/var/..., and on Windows may use a short name
// (RUNNER~1); git rev-parse --path-format=absolute gives the resolved path
// (CI run 44049449185: the hook path compare failed on both).
func realDir(t *testing.T, dir string) string {
	t.Helper()
	d, e := filepath.EvalSymlinks(dir)
	if e != nil {
		t.Fatal(e)
	}
	return d
}

// jr55s8 check 1: with a [models] roster on the board branch, the effort is
// checked per model: hug task models add writes the line and commits it on
// hug/tasks, and the gate reads it from there.
func TestCommitMsgGateChecksModelRoster(t *testing.T) {
	t.Parallel()
	a := app(t)
	dir := realDir(t, a.Cwd)
	execute(t, a, "install", "commit-msg")
	r := execute(t, a, "models", "add", "claude-opus-5-5", "low", "medium", "high", "max")
	if r["changed"] != true {
		t.Fatalf("models add: %v", r)
	}
	if got := git(t, dir, "cat-file", "-p", "hug/tasks:hug.toml"); !strings.Contains(got, "[models]\n\"claude-opus-5-5\" = [\"low\", \"medium\", \"high\", \"max\"]") {
		t.Fatalf("hug.toml on hug/tasks:\n%s", got)
	}
	for _, c := range []struct {
		name, trailer string
		ok            bool
		says          string
	}{
		{"listed", "Model: claude-opus-5-5/high", true, ""},
		{"human", "Model: human", true, ""},
		{"not its level", "Model: claude-opus-5-5/xhigh", false, "effort xhigh is not one of claude-opus-5-5's levels in hug.toml [models] (low, medium, high, max)"},
		{"no effort", "Model: claude-opus-5-5", false, "names no effort"},
		{"unknown model", "Model: gpt-6.1-sol/high", false, "model gpt-6.1-sol is not in hug.toml [models]"},
		{"unknown model fix", "Model: gpt-6.1-sol/high", false, "fix: hug task models add gpt-6.1-sol high && git -c trailer.ifexists=replace commit"},
		{"two", "Model: claude-opus-5-5/high\nModel: human", false, "2 Model: trailers, want exactly one"},
	} {
		out, ok := tryCommit(t, dir, "-m", c.name, "-m", c.trailer)
		if ok != c.ok || !strings.Contains(out, c.says) {
			t.Errorf("%s: ok=%v\n%s", c.name, ok, out)
		}
	}
	// models lists the roster; a second add replaces the model's line.
	execute(t, a, "models", "add", "claude-opus-5-5", "low", "high")
	r = execute(t, a, "models")
	ms, _ := r["models"].([]task.Model)
	if len(ms) != 1 || strings.Join(ms[0].Levels, ",") != "low,high" {
		t.Fatalf("models: %v", r)
	}
	if out, ok := tryCommit(t, dir, "-m", "dropped level", "-m", "Model: claude-opus-5-5/medium"); ok || !strings.Contains(out, "(low, high)") {
		t.Errorf("after the roster changed: ok=%v\n%s", ok, out)
	}
}
