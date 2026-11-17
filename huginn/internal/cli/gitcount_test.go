package cli

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
)

// 0sbvnq: a hug command costs its git processes (about 45 ms each on the
// Windows runner, more under Defender), so their number per command is
// held, counted rather than timed: git on PATH is a wrapper that logs each
// call. The numbers are today's; a change that adds a git call to a
// command fails here, and one that removes one updates the table.
//
// Serial: it sets PATH for the process, which every git call reads.
func TestGitProcessesPerCommand(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the git wrapper is a POSIX sh script")
	}
	a := app(t)
	execute(t, a, "init")
	id := execute(t, a, "new", "Counted", "--goal", "G", "--purpose", "P", "--principle", "Q", "--accept", "C")["task"].(Reply)["id"].(string)
	count := gitCounter(t)
	// Each command is counted on a cache that has caught up with the last
	// write (an ls first): the first command after a write also diffs the
	// new HEAD against the cache's, one more git process.
	want := []struct {
		args []string
		n    int
	}{
		{[]string{"show", id}, 5},
		{[]string{"ls"}, 5},
		{[]string{"board"}, 5},
		{[]string{"note", id, "counted"}, 9},
		{[]string{"check", id, "1"}, 9},
		{[]string{"take", id, "--model", "test-model"}, 15},
		// wait asks who (waiting_asked_by), never where: no place lookup.
		{[]string{"wait", id, "CI"}, 9},
		{[]string{"resume", id}, 9},
		{[]string{"done", id}, 9},
	}
	var got []string
	fail := false
	for _, w := range want {
		execute(t, a, "ls")
		calls := count(func() { execute(t, a, w.args...) })
		got = append(got, w.args[0]+"="+strconv.Itoa(len(calls)))
		if len(calls) != w.n {
			fail = true
			t.Errorf("hug task %s starts %d git processes, want %d:\n  %s", w.args[0], len(calls), w.n, strings.Join(calls, "\n  "))
		}
	}
	if fail {
		t.Logf("all: %s", strings.Join(got, " "))
	}
}

// gitCounter puts a logging git wrapper first on PATH for the rest of the
// test and returns a function that runs f and gives the git calls it made.
func gitCounter(t *testing.T) func(f func()) []string {
	t.Helper()
	real, e := exec.LookPath("git")
	if e != nil {
		t.Fatal(e)
	}
	bin, logFile := t.TempDir(), filepath.Join(t.TempDir(), "calls")
	// One line per process: its first two words, newlines (a commit
	// message) folded into spaces.
	script := "#!/bin/sh\nprintf '%s\\n' \"$(echo \"$*\" | tr '\\n' ' ')\" >> '" + logFile + "'\nexec '" + real + "' \"$@\"\n"
	if e := os.WriteFile(filepath.Join(bin, "git"), []byte(script), 0o755); e != nil {
		t.Fatal(e)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	return func(f func()) []string {
		_ = os.Remove(logFile)
		f()
		b, _ := os.ReadFile(logFile)
		var calls []string
		for _, l := range strings.Split(strings.TrimSpace(string(b)), "\n") {
			if l != "" {
				// The working directory varies; the command is what counts.
				if strings.HasPrefix(l, "-C ") {
					if _, rest, ok := strings.Cut(l[3:], " "); ok {
						l = rest
					}
				}
				calls = append(calls, l)
			}
		}
		return calls
	}
}
