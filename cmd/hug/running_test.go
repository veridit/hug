package main

import (
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"
)

// rvb3j5 check 2: hug version --running lists the running hug processes,
// each with its pid, what it runs, its version and file, and when it
// started, and flags one older than this hug. The rows checked are the
// test's own servers; other hug processes on the machine are only read.
func TestVersionRunningFlagsOlderHug(t *testing.T) {
	if testing.Short() {
		t.Skip("builds the hug binary twice")
	}
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		t.Skip("lists processes on macOS and Linux: TestVersionRunningUnsupported covers the rest")
	}
	dir := t.TempDir()
	oldBin := filepath.Join(dir, "old", "hug")
	sameBin := filepath.Join(dir, "same", "hug")
	thisBin := filepath.Join(dir, "bin", "hug")
	cachedHug(t, oldBin, "0.0.1-old")
	cachedHug(t, sameBin, "0.0.2-new")
	cachedHug(t, thisBin, "0.0.2-new")
	oldCmd, _, _, _ := startMCP(t, filepath.Join(dir, "a"), oldBin)
	sameCmd, _, _, _ := startMCP(t, filepath.Join(dir, "b"), sameBin)
	cmd := exec.Command(thisBin, "version", "--running")
	cmd.Env = mcpEnv(dir, filepath.Join(dir, "home"))
	out, err := cmd.CombinedOutput()
	code := 0
	if ee, ok := err.(*exec.ExitError); ok {
		code = ee.ExitCode()
	} else if err != nil {
		t.Fatal(err)
	}
	got := string(out)
	if !strings.HasPrefix(got, "hug 0.0.2-new\n") {
		t.Errorf("the first line is this hug's version, as hug version prints it:\n%s", got)
	}
	row := func(pid int) string {
		for _, l := range strings.Split(got, "\n") {
			if f := strings.Fields(l); len(f) > 0 && f[0] == strconv.Itoa(pid) {
				return l
			}
		}
		t.Fatalf("no row for pid %d:\n%s", pid, got)
		return ""
	}
	today := time.Now().Format("2006-01-02")
	old := row(oldCmd.Process.Pid)
	for _, want := range []string{" mcp ", " 0.0.1-old ", oldBin, today, "older than this hug"} {
		if !strings.Contains(old, want) {
			t.Errorf("old server's row lacks %q:\n%s\nall:\n%s", want, old, got)
		}
	}
	same := row(sameCmd.Process.Pid)
	for _, want := range []string{" mcp ", " 0.0.2-new ", sameBin, today} {
		if !strings.Contains(same, want) {
			t.Errorf("current server's row lacks %q:\n%s", want, same)
		}
	}
	if strings.Contains(same, "older") {
		t.Errorf("a server on this hug's version is flagged:\n%s", same)
	}
	if code != 5 {
		t.Errorf("exit %d with an older hug running, want 5:\n%s", code, got)
	}
	if !strings.Contains(got, "fix: ") {
		t.Errorf("no fix: line for the older server:\n%s", got)
	}
}
