package tui

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/veridit/hug/internal/tmuxtest"
)

// 24403z checks 1 and 7 in a real terminal: the hug binary in tmux at
// 100x30, five tasks Alpha..Echo (aaaaaa..eeeeee) in one pane. An action
// on the open task that moves it out of the pane (an answer in ASKED, a
// drop in a filtered list and in the board's TODO column) leaves the
// reader where they were: n opens the task after it, p the one before,
// Esc lands on that neighbour, and n after the last says so and stays.
// On main the cursor fell to the pane's first row after the reload
// (clamp), so n opened the second row.
func TestTmuxActionKeepsTheSlot(t *testing.T) {
	if testing.Short() {
		t.Skip("real tmux acceptance disabled by -short")
	}
	if _, err := exec.LookPath("tmux"); err != nil {
		t.Skip("tmux required")
	}
	loadGuard(t)
	root, err := os.MkdirTemp(os.Getenv("JCODE_SCRATCH_DIR"), "hug-24403z-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(root)
	binary := filepath.Join(root, "hug")
	build := exec.Command("nice", "-n", "19", "env", "GOMAXPROCS=2", "GOFLAGS=-p=2", "go", "build", "-o", binary, "./cmd/huginn")
	build.Dir = "../.."
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build: %v\n%s", err, out)
	}
	socket := tmuxtest.Label("24403z")
	defer startTmuxServer(t, socket)()
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
	tmux("set-option", "-s", "exit-empty", "off")

	type pane struct {
		name, open  string   // the view keys, the pane's name in messages
		ask         bool     // the tasks ask jhf a question (ASKED)
		setup, act  []string // keys after the view opens; keys of the action
		where, gone string   // "the last in ..."; the moved message's section
	}
	for _, p := range []pane{
		{name: "overview", ask: true, act: []string{"a", "ok", "Enter"}, where: "ASKED", gone: "NEXT UP"},
		// Drop, not take: one ticket in doing at a time (n2hq8p).
		{name: "list", open: "vl", setup: []string{"/", "state:todo", "Tab"}, act: []string{"x", "dup", "Enter", "Enter"}, where: "the list (filtered)", gone: "RECENTLY DONE (outside this filter)"},
		{name: "board", open: "vb", act: []string{"x", "dup", "Enter", "Enter"}, where: "the TODO column", gone: "DROPPED"},
	} {
		t.Run(p.name, func(t *testing.T) {
			repo := slotRepo(t, filepath.Join(root, p.name), binary, p.ask)
			env := slotEnv(filepath.Join(root, p.name, "home"))
			quoted := make([]string, len(env))
			for i, e := range env {
				quoted[i] = shellQuote(e)
			}
			name := "s" + p.name
			target := name + ":0.0"
			tmux("new-session", "-d", "-s", name, "-x", "100", "-y", "30", "-c", repo, "/bin/sh -c "+shellQuote("env "+strings.Join(quoted, " ")+" "+shellQuote(binary)+" tui"))
			defer exec.Command("tmux", "-N", "-L", socket, "kill-session", "-t", name).Run()
			tmux("set-option", "-t", name, "status", "off")
			capture := func() string { return normalizeCapture(tmux("capture-pane", "-p", "-t", target)) }
			wait := func(wants ...string) string {
				t.Helper()
				last, stable := "", 0
				for deadline := time.Now().Add(15 * time.Second); time.Now().Before(deadline); time.Sleep(50 * time.Millisecond) {
					current := capture()
					if slices.ContainsFunc(wants, func(w string) bool { return strings.Contains(current, w) }) && current == last {
						if stable++; stable >= 2 {
							return current
						}
					} else {
						stable = 0
					}
					last = current
				}
				t.Fatalf("%s: never showed %q:\n%s", p.name, wants, last)
				return ""
			}
			send := func(keys ...string) {
				for _, k := range keys {
					tmux("send-keys", "-t", target, k)
				}
			}
			// The open task is the breadcrumb's last crumb, on the first row.
			open := func(id string) {
				t.Helper()
				screen := wait("› " + id)
				if first, _, _ := strings.Cut(screen, "\n"); !strings.Contains(first, "› "+id) {
					t.Fatalf("%s: %s is not the open task:\n%s", p.name, id, screen)
				}
			}
			act := func(id string) {
				t.Helper()
				send(p.act...)
				// Written once the bottom line names it, as main said it
				// ("answered ID", "dropped ID") or as check 4 says it.
				wait("answered "+id, "dropped "+id, id+" moved to "+p.gone)
			}
			wait("HUG · repo")
			if p.open != "" {
				send(p.open)
				wait("HUG · repo · " + p.name)
			}
			send(p.setup...)
			wait("aaaaaa")
			// Open Charlie, the third row, and move it out of the pane.
			send("j", "j", "Enter")
			open("cccccc")
			act("cccccc")
			send("n")
			open("dddddd") // the task after Charlie (main: Bravo, row 1 + 1)
			send("p")
			open("bbbbbb") // Charlie is gone: the one before Delta
			act("bbbbbb")
			send("p")
			open("aaaaaa") // the task before Bravo
			act("aaaaaa")
			send("Escape")
			send("Enter")
			open("dddddd") // Esc landed on the neighbour after Alpha
			send("n")
			open("eeeeee")
			act("eeeeee")
			send("n")
			wait("eeeeee was the last in " + p.where + "; nothing after it")
			open("eeeeee") // n stayed
			send("Escape")
			send("Enter")
			open("dddddd") // Esc landed on the neighbour before Echo
		})
	}
}

// slotEnv is the environment of a synthetic board for jhf.
func slotEnv(home string) []string {
	return []string{"HOME=" + home, "XDG_CONFIG_HOME=" + filepath.Join(home, ".config"), "GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_NOSYSTEM=1", "GIT_AUTHOR_NAME=Synthetic", "GIT_AUTHOR_EMAIL=synthetic@example.invalid", "GIT_COMMITTER_NAME=Synthetic", "GIT_COMMITTER_EMAIL=synthetic@example.invalid", "HUG_USER=jhf", "HUG_MODEL=", "HUG_SYNC=off", "NO_COLOR=1", "TERM=xterm-256color", "TERM_PROGRAM=", "LC_TERMINAL=", "KITTY_WINDOW_ID=", "GOMAXPROCS=2"}
}

// slotRepo is a git repo under dir with five todo tasks, Alpha..Echo
// (aaaaaa..eeeeee); with ask, crab asks jhf a question on each, in order.
func slotRepo(t *testing.T, dir, binary string, ask bool) string {
	t.Helper()
	home, repo := filepath.Join(dir, "home"), filepath.Join(dir, "repo")
	for _, p := range []string{home, repo} {
		if err := os.MkdirAll(p, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	env := slotEnv(home)
	run := func(name string, args ...string) {
		t.Helper()
		cmd := exec.Command(name, args...)
		cmd.Dir = repo
		cmd.Env = append(os.Environ(), env...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("%s %v: %v\n%s", name, args, err, out)
		}
	}
	run("git", "init", "-q", "-b", "main")
	run("git", "commit", "-q", "--allow-empty", "-m", "fixture")
	run(binary, "init")
	tasks := filepath.Join(repo, ".hug", "tasks")
	for _, title := range []string{"Alpha", "Bravo", "Charlie", "Delta", "Echo"} {
		id := strings.Repeat(strings.ToLower(title[:1]), 6)
		body := "---\nid: \"" + id + "\"\ntitle: \"" + title + " task\"\nstate: \"todo\"\ntype: \"feature\"\npriority: \"high\"\nlabels: []\nassignee: []\ncreated: \"2026-11-01\"\nupdated: \"2026-11-01\"\n---\n\n## Goal\ng\n\n## Purpose\np\n\n## Principle\nq\n\n## Acceptance\n- [ ] one\n- [ ] two\n"
		if err := os.WriteFile(filepath.Join(tasks, "0"+id[:1]+"."+id+".t.md"), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	run("git", "-C", tasks, "add", "-A")
	run("git", "-C", tasks, "commit", "-q", "-m", "fixture tasks")
	if ask {
		for _, id := range []string{"aaaaaa", "bbbbbb", "cccccc", "dddddd", "eeeeee"} {
			run("env", "HUG_USER=crab", binary, "ask", id, "--to", "jhf", "pick one?")
		}
	}
	return repo
}
