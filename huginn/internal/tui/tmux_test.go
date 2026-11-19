package tui

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/veridit/hug/huginn/internal/tui/testfixture"
	"github.com/veridit/hug/internal/tmuxtest"
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
//
// Both waits are on the server itself, never on a tmux client (ghycrq): on
// a loaded machine starting a process can take seconds (under 16 CPU hogs
// /bin/echo took up to 11.6 s), so a client run inside a deadline failed
// servers that were fine. Ready is the server's socket taking a connection
// from this process. Cleanup is SIGTERM, which is what tmux kill-server
// does (cmd-kill-server.c: kill(getpid(), SIGTERM)), then the server's exit.
//
// The server dies with the test, however the test ends (wacc33: crab
// found three huginn-golden/reload servers, parent launchd, alive for
// days). The returned cleanup is also a t.Cleanup, so t.Fatal and a panic
// in a subtest (which skips the parent's defers) stop it. A watchdog shell
// holds the server: it reads a pipe only this process writes, and when
// that pipe closes (cleanup, or this process ended by go test -timeout,
// SIGKILL or a cancelled job) it SIGTERMs the server. A server left anyway
// is reaped by the next run (TestMain, tmuxtest.Reap). Every label starts
// with tmuxtest.Prefix.
func startTmuxServer(t *testing.T, socket string) func() {
	t.Helper()
	if !strings.HasPrefix(socket, tmuxtest.Prefix) {
		t.Fatalf("tmux socket %q: a test server's label starts with %q, so the next run can reap it", socket, tmuxtest.Prefix)
	}
	path, err := tmuxSocketPath(socket)
	if err != nil {
		t.Fatal(err)
	}
	// The test process holds the only write end of sh's stdin. sh starts the
	// server and a reader on a copy of that stdin (an asynchronous list
	// would read /dev/null), then waits on the server. The reader gets end
	// of input when cleanup closes the pipe or this process dies, however it
	// dies, and then SIGTERMs the server.
	server := exec.Command("/bin/sh", "-c", `exec 3<&0
tmux -D -L "$1" -f /dev/null </dev/null 3<&- & s=$!
(read _ || :; kill $s 2>/dev/null) <&3 3<&- >/dev/null 2>&1 &
exec 3<&-
wait $s; st=$?
[ $st -eq 143 ] && exit 0 # 128+SIGTERM: our own cleanup
exit $st`, "sh", socket)
	server.Stdout, server.Stderr = os.Stderr, os.Stderr
	lifeline, err := server.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := server.Start(); err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() { done <- server.Wait() }()
	// A wait past a minute says what it waits on, then goes on waiting: go
	// test's -timeout ends a server that really hangs, with every stack.
	slow := func(what string) *time.Timer {
		return time.AfterFunc(time.Minute, func() { t.Logf("tmux server %s: still waiting for %s", socket, what) })
	}
	var once sync.Once
	cleanup := func() {
		once.Do(func() {
			lifeline.Close()
			defer slow("its exit after SIGTERM").Stop()
			if err := <-done; err != nil {
				t.Errorf("tmux server cleanup: process %v", err)
			}
			procs, err := tmuxtest.Servers()
			if err != nil {
				t.Errorf("tmux server cleanup: %v", err)
			}
			for _, p := range procs {
				if p.Label == socket {
					t.Errorf("tmux server cleanup: server %d (-L %s) still runs", p.PID, socket)
				}
			}
		})
	}
	t.Cleanup(cleanup)
	defer slow(path + " to take a connection").Stop()
	for {
		select {
		case err := <-done:
			t.Fatalf("foreground tmux server did not become ready: it exited (%v) before %s took a connection", err, path)
		default:
		}
		if conn, err := net.Dial("unix", path); err == nil {
			conn.Close()
			return cleanup
		}
		time.Sleep(25 * time.Millisecond)
	}
}

// tmuxSocketPath is where tmux -L label puts its socket: the real path of
// $TMUX_TMPDIR, else /tmp, then tmux-UID/label (tmux.c, make_label).
func tmuxSocketPath(label string) (string, error) {
	dir := os.Getenv("TMUX_TMPDIR")
	if dir == "" {
		dir = "/tmp"
	}
	dir, err := filepath.EvalSymlinks(dir)
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, fmt.Sprintf("tmux-%d", os.Getuid()), label), nil
}

// TestTmuxGolden exercises the actual CLI binary, not a copied model renderer.
// Update deliberately with HUG_UPDATE_GOLDEN=1, review the diff, then test normally.
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
	build := exec.Command("nice", "-n", "19", "env", "GOMAXPROCS=2", "GOFLAGS=-p=2", "go", "build", "-ldflags", "-X github.com/veridit/hug/huginn/run.Version=v0.0.0", "-o", binary, "./cmd/huginn")
	build.Dir = "../.."
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build: %v\n%s", err, out)
	}
	socket := tmuxtest.Label("golden")
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
	baseEnv := append(os.Environ(), "HOME="+home, "XDG_CONFIG_HOME="+filepath.Join(home, ".config"), "GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_NOSYSTEM=1", "GIT_AUTHOR_NAME=Synthetic", "GIT_AUTHOR_EMAIL=synthetic@example.invalid", "GIT_COMMITTER_NAME=Synthetic", "GIT_COMMITTER_EMAIL=synthetic@example.invalid", "HUG_USER=terminal-fixture", "GOMAXPROCS=2", "GOFLAGS=-p=2")
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
	if err := testfixture.Seed(filepath.Join(repo, ".hug", "tasks")); err != nil {
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
				env := []string{"HOME=" + home, "XDG_CONFIG_HOME=" + filepath.Join(home, ".config"), "GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_NOSYSTEM=1", "GIT_AUTHOR_NAME=Synthetic", "GIT_AUTHOR_EMAIL=synthetic@example.invalid", "GIT_COMMITTER_NAME=Synthetic", "GIT_COMMITTER_EMAIL=synthetic@example.invalid", "HUG_USER=terminal-fixture", "TERM=xterm-256color", "COLORTERM=truecolor", "COLORFGBG=15;0", "CLICOLOR_FORCE=1", "NO_COLOR=", "GOMAXPROCS=2", "GOFLAGS=-p=2"}
				if noColor {
					env = append(env, "NO_COLOR=1")
				}
				for i := range env {
					env[i] = shellQuote(env[i])
				}
				launch := "/bin/sh -c " + shellQuote("env "+strings.Join(env, " ")+" "+shellQuote(binary)+" tui\nexit_code=$?\nprintf '%s\\n' \"$exit_code\" > "+shellQuote(exitFile+".tmp")+" && mv "+shellQuote(exitFile+".tmp")+" "+shellQuote(exitFile)+"\nexit \"$exit_code\"")
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
					if os.Getenv("HUG_UPDATE_GOLDEN") == "1" {
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
					if artifacts := os.Getenv("HUG_TUI_ARTIFACTS"); artifacts != "" {
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
				// The first frame is the cache's snapshot without file counts
				// (vnyz8z); ⎘2 appears once the refresh has landed.
				snapshot("overview", "⎘2")
				send("vl")
				snapshot("list", "HUG · repo · list")
				send("vb")
				snapshot("board", "HUG · repo · board")
				special("Escape")
				wait("HUG · repo · overview")
				send("#000001")
				wait("JUMP TO ID")
				special("Enter")
				snapshot("detail", "HUG · repo · list › 000001") // the breadcrumb band (e3sevr)
				special("Escape")
				wait("HUG · repo · list")
				send("#1785")
				wait("JUMP TO ID")
				special("Enter")
				// 000003 is done: its detail renders the Markdown and shows no
				// finding, since lint judges open tasks only (t01kqm).
				snapshot("markdown-done", "rendered code")
				special("Escape")
				wait("HUG · repo · list")
				send(":type ")
				snapshot("palette", "GLOBAL")
				special("Escape") // up a level (05r3pg)
				special("Escape") // closed
				wait("HUG · repo · list")
				send("#178")
				snapshot("jump", "JUMP TO ID")
				special("Escape")
				wait("HUG · repo · list")
				send(":todo")
				special("Enter")
				wait("state:todo")
				send("f")
				wait("a letter picks a field")
				send("t")
				snapshot("quick-filter", "counts include your other filters") // the FILTER picker
				send("feature")
				special("Enter")
				wait("type:feature")
				send("/task")
				special("Tab") // keep the search as a filter (mcvfbz)
				snapshot("chips", "[/ task]")
				send("F")
				wait("filter: none")
				send("?")
				snapshot("help", "EVERYWHERE")
				special("Escape")
				wait("HUG · repo · list")
				// Sort picker (mkzcqr, ahejza): s, priority then type, Enter applies.
				send("s")
				wait("HUG · repo · sort")
				send("p")
				send("t")
				special("Enter")
				snapshot("sort", "sort: priority ↓ · type")
				send(":sort rank")
				special("Enter")
				wait("sort: rank")
				// Grab mode (3xcesq): the grabbed row moves on screen and the
				// key bar shows only the keys that work; Esc writes nothing.
				send("<")
				send("m")
				send("jj")
				snapshot("grab", "(was 1)")
				special("Escape")
				wait("move cancelled; nothing changed")
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
		t.Log("tmux: PASS (48 real-binary goldens, 80x24/140x40, colour/NO_COLOR, all sessions cleaned up)")
	}
}
