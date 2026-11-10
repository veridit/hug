package tui

import (
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

// TestTerminalGoldens drives the real binary, not Model.View or a mock terminal.
// Plain capture-pane output is stable across colour profiles. Both colour modes
// are exercised separately so NO_COLOR cannot accidentally change the layout.
func TestTerminalGoldens(t *testing.T) {
	if testing.Short() {
		t.Skip("real terminal golden workflow requires a binary build")
	}
	for _, tool := range []string{"tmux", "python3", "nice"} {
		if _, err := exec.LookPath(tool); err != nil {
			t.Skip(tool + " unavailable")
		}
	}
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	binary := filepath.Join(t.TempDir(), "muninn")
	// Respect the shared machine's five-minute load ceiling before the nested
	// build. On non-macOS platforms sysctl is absent and the outer test runner's
	// bounded parallelism is still retained.
	for {
		raw, e := exec.Command("sysctl", "-n", "vm.loadavg").Output()
		if e != nil {
			break
		}
		fields := strings.Fields(strings.Trim(string(raw), "{} \n"))
		if len(fields) < 2 {
			break
		}
		load, e := strconv.ParseFloat(strings.ReplaceAll(fields[1], ",", "."), 64)
		if e != nil || load <= 120 {
			break
		}
		t.Logf("waiting for 5-minute load %.2f to fall below 120", load)
		time.Sleep(15 * time.Second)
	}
	build := exec.Command("nice", "-n", "19", "go", "build", "-o", binary, "./cmd/muninn")
	build.Dir = root
	build.Env = append(os.Environ(), "GOMAXPROCS=2", "GOFLAGS=-p=2")
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build: %v\n%s", err, out)
	}
	run := exec.Command("nice", "-n", "19", "python3", filepath.Join(root, "scripts", "tui-pty.py"), binary)
	run.Dir = root
	run.Env = append(os.Environ(), "GOMAXPROCS=2", "GOFLAGS=-p=2")
	out, err := run.CombinedOutput()
	if err != nil {
		t.Fatalf("real-terminal golden workflow: %v\n%s", err, out)
	}
	t.Log(string(out))
}
