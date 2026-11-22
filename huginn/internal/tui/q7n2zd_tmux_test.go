package tui

import (
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/veridit/hug/internal/tmuxtest"
)

// q7n2zd checks 2-4 in a real terminal: the hug binary in tmux at 100x30
// with a fake clipboard. HUG_TEST_CLIPBOARD names a command hug runs as
// "CMD image" and "CMD text" in place of osascript/pbpaste, wl-paste or
// xclip; here a script that prints clip.png and clip.txt from the test's
// folder, failing when one is missing. The real clipboard is never read.
func TestTmuxPasteClipboard(t *testing.T) {
	if testing.Short() {
		t.Skip("real tmux acceptance disabled by -short")
	}
	if runtime.GOOS == "windows" {
		t.Skip("tmux and /bin/sh required")
	}
	if _, err := exec.LookPath("tmux"); err != nil {
		t.Skip("tmux required")
	}
	loadGuard(t)
	root, err := os.MkdirTemp(os.Getenv("JCODE_SCRATCH_DIR"), "hug-q7n2zd-")
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
	repo := slotRepo(t, filepath.Join(root, "b"), binary, false)
	clipDir := filepath.Join(root, "clip")
	if err := os.MkdirAll(clipDir, 0o755); err != nil {
		t.Fatal(err)
	}
	fake := filepath.Join(root, "fake-clipboard")
	script := "#!/bin/sh\ncase \"$1\" in\nimage) [ -f " + shellQuote(filepath.Join(clipDir, "clip.png")) + " ] || exit 1; exec cat " + shellQuote(filepath.Join(clipDir, "clip.png")) + " ;;\ntext) exec cat " + shellQuote(filepath.Join(clipDir, "clip.txt")) + " ;;\nesac\nexit 2\n"
	if err := os.WriteFile(fake, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	setClip := func(png []byte, text string) {
		t.Helper()
		_ = os.Remove(filepath.Join(clipDir, "clip.png"))
		if png != nil {
			if err := os.WriteFile(filepath.Join(clipDir, "clip.png"), png, 0o644); err != nil {
				t.Fatal(err)
			}
		}
		if err := os.WriteFile(filepath.Join(clipDir, "clip.txt"), []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	pastes := func() []string {
		var out []string
		_ = filepath.WalkDir(filepath.Join(repo, ".hug", "tasks"), func(p string, d os.DirEntry, err error) error {
			if err == nil && !d.IsDir() && strings.HasPrefix(d.Name(), "paste-") {
				rel, _ := filepath.Rel(filepath.Join(repo, ".hug", "tasks"), p)
				out = append(out, filepath.ToSlash(rel))
			}
			return nil
		})
		return out
	}

	socket := tmuxtest.Label("q7n2zd")
	defer startTmuxServer(t, socket)()
	tmux := tmuxOn(t, socket)
	tmux("set-option", "-s", "exit-empty", "off")
	env := slotEnv(filepath.Join(root, "b", "home"))
	// Colour on, for the frame; the fake clipboard in place of the real
	// one. SSH would refuse before the fake is asked (check 4).
	env = append(env, "NO_COLOR=", "COLORTERM=truecolor", "HUG_TEST_CLIPBOARD="+fake)
	quoted := make([]string, len(env))
	for i, e := range env {
		quoted[i] = shellQuote(e)
	}
	target := "paste:0.0"
	tmux("new-session", "-d", "-s", "paste", "-x", "100", "-y", "30", "-c", repo, "/bin/sh -c "+shellQuote("env -u SSH_CONNECTION -u SSH_TTY -u SSH_CLIENT "+strings.Join(quoted, " ")+" "+shellQuote(binary)+" tui"))
	tmux("set-option", "-t", "paste", "status", "off")
	capture := func() string { return normalizeCapture(tmux("capture-pane", "-p", "-t", target)) }
	wait := func(want *regexp.Regexp) string {
		t.Helper()
		last, stable := "", 0
		for deadline := time.Now().Add(15 * time.Second); time.Now().Before(deadline); time.Sleep(50 * time.Millisecond) {
			current := capture()
			if want.MatchString(current) && current == last {
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
	lit := regexp.QuoteMeta
	wait(regexp.MustCompile(lit("HUG · repo · overview")))

	// Check 2: in a task V attaches at once. Text with colour is .ans, and
	// FILES shows it in the next frame.
	send("#", "cccccc", "Enter")
	wait(regexp.MustCompile(`› cccccc`))
	setClip(nil, "\x1b[1;32mPASS\x1b[0m ok  github.com/veridit/hug 0.4s\n\x1b[31mFAIL\x1b[0m one\n")
	send("V")
	s := wait(regexp.MustCompile(`clipboard text \(2 lines\) · attached as paste-\d{4}\.ans to cccccc`))
	ans := regexp.MustCompile(`paste-\d{4}\.ans`).FindString(s)
	files := regexp.MustCompile(`(?s)FILES.*` + lit(ans))
	if !files.MatchString(s) {
		t.Fatalf("FILES does not show %s:\n%s", ans, s)
	}
	if got := pastes(); len(got) != 1 || !strings.HasSuffix(got[0], "/"+ans) {
		t.Fatalf("on disk: %v", got)
	}
	if dir := os.Getenv("HUG_TUI_ARTIFACTS"); dir != "" {
		frame := tmux("capture-pane", "-p", "-e", "-t", target)
		_ = os.MkdirAll(filepath.Join(dir, "q7n2zd"), 0o755)
		_ = os.WriteFile(filepath.Join(dir, "q7n2zd", "task-after-V-100x30.ans"), []byte(frame), 0o644)
	}
	// A PNG on the clipboard is the image, not its text.
	setClip(png1, "not this\n")
	send("V")
	s = wait(regexp.MustCompile(`clipboard image \(\d+ bytes\) · attached as paste-\d{4}\.png to cccccc`))
	// Plain text is .txt.
	setClip(nil, "plain words\n")
	send("V")
	s = wait(regexp.MustCompile(`attached as paste-\d{4}\.txt to cccccc`))
	if got := pastes(); len(got) != 3 {
		t.Fatalf("three pastes, on disk: %v", got)
	}

	// Check 4: an empty clipboard says so with a fix: line; nothing more.
	setClip(nil, "")
	send("V")
	wait(regexp.MustCompile(`(?s)the clipboard is empty.*fix: hug task attach cccccc FILE`))
	if got := pastes(); len(got) != 3 {
		t.Fatalf("an empty clipboard attached: %v", got)
	}

	// Check 3: in a list V asks where, as S does; Esc attaches nothing.
	send("Escape")
	wait(regexp.MustCompile(`HUG · repo · overview`))
	setClip(nil, "from a chat\n")
	send("V")
	s = wait(regexp.MustCompile(`Attach the clipboard to: [0-9a-z]{6}`))
	if !strings.Contains(s, "select") || !strings.Contains(s, "create") {
		t.Fatalf("the list's prompt lacks select and create:\n%s", s)
	}
	send("Escape")
	wait(regexp.MustCompile(`clipboard text \(1 lines\) · not attached`))
	if got := pastes(); len(got) != 3 {
		t.Fatalf("Esc attached: %v", got)
	}
	// Enter attaches to the offered row.
	send("V")
	wait(regexp.MustCompile(`Attach the clipboard to:`))
	send("Enter")
	wait(regexp.MustCompile(`attached as paste-\d{4}\.txt to [0-9a-z]{6}`))
	if got := pastes(); len(got) != 4 {
		t.Fatalf("Enter in the list: %v", got)
	}
}
