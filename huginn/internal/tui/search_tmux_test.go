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

// 5cjy88 checks 3, 4, 6 and 7 in a real terminal: the hug binary in tmux at
// 120x30 on a board shaped like Jørgen's statbus case (2026-11-21 08:34Z).
// k5q120 is assigned with a lint finding, so in the overview it is in LINT
// only; hmgadt (done, RECENTLY DONE) names it in its title and notes,
// hm2dne (done) in its notes. On v0.7.93 typing '/ k5q120' kept the cursor
// on hmgadt, ↑↓ did nothing and Enter jumped back to the board.
//
// With FRAMES_DIR set, the frames (tmux capture-pane -e) of typing the id,
// Enter on the results and Esc after Enter are written there.
func TestTmuxSearchResults(t *testing.T) {
	if testing.Short() {
		t.Skip("real tmux acceptance disabled by -short")
	}
	if _, err := exec.LookPath("tmux"); err != nil {
		t.Skip("tmux required")
	}
	loadGuard(t)
	root, err := os.MkdirTemp(os.Getenv("JCODE_SCRATCH_DIR"), "hug-5cjy88-")
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
	repo := searchRepo(t, root, binary)
	socket := tmuxtest.Label("5cjy88")
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
	env := slotEnv(filepath.Join(root, "home"))
	if os.Getenv("FRAMES_DIR") != "" {
		// The frames are for people: colour on.
		for i, e := range env {
			if e == "NO_COLOR=1" {
				env[i] = "NO_COLOR="
			}
		}
	}
	quoted := make([]string, len(env))
	for i, e := range env {
		quoted[i] = shellQuote(e)
	}
	target := "s:0.0"
	tmux("new-session", "-d", "-s", "s", "-x", "120", "-y", "30", "-c", repo, "/bin/sh -c "+shellQuote("env "+strings.Join(quoted, " ")+" "+shellQuote(binary)+" tui"))
	defer exec.Command("tmux", "-N", "-L", socket, "kill-session", "-t", "s").Run()
	tmux("set-option", "-t", "s", "status", "off")
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
		t.Fatalf("never showed %q:\n%s", want, last)
		return ""
	}
	send := func(keys ...string) {
		for _, k := range keys {
			tmux("send-keys", "-t", target, k)
		}
	}
	frame := func(name string) {
		if dir := os.Getenv("FRAMES_DIR"); dir != "" {
			if err := os.WriteFile(filepath.Join(dir, name), []byte(tmux("capture-pane", "-e", "-p", "-t", target)), 0o644); err != nil {
				t.Fatal(err)
			}
		}
	}
	// The selected row's id: the row drawn with '>' (a focused section's
	// heading has one too, and no id).
	row := regexp.MustCompile(`(?m)^>\s+[·!]?([0-9a-z]{6})\s`)
	selected := func(screen string) string {
		if m := row.FindStringSubmatch(screen); m != nil {
			return m[1]
		}
		return ""
	}
	isSel := func(screen, id string) bool { return selected(screen) == id }

	wait("HUG · repo")
	// The cursor in RECENTLY DONE, on hmgadt, as on Jørgen's board.
	send("l", "l", "l", "l")
	if s := wait("RECENTLY DONE"); !isSel(s, "hmgadt") && !isSel(s, "hm2dne") {
		send("j")
	}
	// Check 1, typing: the task with that id, in LINT.
	send("/", "k5q120")
	s := wait("found: / k5q120 · 3 matches")
	frame("1-typing-k5q120.ansi")
	if !isSel(s, "k5q120") {
		t.Fatalf("typing k5q120 selected %q, want k5q120\n%s", selected(s), s)
	}
	// Check 3 and 6, Enter: the results stay; j/k walk them all.
	send("Enter")
	s = wait("found: / k5q120 · 3 matches · Esc all tasks")
	frame("2-enter-results.ansi")
	if !isSel(s, "k5q120") || !strings.Contains(s, "j/k match") {
		t.Fatalf("Enter: %q\n%s", selected(s), s)
	}
	// Check 4, Esc after Enter: all tasks, k5q120 selected and in view.
	send("Escape")
	s = wait("found k5q120, row 1 of 1 in LINT")
	frame("3-esc-after-enter.ansi")
	if !isSel(s, "k5q120") || !strings.Contains(s, "filter: none") {
		t.Fatalf("Esc after Enter: %q\n%s", selected(s), s)
	}
	send("/", "k5q120", "Enter")
	wait("found: / k5q120")
	send("<")
	var walked []string
	for i := 0; i < 3; i++ {
		if i > 0 {
			send("j")
		}
		s = wait("match " + string(rune('1'+i)) + " of 3")
		walked = append(walked, selected(s))
	}
	if walked[2] != "k5q120" || !strings.Contains(s, "match 3 of 3 in LINT") {
		t.Fatalf("j walked %v, want the matches ending in k5q120 in LINT\n%s", walked, s)
	}
	// Enter opens; Esc comes back to the results.
	send("Enter")
	s = wait("› k5q120")
	send("Escape")
	s = wait("found: / k5q120")
	// Esc after a pick: all tasks, the pick selected and in view.
	send("k", "Escape")
	s = wait("found " + walked[1] + ", row")
	if !isSel(s, walked[1]) || strings.Contains(s, "found: /") {
		t.Fatalf("Esc after Enter: %q\n%s", selected(s), s)
	}
	// Esc while typing: back on the task before /.
	send("/", "k5q120", "Escape")
	s = wait("All tasks")
	if !isSel(s, walked[1]) {
		t.Fatalf("Esc while typing: %q, want %s again\n%s", selected(s), walked[1], s)
	}
	// Check 7, Tab: the chip lasts across a task and a view, Esc keeps it.
	send("/", "k5q120", "Tab")
	wait("filter: [/ k5q120]")
	send("Enter")
	wait("› ")
	send("Escape", "v", "b")
	wait("filter: [/ k5q120]")
	send("Escape")
	s = wait("HUG · repo · overview")
	if !strings.Contains(s, "filter: [/ k5q120]") {
		t.Fatalf("Esc cleared the chip\n%s", s)
	}
	send("F")
	s = wait("filter: none")
}

// searchRepo is a git repo under dir shaped as the statbus case: k5q120
// (assigned, a two-sentence Goal: in LINT only), hmgadt (done, names it in
// its title and notes), hm2dne (done, names it in its notes), and three
// tasks that name nothing.
func searchRepo(t *testing.T, dir, binary string) string {
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
	for i, tk := range []struct{ id, title, state, assignee, goal, notes string }{
		{"aaaaaa", "Alpha", "todo", "", "g", ""},
		{"bbbbbb", "Bravo", "todo", "", "g", ""},
		{"k5q120", "Import the statbus units", "todo", "crab", "One sentence. Then another.", ""},
		{"hmgadt", "Follow-up to k5q120", "done", "crab", "g", "carried over to k5q120"},
		{"hm2dne", "Logging", "done", "crab", "g", "see k5q120"},
		{"cccccc", "Charlie", "done", "crab", "g", ""},
	} {
		check := " "
		if tk.state == "done" {
			check = "x"
		}
		assignee := "[]"
		if tk.assignee != "" {
			assignee = `["` + tk.assignee + `"]`
		}
		notes := ""
		if tk.notes != "" {
			notes = "\n## Notes\n" + tk.notes + "\n"
		}
		body := "---\nid: \"" + tk.id + "\"\ntitle: \"" + tk.title + "\"\nstate: \"" + tk.state + "\"\ntype: \"feature\"\npriority: \"high\"\nlabels: []\nassignee: " + assignee + "\ncreated: \"2026-11-01\"\nupdated: \"2026-11-0" + string(rune('1'+i)) + "\"\n---\n\n## Goal\n" + tk.goal + "\n\n## Purpose\np\n\n## Principle\nq\n\n## Acceptance\n- [" + check + "] one\n" + notes
		if err := os.WriteFile(filepath.Join(tasks, "0"+string(rune('a'+i))+"."+tk.id+".t.md"), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	run("git", "-C", tasks, "add", "-A")
	run("git", "-C", tasks, "commit", "-q", "-m", "fixture tasks")
	return repo
}
