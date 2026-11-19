package tui

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/veridit/hug/internal/tmuxtest"
)

// TestTmuxServerSlowStarts holds the tmux harness to a machine where
// starting a process takes seconds (ghycrq). Under 16 CPU hogs /bin/echo
// took up to 11.6 s to run and tmux -V up to 9.7 s, while a running process
// woke at most 174 ms late. startTmuxServer must start and stop its server
// on such a machine: here every tmux process takes 6 s to start.
func TestTmuxServerSlowStarts(t *testing.T) {
	if testing.Short() {
		t.Skip("real tmux acceptance disabled by -short")
	}
	if runtime.GOOS == "windows" {
		t.Skip("no tmux on Windows")
	}
	real, err := exec.LookPath("tmux")
	if err != nil {
		t.Skip("tmux required for terminal acceptance")
	}
	bin := t.TempDir()
	slow := "#!/bin/sh\nsleep 6\nexec " + shellQuote(real) + " \"$@\"\n"
	if err := os.WriteFile(filepath.Join(bin, "tmux"), []byte(slow), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	cleanup := startTmuxServer(t, tmuxtest.Label("slowstart"))
	cleanup()
}
