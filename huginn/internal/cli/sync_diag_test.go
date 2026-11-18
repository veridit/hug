package cli

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	osexec "os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"
	"time"
)

// 6ep7y2: a Windows pass held the run lock for 48 s after "push starting"
// (CI 44309663696) and its phase log could not say what push waited on.
// The test passes write git's trace2 events, one file per git process, and
// a timed-out wait reports the git processes that never exited and what
// each was doing last, plus the machine's live git and shell processes.

// traceDir is where a clone's passes write trace2 events: beside the sync
// state in its git directory, so TempDir cleanup removes it.
func traceDir(clone string) string { return filepath.Join(clone, ".git", "hug", "trace2") }

// traceDirOf is traceDir from the phase log path (<git dir>/hug/sync/x.log).
func traceDirOf(log string) string { return filepath.Join(filepath.Dir(filepath.Dir(log)), "trace2") }

// unfinishedGit lists the git processes in dir's trace2 files with no exit
// event: argv, age at its last event, and its last 4 events.
func unfinishedGit(dir string) string {
	ents, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Sprintf("reading trace2: %v\n", err)
	}
	type ev struct {
		Event    string   `json:"event"`
		Time     string   `json:"time"`
		Argv     []string `json:"argv"`
		Name     string   `json:"name"`
		ChildID  *int     `json:"child_id"`
		Class    string   `json:"child_class"`
		HookName string   `json:"hook_name"`
		Category string   `json:"category"`
		Label    string   `json:"label"`
		Code     *int     `json:"code"`
	}
	var out []string
	for _, e := range ents {
		f, err := os.Open(filepath.Join(dir, e.Name()))
		if err != nil {
			continue
		}
		var argv []string
		var last []string
		exited := false
		sc := bufio.NewScanner(f)
		sc.Buffer(make([]byte, 1<<20), 1<<20)
		for sc.Scan() {
			var x ev
			if json.Unmarshal(sc.Bytes(), &x) != nil {
				continue
			}
			switch x.Event {
			case "start":
				argv = x.Argv
			case "exit", "atexit":
				exited = true
			case "version", "cmd_ancestry", "def_param", "def_repo", "data", "data_json", "counter", "timer", "too_many_files", "printf", "perf":
				continue
			}
			s := x.Time + " " + x.Event
			if x.ChildID != nil {
				s += fmt.Sprintf(" child %d %s%s %q", *x.ChildID, x.Class, x.HookName, x.Argv)
			}
			if x.Category != "" {
				s += " " + x.Category + "/" + x.Label
			}
			if x.Name != "" {
				s += " " + x.Name
			}
			if x.Code != nil {
				s += fmt.Sprintf(" code %d", *x.Code)
			}
			last = append(last, s)
			if len(last) > 4 {
				last = last[1:]
			}
		}
		f.Close()
		if !exited && argv != nil {
			out = append(out, fmt.Sprintf("git never exited: %q\n    %s", argv, strings.Join(last, "\n    ")))
		}
	}
	sort.Strings(out)
	if len(out) == 0 {
		return fmt.Sprintf("trace2: every one of %d git processes exited\n", len(ents))
	}
	return fmt.Sprintf("trace2: %d of %d git processes never exited:\n%s\n", len(out), len(ents), strings.Join(out, "\n"))
}

// liveProcesses lists the machine's git, shell and test processes, bounded 20 s.
func liveProcesses() string {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	var c *osexec.Cmd
	if runtime.GOOS == "windows" {
		c = osexec.CommandContext(ctx, "powershell", "-NoProfile", "-Command",
			"Get-CimInstance Win32_Process | Where-Object { $_.Name -match '^(git.*|sh|bash|cat|cli\\.test)\\.exe$' } | ForEach-Object { '{0} ppid {1} start {2:o} {3}' -f $_.ProcessId, $_.ParentProcessId, $_.CreationDate, $_.CommandLine }")
	} else {
		c = osexec.CommandContext(ctx, "ps", "-eo", "pid,ppid,etime,args")
	}
	b, err := c.CombinedOutput()
	lines := strings.Split(strings.TrimSpace(string(b)), "\n")
	if runtime.GOOS != "windows" {
		keep := lines[:0]
		for i, l := range lines {
			if i == 0 || strings.Contains(l, "git") || strings.Contains(l, "cli.test") {
				keep = append(keep, l)
			}
		}
		lines = keep
	}
	return fmt.Sprintf("processes (err %v):\n%s\n", err, strings.Join(lines, "\n"))
}

// The stall report names a git process still running and leaves out the
// ones that exited: a cat-file waiting on its stdin stands in for a push.
func TestUnfinishedGitNamesARunningGit(t *testing.T) {
	dir := t.TempDir()
	repo := filepath.Join(dir, "repo")
	trace := filepath.Join(dir, "trace2")
	if e := os.MkdirAll(trace, 0o700); e != nil {
		t.Fatal(e)
	}
	git(t, dir, "init", "-q", repo)
	run := func(args ...string) *osexec.Cmd {
		c := osexec.Command("git", append([]string{"-C", repo}, args...)...)
		c.Env = append(os.Environ(), "GIT_TRACE2_EVENT="+trace)
		return c
	}
	for i := 0; i < 3; i++ {
		if out, e := run("rev-parse", "--git-dir").CombinedOutput(); e != nil {
			t.Fatalf("%v: %s", e, out)
		}
	}
	hung := run("cat-file", "--batch")
	stdin, e := hung.StdinPipe()
	if e != nil {
		t.Fatal(e)
	}
	if e = hung.Start(); e != nil {
		t.Fatal(e)
	}
	defer func() { stdin.Close(); _ = hung.Wait() }()
	var report string
	if !until(10*time.Second, func() bool {
		report = unfinishedGit(trace)
		return strings.Contains(report, "cat-file")
	}) {
		t.Fatalf("report does not name the running git:\n%s", report)
	}
	if !strings.Contains(report, "trace2: 1 of 4 git processes never exited") || strings.Contains(report, "rev-parse") {
		t.Fatalf("report counts the exited gits as running:\n%s", report)
	}
}
