package main

import (
	"context"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// lockedBuf is a stderr the exec'ed process writes while the test reads.
type lockedBuf struct {
	mu sync.Mutex
	b  strings.Builder
}

func (l *lockedBuf) Write(p []byte) (int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.b.Write(p)
}

func (l *lockedBuf) String() string {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.b.String()
}

// reloadRig builds hug twice (versions old and new) and starts the old one
// as an app would, at a path the test can replace under it (mbdfzs).
type reloadRig struct {
	t      *testing.T
	launch string // the path the app started
	newBin string
	hold   string
	stderr *lockedBuf
	cs     *sdk.ClientSession
	cmd    *exec.Cmd
	ctx    context.Context
}

func buildHug(t *testing.T, out, version string) {
	t.Helper()
	b := exec.Command("go", "build", "-ldflags", "-X main.Version="+version, "-o", out, ".")
	b.Env = append(os.Environ(), "GOMAXPROCS=2", "GOFLAGS=-p=2")
	if o, err := b.CombinedOutput(); err != nil {
		t.Fatalf("build %s: %v\n%s", version, err, o)
	}
}

func newReloadRig(t *testing.T) *reloadRig {
	if testing.Short() {
		t.Skip("builds the hug binary twice")
	}
	dir := t.TempDir()
	exe := ""
	if runtime.GOOS == "windows" {
		exe = ".exe"
	}
	r := &reloadRig{t: t, launch: filepath.Join(dir, "bin", "hug"+exe), newBin: filepath.Join(dir, "hug-new"+exe), hold: filepath.Join(dir, "hold")}
	os.MkdirAll(filepath.Dir(r.launch), 0o755)
	buildHug(t, r.launch, "0.0.1-old")
	buildHug(t, r.newBin, "0.0.2-new")
	repo := filepath.Join(dir, "repo")
	home := filepath.Join(dir, "home")
	for _, args := range [][]string{{"init", "-q", repo}, {"-C", repo, "config", "user.email", "t@example.com"}, {"-C", repo, "config", "user.name", "t"}} {
		if o, err := exec.Command("git", args...).CombinedOutput(); err != nil {
			t.Skipf("git unavailable: %v %s", err, o)
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	t.Cleanup(cancel)
	r.ctx = ctx
	r.cmd = exec.Command(r.launch, "mcp")
	r.cmd.Dir = repo
	r.cmd.Env = append(os.Environ(), "HOME="+home, "USERPROFILE="+home, "HUG_HOME="+filepath.Join(dir, "mem"), "JCODE_HOME="+filepath.Join(home, "no-jcode"),
		"HUG_NO_EMBED=1", "HUG_SYNC=", "GOMAXPROCS=2", "GIT_CONFIG_NOSYSTEM=1", "HUG_MCP_TEST_HOLD="+r.hold)
	r.stderr = &lockedBuf{}
	r.cmd.Stderr = r.stderr
	cs, err := sdk.NewClient(&sdk.Implementation{Name: "t", Version: "0"}, nil).Connect(ctx, &sdk.CommandTransport{Command: r.cmd}, nil)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	r.cs = cs
	t.Cleanup(func() { cs.Close() })
	return r
}

// call runs a task_* tool and returns the version that served it.
func (r *reloadRig) call(name string) (string, error) {
	res, err := r.cs.CallTool(r.ctx, &sdk.CallToolParams{Name: name, Arguments: map[string]any{}})
	if err != nil {
		return "", err
	}
	if res.IsError {
		r.t.Fatalf("%s: tool error %v", name, res.Content)
	}
	v, _ := res.GetMeta()["hug/version"].(string)
	return v, nil
}

// upgrade replaces the launched hug with the new build as a package manager
// does: a new file renamed over the old path.
func (r *reloadRig) upgrade() {
	r.t.Helper()
	b, err := os.ReadFile(r.newBin)
	if err != nil {
		r.t.Fatal(err)
	}
	tmp := r.launch + ".tmp"
	if err := os.WriteFile(tmp, b, 0o755); err != nil {
		r.t.Fatal(err)
	}
	if err := os.Rename(tmp, r.launch); err != nil {
		r.t.Fatal(err)
	}
}

// mbdfzs check 1: the binary changes under a running hug mcp; the next
// request is answered by the new hug, on the same connection and pid.
func TestMCPReloadsChangedBinary(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("no exec on Windows: TestMCPStopsForRestartWithoutExec covers it")
	}
	r := newReloadRig(t)
	os.WriteFile(r.hold, nil, 0o644) // task_types is not held here
	pid := r.cmd.Process.Pid
	if v, err := r.call("task_list"); err != nil || v != "0.0.1-old" {
		t.Fatalf("before the upgrade: version %q, err %v", v, err)
	}
	r.upgrade()
	v, err := r.call("task_list")
	if err != nil || v != "0.0.2-new" {
		t.Fatalf("after the upgrade: version %q, err %v; stderr:\n%s", v, err, r.stderr)
	}
	if r.cmd.Process.Pid != pid {
		t.Fatalf("pid changed")
	}
	if v, err := r.call("task_types"); err != nil || v != "0.0.2-new" {
		t.Fatalf("second call after: version %q, err %v", v, err)
	}
	if !strings.Contains(r.stderr.String(), "reloaded as hug 0.0.2-new (was 0.0.1-old), same connection") {
		t.Errorf("stderr:\n%s", r.stderr)
	}
}

// mbdfzs check 2: a request running when the binary changes is finished by
// the old hug; the next one is served by the new.
func TestMCPReloadWaitsForRunningRequest(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("no exec on Windows")
	}
	r := newReloadRig(t)
	type reply struct {
		v   string
		err error
	}
	held := make(chan reply, 1)
	go func() { v, err := r.call("task_types"); held <- reply{v, err} }()
	for i := 0; ; i++ {
		if _, err := os.Stat(r.hold + ".started"); err == nil {
			break
		}
		if i > 1000 {
			t.Fatal("task_types never started")
		}
		time.Sleep(10 * time.Millisecond)
	}
	r.upgrade()
	next := make(chan reply, 1)
	go func() { v, err := r.call("task_list"); next <- reply{v, err} }()
	select {
	case got := <-next:
		t.Fatalf("task_list answered (%q, %v) while task_types was still running", got.v, got.err)
	case <-time.After(300 * time.Millisecond):
	}
	os.WriteFile(r.hold, nil, 0o644)
	if got := <-held; got.err != nil || got.v != "0.0.1-old" {
		t.Fatalf("running request: version %q, err %v", got.v, got.err)
	}
	if got := <-next; got.err != nil || got.v != "0.0.2-new" {
		t.Fatalf("next request: version %q, err %v; stderr:\n%s", got.v, got.err, r.stderr)
	}
}

// TestMCPStopsForRestartWithoutExec: where there is no exec (Windows), a
// changed binary makes the server answer each request it has read with an
// error saying to run it again, then exit so the app starts the new hug. No
// request is left without an answer. Runs in-process on every system.
func TestMCPStopsForRestartWithoutExec(t *testing.T) {
	bin := filepath.Join(t.TempDir(), "hug")
	os.WriteFile(bin, []byte("old"), 0o755)
	var out lockedBuf
	var log lockedBuf
	r := newReloader(bin, &out, &log)
	r.exec = nil
	exited := make(chan int, 1)
	r.exit = func(c int) { exited <- c }
	go func() {
		buf := make([]byte, 4096)
		for {
			if _, e := r.Read(buf); e != nil {
				return
			}
		}
	}()
	src, w := io.Pipe()
	go r.pump(src, nil)
	w.Write([]byte(`{"jsonrpc":"2.0","method":"notifications/initialized"}` + "\n"))
	time.Sleep(50 * time.Millisecond)
	os.Remove(bin)
	os.WriteFile(bin, []byte("new binary"), 0o755)
	w.Write([]byte(`{"jsonrpc":"2.0","id":7,"method":"tools/call","params":{"name":"task_list"}}` + "\n"))
	select {
	case c := <-exited:
		if c != 0 {
			t.Errorf("exit %d", c)
		}
	case <-time.After(10 * time.Second):
		t.Fatalf("no exit; out:\n%s\nlog:\n%s", out.String(), log.String())
	}
	if got := out.String(); !strings.Contains(got, `"id":7`) || !strings.Contains(got, "run the call again") {
		t.Errorf("reply:\n%s", got)
	}
	if !strings.Contains(log.String(), "stopping so the app starts it") {
		t.Errorf("log:\n%s", log.String())
	}
}

// TestMCPResumeReplaysInitialize: the new hug gets the client's initialize
// (under its own id) and initialized again, then the held request; its reply
// to the replayed initialize never reaches the client, which already has one.
func TestMCPResumeReplaysInitialize(t *testing.T) {
	var out, log lockedBuf
	r := newReloader("", &out, &log)
	init := `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18"}}` + "\n"
	held := `{"jsonrpc":"2.0","id":9,"method":"tools/call","params":{"name":"task_list"}}` + "\n"
	go r.pump(strings.NewReader(""), &resumeState{Init: []byte(init), Initialized: true, Pending: []byte(held), From: "0.0.1-old"})
	got, _ := io.ReadAll(r)
	lines := strings.Split(strings.TrimSpace(string(got)), "\n")
	if len(lines) != 3 || !strings.Contains(lines[0], `"id":"hug-reload-initialize"`) || !strings.Contains(lines[0], `"protocolVersion":"2025-06-18"`) ||
		!strings.Contains(lines[1], "notifications/initialized") || lines[2] != strings.TrimSpace(held) {
		t.Fatalf("the new server read:\n%s", got)
	}
	r.Write([]byte(`{"jsonrpc":"2.0","id":"hug-reload-initialize","result":{}}` + "\n"))
	r.Write([]byte(`{"jsonrpc":"2.0","id":9,"result":{}}` + "\n"))
	if o := out.String(); strings.Contains(o, "hug-reload-initialize") || !strings.Contains(o, `"id":9`) {
		t.Fatalf("the client got:\n%s", o)
	}
}

// An unchanged binary is never reloaded, and touching nothing keeps serving.
func TestMCPNoReloadWhenUnchanged(t *testing.T) {
	r := newReloadRig(t)
	for i := 0; i < 3; i++ {
		if v, err := r.call("task_list"); err != nil || v != "0.0.1-old" {
			t.Fatalf("call %d: version %q, err %v", i, v, err)
		}
	}
	if strings.Contains(r.stderr.String(), "reload") {
		t.Errorf("stderr:\n%s", r.stderr)
	}
}
