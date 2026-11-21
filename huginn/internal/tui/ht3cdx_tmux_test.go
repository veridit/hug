package tui

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/veridit/hug/internal/tmuxtest"
)

// ht3cdx checks 2-5 in a real terminal: the hug binary in tmux at 120x30
// on five tasks. aaaaaa is taken (IN PROGRESS, DOING), bbbbbb dropped
// (RECENTLY DONE, DROPPED), cccccc..eeeeee todo (NEXT UP, TODO). In the
// overview and on the board, a filter (the picker's Enter, ': todo', ':
// state X'), a sort from the s picker, a task created with S c, # and ':
// goto' leave the view as it was (Jørgen 2026-11-20: "I think I should
// just keep the view I'm in"). On v0.7.96 the filters, S c, # and goto
// went to the list; the sort kept the view.
func TestTmuxActionsKeepTheView(t *testing.T) {
	if testing.Short() {
		t.Skip("real tmux acceptance disabled by -short")
	}
	if _, err := exec.LookPath("tmux"); err != nil {
		t.Skip("tmux required")
	}
	loadGuard(t)
	root, err := os.MkdirTemp(os.Getenv("JCODE_SCRATCH_DIR"), "hug-ht3cdx-")
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
	socket := tmuxtest.Label("ht3cdx")
	defer startTmuxServer(t, socket)()
	tmuxOn(t, socket)("set-option", "-s", "exit-empty", "off")
	for _, view := range []string{"overview", "board"} {
		for _, check := range []string{"2 filter", "3 sort", "4 create", "5 jump"} {
			t.Run(view+"/"+check, func(t *testing.T) {
				keepView(t, tmuxOn(t, socket), socket, filepath.Join(root, view+"-"+check[:1]), binary, view, check[:1])
			})
		}
	}
}

// keepView runs check (2-5) of ht3cdx in view on a fresh board, in its own
// tmux session.
func keepView(t *testing.T, tmux func(...string) string, socket, dir, binary, view, check string) {
	// The pane each task is in, by view: the breadcrumb's second crumb.
	panes := map[string]map[string]string{
		"overview": {"bbbbbb": "RECENTLY DONE", "dddddd": "NEXT UP"},
		"board":    {"bbbbbb": "DROPPED", "dddddd": "TODO"},
	}[view]
	repo := viewRepo(t, dir, binary)
	env := slotEnv(filepath.Join(dir, "home"))
	// S copies the screen: never to this machine's clipboard.
	env = append(env, "SSH_CONNECTION=ht3cdx-test")
	quoted := make([]string, len(env))
	for i, e := range env {
		quoted[i] = shellQuote(e)
	}
	name := "v" + view + check
	target := name + ":0.0"
	tmux("new-session", "-d", "-s", name, "-x", "120", "-y", "30", "-c", repo, "/bin/sh -c "+shellQuote("env "+strings.Join(quoted, " ")+" "+shellQuote(binary)+" tui"))
	defer exec.Command("tmux", "-N", "-L", socket, "kill-session", "-t", name).Run()
	tmux("set-option", "-t", name, "status", "off")
	capture := func() string { return normalizeCapture(tmux("capture-pane", "-p", "-t", target)) }
	wait := func(want string) string {
		t.Helper()
		last, stable := "", 0
		for deadline := time.Now().Add(15 * time.Second); time.Now().Before(deadline); time.Sleep(50 * time.Millisecond) {
			current := capture()
			if strings.Contains(current, want) && current == last {
				if stable++; stable >= 2 {
					return current
				}
			} else {
				stable = 0
			}
			last = current
		}
		t.Fatalf("%s: never showed %q:\n%s", view, want, last)
		return ""
	}
	send := func(keys ...string) {
		for _, k := range keys {
			tmux("send-keys", "-t", target, k)
		}
	}
	first := func(screen string) string {
		line, _, _ := strings.Cut(screen, "\n")
		return line
	}
	// The rows view: its name heads the first line, with no crumb after
	// it (a task's first line is its breadcrumb).
	inView := func(what, screen string) {
		t.Helper()
		if line := first(screen); !strings.HasPrefix(line, "HUG · repo · "+view+" ") || strings.Contains(line, "›") {
			t.Fatalf("%s: after %s the view is not the %s:\n%s", view, what, view, screen)
		}
	}
	selected := func(id, screen string) {
		t.Helper()
		// A board column after the first starts mid-line.
		if !regexp.MustCompile(`(?m)(?:^|\s)>\s+[·!]?` + id + `\s`).MatchString(screen) {
			t.Fatalf("%s: %s is not the selected row:\n%s", view, id, screen)
		}
	}
	// A task open over the view: 'view › PANE › ID' (24403z).
	opened := func(what, id string) {
		t.Helper()
		crumb := "HUG · repo · " + view + " › " + panes[id] + " › " + id
		if screen := wait("› " + id); !strings.HasPrefix(first(screen), crumb) {
			t.Fatalf("%s: %s did not open %s over the %s (want %q):\n%s", view, what, id, view, crumb, screen)
		}
	}
	s := wait("HUG · repo · overview")
	if view == "board" {
		send("v", "b")
		s = wait("HUG · repo · board")
	}
	inView("start", s)
	switch check {
	case "2":
		// The filter picker's Enter, ': todo', ': state X'. A section
		// the filter empties says so, with the filter.
		send("f", "s", "t", "o", "d", "o", "Enter")
		s = wait("[state:todo ×]")
		inView("the filter picker's Enter", s)
		if view == "overview" && !strings.Contains(s, "IN PROGRESS · assigned to  [0] · None with state:todo") {
			t.Fatalf("an empty section does not name the filter:\n%s", s)
		}
		send("F")
		inView("F", wait("filter: none"))
		send(":", "todo", "Enter")
		inView("': todo'", wait("[state:todo ×]"))
		send("F")
		wait("filter: none")
		send(":", "state dropped", "Enter")
		s = wait("[state:dropped ×]")
		inView("': state dropped'", s)
		// The focus goes to DROPPED, the first column with rows (crab
		// 15:25Z); at 120 columns the board shows DEPENDS, DONE, DROPPED.
		want := map[string]string{"overview": "NEXT UP · rank order  [0] · None with state:dropped", "board": "DONE [0] · None with state:dropped    > DROPPED [1]"}[view]
		if !strings.Contains(s, want) {
			t.Fatalf("%s: an empty section does not name the filter (want %q):\n%s", view, want, s)
		}
	case "3":
		// A sort from the s picker, and back to rank order.
		send("s", "p", "Enter")
		inView("a sort from the s picker", wait("sort: priority"))
		send("s", "r")
		inView("s r", wait("sort: rank"))
	case "4":
		// S, c and a title create the task and open it over the view;
		// Esc goes back with it selected.
		send("S")
		wait("Attach this screen to:")
		send("c")
		wait("new task with this screen, title:")
		tmux("send-keys", "-t", target, "-l", "Foxtrot task")
		send("Enter")
		s = wait("created ")
		id := regexp.MustCompile(`created ([0-9a-z]{6})`).FindStringSubmatch(s)
		if id == nil {
			t.Fatalf("%s: no new id:\n%s", view, s)
		}
		panes[id[1]] = map[string]string{"overview": "NEXT UP", "board": "TODO"}[view]
		opened("S c", id[1])
		send("Escape")
		s = wait("HUG · repo · " + view + " ")
		inView("Esc from the new task", s)
		selected(id[1], s)
	case "5":
		// # and ': goto' open over the view; Esc goes back to it.
		send("#", "dddddd", "Enter")
		opened("# dddddd", "dddddd")
		send("Escape")
		s = wait("HUG · repo · " + view + " ")
		inView("Esc from # dddddd", s)
		selected("dddddd", s)
		send(":", "goto bbbbbb", "Enter")
		opened("': goto bbbbbb'", "bbbbbb")
		send("Escape")
		s = wait("HUG · repo · " + view + " ")
		inView("Esc from ': goto bbbbbb'", s)
		selected("bbbbbb", s)
	}
}

// tmuxOn runs tmux on the test server socket for t.
func tmuxOn(t *testing.T, socket string) func(...string) string {
	return func(args ...string) string {
		t.Helper()
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		out, err := exec.CommandContext(ctx, "tmux", append([]string{"-N", "-L", socket, "-f", "/dev/null"}, args...)...).CombinedOutput()
		if err != nil {
			t.Fatalf("tmux %v: %v\n%s", args, err, out)
		}
		return string(out)
	}
}

// viewRepo is slotRepo's five tasks with aaaaaa taken by jhf and bbbbbb
// dropped.
func viewRepo(t *testing.T, dir, binary string) string {
	t.Helper()
	repo := slotRepo(t, dir, binary, false)
	env := slotEnv(filepath.Join(dir, "home"))
	for _, args := range [][]string{{"take", "aaaaaa", "--model", "human"}, {"drop", "bbbbbb", "not needed"}} {
		cmd := exec.Command(binary, args...)
		cmd.Dir = repo
		cmd.Env = append(os.Environ(), env...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("%v: %v\n%s", args, err, out)
		}
	}
	return repo
}
