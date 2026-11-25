package main

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

// fakeIdle is an idleSource the test drives: lines it is given are ready to
// read; wait reports a tick (or a nudge) when there are none.
type fakeIdle struct {
	mu     sync.Mutex
	buf    []byte
	closed bool
	nudge  bool
	ch     chan struct{}
}

func newFakeIdle() *fakeIdle { return &fakeIdle{ch: make(chan struct{}, 1)} }

func (f *fakeIdle) poke() {
	select {
	case f.ch <- struct{}{}:
	default:
	}
}

func (f *fakeIdle) send(s string) { f.mu.Lock(); f.buf = append(f.buf, s...); f.mu.Unlock(); f.poke() }
func (f *fakeIdle) hangup()       { f.mu.Lock(); f.nudge = true; f.mu.Unlock(); f.poke() }

func (f *fakeIdle) Read(p []byte) (int, error) {
	for {
		f.mu.Lock()
		if len(f.buf) > 0 {
			n := copy(p, f.buf)
			f.buf = f.buf[n:]
			f.mu.Unlock()
			return n, nil
		}
		if f.closed {
			f.mu.Unlock()
			return 0, io.EOF
		}
		f.mu.Unlock()
		<-f.ch
	}
}

func (f *fakeIdle) wait(d time.Duration) (bool, bool, error) {
	select {
	case <-f.ch:
	case <-time.After(d):
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	n := f.nudge
	f.nudge = false
	ready := len(f.buf) > 0 || f.closed
	if ready {
		f.poke() // Read must not block on the token wait took
	}
	return ready, n, nil
}

// idleRig is a reloader on a temp brew layout, with exec recorded.
type idleRig struct {
	dir, link string
	r         *reloader
	src       *fakeIdle
	out, log  lockedBuf
	execs     chan string // the resume file's Pending, per exec
	server    lockedBuf   // what the server read
}

func newIdleRig(t *testing.T, tick time.Duration) *idleRig {
	t.Helper()
	g := &idleRig{dir: t.TempDir(), execs: make(chan string, 4)}
	g.link = filepath.Join(g.dir, "bin", "hug")
	os.MkdirAll(filepath.Dir(g.link), 0o755)
	g.keg("0.0.1", "old")
	if err := os.Symlink(filepath.Join("..", "Cellar", "hug", "0.0.1", "bin", "hug"), g.link); err != nil {
		t.Skipf("no symlinks: %v", err)
	}
	g.r = newReloader(g.link, &g.out, &g.log)
	g.r.tick = tick
	g.r.probe = func(string) error { return nil }
	g.r.exec = func(path string, argv, env []string) error {
		for _, e := range env {
			if p, ok := strings.CutPrefix(e, resumeEnv+"="); ok {
				st, _ := os.ReadFile(p)
				g.execs <- string(st)
			}
		}
		return os.ErrInvalid // stay in this process
	}
	go func() {
		b := make([]byte, 4096)
		for {
			n, err := g.r.Read(b)
			g.server.Write(b[:n])
			if err != nil {
				return
			}
		}
	}()
	g.src = newFakeIdle()
	go g.r.pump(g.src, nil)
	t.Cleanup(func() { g.src.mu.Lock(); g.src.closed = true; g.src.mu.Unlock(); g.src.poke() })
	return g
}

// initialize sends the client's handshake and answers it once the server
// has read it, as the SDK would.
func (g *idleRig) initialize(t *testing.T) {
	t.Helper()
	g.src.send(initLine + initedLine)
	for i := 0; !strings.Contains(g.server.String(), "notifications/initialized"); i++ {
		if i > 500 {
			t.Fatal("the server never read the handshake")
		}
		time.Sleep(10 * time.Millisecond)
	}
	g.r.Write([]byte(`{"jsonrpc":"2.0","id":1,"result":{}}` + "\n"))
}

func (g *idleRig) keg(v, body string) {
	p := filepath.Join(g.dir, "Cellar", "hug", v, "bin", "hug")
	os.MkdirAll(filepath.Dir(p), 0o755)
	os.WriteFile(p, []byte(body), 0o755)
}

// brew moves bin/hug to a new keg and removes the old one.
func (g *idleRig) brewUpgrade(t *testing.T) {
	g.keg("0.0.2", "new binary")
	os.Remove(g.link)
	if err := os.Symlink(filepath.Join("..", "Cellar", "hug", "0.0.2", "bin", "hug"), g.link); err != nil {
		t.Fatal(err)
	}
	os.RemoveAll(filepath.Join(g.dir, "Cellar", "hug", "0.0.1"))
}

func (g *idleRig) execd(d time.Duration) (string, bool) {
	select {
	case p := <-g.execs:
		return p, true
	case <-time.After(d):
		return "", false
	}
}

const initLine = `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18"}}` + "\n"
const initedLine = `{"jsonrpc":"2.0","method":"notifications/initialized"}` + "\n"

// The symlink moves to a new keg; nothing is sent; the idle server execs
// the launch path, handing over the client's initialize to replay.
func TestIdleReloadFollowsSymlink(t *testing.T) {
	g := newIdleRig(t, 20*time.Millisecond)
	g.initialize(t)
	time.Sleep(100 * time.Millisecond)
	if _, ok := g.execd(0); ok {
		t.Fatal("reloaded before any upgrade")
	}
	g.brewUpgrade(t)
	st, ok := g.execd(5 * time.Second)
	if !ok {
		t.Fatalf("no idle reload after the symlink moved; log:\n%s", g.log.String())
	}
	if !strings.Contains(st, `"init":`) || !strings.Contains(st, `"initialized":true`) {
		t.Errorf("the hand-over lacks the initialize to replay: %s", st)
	}
	if strings.Contains(st, `"pending"`) {
		t.Errorf("an idle reload has no request to hand on: %s", st)
	}
	if !strings.Contains(g.log.String(), "changed while no request ran") {
		t.Errorf("log:\n%s", g.log.String())
	}
}

// A file being copied (it changes between ticks) is not exec'ed until it
// holds still for one tick.
func TestIdleReloadWaitsForStillFile(t *testing.T) {
	g := newIdleRig(t, 50*time.Millisecond)
	g.brewUpgrade(t)
	stop := time.Now().Add(400 * time.Millisecond)
	for i := 0; time.Now().Before(stop); i++ {
		g.keg("0.0.2", strings.Repeat("x", i+20))
		os.Chtimes(filepath.Join(g.dir, "Cellar", "hug", "0.0.2", "bin", "hug"), time.Now(), time.Now().Add(time.Duration(i)*time.Second))
		time.Sleep(10 * time.Millisecond)
	}
	if _, ok := g.execd(0); ok {
		t.Fatal("exec'ed a file still being written")
	}
	if _, ok := g.execd(5 * time.Second); !ok {
		t.Fatalf("no reload once the file held still; log:\n%s", g.log.String())
	}
}

// No idle reload while a request runs: it waits for the reply.
func TestIdleReloadWaitsForRunningRequest(t *testing.T) {
	g := newIdleRig(t, 20*time.Millisecond)
	g.initialize(t)
	g.src.send(`{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"task_list"}}` + "\n")
	time.Sleep(50 * time.Millisecond)
	g.brewUpgrade(t)
	if _, ok := g.execd(300 * time.Millisecond); ok {
		t.Fatal("reloaded while request 2 ran")
	}
	g.r.Write([]byte(`{"jsonrpc":"2.0","id":2,"result":{}}` + "\n"))
	if _, ok := g.execd(5 * time.Second); !ok {
		t.Fatalf("no reload after request 2 was answered; log:\n%s", g.log.String())
	}
	if !strings.Contains(g.out.String(), `"id":2`) {
		t.Errorf("request 2's reply never reached the client:\n%s", g.out.String())
	}
}

// SIGHUP on an unchanged hug: says so, no exec; after an upgrade: reloads
// at once, without waiting for the file to hold still a tick.
func TestHangupReloadsNow(t *testing.T) {
	g := newIdleRig(t, time.Hour) // only the nudge can wake it
	g.src.hangup()
	for i := 0; i < 200 && !strings.Contains(g.log.String(), "is unchanged"); i++ {
		time.Sleep(10 * time.Millisecond)
	}
	if !strings.Contains(g.log.String(), "SIGHUP: "+g.link+" is unchanged") {
		t.Fatalf("log:\n%s", g.log.String())
	}
	g.brewUpgrade(t)
	g.src.hangup()
	if _, ok := g.execd(2 * time.Second); !ok {
		t.Fatalf("SIGHUP after an upgrade did not reload; log:\n%s", g.log.String())
	}
}

// Every byte the client sent before an exec is handed over: a request that
// arrives just as the file changes is served by the new hug.
func TestReloadAtRequestHandsOverHeldBytes(t *testing.T) {
	g := newIdleRig(t, time.Hour)
	g.initialize(t)
	time.Sleep(50 * time.Millisecond)
	g.brewUpgrade(t)
	two := `{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"task_list"}}` + "\n"
	three := `{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"task_types"}}` + "\n"
	g.src.send(two + three)
	st, ok := g.execd(5 * time.Second)
	if !ok {
		t.Fatalf("no reload at the request; log:\n%s", g.log.String())
	}
	var rs resumeState
	if err := json.Unmarshal([]byte(st), &rs); err != nil {
		t.Fatal(err)
	}
	if string(rs.Pending) != two+three {
		t.Errorf("hand-over pending = %q, want both requests %q", rs.Pending, two+three)
	}
}

func TestVersionOlder(t *testing.T) {
	for _, c := range []struct {
		a, b string
		want bool
	}{
		{"0.7.108", "0.7.112", true},
		{"0.7.112", "0.7.108", false},
		{"0.7.112", "0.7.112", false},
		{"0.0.1-old", "0.0.2-new", true},
		{"0.9.0", "0.10.0", true},
		{"1.0.0", "0.99.99", false},
		{"", "0.7.1", false},
		{"0.7.1", "dev", false},
	} {
		if got := versionOlder(c.a, c.b); got != c.want {
			t.Errorf("versionOlder(%q, %q) = %v, want %v", c.a, c.b, got, c.want)
		}
	}
}

// What ps and lsof showed on Jørgen's Mac on 2026-11-25, read into rows.
func TestRunningFromPSAndLsof(t *testing.T) {
	ps := []byte(` 1402 Wed Nov 18 19:55:22 2026     /opt/homebrew/bin/hug mcp --jcode-home /Users/jhf/.jcode
 8265 Thu Nov 19 17:52:32 2026     /opt/homebrew/bin/hug mcp --jcode-home /Users/jhf/Projects/Brovs/.jcode
58480 Sun Nov 22 23:11:16 2026     hug mcp
32504 Mon Nov 23 09:00:01 2026     hug tui
 4002 Tue Nov 25 10:00:00 2026     /opt/homebrew/bin/hug task ls --json
 4003 Tue Nov 25 10:00:00 2026     muninn mcp
 4004 Tue Nov 25 10:00:00 2026     /usr/bin/vim hug mcp
 4005 Tue Nov 25 10:00:00 2026     /opt/homebrew/bin/hug mem serve
  777 Tue Nov 25 10:00:00 2026     /opt/homebrew/bin/hug version --running
`)
	procs := pickHug(parsePSStart(ps), 777)
	var got []string
	for _, p := range procs {
		got = append(got, p.Runs)
	}
	if strings.Join(got, ",") != "mcp,mcp,mcp,tui,task ls,mem mcp,mem serve" {
		t.Fatalf("runs: %v", got)
	}
	if procs[0].Started.Format("2006-01-02 15:04:05") != "2026-11-18 19:55:22" {
		t.Errorf("start: %v", procs[0].Started)
	}
	lsof := parseLsofExe([]byte("p1402\nftxt\ni1905458927\nn/hug\nftxt\ni1\nn/usr/lib/dyld\np8265\nftxt\ni42\nn/opt/homebrew/Cellar/hug/0.7.113/bin/hug\n"))
	if lsof[1402].name != "/hug" || lsof[1402].inode != 1905458927 || lsof[8265].name != "/opt/homebrew/Cellar/hug/0.7.113/bin/hug" {
		t.Fatalf("lsof: %+v", lsof)
	}
	procs[0].Exe, procs[0].Gone = "/hug", true
	procs[1].Exe, procs[1].Version = "/opt/homebrew/Cellar/hug/0.7.113/bin/hug", "0.7.113"
	procs[2].Exe, procs[2].Version = "/opt/homebrew/Cellar/hug/0.7.114/bin/hug", "0.7.114"
	procs[3].Exe, procs[3].Version = "/opt/homebrew/Cellar/hug/0.7.108/bin/hug", "0.7.108"
	var b strings.Builder
	code := printRunning(&b, procs, "0.7.114")
	out := b.String()
	if code != 5 {
		t.Errorf("exit %d, want 5", code)
	}
	for _, want := range []string{"mcp: pid 8265 run an older hug", "tui: pid 32504 run an older hug", "fix: a hug mcp from this release on reloads", "/hug (removed or replaced since it started)", "version unknown"} {
		if !strings.Contains(out, want) {
			t.Errorf("lacks %q:\n%s", want, out)
		}
	}
	for _, l := range strings.Split(out, "\n") {
		if strings.HasPrefix(strings.TrimSpace(l), "58480 ") && strings.Contains(l, "older") {
			t.Errorf("0.7.114 flagged older than 0.7.114: %s", l)
		}
	}
}

// Where hug cannot list processes it says so and how to look, exit 1.
func TestVersionRunningUnsupported(t *testing.T) {
	var b strings.Builder
	if code := versionRunning(&b, "windows"); code != 1 {
		t.Errorf("exit %d, want 1", code)
	}
	if got := b.String(); !strings.Contains(got, "not supported on windows") || !strings.Contains(got, "\nfix: ") {
		t.Errorf("got:\n%s", got)
	}
}

// countingIdle counts the waits: a pump that never advances its tick spins.
type countingIdle struct {
	*fakeIdle
	mu sync.Mutex
	n  int
}

func (c *countingIdle) wait(d time.Duration) (bool, bool, error) {
	c.mu.Lock()
	c.n++
	c.mu.Unlock()
	return c.fakeIdle.wait(d)
}

// With no known hug file (launch ""), the idle pump still waits a tick
// between looks: it never spins.
func TestIdlePumpWithoutLaunchDoesNotSpin(t *testing.T) {
	var out, log lockedBuf
	r := newReloader("", &out, &log)
	r.tick = 20 * time.Millisecond
	src := &countingIdle{fakeIdle: newFakeIdle()}
	go r.pump(src, nil)
	time.Sleep(300 * time.Millisecond)
	src.mu.Lock()
	n := src.n
	src.mu.Unlock()
	src.fakeIdle.mu.Lock()
	src.fakeIdle.closed = true
	src.fakeIdle.mu.Unlock()
	src.fakeIdle.poke()
	if n < 5 || n > 60 { // a spin calls wait once, then never again
		t.Fatalf("%d waits in 300 ms at a 20 ms tick; want about 15", n)
	}
}
