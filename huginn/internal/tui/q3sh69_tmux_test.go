package tui

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/charmbracelet/x/ansi"
	"github.com/veridit/hug/internal/tmuxtest"
)

// q3sh69 checks 1, 2 and 4 in a real terminal: the hug binary in tmux at
// 120x30, in colour. A task asks jhf about its check 2 and has a page far
// taller than the screen.
//
//   - check 2: in the task's page, with the asked check focused, > and End
//     go to the bottom, < and Home to the top, and the check keeps focus;
//   - checks 1 and 4: the focus sign pulses (bold, then not) in the list
//     and in the page, never underlined; while / takes text and while the
//     answer box is open it is on and still, and the box cursor blinks.
//
// With FRAMES_DIR set, two frames per state (capture-pane -e, sign or
// cursor on, then off) are written there: list, page, answer box.
func TestTmuxFocusPulse(t *testing.T) {
	if testing.Short() {
		t.Skip("real tmux acceptance disabled by -short")
	}
	if _, err := exec.LookPath("tmux"); err != nil {
		t.Skip("tmux required")
	}
	loadGuard(t)
	root, err := os.MkdirTemp(os.Getenv("JCODE_SCRATCH_DIR"), "hug-q3sh69-")
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
	repo := pulseRepo(t, root, binary)
	socket := tmuxtest.Label("q3sh69")
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
	for i, e := range env {
		if e == "NO_COLOR=1" {
			env[i] = "NO_COLOR=" // the pulse is a colour and a weight
		}
	}
	// CI=true makes termenv take tmux for no terminal (Ascii, no bold at
	// all); CLICOLOR_FORCE keeps the colour, as TestTmuxGolden does.
	env = append(env, "COLORTERM=truecolor", "COLORFGBG=15;0", "CLICOLOR_FORCE=1")
	quoted := make([]string, len(env))
	for i, e := range env {
		quoted[i] = shellQuote(e)
	}
	target := "p:0.0"
	tmux("new-session", "-d", "-s", "p", "-x", "120", "-y", "30", "-c", repo, "/bin/sh -c "+shellQuote("env "+strings.Join(quoted, " ")+" "+shellQuote(binary)+" tui"))
	defer exec.Command("tmux", "-N", "-L", socket, "kill-session", "-t", "p").Run()
	tmux("set-option", "-t", "p", "status", "off")
	capture := func() string { return normalizeCapture(tmux("capture-pane", "-p", "-t", target)) }
	colour := func() string { return tmux("capture-pane", "-e", "-p", "-t", target) }
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
	// at is the cell of the colour capture's line holding want at the
	// column of the first of signs on it (or of want itself).
	at := func(screen, want, signs string) (cell, bool) {
		for _, l := range strings.Split(screen, "\n") {
			plain := ansi.Strip(l)
			i := strings.Index(plain, want)
			if i < 0 {
				continue
			}
			if signs != "" {
				if i = strings.IndexAny(plain, signs); i < 0 {
					continue // not the focused row
				}
			}
			return sgrAt(l, utf8.RuneCountInString(plain[:i])), true
		}
		return cell{}, false
	}
	// watch captures the screen for d, and returns a capture where pick
	// says on, one where it says off, and how many of each it saw.
	type seen struct {
		on, off      string
		ons, offs, n int
	}
	watch := func(d time.Duration, pick func(string) (on, ok bool)) seen {
		var s seen
		for deadline := time.Now().Add(d); time.Now().Before(deadline); time.Sleep(60 * time.Millisecond) {
			screen := colour()
			on, ok := pick(screen)
			if !ok {
				continue
			}
			s.n++
			if on {
				s.ons++
				if s.on == "" {
					s.on = screen
				}
			} else {
				s.offs++
				if s.off == "" {
					s.off = screen
				}
			}
		}
		return s
	}
	frames := func(state string, s seen) {
		dir := os.Getenv("FRAMES_DIR")
		if dir == "" {
			return
		}
		for name, frame := range map[string]string{state + "-on.ansi": s.on, state + "-off.ansi": s.off} {
			if err := os.WriteFile(filepath.Join(dir, name), []byte(frame), 0o644); err != nil {
				t.Fatal(err)
			}
		}
	}
	// sign is the focus sign's phase on the line holding want: bold is on.
	sign := func(want, signs string) func(string) (bool, bool) {
		return func(screen string) (bool, bool) {
			c, ok := at(screen, want, signs)
			if ok && (c.underline || c.bg) {
				t.Fatalf("the sign on %q is underlined or filled:\n%q", want, screen)
			}
			return c.bold, ok
		}
	}
	pulses := func(where string, s seen) {
		t.Helper()
		if s.ons == 0 || s.offs == 0 {
			t.Fatalf("%s: over %d captures the sign was on %d times, off %d: it does not pulse", where, s.n, s.ons, s.offs)
		}
	}
	still := func(where string, s seen) {
		t.Helper()
		if s.n == 0 || s.offs != 0 {
			t.Fatalf("%s: over %d captures the sign was off %d times: it moves while text has the cursor", where, s.n, s.offs)
		}
	}

	wait("HUG · repo")
	send("v", "l")
	s := wait("HUG · repo · list")
	// Check 4: the list's > pulses.
	list := watch(2500*time.Millisecond, sign("Task ", ">⋗"))
	pulses("list", list)
	frames("list", list)
	// While / takes text the sign is still.
	send("/", "T")
	wait("found: / T")
	still("typing in /", watch(1500*time.Millisecond, sign("Task ", ">⋗")))
	send("Escape")
	wait("state=!done · F clears") // the list's default chip stays (7zg9mf)
	// The task's page: the asked check has the focus and its > pulses.
	send("#", "tttttt", "Enter")
	s = wait("> ? ☐ 2. second")
	if !strings.Contains(s, "ACCEPTANCE") || strings.Contains(s, "note 80") {
		t.Fatalf("the page does not open at its top:\n%s", s)
	}
	top := s
	page := watch(2500*time.Millisecond, sign("> ? ☐ 2. second", ">"))
	pulses("page", page)
	frames("page", page)
	// Check 2: > and End to the bottom, < and Home to the top, the check
	// keeping the focus.
	for _, k := range [][2]string{{">", "<"}, {"End", "Home"}} {
		send(k[0])
		bottom := wait("note 80")
		if strings.Contains(bottom, "ACCEPTANCE") {
			t.Fatalf("%s: not at the bottom:\n%s", k[0], bottom)
		}
		send("C-d")
		time.Sleep(300 * time.Millisecond)
		if again := capture(); again != bottom {
			t.Fatalf("%s left the page short of its bottom: Ctrl-d still scrolled\n%s", k[0], again)
		}
		send(k[1])
		if back := wait("> ? ☐ 2. second"); back != top {
			t.Fatalf("%s: not back at the top, or the focus moved:\n%s\nwant\n%s", k[1], back, top)
		}
	}
	// Check 1: the answer box open, its cursor blinks and the > is still.
	send("a")
	wait("answer:")
	box := watch(2500*time.Millisecond, func(screen string) (bool, bool) {
		// Off, the cursor's cell is a plain space, which capture-pane
		// trims: the line then ends at "answer:". On, an underlined cell
		// follows it.
		l := screenLine(screen, "answer:")
		if l == "" {
			return false, false
		}
		plain := ansi.Strip(l)
		col := utf8.RuneCountInString(plain[:strings.Index(plain, "answer:")]) + len("answer: ")
		return sgrAt(l, col).underline, true
	})
	if box.ons == 0 || box.offs == 0 {
		t.Fatalf("the box cursor does not blink: on %d, off %d of %d\n%q", box.ons, box.offs, box.n, screenLine(box.on+box.off, "answer: "))
	}
	frames("answer-box", box)
	still("answer box open", watch(1500*time.Millisecond, sign("> ? ☐ 2. second", ">")))
	send("Escape")
	pulses("page after the box", watch(2000*time.Millisecond, sign("> ? ☐ 2. second", ">")))
}

// pulseRepo is a git repo under dir with tttttt, whose check 2 crab asks
// jhf about, and Notes of 80 lines, so its page is far taller than 30
// rows; and two more tasks for a list.
func pulseRepo(t *testing.T, dir, binary string) string {
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
	var notes strings.Builder
	for i := 1; i <= 80; i++ {
		fmt.Fprintf(&notes, "- note %d\n", i)
	}
	for i, id := range []string{"aaaaaa", "tttttt", "zzzzzz"} {
		body := "---\nid: \"" + id + "\"\ntitle: \"Task " + id + "\"\nstate: \"todo\"\ntype: \"feature\"\npriority: \"high\"\nlabels: []\nassignee: []\ncreated: \"2026-11-01\"\nupdated: \"2026-11-01\"\n---\n\n## Goal\ng\n\n## Purpose\np\n\n## Principle\nq\n\n## Acceptance\n- [ ] first\n- [ ] second\n"
		if id == "tttttt" {
			// A long page to scroll: a section of its own, as Notes fold
			// to one line (8a0ybb).
			body += "\n## Log\n" + notes.String()
		}
		if err := os.WriteFile(filepath.Join(tasks, "0"+string(rune('a'+i))+"."+id+".t.md"), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	run("git", "-C", tasks, "add", "-A")
	run("git", "-C", tasks, "commit", "-q", "-m", "fixture tasks")
	run("env", "HUG_USER=crab", binary, "ask", "tttttt", "--to", "jhf", "--check", "2", "is the second check right?")
	return repo
}

// screenLine is the first line of screen whose plain text holds want.
func screenLine(screen, want string) string {
	for _, l := range strings.Split(screen, "\n") {
		if strings.Contains(ansi.Strip(l), want) {
			return l
		}
	}
	return ""
}
