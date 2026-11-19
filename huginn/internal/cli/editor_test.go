package cli

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/veridit/hug/internal/homeguard"
)

// The test binary doubles as $VISUAL for hug task edit: a shell script with
// a #!/bin/sh line cannot be started on Windows ("%1 is not a valid Win32
// application", 4emme8), and perl and sh are not on every PATH there.
const (
	editFrom = "HUG_TEST_EDIT_FROM"
	editTo   = "HUG_TEST_EDIT_TO"
)

func TestMain(m *testing.M) {
	if len(os.Args) == 4 && os.Args[3] == SyncChildArg {
		runSyncChild() // a background pass spawned by sync_test.go
	}
	if from, ok := os.LookupEnv(editFrom); ok && len(os.Args) > 1 {
		os.Exit(fakeEdit(os.Args[len(os.Args)-1], from, os.Getenv(editTo)))
	}
	quietGit()
	// No test reaches this machine's real harness configs (ynh84t).
	os.Exit(homeguard.Run(m))
}

// quietGit turns off git's auto maintenance for every git this test binary
// and its children start (0n4rnx): after each commit, fetch and merge git
// runs `git maintenance run --auto`, which Windows git runs in the
// foreground (daemonize is ENOSYS there). One catch-up test started 31 of
// them (trace2, 2026-11-18). The variables reach hug's own git calls and
// background passes, which inherit the environment; a fixture remote's
// receive-pack does not read them, so quietRemote sets the remotes' config.
// Test fixtures only: hug never sets this for a user.
func quietGit() {
	n := 0
	if s, ok := os.LookupEnv("GIT_CONFIG_COUNT"); ok {
		fmt.Sscan(s, &n)
	}
	os.Setenv(fmt.Sprintf("GIT_CONFIG_KEY_%d", n), "maintenance.auto")
	os.Setenv(fmt.Sprintf("GIT_CONFIG_VALUE_%d", n), "false")
	os.Setenv("GIT_CONFIG_COUNT", fmt.Sprint(n+1))
}

// quietRemote sets maintenance.auto=false in a fixture remote: receive-pack
// starts `git maintenance run --auto` after every push and reads only the
// remote's own config (not GIT_CONFIG_COUNT from quietGit).
func quietRemote(t *testing.T, bare string) {
	t.Helper()
	git(t, bare, "config", "maintenance.auto", "false")
}

// 0n4rnx: a fixture commit and a push to a fixture remote start no
// `git maintenance run --auto`. Without quietGit and quietRemote, git 2.54
// started one per commit and one per receive-pack.
func TestFixtureGitStartsNoMaintenance(t *testing.T) {
	t.Parallel()
	a := app(t)
	bare := filepath.Join(t.TempDir(), "remote.git")
	git(t, a.Cwd, "init", "--bare", "-b", "main", bare)
	quietRemote(t, bare)
	git(t, a.Cwd, "remote", "add", "origin", bare)
	trace := filepath.Join(t.TempDir(), "trace2")
	for _, args := range [][]string{{"commit", "--allow-empty", "-m", "x"}, {"push", "-q", "origin", "main"}} {
		c := exec.Command("git", append([]string{"-C", a.Cwd}, args...)...)
		c.Env = append(os.Environ(), "GIT_TRACE2_EVENT="+trace)
		if out, e := c.CombinedOutput(); e != nil {
			t.Fatalf("git %v: %v\n%s", args, e, out)
		}
	}
	b, e := os.ReadFile(trace)
	if e != nil {
		t.Fatal(e)
	}
	if n := bytes.Count(b, []byte(`"maintenance","run"`)); n != 0 {
		t.Errorf("fixture commit and push started git maintenance %d times, want 0", n)
	}
}

// fakeEdit replaces the first from with to in path, as the editor would.
func fakeEdit(path, from, to string) int {
	b, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	if from != "" && !strings.Contains(string(b), from) {
		fmt.Fprintf(os.Stderr, "fake editor: %q not in %s\n", from, path)
		return 1
	}
	if err := os.WriteFile(path, []byte(strings.Replace(string(b), from, to, 1)), 0o644); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	return 0
}

// useEditor makes hug task edit replace the first from with to in the task
// file; from "" leaves the file as it is.
func useEditor(t *testing.T, a *App, from, to string) {
	t.Helper()
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	if strings.ContainsAny(exe, " \t") {
		t.Fatalf("test binary path %q has a space; editor.Command splits $VISUAL on spaces", exe)
	}
	// a's environment, passed to the editor it starts: the process's stays
	// as it is, so this test can run in parallel with others (0sbvnq).
	a.Env["VISUAL"], a.Env[editFrom], a.Env[editTo] = exe, from, to
}
