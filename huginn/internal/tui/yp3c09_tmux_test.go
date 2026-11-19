package tui

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/veridit/hug/internal/tmuxtest"
)

// yp3c09 in a real terminal: the hug binary in tmux at 100x30 and 160x45,
// p87ngv with before.md (Jørgen's file, hug/tasks 4cc90adf), an 80-line
// text file and a picture. Space on before.md, M for its source, n to the
// picture and the text: every row between the file's rules and the keys
// says something, none is blank (goldens in testdata/yp3c09/tmux).
func TestTmuxFileViewFillsToTheKeys(t *testing.T) {
	if testing.Short() {
		t.Skip("real tmux acceptance disabled by -short")
	}
	if _, err := exec.LookPath("tmux"); err != nil {
		t.Skip("tmux required")
	}
	loadGuard(t)
	root, err := os.MkdirTemp(os.Getenv("JCODE_SCRATCH_DIR"), "hug-yp3c09-")
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
	home, repo := filepath.Join(root, "home"), filepath.Join(root, "repo")
	for _, p := range []string{home, repo} {
		if err := os.MkdirAll(p, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	env := []string{"HOME=" + home, "XDG_CONFIG_HOME=" + filepath.Join(home, ".config"), "GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_NOSYSTEM=1", "GIT_AUTHOR_NAME=Synthetic", "GIT_AUTHOR_EMAIL=synthetic@example.invalid", "GIT_COMMITTER_NAME=Synthetic", "GIT_COMMITTER_EMAIL=synthetic@example.invalid", "HUG_USER=jhf", "HUG_MODEL=", "HUG_SYNC=off", "NO_COLOR=1", "TERM=xterm-256color", "TERM_PROGRAM=", "LC_TERMINAL=", "KITTY_WINDOW_ID=", "GOMAXPROCS=2"}
	run := func(name string, args ...string) string {
		t.Helper()
		cmd := exec.Command(name, args...)
		cmd.Dir = repo
		cmd.Env = append(os.Environ(), env...)
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("%s %v: %v\n%s", name, args, err, out)
		}
		return string(out)
	}
	run("git", "init", "-q", "-b", "main")
	run("git", "commit", "-q", "--allow-empty", "-m", "fixture")
	run(binary, "init")
	tasks := filepath.Join(repo, ".hug", "tasks")
	long := ""
	for i := 1; i <= 80; i++ {
		long += fmt.Sprintf("line %d of 80\n", i)
	}
	before, err := os.ReadFile("testdata/yp3c09/before.md")
	if err != nil {
		t.Fatal(err)
	}
	before = bytes.ReplaceAll(before, []byte("\r\n"), []byte("\n")) // LF, as on hug's board
	card, err := os.ReadFile("testdata/p87ngv/five-questions.md")
	if err != nil {
		t.Fatal(err)
	}
	for rel, b := range map[string][]byte{"p/p87ngv.md": card, "p/p87ngv/before.md": before, "p/p87ngv/long.txt": []byte(long), "p/p87ngv/shot.png": png1} {
		if e := os.MkdirAll(filepath.Dir(filepath.Join(tasks, rel)), 0o755); e != nil {
			t.Fatal(e)
		}
		if e := os.WriteFile(filepath.Join(tasks, rel), b, 0o644); e != nil {
			t.Fatal(e)
		}
	}
	run("git", "-C", tasks, "add", "-A")
	run("git", "-C", tasks, "commit", "-q", "-m", "p87ngv with files")

	socket := tmuxtest.Label("yp3c09")
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
	quoted := make([]string, len(env))
	for i, e := range env {
		quoted[i] = shellQuote(e)
	}
	tmux("set-option", "-s", "exit-empty", "off")
	for _, size := range [][2]int{{100, 30}, {160, 45}} {
		w, h := size[0], size[1]
		name := fmt.Sprintf("%dx%d", w, h)
		t.Run(name, func(t *testing.T) {
			target := name + ":0.0"
			tmux("new-session", "-d", "-s", name, "-x", fmt.Sprint(w), "-y", fmt.Sprint(h), "-c", repo, "/bin/sh -c "+shellQuote("env "+strings.Join(quoted, " ")+" "+shellQuote(binary)+" tui"))
			defer exec.Command("tmux", "-N", "-L", socket, "kill-session", "-t", name).Run()
			tmux("set-option", "-t", name, "status", "off")
			tmux("resize-window", "-t", name, "-x", fmt.Sprint(w), "-y", fmt.Sprint(h))
			capture := func() string { return normalizeCapture(tmux("capture-pane", "-p", "-t", target)) }
			wait := func(want string) string {
				t.Helper()
				last, stable := "", 0
				for deadline := time.Now().Add(12 * time.Second); time.Now().Before(deadline); time.Sleep(50 * time.Millisecond) {
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
			send := func(keys ...string) { tmux(append([]string{"send-keys", "-t", target}, keys...)...) }
			wait("HUG · repo")
			send("#p87ngv")
			wait("JUMP TO ID")
			send("Enter")
			wait("› p87ngv")
			for i := 0; i < 20 && !strings.Contains(capture(), "▸ FILES"); i++ {
				send("Tab")
				time.Sleep(120 * time.Millisecond)
			}
			wait("> ⎘ before.md")
			var golden []string
			snap := func(view, want string) {
				t.Helper()
				screen := wait(want)
				lines := strings.Split(strings.TrimSuffix(screen, "\n"), "\n")
				if len(lines) != h {
					t.Fatalf("%s: %d rows, want %d", view, len(lines), h)
				}
				rule := strings.Repeat("─", w)
				bottom := -1
				for i := len(lines) - 1; i > 0; i-- {
					if lines[i] == rule {
						bottom = i
						break
					}
				}
				if bottom < 0 {
					t.Fatalf("%s: no bottom rule:\n%s", view, screen)
				}
				for i := bottom + 1; i < len(lines)-1; i++ {
					if strings.TrimSpace(lines[i]) == "" {
						t.Errorf("%s at %s: row %d of %d, between the bottom rule (row %d) and the keys, is blank:\n%s", view, name, i+1, h, bottom+1, screen)
						break
					}
				}
				if bottom != h-2 {
					t.Errorf("%s at %s: the bottom rule is on row %d, want %d, on the keys:\n%s", view, name, bottom+1, h-1, screen)
				}
				if view == "picture" {
					// The keys name the system viewer, Preview on macOS
					// (viewerName): the golden keeps the frame, not the OS.
					lines[len(lines)-1] = "(the picture's keys, naming the system viewer)"
					screen = strings.Join(lines, "\n") + "\n"
				}
				golden = append(golden, "== "+view, screen)
			}
			send("Space")
			snap("markdown", "before.md · 1.8 KiB · 1 of 3")
			send("M")
			snap("source", "M Markdown")
			send("n")
			snap("text", "long.txt · ")
			send("n")
			snap("picture", "shot.png · 67 B")
			goldenFile(t, "testdata/yp3c09/tmux-"+name+".txt", strings.Join(golden, "\n"))
		})
	}
}
