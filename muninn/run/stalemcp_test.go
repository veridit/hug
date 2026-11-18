package run

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/veridit/hug/muninn/internal/config"
	"github.com/veridit/hug/muninn/internal/harness"
	"github.com/veridit/hug/muninn/internal/scopetest"
)

// A copy of this test binary started with HUG_TEST_FAKE_MCP=1 is a hug mcp
// server that only waits: its file sits where Homebrew put an old hug.
func init() {
	if os.Getenv("HUG_TEST_FAKE_MCP") == "1" {
		time.Sleep(2 * time.Minute)
		os.Exit(0)
	}
}

// startFakeMCP runs a copy of this test binary at DIR/Cellar/hug/VERSION/bin/hug
// as "hug mcp --jcode-home HOME" and returns its pid.
func startFakeMCP(t *testing.T, version, jcodeHome string) int {
	t.Helper()
	self, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	bin := filepath.Join(t.TempDir(), "Cellar", "hug", version, "bin", "hug")
	os.MkdirAll(filepath.Dir(bin), 0o755)
	src, err := os.Open(self)
	if err != nil {
		t.Fatal(err)
	}
	dst, err := os.OpenFile(bin, os.O_CREATE|os.O_WRONLY, 0o755)
	if err != nil {
		t.Fatal(err)
	}
	io.Copy(dst, src)
	src.Close()
	dst.Close()
	cmd := exec.Command(bin, "mcp", "--jcode-home", jcodeHome)
	cmd.Env = append(os.Environ(), "HUG_TEST_FAKE_MCP=1")
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { cmd.Process.Kill(); cmd.Wait() })
	return cmd.Process.Pid
}

// onlyFakes keeps the scan to this test's fake servers: the machine's own
// are not this test's.
func onlyFakes(pids ...int) func(context.Context) harness.MCPScan {
	return func(ctx context.Context) harness.MCPScan {
		sc := harness.ScanMCP(ctx)
		var mine []harness.StaleMCP
		for _, s := range sc.Stale {
			for _, p := range pids {
				if s.PID == p {
					mine = append(mine, s)
				}
			}
		}
		sc.Stale = mine
		return sc
	}
}

// staleTurns runs one jcode turn per message and returns each first tool
// result.
func staleTurn(t *testing.T, j *jcodeTurns, id, text string) string {
	t.Helper()
	j.user(id, text)
	if out := j.event("turn_start"); out != "" {
		t.Fatalf("turn_start printed %q", out)
	}
	j.toolCall()
	out := j.event("post_tool_feedback")
	j.answer()
	j.event("turn_end")
	return out
}

// z77pem check 1 (crab's review): the hug mcp serving this session (its
// jcode home) is older than v0.7.54 (no reload, mbdfzs): the session's
// first turn names it in one line with the fix this agent can run, with
// the turn's recall at its first tool result. Another app's stale server
// and a server new enough to reload get no line here, only hook.log. A
// later turn says nothing more, and nothing is stopped.
func TestHookNamesItsOwnStaleMCPServer(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		t.Skip("ps and the executable path: macOS and Linux")
	}
	f := scopetest.New(t)
	f.Bind(t, "global")
	t.Chdir(f.Cwd)
	j := newJcodeTurns(t, f)
	own := startFakeMCP(t, "0.7.2", j.home)
	other := startFakeMCP(t, "0.7.14", "/srv/statbus/.jcode")
	fresh := startFakeMCP(t, "0.7.54", j.home)
	staleMCPScan = onlyFakes(own, other, fresh)
	defer func() { staleMCPScan = harness.ScanMCP }()

	out := staleTurn(t, j, "message_1", scopetest.Query)
	want := "hug: your hug mcp server (pid " + strconv.Itoa(own) + ") runs hug 0.7.2, older than v0.7.54"
	if !strings.Contains(out, want) || !strings.Contains(out, "\nfix: mcp action=reload server=hug\n") {
		t.Fatalf("the first tool result does not name this session's server (want %q and its fix):\n%s", want, out)
	}
	if !strings.Contains(out, recallMarker) {
		t.Fatalf("the turn's recall is gone:\n%s", out)
	}
	for _, p := range []int{other, fresh} {
		if strings.Contains(out, strconv.Itoa(p)) {
			t.Fatalf("pid %d (another app's, or one that reloads) is told to this agent:\n%s", p, out)
		}
	}
	// A person finds the other app's server in hook.log.
	log, _ := os.ReadFile(filepath.Join(config.DataDir(), "hook.log"))
	if !strings.Contains(string(log), "stale hug mcp of other apps (not told to this agent): pid "+strconv.Itoa(other)+" hug 0.7.14 in jcode /srv/statbus/.jcode") {
		t.Fatalf("hook.log lacks the other app's server:\n%s", log)
	}
	// Once per session: the next turn says nothing of it.
	if again := staleTurn(t, j, "message_2", "something unrelated to any memory here"); strings.Contains(again, "older than v0.7.54") {
		t.Fatalf("second turn said it again:\n%s", again)
	}
	// Tell, never kill: the old servers still run.
	for _, p := range []int{own, other} {
		if pr, err := os.FindProcess(p); err != nil || pr.Signal(syscall.Signal(0)) != nil {
			t.Fatalf("the old server %d was stopped", p)
		}
	}
	j.noTranscript()
}

// crab's review: a session whose own server is current, while another app's
// is stale, gets no notice at all; hook.log has it.
func TestHookSilentWhenOnlyOthersAreStale(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		t.Skip("ps and the executable path: macOS and Linux")
	}
	f := scopetest.New(t)
	f.Bind(t, "global")
	t.Chdir(f.Cwd)
	j := newJcodeTurns(t, f)
	fresh := startFakeMCP(t, "0.7.66", j.home)
	other := startFakeMCP(t, "0.7.2", "/srv/statbus/.jcode")
	staleMCPScan = onlyFakes(fresh, other)
	defer func() { staleMCPScan = harness.ScanMCP }()

	out := staleTurn(t, j, "message_1", scopetest.Query)
	if strings.Contains(out, "hug mcp") || strings.Contains(out, strconv.Itoa(other)) {
		t.Fatalf("a session on a current server is told of another app's:\n%s", out)
	}
	log, _ := os.ReadFile(filepath.Join(config.DataDir(), "hook.log"))
	if !strings.Contains(string(log), "pid "+strconv.Itoa(other)+" hug 0.7.2 in jcode /srv/statbus/.jcode") {
		t.Fatalf("hook.log lacks the other app's server:\n%s", log)
	}
}

// The Claude Code / Codex prompt hook: the server whose app this hook runs
// under is named once per session; a server it cannot tie to this session
// gets no line, only hook.log.
func TestPromptHookNamesItsOwnStaleMCPServerOnce(t *testing.T) {
	memHome(t)
	// The scan is injected, so this runs on every system, Windows too.
	listed := staleMCPListed
	staleMCPListed = true
	defer func() { staleMCPListed = listed }()
	self := os.Getppid() // the hook's own ancestors include its parent
	mine := harness.StaleMCP{PID: 77273, Version: "0.7.43", App: "Claude", Parents: []int{self}}
	theirs := harness.StaleMCP{PID: 3175, Version: "0.7.2", App: "jcode /srv/statbus/.jcode", Parents: []int{424242}}
	stale := []harness.StaleMCP{mine, theirs}
	staleMCPScan = func(context.Context) harness.MCPScan {
		return harness.MCPScan{Stale: stale, Self: []int{os.Getpid(), self}}
	}
	defer func() { staleMCPScan = harness.ScanMCP }()
	run := func(sid string) string {
		t.Helper()
		// Marshalled, never concatenated: a Windows TempDir (C:\Users\...)
		// is no valid JSON string as it stands (CI 44371620428).
		b, _ := json.Marshal(map[string]string{"session_id": sid, "prompt": "hello", "cwd": t.TempDir(), "hook_event_name": "UserPromptSubmit"})
		in := string(b)
		r, w, _ := os.Pipe()
		w.WriteString(in)
		w.Close()
		prev := os.Stdin
		os.Stdin = r
		defer func() { os.Stdin = prev }()
		out, _ := captureSurface(t, func() error { return runHook("claude", 10*time.Second) })
		return out
	}
	out := run("c1")
	if !strings.Contains(out, "your hug mcp server (pid 77273) runs hug 0.7.43") || !strings.Contains(out, "restart hug's MCP server in Claude (quit and reopen it)") {
		t.Fatalf("the prompt hook does not name its own server:\n%s", out)
	}
	if strings.Contains(out, "3175") {
		t.Fatalf("another app's server is told to this agent:\n%s", out)
	}
	if out := run("c1"); strings.Contains(out, "pid 77273") {
		t.Fatalf("said twice in one session:\n%s", out)
	}
	// Not identified: its app is no ancestor of this hook.
	stale = []harness.StaleMCP{theirs}
	if out := run("c2"); strings.Contains(out, "hug mcp") {
		t.Fatalf("a server not tied to this session is told:\n%s", out)
	}
}

// hug mem stats shows a person every stale server on the machine.
func TestStatsNamesStaleMCPServers(t *testing.T) {
	memHome(t)
	staleMCPScan = func(context.Context) harness.MCPScan {
		return harness.MCPScan{Stale: []harness.StaleMCP{{PID: 3175, Version: "0.7.2", App: "jcode /srv/statbus/.jcode"}, {PID: 77273, Version: "0.7.43", App: "Claude"}}}
	}
	defer func() { staleMCPScan = harness.ScanMCP }()
	out, errOut, code := runCLI(t, "", "stats")
	if code != 0 {
		t.Fatalf("stats: exit %d\n%s", code, errOut)
	}
	if !strings.Contains(out, `"stale_mcp": "pid 3175 hug 0.7.2 in jcode /srv/statbus/.jcode; pid 77273 hug 0.7.43 in Claude (older than v0.7.54`) {
		t.Fatalf("stats lacks the stale servers:\n%s", out)
	}
}

// Where the scan cannot list processes (Windows), the first turn costs
// nothing: no scan runs, no marker is written, nothing is said.
func TestStaleMCPNothingWhereProcessesAreNotListed(t *testing.T) {
	memHome(t)
	listed := staleMCPListed
	staleMCPListed = false
	defer func() { staleMCPListed = listed }()
	scanned := false
	staleMCPScan = func(context.Context) harness.MCPScan {
		scanned = true
		return harness.MCPScan{Stale: []harness.StaleMCP{{PID: 77273, Version: "0.7.43", App: "Claude", Parents: []int{os.Getppid()}}}, Self: []int{os.Getpid(), os.Getppid()}}
	}
	defer func() { staleMCPScan = harness.ScanMCP }()
	if out := staleMCPOnce(context.Background(), "w1", ""); out != "" {
		t.Fatalf("said %q", out)
	}
	if scanned {
		t.Fatal("scanned where processes are not listed")
	}
	if _, err := os.Stat(staleMCPMarker("w1")); !os.IsNotExist(err) {
		t.Fatalf("marker written: %v", err)
	}
}
