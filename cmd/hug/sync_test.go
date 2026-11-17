package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/veridit/hug/internal/flock"
)

// 0hkaa5, end to end with the real binary and its real detached syncers:
// two clones on one bare remote. A task note and a project memory made in
// clone A reach clone B within 10 s, and nobody runs sync.
func TestBackgroundSyncTwoClonesRealBinary(t *testing.T) {
	if testing.Short() {
		t.Skip("builds the hug binary")
	}
	root := t.TempDir()
	// The detached syncers run the binary and may still be running when the
	// test ends; Windows cannot delete a running .exe, so the binary lives
	// outside root, and the test waits for the syncers before cleanup.
	binDir, err := os.MkdirTemp("", "hug-sync-bin-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		for i := 0; i < 50 && os.RemoveAll(binDir) != nil; i++ {
			time.Sleep(200 * time.Millisecond)
		}
	})
	bin := filepath.Join(binDir, "hug")
	if runtime.GOOS == "windows" {
		bin += ".exe"
	}
	build := exec.Command("go", "build", "-o", bin, ".")
	build.Env = append(os.Environ(), "GOMAXPROCS=2", "GOFLAGS=-p=2")
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build: %v\n%s", err, out)
	}
	home := filepath.Join(root, "home")
	env := append(os.Environ(), "HOME="+home, "USERPROFILE="+home, "XDG_CONFIG_HOME="+filepath.Join(home, ".config"), "XDG_DATA_HOME="+filepath.Join(home, ".data"),
		"GIT_CONFIG_GLOBAL="+os.DevNull, "GIT_CONFIG_NOSYSTEM=1", "GIT_AUTHOR_NAME=Synthetic", "GIT_AUTHOR_EMAIL=synthetic@example.invalid",
		"GIT_COMMITTER_NAME=Synthetic", "GIT_COMMITTER_EMAIL=synthetic@example.invalid", "HUG_USER=tester", "HUG_MODEL=test-model", "JCODE_SESSION_ID=",
		"HUG_NO_EMBED=1", "JCODE_HOME="+filepath.Join(home, "no-jcode"), "HUG_SYNC=", "GOMAXPROCS=2")
	try := func(dir, muninnHome string, args ...string) (string, error) {
		c := exec.Command(args[0], args[1:]...)
		c.Dir, c.Env = dir, append(env, "HUG_HOME="+muninnHome)
		out, err := c.CombinedOutput()
		return string(out), err
	}
	run := func(dir, muninnHome string, args ...string) string {
		t.Helper()
		out, err := try(dir, muninnHome, args...)
		if err != nil {
			t.Fatalf("%v in %s: %v\n%s", args, dir, err, out)
		}
		return out
	}
	has := func(want, dir, muninnHome string, args ...string) bool {
		out, err := try(dir, muninnHome, args...)
		return err == nil && strings.Contains(out, want)
	}
	bare, one, two := filepath.Join(root, "remote.git"), filepath.Join(root, "one"), filepath.Join(root, "two")
	homeA, homeB := filepath.Join(root, "mem-a"), filepath.Join(root, "mem-b")
	run(root, homeA, "git", "init", "-q", "--bare", "-b", "main", bare)
	run(root, homeA, "git", "init", "-q", "-b", "main", one)
	run(one, homeA, "git", "commit", "-q", "--allow-empty", "-m", "init")
	run(one, homeA, "git", "remote", "add", "origin", bare)
	run(one, homeA, "git", "push", "-q", "origin", "main")
	run(root, homeB, "git", "clone", "-q", bare, two)
	run(one, homeA, bin, "task", "init")
	run(one, homeA, bin, "task", "sync")
	run(two, homeB, bin, "task", "init")
	out := run(one, homeA, bin, "task", "new", "Shared work", "--goal", "G", "--purpose", "P", "--principle", "Q", "--accept", "C", "--json")
	id := between(out, `"id":"`, `"`)
	if id == "" {
		t.Fatalf("no id in %s", out)
	}
	// The project memory branch exists on the remote once (as a person's
	// first hug mem sync does); from then on nobody syncs.
	run(one, homeA, bin, "mem", "remember", "--project", "Synthetic setup fact for the branch.")
	run(one, homeA, bin, "mem", "sync")
	run(two, homeB, bin, "mem", "sync")

	start := time.Now()
	run(one, homeA, bin, "task", "note", id, "note from clone A")
	run(one, homeA, bin, "mem", "remember", "--project", "Quartzite releases need a signed checklist.")
	// 10 s, the acceptance bound; the Windows CI runner spends ~1.9 s on one
	// hug command (run 43943367365), so the pipeline gets four times that.
	limit := 10 * time.Second
	if runtime.GOOS == "windows" {
		limit = 40 * time.Second
	}
	seen := func(f func() bool) time.Duration {
		for time.Since(start) < limit {
			if f() {
				return time.Since(start)
			}
			time.Sleep(200 * time.Millisecond)
		}
		return -1
	}
	note := seen(func() bool { return has("note from clone A", two, homeB, bin, "task", "show", id) })
	mem := seen(func() bool {
		return has("Quartzite releases need a signed checklist", two, homeB, bin, "mem", "search", "Quartzite checklist")
	})
	logs, _ := filepath.Glob(filepath.Join(one, ".git", "hug", "sync", "*.log"))
	var logText string
	for _, l := range logs {
		b, _ := os.ReadFile(l)
		logText += l + ":\n" + string(b)
	}
	// Let the syncers both clones started finish before the temp dirs go.
	t.Cleanup(func() { quiet(t, one, two) })
	if note < 0 || mem < 0 {
		t.Fatalf("clone B: note after %v, memory after %v (-1: not within %v)\n%s", note, mem, limit, logText)
	}
	t.Logf("clone B saw the note after %v and the memory after %v", note.Round(10*time.Millisecond), mem.Round(10*time.Millisecond))
}

// quiet waits up to 60 s until no background pass holds or waits for a run
// lock in the clones (their .git/hug/sync/*.run and *.queue files).
//
// One look was not enough (44w1ev, macOS CI run 44049348743 and 1 in 5 local
// runs on 8e9df0b: "TempDir RemoveAll cleanup: ... directory not empty"): a
// read in the test's last poll spawns a syncer that has not taken its queue
// yet (only its .pending marker exists), and a pass moves from the queue to
// the run lock after the run lock was looked at. So every look globs again,
// counts a fresh .pending marker as busy, and the clones count as quiet only
// after two idle looks 500 ms apart.
func quiet(t *testing.T, clones ...string) {
	deadline := time.Now().Add(60 * time.Second)
	idle := 0
	for idle < 2 && time.Now().Before(deadline) {
		if busySyncers(clones) {
			idle = 0
			time.Sleep(100 * time.Millisecond)
			continue
		}
		if idle++; idle < 2 {
			time.Sleep(500 * time.Millisecond)
		}
	}
	if idle < 2 {
		t.Logf("background passes still running after 60 s")
	}
}

// busySyncers reports a syncer pending, queued or running in any clone.
func busySyncers(clones []string) bool {
	for _, c := range clones {
		dir := filepath.Join(c, ".git", "hug", "sync")
		pending, _ := filepath.Glob(filepath.Join(dir, "*.pending"))
		for _, p := range pending {
			if st, err := os.Stat(p); err == nil && time.Since(st.ModTime()) < 5*time.Second {
				return true
			}
		}
		queues, _ := filepath.Glob(filepath.Join(dir, "*.queue"))
		runs, _ := filepath.Glob(filepath.Join(dir, "*.run"))
		for _, l := range append(queues, runs...) {
			release, ok, err := flock.TryAcquire(l)
			if err != nil {
				continue
			}
			if !ok {
				return true
			}
			release()
		}
	}
	return false
}

func between(s, from, to string) string {
	_, rest, ok := strings.Cut(s, from)
	if !ok {
		return ""
	}
	v, _, _ := strings.Cut(rest, to)
	return v
}
