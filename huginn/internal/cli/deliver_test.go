package cli

import (
	"bufio"
	"context"
	"encoding/json"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

// fakeJcode is a jcode daemon on a Unix socket, as far as notify_session
// goes (jcode-app-core server/client_actions.rs handle_notify_session): a
// live session gets the message appended to its journal as a user
// message (its turn, or a soft interrupt), then ack and done; any other
// session is refused, "Session 'S' is not currently live". hook, when set,
// is what the session's hug hook does at the turn the message starts.
type fakeJcode struct {
	home string
	live map[string]bool
	hook func(sid string)
	mu   sync.Mutex
	got  []string
}

func newFakeJcode(t *testing.T, live ...string) *fakeJcode {
	t.Helper()
	// A short path: a Unix socket path is capped near 104 bytes on macOS.
	dir, err := os.MkdirTemp("", "jc")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(dir) })
	f := &fakeJcode{home: dir, live: map[string]bool{}}
	os.MkdirAll(filepath.Join(dir, "sessions"), 0o700)
	os.MkdirAll(filepath.Join(dir, "runtime"), 0o700)
	for _, s := range live {
		f.live[s] = true
	}
	l, err := net.Listen("unix", filepath.Join(dir, "runtime", "jcode.sock"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { l.Close() })
	go func() {
		for {
			c, err := l.Accept()
			if err != nil {
				return
			}
			go f.serve(c)
		}
	}()
	return f
}

// session makes a session file named name in the fake's home.
func (f *fakeJcode) session(t *testing.T, name string, ms int64, live bool) string {
	t.Helper()
	sid := "session_" + name + "_" + itoaTest(ms) + "_00ab"
	if err := os.WriteFile(filepath.Join(f.home, "sessions", sid+".json"), []byte(`{"id":"`+sid+`","messages":[]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	f.mu.Lock()
	f.live[sid] = live
	f.mu.Unlock()
	return sid
}

func itoaTest(n int64) string { b, _ := json.Marshal(n); return string(b) }

func (f *fakeJcode) serve(c net.Conn) {
	defer c.Close()
	sc := bufio.NewScanner(c)
	sc.Buffer(make([]byte, 0, 64<<10), 16<<20)
	for sc.Scan() {
		var req struct {
			Type, SessionID, Message string
			ID                       int
		}
		var raw map[string]any
		json.Unmarshal(sc.Bytes(), &raw)
		req.Type, _ = raw["type"].(string)
		req.SessionID, _ = raw["session_id"].(string)
		req.Message, _ = raw["message"].(string)
		if n, ok := raw["id"].(float64); ok {
			req.ID = int(n)
		}
		reply := func(v map[string]any) { b, _ := json.Marshal(v); c.Write(append(b, '\n')) }
		reply(map[string]any{"type": "ack", "id": req.ID})
		if req.Type != "notify_session" {
			continue
		}
		f.mu.Lock()
		live := f.live[req.SessionID]
		f.mu.Unlock()
		if !live {
			reply(map[string]any{"type": "error", "id": req.ID, "message": "Session '" + req.SessionID + "' is not currently live"})
			continue
		}
		// The turn starts with the message in the transcript (jcode
		// appends it before turn_start fires), then the hook runs.
		line, _ := json.Marshal(map[string]any{"meta": map[string]any{}, "append_messages": []any{map[string]any{"id": "m1", "role": "user", "display_role": "system", "content": []any{map[string]any{"type": "text", "text": req.Message}}}}})
		jf, _ := os.OpenFile(filepath.Join(f.home, "sessions", req.SessionID+".journal.jsonl"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
		jf.Write(append(line, '\n'))
		jf.Close()
		f.mu.Lock()
		f.got = append(f.got, req.SessionID+"\x00"+req.Message)
		f.mu.Unlock()
		reply(map[string]any{"type": "done", "id": req.ID})
		if f.hook != nil {
			go f.hook(req.SessionID)
		}
	}
}

func (f *fakeJcode) messages() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]string(nil), f.got...)
}

// gmzd87 check 1 (end to end, the daemon faked at its socket): a request
// written while the coordinator's session is idle reaches it through hug
// itself (no poller): the daemon gets notify_session for crab's newest
// session with the request verbatim and the resolve command; History says
// delivered; crab's hook, at the turn the message starts, finds it in the
// transcript and writes acknowledged; crab's first write marks it in
// progress; resolve answers it. A coordinator with no live session leaves
// "pending, not delivered: REASON" in hug task requests.
func TestRequestReachesCoordinator(t *testing.T) {
	t.Parallel()
	jc := newFakeJcode(t)
	old := jc.session(t, "crab", 1000, false)
	crab := jc.session(t, "crab", 2000, true)
	// The older, dead session was written last: delivery must still find
	// the live one.
	now := time.Now()
	os.Chtimes(filepath.Join(jc.home, "sessions", old+".json"), now, now)
	acked := make(chan []string, 4)
	jc.hook = func(sid string) {
		// No t here: the hook may run after the test has failed.
		got, err := Acknowledge(context.Background(), jc.home, sid)
		if err != nil {
			got = append(got, "error: "+err.Error())
		}
		acked <- got
	}
	a := app(t)
	a.JcodeHomes = []string{jc.home}
	execute(t, a, "init")
	id := execute(t, a, "new", "Tags", "--goal", "g", "--purpose", "p", "--principle", "q", "--accept", "a")["task"].(Reply)["id"].(string)
	// No coordinator: not delivered, and it says why.
	r := execute(t, a, "request", id, "first ask", "--as", "jhf")
	if r["receipt"] != "pending, not delivered: no coordinator set (hug task coordinator NAME)" {
		t.Fatalf("no coordinator: receipt %v", r["receipt"])
	}
	if _, out := runText(a, "requests"); !strings.Contains(out, "  receipt: pending, not delivered: no coordinator set (hug task coordinator NAME)\n") {
		t.Fatalf("requests without a coordinator:\n%s", out)
	}
	execute(t, a, "coordinator", "crab")
	a.In = strings.NewReader("make it per board: \"tags\"\nand split the TUI part\n")
	r = execute(t, a, "request", id, "-", "--as", "jhf")
	if r["receipt"] != "pending, delivered to crab ("+crab+")" {
		t.Fatalf("receipt: %v", r["receipt"])
	}
	msgs := jc.messages()
	if len(msgs) != 1 {
		t.Fatalf("daemon got %d messages: %q", len(msgs), msgs)
	}
	sid, msg, _ := strings.Cut(msgs[0], "\x00")
	board := realPath(t, a.Cwd)
	for _, want := range []string{
		"hug: jhf wrote request 2 on " + id + " (Tags) on board " + filepath.Base(board) + " (" + board + "), to you as the board coordinator:\n",
		"  make it per board: \"tags\"\n  and split the TUI part\n",
		"Turn it into changes to the ticket, then (in " + board + "): hug task resolve " + id + " 2 \"what changed\"\n",
		"[hug " + id + " request 2 ",
	} {
		if sid != crab || !strings.Contains(msg, want) {
			t.Fatalf("message to %s lacks %q:\n%s", sid, want, msg)
		}
	}
	// The hook answers on every path (an error too), so wait for it, not
	// for a clock (awht8w): at load 95-500 a niced git took up to 14 s to
	// start and finish, and the delivery and the ack are eight git
	// processes, so a 20 s budget failed a hook that was only slow. A hook
	// that never answers stops at go test -timeout, with its stack.
	if got := <-acked; len(got) != 1 || got[0] != id+" request 2" {
		t.Fatalf("acknowledged %v", got)
	}
	if _, out := runText(a, "requests", id); !strings.Contains(out, "request 2 by jhf") || !strings.Contains(out, "  receipt: acknowledged by crab\n") {
		t.Fatalf("requests after the ack:\n%s", out)
	}
	// crab's first write on the task: in progress (crab's own session, so
	// HUG_USER is crab, as JCODE_SESSION_ID makes it in jcode).
	a.Env["HUG_USER"] = "crab"
	execute(t, a, "set", id, "--goal", "per board")
	a.Env["HUG_USER"] = "tester"
	if _, out := runText(a, "requests", id); !strings.Contains(out, "  receipt: in progress by crab\n") {
		t.Fatalf("requests after crab's write:\n%s", out)
	}
	execute(t, a, "resolve", id, "2", "goal per board", "--as", "crab")
	if _, out := runText(a, "requests", id, "--all"); !strings.Contains(out, "  receipt: answered\n") {
		t.Fatalf("requests after resolve:\n%s", out)
	}
	f := taskFile(t, a, id)
	for _, want := range []string{" request 1 not delivered: no coordinator set (hug task coordinator NAME)\n", " request 2 delivered to crab (" + crab + ")", " request 2 acknowledged by crab", " request 2 in progress by crab: hug task set " + id + "\n"} {
		if !strings.Contains(f, want) {
			t.Fatalf("History lacks %q:\n%s", want, f)
		}
	}
	// The coordinator's session is gone: pending, not delivered, and why.
	jc.mu.Lock()
	jc.live[crab] = false
	jc.mu.Unlock()
	r = execute(t, a, "request", id, "third", "--as", "jhf")
	if want := "pending, not delivered: crab's jcode session " + old; !strings.HasPrefix(r["receipt"].(string), "pending, not delivered: crab's jcode session ") || !strings.HasSuffix(r["receipt"].(string), " is not live in a jcode daemon") {
		t.Fatalf("dead coordinator: receipt %v (want like %q)", r["receipt"], want)
	}
	// Nothing waits in the inbox for a delivery that did not happen.
	for _, s := range []string{old, crab} {
		if ents, _ := os.ReadDir(filepath.Join(jc.home, "hug-inbox", s)); len(ents) != 0 {
			t.Fatalf("inbox of %s keeps %d items", s, len(ents))
		}
	}
}

// gmzd87 (Jørgen 11:59Z): a person's answer goes the same way, with the
// question, the answer verbatim, who asked and the next command. 5yh7hr
// (Jørgen 2026-11-26 22:01) narrowed "to the coordinator even when a
// worker asked": the agent doing the task hears first, and the
// coordinator only when that agent has no live session, as here
// (nautilus has none), told so.
func TestAnswerReachesCoordinator(t *testing.T) {
	t.Parallel()
	jc := newFakeJcode(t)
	crab := jc.session(t, "crab", 2000, true)
	a := app(t)
	a.JcodeHomes = []string{jc.home}
	execute(t, a, "init")
	execute(t, a, "coordinator", "crab")
	id := execute(t, a, "new", "Tags", "--goal", "g", "--purpose", "p", "--principle", "q", "--accept", "a", "--accept", "b")["task"].(Reply)["id"].(string)
	execute(t, a, "take", id, "--as", "nautilus", "--model", "m")
	execute(t, a, "ask", id, "--to", "jhf", "--check", "2", "type stays, or drop?", "--as", "nautilus")
	r := execute(t, a, "answer", id, "drop it\nthe column says it", "--as", "jhf")
	if r["receipt"] != "pending, delivered to crab ("+crab+"); not delivered live to nautilus, who is doing it: no jcode session named nautilus on this machine" {
		t.Fatalf("receipt: %v", r["receipt"])
	}
	msgs := jc.messages()
	if len(msgs) != 1 {
		t.Fatalf("daemon got %q", msgs)
	}
	_, msg, _ := strings.Cut(msgs[0], "\x00")
	board := realPath(t, a.Cwd)
	for _, want := range []string{
		"hug: jhf answered on " + id + " (Tags), check 2, the question nautilus asked on board " + filepath.Base(board) + " (" + board + "), to you as the board coordinator:\n",
		"> type stays, or drop?\n",
		"  drop it\n  the column says it\n",
		"Route it to nautilus, who asked, or act on it: hug task show " + id + "\n",
	} {
		if !strings.Contains(msg, want) {
			t.Fatalf("message lacks %q:\n%s", want, msg)
		}
	}
	if f := taskFile(t, a, id); !strings.Contains(f, " answer 1 delivered to crab ("+crab+")") {
		t.Fatalf("History:\n%s", f)
	}
}

// realPath is the board's root as git names it, which is what hug puts in
// the message (macOS: /var is /private/var; Windows: forward slashes).
func realPath(t *testing.T, dir string) string {
	t.Helper()
	out, err := exec.Command("git", "-C", dir, "rev-parse", "--show-toplevel").Output()
	if err != nil {
		t.Fatal(err)
	}
	return strings.TrimSpace(string(out))
}

// qsqhpe check 3: the person asked ticking the check the question is about
// answers it, and that answer reaches the coordinator like a typed one; a
// tick by anyone else sends nothing.
func TestTickAnswerReachesCoordinator(t *testing.T) {
	t.Parallel()
	jc := newFakeJcode(t)
	crab := jc.session(t, "crab", 2000, true)
	a := app(t)
	a.JcodeHomes = []string{jc.home}
	execute(t, a, "init")
	execute(t, a, "coordinator", "crab")
	id := execute(t, a, "new", "Tags", "--goal", "g", "--purpose", "p", "--principle", "q", "--accept", "a", "--accept", "b")["task"].(Reply)["id"].(string)
	execute(t, a, "take", id, "--as", "nautilus", "--model", "m")
	execute(t, a, "ask", id, "--to", "jhf", "--check", "2", "type stays, or drop?", "--as", "nautilus")
	// check has no --as: who ticks is HUG_USER (or the agent session).
	// jhf ticking another check: still asked, nothing sent (the asker's
	// own tick is TestTickByOthersLeavesQuestion).
	a.Env["HUG_USER"] = "jhf"
	if r := execute(t, a, "check", id, "1"); r["receipt"] != nil || r["answered"] != nil {
		t.Fatalf("another check answered: %v", r)
	}
	r := execute(t, a, "check", id, "2")
	// nautilus, doing it, has no live session (5yh7hr): crab has it, told so.
	if r["receipt"] != "pending, delivered to crab ("+crab+"); not delivered live to nautilus, who is doing it: no jcode session named nautilus on this machine" {
		t.Fatalf("receipt: %v", r["receipt"])
	}
	if st := r["task"].(Reply)["stage"]; st != "active" {
		t.Fatalf("stage after the tick: %v", st)
	}
	msgs := jc.messages()
	if len(msgs) != 1 {
		t.Fatalf("daemon got %q", msgs)
	}
	_, msg, _ := strings.Cut(msgs[0], "\x00")
	board := realPath(t, a.Cwd)
	for _, want := range []string{
		"hug: jhf answered on " + id + " (Tags), check 2, the question nautilus asked on board " + filepath.Base(board) + " (" + board + "), to you as the board coordinator:\n",
		"> type stays, or drop?\n",
		"  answered by ticking check 2\n",
		"Route it to nautilus, who asked, or act on it: hug task show " + id + "\n",
	} {
		if !strings.Contains(msg, want) {
			t.Fatalf("message lacks %q:\n%s", want, msg)
		}
	}
	if f := taskFile(t, a, id); !strings.Contains(f, " answer 1 delivered to crab ("+crab+")") || !strings.Contains(f, "  answered by ticking check 2\n") {
		t.Fatalf("History:\n%s", f)
	}
}
