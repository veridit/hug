package tui

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/veridit/hug/internal/tmuxtest"
)

// TestTmuxAutoRefresh runs the real binary in tmux: another process runs
// hug task note and hug task set, and the open TUI shows both without a key
// press, in the list and in the open detail (nfv18p).
func TestTmuxAutoRefresh(t *testing.T) {
	if testing.Short() {
		t.Skip("real tmux acceptance disabled by -short")
	}
	if _, err := exec.LookPath("tmux"); err != nil {
		t.Skip("tmux required for terminal acceptance")
	}
	loadGuard(t)
	root, err := os.MkdirTemp(os.Getenv("JCODE_SCRATCH_DIR"), "hug-refresh-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(root)
	binary := filepath.Join(root, "hug")
	build := exec.Command("nice", "-n", "19", "env", "GOMAXPROCS=2", "GOFLAGS=-p=2", "go", "build", "-o", binary, "./cmd/hug")
	build.Dir = "../../.."
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build: %v\n%s", err, out)
	}
	home, repo := filepath.Join(root, "home"), filepath.Join(root, "repo")
	for _, p := range []string{home, repo} {
		if err := os.MkdirAll(p, 0755); err != nil {
			t.Fatal(err)
		}
	}
	env := []string{"HOME=" + home, "XDG_CONFIG_HOME=" + filepath.Join(home, ".config"), "GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_NOSYSTEM=1", "GIT_AUTHOR_NAME=Synthetic", "GIT_AUTHOR_EMAIL=synthetic@example.invalid", "GIT_COMMITTER_NAME=Synthetic", "GIT_COMMITTER_EMAIL=synthetic@example.invalid", "HUG_USER=other-terminal", "GOMAXPROCS=2", "GOFLAGS=-p=2"}
	// The other terminal also sets an old variable name, so every hug it
	// runs prints the stale-name warning on stderr (cmd/hug/env.go,
	// "hug: hug reads only HUG_* variables: ..."). ladybug 2026-11-21 ran
	// the suite with the old golden variable set and this parse failed:
	// "invalid character 'h' looking for beginning of value" (tfe9nr).
	// Spelt in parts so TestNoOldNamesInRepository does not match it.
	stale := "HUG" + "INN_UPDATE_GOLDEN=1"
	run := func(name string, args ...string) string {
		t.Helper()
		cmd := exec.Command(name, args...)
		cmd.Dir = repo
		cmd.Env = append(append(os.Environ(), env...), stale)
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("%s %v: %v\n%s", name, args, err, out)
		}
		return string(out)
	}
	run("git", "init", "-q", "-b", "main")
	run("git", "commit", "-q", "--allow-empty", "-m", "synthetic fixture")
	run(binary, "task", "init")
	ids := map[string]string{}
	for _, title := range []string{"Alpha", "Beta", "Gamma"} {
		var r struct {
			Task struct{ ID string } `json:"task"`
		}
		if err := json.Unmarshal([]byte(run(binary, "task", "new", title, "--goal", "G", "--purpose", "P", "--principle", "Q", "--accept", "C", "--json")), &r); err != nil {
			t.Fatal(err)
		}
		ids[title] = r.Task.ID
	}
	socket := tmuxtest.Label("refresh")
	cleanupServer := startTmuxServer(t, socket)
	defer cleanupServer()
	tmux := func(args ...string) string {
		t.Helper()
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		out, err := exec.CommandContext(ctx, "tmux", append([]string{"-N", "-L", socket, "-f", "/dev/null"}, args...)...).CombinedOutput()
		if err != nil {
			t.Fatalf("tmux %v: %v\n%s", args, err, out)
		}
		return string(out)
	}
	quoted := []string{"TERM=xterm-256color", "NO_COLOR=1", "HUG_USER=tui-reader"}
	for _, e := range env {
		if !strings.HasPrefix(e, "HUG_USER=") {
			quoted = append(quoted, e)
		}
	}
	for i := range quoted {
		quoted[i] = shellQuote(quoted[i])
	}
	tmux("new-session", "-d", "-s", "r", "-x", "120", "-y", "30", "-c", repo, "env "+strings.Join(quoted, " ")+" "+shellQuote(binary)+" task tui")
	tmux("set-option", "-s", "exit-empty", "off")
	defer exec.Command("tmux", "-N", "-L", socket, "kill-session", "-t", "r").Run()
	capture := func() string { return tmux("capture-pane", "-p", "-t", "r:0.0") }
	// wait returns how long the screen took to show want.
	wait := func(want string, within time.Duration) time.Duration {
		t.Helper()
		start := time.Now()
		for time.Since(start) < within {
			if strings.Contains(capture(), want) {
				return time.Since(start)
			}
			time.Sleep(50 * time.Millisecond)
		}
		t.Fatalf("screen never showed %q within %v\n%s", want, within, capture())
		return 0
	}
	keys := func(k ...string) { tmux(append([]string{"send-keys", "-t", "r:0.0"}, k...)...) }
	type_ := func(text string) { tmux("send-keys", "-t", "r:0.0", "-l", text) }
	wait("HUG · repo · overview", 10*time.Second)
	keys("v", "l") // the list (vl, qh9ep1)
	wait("HUG · repo · list", 5*time.Second)
	// List: another terminal renames Gamma.
	run(binary, "task", "set", ids["Gamma"], "--title", "Gamma renamed by another terminal")
	took := wait("Gamma renamed by another terminal", 3*time.Second)
	t.Logf("list: shown %v after hug task set returned, no key pressed", took)
	// Detail: open Beta, another terminal adds a note.
	keys("j", "Enter")
	wait("HUG · repo · list ›", 5*time.Second) // the breadcrumb band (e3sevr)
	run(binary, "task", "note", ids["Beta"], "note from another terminal")
	// The Notes are a long section, folded to their count (8a0ybb): the
	// new one shows as NOTES (1), and Enter on that heading shows it.
	took = wait("› NOTES (1)", 3*time.Second)
	t.Logf("detail: shown %v after hug task note returned, no key pressed", took)
	for n := 0; n < 8 && !strings.Contains(capture(), "▸ NOTES (1)"); n++ {
		keys("BTab")
		time.Sleep(100 * time.Millisecond)
	}
	keys("Enter")
	wait("note from another terminal", 3*time.Second)
	if !strings.Contains(capture(), "Beta") {
		t.Fatal("the open detail is no longer Beta\n" + capture())
	}
	// A prompt stays open and keeps what was typed while a change arrives;
	// the change shows once it closes.
	keys("Escape")
	wait("HUG · repo · list", 5*time.Second)
	keys("/")
	type_("Alp")
	wait("/ Alp", 5*time.Second)
	run(binary, "task", "set", ids["Alpha"], "--title", "Alpha renamed while typing")
	time.Sleep(1500 * time.Millisecond)
	if screen := capture(); !strings.Contains(screen, "/ Alp") || strings.Contains(screen, "Alpha renamed while typing") {
		t.Fatal("the prompt was interrupted or reloaded under it\n" + screen)
	}
	keys("Escape")
	wait("Alpha renamed while typing", 3*time.Second)
	keys("q")
	t.Log("tmux: auto-refresh PASS (3 changes by other processes shown without a key press; the prompt kept its text)")
}
