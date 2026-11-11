package tui

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/veridit/huginn/internal/tui/testfixture"
)

func loadGuard(t *testing.T) {
	t.Helper()
	if runtime.GOOS != "darwin" {
		return
	}
	for {
		out, err := exec.Command("sysctl", "-n", "vm.loadavg").Output()
		if err != nil {
			t.Fatal(err)
		}
		values := regexp.MustCompile(`[0-9]+[.,][0-9]+`).FindAllString(string(out), -1)
		if len(values) != 3 {
			t.Fatalf("cannot read load guard: %s", out)
		}
		load, _ := strconv.ParseFloat(strings.ReplaceAll(values[1], ",", "."), 64)
		if load <= 120 {
			return
		}
		t.Logf("load guard: paused before heavy work, 5-minute load %.2f > 120", load)
		time.Sleep(30 * time.Second)
	}
}
func shellQuote(s string) string { return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'" }
// ansiColour matches any SGR sequence that sets a foreground or background colour.
var ansiColour = regexp.MustCompile(`\x1b\[(?:[0-9;]*;)?(?:3[0-7]|4[0-7]|9[0-7]|10[0-7]|38;|48;)[0-9;]*m`)

func normalizeCapture(s string) string {
	lines := strings.Split(strings.TrimSuffix(s, "\n"), "\n")
	for i := range lines {
		lines[i] = strings.TrimRight(lines[i], " ")
	}
	return strings.Join(lines, "\n") + "\n"
}

// A foreground server gives the test an exact process to wait for and clean up.
// Clients never auto-start a replacement server during startup or shutdown.
func startTmuxServer(t *testing.T, socket string) func() {
	t.Helper()
	server := exec.Command("tmux", "-D", "-L", socket, "-f", "/dev/null")
	server.Stdout, server.Stderr = os.Stderr, os.Stderr
	if err := server.Start(); err != nil {
		t.Fatal(err)
	}
	cleanup := func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		kill := exec.CommandContext(ctx, "tmux", "-N", "-L", socket, "kill-server")
		killErr := kill.Run()
		done := make(chan error, 1)
		go func() { done <- server.Wait() }()
		select {
		case err := <-done:
			if err != nil || killErr != nil {
				t.Errorf("tmux server cleanup: command %v, process %v", killErr, err)
			}
		case <-time.After(3 * time.Second):
			_ = server.Process.Kill()
			<-done
			t.Error("tmux server did not exit after explicit cleanup")
		}
	}
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		err := exec.CommandContext(ctx, "tmux", "-N", "-L", socket, "show-options", "-s", "-v", "exit-empty").Run()
		cancel()
		if err == nil {
			return cleanup
		}
		time.Sleep(25 * time.Millisecond)
	}
	cleanup()
	t.Fatal("foreground tmux server did not become ready")
	return nil
}

// TestTmuxGolden exercises the actual CLI binary, not a copied model renderer.
// Update deliberately with HUGINN_UPDATE_GOLDEN=1, review the diff, then test normally.
func TestTmuxGolden(t *testing.T) {
	if testing.Short() {
		t.Skip("real tmux acceptance disabled by -short")
	}
	if _, err := exec.LookPath("tmux"); err != nil {
		t.Skip("tmux required for terminal golden acceptance")
	}
	loadGuard(t)
	root, err := os.MkdirTemp(os.Getenv("JCODE_SCRATCH_DIR"), "huginn-tmux-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(root)
	binary := filepath.Join(root, "huginn")
	build := exec.Command("nice", "-n", "19", "env", "GOMAXPROCS=2", "GOFLAGS=-p=2", "go", "build", "-o", binary, "./cmd/huginn")
	build.Dir = "../.."
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build: %v\n%s", err, out)
	}
	socket := fmt.Sprintf("huginn-golden-%d", os.Getpid())
	cleanupServer := startTmuxServer(t, socket)
	defer cleanupServer()
	tmux := func(t *testing.T, args ...string) string {
		t.Helper()
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		cmd := exec.CommandContext(ctx, "tmux", append([]string{"-N", "-L", socket, "-f", "/dev/null"}, args...)...)
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("tmux %v: %v\n%s", args, err, out)
		}
		return string(out)
	}
	home := filepath.Join(root, "home")
	repo := filepath.Join(root, "repo")
	for _, p := range []string{home, repo} {
		if err := os.MkdirAll(p, 0755); err != nil {
			t.Fatal(err)
		}
	}
	baseEnv := append(os.Environ(), "HOME="+home, "XDG_CONFIG_HOME="+filepath.Join(home, ".config"), "GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_NOSYSTEM=1", "GIT_AUTHOR_NAME=Synthetic", "GIT_AUTHOR_EMAIL=synthetic@example.invalid", "GIT_COMMITTER_NAME=Synthetic", "GIT_COMMITTER_EMAIL=synthetic@example.invalid", "HUGINN_USER=terminal-fixture", "GOMAXPROCS=2", "GOFLAGS=-p=2")
	run := func(name string, args ...string) {
		t.Helper()
		cmd := exec.Command(name, args...)
		cmd.Dir = repo
		cmd.Env = baseEnv
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("%s %v: %v\n%s", name, args, err, out)
		}
	}
	run("git", "init", "-b", "main")
	run("git", "commit", "--allow-empty", "-m", "synthetic fixture")
	run(binary, "init")
	if err := testfixture.Seed(filepath.Join(repo, ".huginn")); err != nil {
		t.Fatal(err)
	}
	for _, size := range [][2]int{{80, 24}, {140, 40}} {
		for _, noColor := range []bool{false, true} {
			mode := "color"
			if noColor {
				mode = "no-color"
			}
			name := fmt.Sprintf("%dx%d-%s", size[0], size[1], mode)
			t.Run(name, func(t *testing.T) {
				loadGuard(t)
				target := name + ":0.0"
				exitFile := filepath.Join(root, name+".exit")
				env := []string{"HOME=" + home, "XDG_CONFIG_HOME=" + filepath.Join(home, ".config"), "GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_NOSYSTEM=1", "GIT_AUTHOR_NAME=Synthetic", "GIT_AUTHOR_EMAIL=synthetic@example.invalid", "GIT_COMMITTER_NAME=Synthetic", "GIT_COMMITTER_EMAIL=synthetic@example.invalid", "HUGINN_USER=terminal-fixture", "TERM=xterm-256color", "COLORTERM=truecolor", "COLORFGBG=15;0", "CLICOLOR_FORCE=1", "NO_COLOR=", "GOMAXPROCS=2", "GOFLAGS=-p=2"}
				if noColor {
					env = append(env, "NO_COLOR=1")
				}
				for i := range env {
					env[i] = shellQuote(env[i])
				}
				launch := "/bin/sh -c " + shellQuote("env "+strings.Join(env, " ")+" "+shellQuote(binary)+" tui\nexit_code=$?\nprintf '%s\\n' \"$exit_code\" > "+shellQuote(exitFile)+"\nexit \"$exit_code\"")
				tmux(t, "new-session", "-d", "-s", name, "-x", strconv.Itoa(size[0]), "-y", strconv.Itoa(size[1]), "-c", repo, launch)
				// Keep the dedicated server alive until explicit cleanup, rather than
				// connecting a cleanup client during automatic last-session shutdown.
				tmux(t, "set-option", "-s", "exit-empty", "off")
				defer exec.Command("tmux", "-N", "-L", socket, "kill-session", "-t", name).Run()
				tmux(t, "set-option", "-g", "status", "off")
				tmux(t, "resize-window", "-t", name, "-x", strconv.Itoa(size[0]), "-y", strconv.Itoa(size[1]))
				dimensions := strings.TrimSpace(tmux(t, "display-message", "-p", "-t", target, "#{pane_width}x#{pane_height}"))
				if dimensions != fmt.Sprintf("%dx%d", size[0], size[1]) {
					t.Fatal("wrong terminal size", dimensions)
				}
				send := func(text string) { tmux(t, "send-keys", "-t", target, "-l", text) }
				special := func(key string) { tmux(t, "send-keys", "-t", target, key) }
				wait := func(contains string) string {
					t.Helper()
					deadline := time.Now().Add(12 * time.Second)
					last := ""
					stable := 0
					for time.Now().Before(deadline) {
						current := tmux(t, "capture-pane", "-p", "-t", target)
						if strings.Contains(current, contains) && current == last {
							stable++
							if stable >= 2 {
								return current
							}
						} else {
							stable = 0
						}
						last = current
						time.Sleep(45 * time.Millisecond)
					}
					t.Fatalf("screen never settled on %q\n%s", contains, last)
					return ""
				}
				snapshot := func(view, contains string) {
					t.Helper()
					plain := normalizeCapture(wait(contains))
					colored := tmux(t, "capture-pane", "-p", "-e", "-t", target)
					if len(strings.Split(strings.TrimSuffix(plain, "\n"), "\n")) != size[1] {
						t.Fatalf("capture height: %q", plain)
					}
					for _, line := range strings.Split(strings.TrimSuffix(plain, "\n"), "\n") {
						if lipgloss.Width(line) > size[0] {
							t.Fatalf("capture exceeds width: %q", line)
						}
					}
					// Any foreground/background SGR counts: truecolor/256 (38;/48;) on a
					// rich terminal, basic 30-37/90-97/40-47/100-107 on CI's tmux.
					hasColor := ansiColour.MatchString(colored)
					if noColor && hasColor || !noColor && !hasColor {
						t.Fatalf("colour contract noColor=%v hasColor=%v\n%q", noColor, hasColor, colored)
					}
					path := filepath.Join("testdata", "tmux", name, view+".txt")
					if os.Getenv("HUGINN_UPDATE_GOLDEN") == "1" {
						if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
							t.Fatal(err)
						}
						if err := os.WriteFile(path, []byte(plain), 0644); err != nil {
							t.Fatal(err)
						}
					}
					want, err := os.ReadFile(path)
					if err != nil {
						t.Fatal(err)
					}
					if string(want) != plain {
						t.Errorf("golden drift %s\nGOT:\n%s\nWANT:\n%s", path, plain, want)
					}
					if artifacts := os.Getenv("HUGINN_TUI_ARTIFACTS"); artifacts != "" {
						dir := filepath.Join(artifacts, name)
						if err := os.MkdirAll(dir, 0755); err != nil {
							t.Fatal(err)
						}
						if err := os.WriteFile(filepath.Join(dir, view+".txt"), []byte(plain), 0644); err != nil {
							t.Fatal(err)
						}
						if err := os.WriteFile(filepath.Join(dir, view+".ansi"), []byte(colored), 0644); err != nil {
							t.Fatal(err)
						}
					}
				}
				snapshot("overview", "HUGINN · overview")
				send("v")
				snapshot("list", "HUGINN · list")
				send("b")
				snapshot("board", "HUGINN · board")
				special("Escape")
				wait("HUGINN · overview")
				send("#000001")
				wait("JUMP TO ID")
				special("Enter")
				snapshot("detail", "HUGINN · detail")
				special("Escape")
				wait("HUGINN · list")
				send("#1785")
				wait("JUMP TO ID")
				special("Enter")
				snapshot("markdown-warning", "⚠ Acceptance requires at least one item")
				special("Escape")
				wait("HUGINN · list")
				send(":type ")
				snapshot("palette", "COMMAND PALETTE")
				special("Escape")
				wait("HUGINN · list")
				send("#178")
				snapshot("jump", "JUMP TO ID")
				special("Escape")
				wait("HUGINN · list")
				send("1")
				wait("status:todo")
				send("f")
				wait("choose a field")
				send("t")
				snapshot("quick-filter", "counts include your other filters")
				send("feature")
				special("Enter")
				wait("type:feature")
				send("/task")
				special("Enter")
				snapshot("chips", "[/ task]")
				send("F")
				wait("No filters")
				send("?")
				snapshot("help", "NAVIGATION")
				special("Escape")
				wait("HUGINN · list")
				send("q")
				deadline := time.Now().Add(8 * time.Second)
				for time.Now().Before(deadline) {
					if code, err := os.ReadFile(exitFile); err == nil {
						if strings.TrimSpace(string(code)) != "0" {
							t.Fatalf("CLI exit: %s", code)
						}
						return
					}
					time.Sleep(50 * time.Millisecond)
				}
				t.Fatal("TUI did not quit cleanly")
			})
		}
	}
	if !t.Failed() {
		t.Log("tmux: PASS (40 real-binary goldens, 80x24/140x40, colour/NO_COLOR, all sessions cleaned up)")
	}
}
