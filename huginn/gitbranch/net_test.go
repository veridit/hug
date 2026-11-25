package gitbranch

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/veridit/hug/internal/timing"
)

// 0hkaa5: a remote that never answers does not hang a command: the network
// call is killed at NetTimeout and says so, with a fix line.
func TestNetworkCallTimesOut(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses a shell script as the ssh command")
	}
	timing.Budget(t)
	root, _ := fixture(t, false)
	// An ssh that never answers, a remote that uses it.
	hang := filepath.Join(t.TempDir(), "hang-ssh")
	must(t, os.WriteFile(hang, []byte("#!/bin/sh\nexec sleep 60\n"), 0o755))
	command(t, root, "remote", "add", "origin", "ssh://example.invalid/x.git")
	t.Setenv("GIT_SSH_COMMAND", hang)
	old := NetTimeout
	NetTimeout = 500 * time.Millisecond
	defer func() { NetTimeout = old }()
	r := &Repository{Dir: root, Branch: "hug/tasks", Remote: "origin"}
	start := time.Now()
	_, e := r.fetch()
	if d := time.Since(start); d > 5*time.Second {
		t.Fatalf("fetch took %v against a hanging remote, NetTimeout 500ms", d)
	}
	var ge *Error
	if !errors.Is(e, ErrNetTimeout) || !errors.As(e, &ge) || !strings.Contains(ge.Error(), "did not answer within 500ms") || !strings.Contains(ge.Error(), "\nfix: ") {
		t.Fatalf("want a timeout error with a fix line, got %v", e)
	}
}

// Network calls never prompt: ssh gets BatchMode and a connect timeout,
// unless the user has an ssh command of their own, which is kept.
func TestNetworkEnvNeverPrompts(t *testing.T) {
	root, _ := fixture(t, false)
	t.Setenv("GIT_SSH_COMMAND", "")
	t.Setenv("GIT_SSH", "")
	env := strings.Join(netEnv(root, nil), "\n")
	for _, want := range []string{"GIT_SSH_COMMAND=ssh -o BatchMode=yes -o ConnectTimeout=10", "GIT_ASKPASS=", "SSH_ASKPASS_REQUIRE=never", "GCM_INTERACTIVE=never"} {
		if !strings.Contains(env, want) {
			t.Fatalf("network env misses %q:\n%s", want, env)
		}
	}
	command(t, root, "config", "core.sshCommand", "ssh -i /my/key")
	if env := strings.Join(netEnv(root, nil), "\n"); strings.Contains(env, "GIT_SSH_COMMAND=") {
		t.Fatalf("overrides the user's core.sshCommand:\n%s", env)
	}
	if !isNet([]string{"-c", "x=y", "push", "origin"}) || isNet([]string{"commit", "-m", "push"}) {
		t.Fatal("isNet misjudges the verb")
	}
}

// An unreachable remote is named as such, not as a key problem.
func TestUnreachableRemoteIsNamed(t *testing.T) {
	root, _ := fixture(t, false)
	command(t, root, "remote", "add", "origin", filepath.Join(t.TempDir(), "gone.git"))
	r := &Repository{Dir: root, Branch: "hug/tasks", Remote: "origin"}
	_, e := r.fetch()
	if e == nil || !strings.Contains(e.Error(), "the remote is unreachable: ") || strings.Contains(e.Error(), "SSH credentials") {
		t.Fatalf("want an unreachable-remote error, got %v", e)
	}
}
