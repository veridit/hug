//go:build !windows

package tui

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/veridit/hug/internal/tmuxtest"
)

// tmuxChild runs TestTmuxServerChild in a child test process (wacc33): the
// value says how that test ends.
const tmuxChild = "HUG_TEST_TMUX_CHILD"

// serverAlive says whether tmux server pid with this label still runs.
func serverAlive(t *testing.T, pid int, label string) bool {
	t.Helper()
	alive, err := tmuxtest.Alive(pid, label)
	if err != nil {
		t.Fatalf("ps: %v", err)
	}
	return alive
}

// gone waits up to 10 s for server pid to end.
func gone(t *testing.T, pid int, label string) bool {
	t.Helper()
	for deadline := time.Now().Add(10 * time.Second); time.Now().Before(deadline); time.Sleep(50 * time.Millisecond) {
		if !serverAlive(t, pid, label) {
			return true
		}
	}
	return false
}

// TestTmuxServerChild is the test a child process runs for
// TestTmuxServerGoneWhenTestEnds. It starts a server the way the tmux tests
// do, says which, and then ends as HUG_TEST_TMUX_CHILD says: t.Fatal, a
// panic in a subtest, go test's -timeout, or SIGKILL from the parent.
func TestTmuxServerChild(t *testing.T) {
	mode := os.Getenv(tmuxChild)
	if mode == "" {
		t.Skip("run as a child by TestTmuxServerGoneWhenTestEnds")
	}
	label := tmuxtest.Label("wacc33")
	defer startTmuxServer(t, label)()
	procs, err := tmuxtest.Servers()
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range procs {
		if p.Label == label {
			fmt.Printf("tmux-server %d %s\n", p.PID, label)
		}
	}
	switch mode {
	case "fatal":
		t.Fatal("mid-run failure")
	case "panic":
		t.Run("child", func(t *testing.T) { panic("mid-run panic") })
	case "timeout", "kill":
		select {} // until -test.timeout or SIGKILL ends the process
	}
}

// A test that fails, panics, times out or is killed mid-run leaves no tmux
// server behind (wacc33: crab found three huginn-golden/reload servers,
// parent launchd, alive for days).
func TestTmuxServerGoneWhenTestEnds(t *testing.T) {
	if testing.Short() {
		t.Skip("real tmux acceptance disabled by -short")
	}
	if _, err := exec.LookPath("tmux"); err != nil {
		t.Skip("tmux required for terminal acceptance")
	}
	self, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	for _, mode := range []string{"fatal", "panic", "timeout", "kill"} {
		t.Run(mode, func(t *testing.T) {
			t.Parallel()
			cmd := exec.Command(self, "-test.run=^TestTmuxServerChild$", "-test.timeout=15s")
			cmd.Env = append(os.Environ(), tmuxChild+"="+mode)
			// The server writes to the child's stderr and may outlive it: a
			// file, not a pipe, so reading it never waits on the server.
			stderr, err := os.CreateTemp(t.TempDir(), "stderr")
			if err != nil {
				t.Fatal(err)
			}
			defer stderr.Close()
			cmd.Stderr = stderr
			stdout, err := cmd.StdoutPipe()
			if err != nil {
				t.Fatal(err)
			}
			if err := cmd.Start(); err != nil {
				t.Fatal(err)
			}
			reported := make(chan string, 1)
			read := make(chan struct{})
			var output strings.Builder
			go func() {
				defer close(read)
				sc := bufio.NewScanner(stdout)
				for sc.Scan() {
					output.WriteString(sc.Text() + "\n")
					if strings.HasPrefix(sc.Text(), "tmux-server ") {
						reported <- sc.Text()
					}
				}
			}()
			var pid int
			var label string
			select {
			case line := <-reported:
				fmt.Sscanf(line, "tmux-server %d %s", &pid, &label)
			case <-read:
			case <-time.After(2 * time.Minute):
				cmd.Process.Kill()
			}
			if pid != 0 {
				// This test must not leave the server it looks for either.
				t.Cleanup(func() {
					if serverAlive(t, pid, label) {
						syscall.Kill(pid, syscall.SIGTERM)
					}
				})
			}
			if mode == "kill" {
				cmd.Process.Kill()
			}
			<-read
			waitErr := cmd.Wait()
			errText, _ := os.ReadFile(stderr.Name())
			if pid == 0 {
				t.Fatalf("%s: the child never reported its tmux server (%v)\nstdout:\n%s\nstderr:\n%s", mode, waitErr, output.String(), errText)
			}
			if waitErr == nil {
				t.Fatalf("%s: the child passed; it must fail\n%s", mode, output.String())
			}
			if !gone(t, pid, label) {
				t.Errorf("%s: tmux server %d (-L %s) still runs 10s after its test process ended (%v)", mode, pid, label, waitErr)
			}
		})
	}
}

// A server left by a run that died anyway (from before wacc33, or with its
// watchdog killed too) is reaped by the next run: TestMain calls
// tmuxtest.Reap. Its owner pid, the end of its label, no longer runs.
func TestTmuxOrphanReaped(t *testing.T) {
	if testing.Short() {
		t.Skip("real tmux acceptance disabled by -short")
	}
	if _, err := exec.LookPath("tmux"); err != nil {
		t.Skip("tmux required for terminal acceptance")
	}
	dead := exec.Command("true")
	if err := dead.Run(); err != nil {
		t.Fatal(err)
	}
	label := fmt.Sprintf("%swacc33-reap-%d", tmuxtest.Prefix, dead.Process.Pid)
	path, err := tmuxSocketPath(label)
	if err != nil {
		t.Fatal(err)
	}
	server := exec.Command("tmux", "-D", "-L", label, "-f", "/dev/null")
	if err := server.Start(); err != nil {
		t.Fatal(err)
	}
	done := make(chan struct{})
	go func() { server.Wait(); close(done) }()
	t.Cleanup(func() {
		server.Process.Signal(syscall.SIGTERM)
		<-done
	})
	for deadline := time.Now().Add(time.Minute); ; time.Sleep(25 * time.Millisecond) {
		if _, err := os.Stat(path); err == nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("tmux -L %s never made %s", label, path)
		}
	}
	reaped, err := tmuxtest.Reap()
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, p := range reaped {
		if p.PID == server.Process.Pid {
			found = true
		}
		if tmuxtest.Owner(p.Label) == os.Getpid() {
			t.Errorf("reaped %v, a server of this live test process", p)
		}
	}
	if !found {
		// Another tui test binary starting now (TestMain) may have reaped it first.
		select {
		case <-done:
			return
		case <-time.After(10 * time.Second):
		}
		t.Fatalf("Reap did not end server %d (-L %s, owner %d gone): reaped %v", server.Process.Pid, label, dead.Process.Pid, reaped)
	}
	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatalf("server %d (-L %s) still runs 10s after Reap", server.Process.Pid, label)
	}
}
