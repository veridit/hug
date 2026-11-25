package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// hug mcp serves every request with the hug installed now (mbdfzs): an app
// keeps its server alive across upgrades, and an old server writes the board
// with old rules (5408358f on hug/tasks came from one). The reloader sits
// between stdin/stdout and the SDK. It sees each JSON-RPC line, keeps the ids
// of requests still running, and before it passes on a new request it checks
// the hug the app started. If that file changed, it waits until no request
// is running, then (macOS, Linux) execs the new hug on the same pid and fds;
// the new process replays the client's initialize (its reply is dropped) and
// carries on with the request that was held, so the client sees no
// disconnect. Windows has no exec: there the held request gets an error that
// says to run it again, and the server exits so the app starts the new one.

// resumeEnv names the file a reloading hug mcp hands to the next one.
const resumeEnv = "HUG_MCP_RESUME"

// resumeProbe is what hug mcp --can-resume prints: a hug that cannot take
// over a session (older than mbdfzs) prints something else and is not
// exec'ed.
const resumeProbe = "hug mcp resume 1"

// replayID is the id of the replayed initialize; its reply is not sent on.
const replayID = `"hug-reload-initialize"`

// drainWait bounds the wait for running requests before a reload; past it the
// request is served by this binary and the reload is tried at the next one.
var drainWait = 30 * time.Second

// idleTick is how often a hug mcp with no traffic looks at its hug (rvb3j5).
// A stat every 5 s costs nothing; a changed file reloads after it holds
// still one more tick, so 5-10 s after an upgrade. A timer, not fsnotify:
// brew moves the bin/hug symlink (a watch on the file it pointed at never
// fires, and a watch on /opt/homebrew/bin fires for every formula, with a
// kqueue fd per file there), while a stat of the launch path follows it
// exactly as the exec does, the same on every system.
var idleTick = 5 * time.Second

type resumeState struct {
	Init        []byte `json:"init,omitempty"`        // the client's initialize request
	Initialized bool   `json:"initialized,omitempty"` // notifications/initialized was sent
	Pending     []byte `json:"pending,omitempty"`     // bytes read from stdin, not yet served
	From        string `json:"from,omitempty"`        // the version that handed over
}

type fileID struct {
	info os.FileInfo
}

func (a fileID) same(b fileID) bool {
	if a.info == nil || b.info == nil {
		return a.info == b.info
	}
	return os.SameFile(a.info, b.info) && a.info.ModTime().Equal(b.info.ModTime()) && a.info.Size() == b.info.Size()
}

type reloader struct {
	launch   string // the hug the app started; "" when unknown (no reload)
	start    fileID
	rejected fileID // a changed file that failed the probe: not tried again
	out      io.Writer
	logw     io.Writer
	pr       *io.PipeReader
	pw       *io.PipeWriter

	mu       sync.Mutex
	cond     *sync.Cond
	inflight map[string]bool

	writeMu     sync.Mutex
	init        []byte
	initialized bool

	exec  func(path string, argv, env []string) error // nil: no exec (Windows)
	exit  func(int)
	probe func(path string) error

	// The idle watch (rvb3j5), run by the pump goroutine only.
	tick     time.Duration // how often an idle server looks at its hug
	lastIdle time.Time
	seen     fileID // a changed file seen once: reloaded when it holds still
	nudged   bool   // SIGHUP: reload as soon as no request runs
}

func newReloader(launch string, out, logw io.Writer) *reloader {
	r := &reloader{launch: launch, out: out, logw: logw, inflight: map[string]bool{}, exec: execHug, exit: os.Exit, probe: probe, tick: idleTick}
	r.cond = sync.NewCond(&r.mu)
	r.pr, r.pw = io.Pipe()
	if launch != "" {
		r.start = statID(launch)
	}
	return r
}

func statID(p string) fileID {
	info, err := os.Stat(p)
	if err != nil {
		return fileID{}
	}
	return fileID{info}
}

// launchPath is the hug the app started: os.Args[0] as a path, else found on
// PATH as the app's spawn found it.
func launchPath() string {
	arg0 := os.Args[0]
	if !strings.ContainsAny(arg0, `/\`) {
		p, err := exec.LookPath(arg0)
		if err != nil {
			return ""
		}
		arg0 = p
	}
	p, err := filepath.Abs(arg0)
	if err != nil {
		return ""
	}
	return p
}

// takeResume reads and removes the file a reloading hug mcp left.
func takeResume() (*resumeState, error) {
	p := os.Getenv(resumeEnv)
	if p == "" {
		return nil, nil
	}
	os.Unsetenv(resumeEnv)
	b, err := os.ReadFile(p)
	os.Remove(p)
	if err != nil {
		return nil, err
	}
	var st resumeState
	if err := json.Unmarshal(b, &st); err != nil {
		return nil, err
	}
	return &st, nil
}

// msg is the part of a JSON-RPC message the reloader reads.
type msg struct {
	ID     json.RawMessage `json:"id"`
	Method *string         `json:"method"`
	Params struct {
		RequestID json.RawMessage `json:"requestId"`
	} `json:"params"`
}

func (m msg) request() bool  { return m.Method != nil && len(m.ID) > 0 && string(m.ID) != "null" }
func (m msg) response() bool { return m.Method == nil && len(m.ID) > 0 && string(m.ID) != "null" }
func (m msg) key() string    { return idKey(m.ID) }

func idKey(id json.RawMessage) string {
	var b bytes.Buffer
	if json.Compact(&b, id) != nil {
		return string(id)
	}
	return b.String()
}

func parseMsgs(line []byte) []msg {
	t := bytes.TrimSpace(line)
	if len(t) == 0 {
		return nil
	}
	if t[0] == '[' {
		var ms []msg
		if json.Unmarshal(t, &ms) != nil {
			return nil
		}
		return ms
	}
	var m msg
	if json.Unmarshal(t, &m) != nil {
		return nil
	}
	return []msg{m}
}

// Write takes one message from the server to the client.
func (r *reloader) Write(p []byte) (int, error) {
	ms := parseMsgs(p)
	if len(ms) == 1 && ms[0].response() && ms[0].key() == replayID {
		return len(p), nil // the replayed initialize: the client has its reply
	}
	r.writeMu.Lock()
	n, err := r.out.Write(p)
	r.writeMu.Unlock()
	r.mu.Lock()
	for _, m := range ms {
		if m.response() {
			delete(r.inflight, m.key())
		}
	}
	r.cond.Broadcast()
	r.mu.Unlock()
	return n, err
}

func (r *reloader) Close() error { return nil }

// Read is the server's stdin.
func (r *reloader) Read(p []byte) (int, error) { return r.pr.Read(p) }

// idleSource is a stdin the pump can wait on without reading it (rvb3j5):
// while it waits, no byte the client sent is held in this process, so an
// idle reload hands the new hug everything read so far.
type idleSource interface {
	io.Reader
	// wait blocks until there is something to read (ready), d passes, or a
	// nudge (SIGHUP) arrives (nudged). An error means it cannot wait: the
	// pump then reads as before and reloads only at a request.
	wait(d time.Duration) (ready, nudged bool, err error)
}

// pump copies client lines to the server, reloading between requests and,
// with an idleSource, while no request runs.
func (r *reloader) pump(src io.Reader, resumed *resumeState) {
	var pending []byte // read from src, not yet passed on: at most part of a line
	if resumed != nil {
		r.replay(resumed)
		pending = append(pending, resumed.Pending...)
	}
	ws, idle := src.(idleSource)
	buf := make([]byte, 1<<16)
	r.lastIdle = time.Now()
	for {
		for {
			i := bytes.IndexByte(pending, '\n')
			if i < 0 {
				break
			}
			line := append([]byte(nil), pending[:i+1]...)
			if !r.serve(line, pending) {
				return
			}
			pending = pending[i+1:]
		}
		pending = append([]byte(nil), pending...)
		if idle {
			d := r.tick - time.Since(r.lastIdle)
			if d <= 0 {
				r.idleCheck(pending, false) // also under steady notifications
				continue
			}
			ready, nudged, err := ws.wait(d)
			switch {
			case err != nil:
				fmt.Fprintf(r.logw, "hug mcp: cannot wait on stdin (%v); this hug reloads only at a request\n", err)
				idle = false
			case nudged:
				r.idleCheck(pending, true)
				continue
			case !ready:
				continue
			}
		}
		n, err := src.Read(buf)
		pending = append(pending, buf[:n]...)
		if err != nil {
			for len(pending) > 0 {
				i := bytes.IndexByte(pending, '\n') + 1
				if i == 0 {
					i = len(pending)
				}
				if !r.serve(append([]byte(nil), pending[:i]...), pending) {
					return
				}
				pending = pending[i:]
			}
			r.pw.Close()
			return
		}
	}
}

// serve passes one line to the server; held is the line and every byte read
// after it, which a reload hands on. false: the server is gone.
func (r *reloader) serve(line, held []byte) bool {
	ms := parseMsgs(line)
	if r.newRequest(ms) && r.changed() {
		r.reload(held)
	}
	r.mu.Lock()
	for _, m := range ms {
		if m.request() {
			r.inflight[m.key()] = true
		}
		if m.Method != nil && *m.Method == "initialize" && len(ms) == 1 {
			r.init = append([]byte(nil), line...)
		}
		if m.Method != nil && *m.Method == "notifications/initialized" {
			r.initialized = true
		}
		// A cancelled request gets no reply: it is not running.
		if m.Method != nil && *m.Method == "notifications/cancelled" && len(m.Params.RequestID) > 0 {
			delete(r.inflight, idKey(m.Params.RequestID))
			r.cond.Broadcast()
		}
	}
	r.mu.Unlock()
	_, werr := r.pw.Write(line)
	return werr == nil
}

// idleCheck runs every tick whatever the client sends (rvb3j5), and at once
// on SIGHUP: it looks at the hug the app started. A changed file that
// has held still for a tick (a copy in progress does not count; SIGHUP does
// not wait) is reloaded when no request is running and no message is half
// read, with the same hand-over as at a request.
func (r *reloader) idleCheck(partial []byte, nudged bool) {
	r.lastIdle = time.Now()
	if r.launch == "" {
		if nudged {
			fmt.Fprintf(r.logw, "hug mcp: SIGHUP: the hug file this server started from is unknown, so it cannot reload; this hug %s serves on\nfix: restart the app to start the installed hug\n", Version)
		}
		return
	}
	if nudged {
		r.nudged = true
	}
	now := statID(r.launch)
	if !r.changedTo(now) {
		r.seen = fileID{}
		if r.nudged {
			r.nudged = false
			fmt.Fprintf(r.logw, "hug mcp: SIGHUP: %s is unchanged; this hug %s serves on\n", r.launch, Version)
		}
		return
	}
	if !r.nudged && !now.same(r.seen) {
		r.seen = now // changed: wait one more tick for it to hold still
		return
	}
	r.mu.Lock()
	busy := len(r.inflight)
	r.mu.Unlock()
	if busy > 0 || len(partial) > 0 {
		return // a request is running or arriving: the next tick, or the request, reloads
	}
	r.nudged, r.seen = false, fileID{}
	fmt.Fprintf(r.logw, "hug mcp: %s changed while no request ran\n", r.launch)
	r.reload(nil)
}

func (r *reloader) newRequest(ms []msg) bool {
	for _, m := range ms {
		if m.request() && *m.Method != "initialize" {
			return true
		}
	}
	return false
}

// replay gives a resumed server the session the client set up.
func (r *reloader) replay(st *resumeState) {
	r.init, r.initialized = st.Init, st.Initialized
	if len(st.Init) > 0 {
		var req map[string]json.RawMessage
		if json.Unmarshal(st.Init, &req) == nil {
			req["id"] = json.RawMessage(replayID)
			if b, err := json.Marshal(req); err == nil {
				r.pw.Write(append(b, '\n'))
			}
		}
	}
	if st.Initialized {
		r.pw.Write([]byte(`{"jsonrpc":"2.0","method":"notifications/initialized"}` + "\n"))
	}
	fmt.Fprintf(r.logw, "hug mcp: reloaded as hug %s (was %s), same connection\n", Version, st.From)
}

func (r *reloader) changed() bool {
	if r.launch == "" {
		return false
	}
	return r.changedTo(statID(r.launch))
}

// changedTo says now, the file the launch path leads to (a symlink is
// followed: brew moves bin/hug to a new keg), is another hug than the one
// this server started from, and not one that failed the probe.
func (r *reloader) changedTo(now fileID) bool {
	return now.info != nil && !now.same(r.start) && !now.same(r.rejected)
}

// drained waits until no request is running, at most d.
func (r *reloader) drained(d time.Duration) bool {
	deadline := time.Now().Add(d)
	t := time.AfterFunc(d, func() { r.mu.Lock(); r.cond.Broadcast(); r.mu.Unlock() })
	defer t.Stop()
	r.mu.Lock()
	defer r.mu.Unlock()
	for len(r.inflight) > 0 && time.Now().Before(deadline) {
		r.cond.Wait()
	}
	return len(r.inflight) == 0
}

// probe asks the changed hug whether it can take over this session.
func probe(path string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, path, "mcp", "--can-resume")
	cmd.Env = withoutEnv(os.Environ(), resumeEnv)
	out, err := cmd.Output()
	if strings.TrimSpace(string(out)) != resumeProbe {
		if err == nil {
			err = fmt.Errorf("it printed %q", strings.TrimSpace(string(out)))
		}
		return err
	}
	return nil
}

func withoutEnv(env []string, key string) []string {
	var out []string
	for _, e := range env {
		if !strings.HasPrefix(e, key+"=") {
			out = append(out, e)
		}
	}
	return out
}

// reload hands the session to the changed hug, or returns to serve held
// here. held is every byte read from the client and not yet served.
func (r *reloader) reload(held []byte) {
	if !r.drained(drainWait) {
		fmt.Fprintf(r.logw, "hug mcp: %s changed, but requests are still running after %s; this hug serves on and reloads at a later request\n", r.launch, drainWait)
		return
	}
	now := statID(r.launch)
	if r.exec == nil {
		r.stopForRestart(append([]byte(nil), held...))
		return
	}
	if err := r.probe(r.launch); err != nil {
		r.rejected = now
		fmt.Fprintf(r.logw, "hug mcp: %s changed, but it cannot take over this session (%v); this hug serves on\nfix: restart the app to start the installed hug\n", r.launch, err)
		return
	}
	r.mu.Lock()
	st := resumeState{Init: r.init, Initialized: r.initialized, Pending: append([]byte(nil), held...), From: Version}
	r.mu.Unlock()
	b, _ := json.Marshal(st)
	f, err := os.CreateTemp("", "hug-mcp-resume-*.json")
	if err == nil {
		_, err = f.Write(b)
		if cerr := f.Close(); err == nil {
			err = cerr
		}
	}
	if err != nil {
		fmt.Fprintf(r.logw, "hug mcp: cannot hand the session to %s: %v; this hug serves on\n", r.launch, err)
		r.rejected = now
		return
	}
	env := append(withoutEnv(os.Environ(), resumeEnv), resumeEnv+"="+f.Name())
	fmt.Fprintf(r.logw, "hug mcp: %s changed; reloading it on this connection\n", r.launch)
	r.writeMu.Lock() // nothing more goes out from this binary
	err = r.exec(r.launch, os.Args, env)
	r.writeMu.Unlock()
	os.Remove(f.Name())
	r.rejected = now
	fmt.Fprintf(r.logw, "hug mcp: exec %s: %v; this hug serves on\nfix: restart the app to start the installed hug\n", r.launch, err)
}

// stopForRestart answers every request read and not yet served with an
// error that says to run it again, and exits: on Windows the app starts the
// new hug when this one is gone. No request goes unanswered.
func (r *reloader) stopForRestart(held []byte) {
	r.writeMu.Lock()
	var ms []msg
	for _, l := range bytes.Split(held, []byte("\n")) {
		ms = append(ms, parseMsgs(l)...)
	}
	for _, m := range ms {
		if !m.request() {
			continue
		}
		e, _ := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": json.RawMessage(m.ID), "error": map[string]any{
			"code":    -32000,
			"message": "hug was upgraded while this server ran: it stops now so the app starts the installed hug; run the call again (fix: if the app does not restart it, restart the app)",
		}})
		r.out.Write(append(e, '\n'))
	}
	fmt.Fprintf(r.logw, "hug mcp: %s changed; stopping so the app starts it (no exec on this system)\n", r.launch)
	r.exit(0)
}
