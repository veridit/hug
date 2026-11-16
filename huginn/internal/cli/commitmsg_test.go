package cli

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
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
	a := app(t)
	dir := a.Cwd
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
	if ok || !strings.Contains(out, "commit refused: no Model: trailer") || !strings.Contains(out, "\nfix: git -c trailer.ifexists=replace commit --cleanup=strip -F '") || !strings.Contains(out, "--trailer 'Model: <model-id-or-human>'") {
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
	sh = exec.Command("sh", "-c", strings.Replace(fix, "<model-id-or-human>", "gpt-6.1-sol", 1))
	sh.Dir = dir
	if b, e := sh.CombinedOutput(); e != nil {
		t.Fatalf("filled fix: %v\n%s", e, b)
	}
	if got := git(t, dir, "log", "-1", "--format=%s|%(trailers:key=Model,valueonly)"); got != "no trailer|gpt-6.1-sol" {
		t.Fatalf("after the fix: %q", got)
	}

	for _, c := range []struct {
		name string
		args []string
		ok   bool
		says string
	}{
		{"model", []string{"-m", "work", "-m", "Model: claude-opus-5-5"}, true, ""},
		{"human", []string{"-m", "my own", "-m", "Model: human"}, true, ""},
		{"two", []string{"-m", "two", "-m", "Model: a\nModel: b"}, false, "2 Model: trailers, want exactly one"},
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

	// Cheap: under 50 ms a commit on a quiet machine (fastest of 7 each,
	// gate minus no-verify); CI and Windows only log it.
	gate, plain := fastest(t, dir)
	t.Logf("commit with gate %v, --no-verify %v (fastest of 7, interleaved)", gate, plain)
	if limit := 50 * time.Millisecond; gate-plain > limit && os.Getenv("CI") == "" && runtime.GOOS != "windows" {
		t.Errorf("the gate costs %v per commit, over %v", gate-plain, limit)
	}

	r = execute(t, a, "uninstall", "commit-msg")
	if _, e := os.Stat(hook); !os.IsNotExist(e) {
		t.Fatalf("uninstall left %s: %v", hook, r)
	}
	if _, ok := tryCommit(t, dir, "-m", "gate gone"); !ok {
		t.Fatal("commit refused after uninstall")
	}
}

// fastest times commits with and without the gate, interleaved so both see
// the same machine load, and returns the fastest of each: load only ever
// adds time, so the minimum is the closest to the gate's own cost (a median
// of two separate blocks read 79 ms with 14 test packages running, 39 ms
// quiet).
func fastest(t *testing.T, dir string) (gate, plain time.Duration) {
	one := func(noVerify bool) time.Duration {
		args := []string{"-q", "-m", "timing", "-m", "Model: human"}
		if noVerify {
			args = append(args, "--no-verify")
		}
		start := time.Now()
		if out, ok := tryCommit(t, dir, args...); !ok {
			t.Fatalf("timing commit: %s", out)
		}
		return time.Since(start)
	}
	for i := 0; i < 7; i++ {
		if d := one(false); gate == 0 || d < gate {
			gate = d
		}
		if d := one(true); plain == 0 || d < plain {
			plain = d
		}
	}
	return gate, plain
}

// A commit-msg hook already in .git/hooks keeps running, first; uninstall
// puts it back.
func TestCommitMsgGateChainsExistingHook(t *testing.T) {
	a := app(t)
	dir := a.Cwd
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
	a := app(t)
	dir := a.Cwd
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
