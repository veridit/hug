package main

import (
	"bufio"
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

	exec func(path string, argv, env []string) error // nil: no exec (Windows)
	exit func(int)
}

func newReloader(launch string, out, logw io.Writer) *reloader {
	r := &reloader{launch: launch, out: out, logw: logw, inflight: map[string]bool{}, exec: execHug, exit: os.Exit}
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

// pump copies client lines to the server, reloading between requests.
func (r *reloader) pump(src io.Reader, resumed *resumeState) {
	if resumed != nil {
		r.replay(resumed)
		src = io.MultiReader(bytes.NewReader(resumed.Pending), src)
	}
	br := bufio.NewReaderSize(src, 1<<16)
	for {
		line, err := br.ReadBytes('\n')
		if len(line) > 0 {
			ms := parseMsgs(line)
			if r.newRequest(ms) && r.changed() {
				r.reload(line, br)
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
			if _, werr := r.pw.Write(line); werr != nil {
				return
			}
		}
		if err != nil {
			r.pw.Close()
			return
		}
	}
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
	now := statID(r.launch)
	if now.info == nil || now.same(r.start) || now.same(r.rejected) {
		return false
	}
	return true
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

// reload hands the session to the changed hug, or returns to serve line here.
func (r *reloader) reload(line []byte, br *bufio.Reader) {
	if !r.drained(drainWait) {
		fmt.Fprintf(r.logw, "hug mcp: %s changed, but requests are still running after %s; this hug serves on and reloads at a later request\n", r.launch, drainWait)
		return
	}
	now := statID(r.launch)
	if r.exec == nil {
		buffered, _ := br.Peek(br.Buffered())
		r.stopForRestart(append(append([]byte(nil), line...), buffered...))
		return
	}
	if err := probe(r.launch); err != nil {
		r.rejected = now
		fmt.Fprintf(r.logw, "hug mcp: %s changed, but it cannot take over this session (%v); this hug serves on\nfix: restart the app to start the installed hug\n", r.launch, err)
		return
	}
	buffered, _ := br.Peek(br.Buffered())
	r.mu.Lock()
	st := resumeState{Init: r.init, Initialized: r.initialized, Pending: append(append([]byte(nil), line...), buffered...), From: Version}
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
