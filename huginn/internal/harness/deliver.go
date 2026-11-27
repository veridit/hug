package harness

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Delivery to a jcode session (gmzd87, Jørgen 11:21Z: by design, not by an
// agent's poller). jcode's daemon takes a line of JSON on its socket,
// {"type":"notify_session","id":N,"session_id":S,"message":M}, the request
// jcode's own scheduler uses: an idle live session starts a turn with M at
// once, a busy one gets M as a soft interrupt at its next safe point, and a
// session no daemon has live is refused ("Session 'S' is not currently
// live"). The reply is {"type":"ack"}, then {"type":"done"} or
// {"type":"error","message":...} with the same id.

// Session is one jcode session file: session_<name>_<ms>_<hex>.json.
type Session struct {
	ID    string // session_<name>_<ms>_<hex>
	Home  string // the JCODE_HOME it lives in
	Start time.Time
	Last  time.Time
}

var sessionFile = regexp.MustCompile(`^(session_([a-z0-9]+)_(\d+)_[0-9a-f]+)\.json$`)

// SessionsNamed lists the sessions called name in homes, the most recently
// written first.
func SessionsNamed(homes []string, name string) []Session {
	name = strings.ToLower(strings.TrimSpace(name))
	var out []Session
	for _, h := range homes {
		ents, err := os.ReadDir(filepath.Join(h, "sessions"))
		if err != nil {
			continue
		}
		for _, e := range ents {
			m := sessionFile.FindStringSubmatch(e.Name())
			if m == nil || m[2] != name {
				continue
			}
			ms, _ := strconv.ParseInt(m[3], 10, 64)
			s := Session{ID: m[1], Home: h, Start: time.UnixMilli(ms)}
			if fi, err := e.Info(); err == nil {
				s.Last = fi.ModTime()
			}
			out = append(out, s)
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Last.After(out[j].Last) })
	return out
}

// Sockets are the daemon sockets to try for a session in home, in order:
// the instance's own runtime folder (a per-project wrapper sets
// JCODE_RUNTIME_DIR=<home>/runtime), then the process's JCODE_SOCKET,
// JCODE_RUNTIME_DIR, XDG_RUNTIME_DIR and, on macOS, TMPDIR, as jcode's
// runtime_dir() picks them. Only paths that exist.
func Sockets(home string, getenv func(string) string) []string {
	var out []string
	seen := map[string]bool{}
	add := func(p string) {
		if p == "" {
			return
		}
		p = filepath.Clean(p)
		if seen[p] {
			return
		}
		seen[p] = true
		if fi, err := os.Stat(p); err == nil && fi.Mode()&os.ModeSocket != 0 {
			out = append(out, p)
		}
	}
	add(filepath.Join(home, "runtime", "jcode.sock"))
	add(getenv("JCODE_SOCKET"))
	for _, k := range []string{"JCODE_RUNTIME_DIR", "XDG_RUNTIME_DIR", "TMPDIR"} {
		if d := getenv(k); d != "" {
			add(filepath.Join(d, "jcode.sock"))
		}
	}
	return out
}

// ErrNotLive is a daemon's refusal of a session it does not have live.
var ErrNotLive = errors.New("not currently live")

// Notify hands message to session through the daemon at socket: nil once
// the daemon says done (the session has it: a turn started, or a soft
// interrupt queued), ErrNotLive (wrapped) when that daemon does not have the
// session live, another error when the socket does not answer.
func Notify(socket, session, message string, timeout time.Duration) error {
	c, err := net.DialTimeout("unix", socket, timeout)
	if err != nil {
		return err
	}
	defer c.Close()
	c.SetDeadline(time.Now().Add(timeout))
	const id = 1
	req, _ := json.Marshal(map[string]any{"type": "notify_session", "id": id, "session_id": session, "message": message})
	if _, err := c.Write(append(req, '\n')); err != nil {
		return err
	}
	sc := bufio.NewScanner(c)
	sc.Buffer(make([]byte, 0, 64<<10), 16<<20)
	for sc.Scan() {
		var ev struct {
			Type    string `json:"type"`
			ID      int    `json:"id"`
			Message string `json:"message"`
		}
		if json.Unmarshal(sc.Bytes(), &ev) != nil || ev.ID != id {
			continue
		}
		switch ev.Type {
		case "done":
			return nil
		case "error":
			if strings.Contains(ev.Message, "not currently live") {
				return fmt.Errorf("%w: %s", ErrNotLive, ev.Message)
			}
			return errors.New(ev.Message)
		}
	}
	if err := sc.Err(); err != nil {
		return err
	}
	return errors.New("the jcode daemon closed the socket without an answer")
}

// Deliver hands message to the newest live session called name in homes,
// trying its three newest sessions on each of their sockets. It returns
// the session that took it, or why none did, in words for a receipt line.
// it goes into a session's inbox before the daemon is asked: an idle
// session starts its turn (and hug's hook in it looks) before Notify
// returns; a session that did not take it gets it out again.
func Deliver(homes []string, getenv func(string) string, name, message string, it InboxItem) (Session, string) {
	ss := SessionsNamed(homes, name)
	if len(ss) == 0 {
		return Session{}, "no jcode session named " + name + " on this machine"
	}
	if len(ss) > 3 {
		ss = ss[:3]
	}
	tried := false
	for _, s := range ss {
		ok, sockets, err := DeliverTo(s, getenv, message, it)
		if err != nil {
			return Session{}, "cannot write " + name + "'s hug inbox: " + err.Error()
		}
		if ok {
			return s, ""
		}
		tried = tried || sockets > 0
	}
	if !tried {
		return Session{}, "no jcode daemon socket for " + name + "'s session " + ss[0].ID
	}
	return Session{}, name + "'s jcode session " + ss[0].ID + " is not live in a jcode daemon"
}

// DeliverTo hands message to session s on each of its daemon sockets in
// turn, with it in s's inbox first (an idle session's hook looks before
// Notify returns); true once a daemon took it. sockets is how many it
// tried; err only when the inbox cannot be written.
func DeliverTo(s Session, getenv func(string) string, message string, it InboxItem) (ok bool, sockets int, err error) {
	for _, sock := range Sockets(s.Home, getenv) {
		sockets++
		if err := AddInbox(s.Home, s.ID, it); err != nil {
			return false, sockets, err
		}
		if Notify(sock, s.ID, message, 5*time.Second) == nil {
			return true, sockets, nil
		}
		RemoveInbox(s.Home, s.ID, it)
	}
	return false, sockets, nil
}

// Pending: what could not be delivered to coordinator NAME (no live
// session) waits for NAME's next session (bbxdd7). One file per item,
// <JCODE_HOME>/hug-inbox/pending/<name>/<task>-<kind>-<n>.json, in every
// jcode home hug knows; the hook in a session called NAME reads that one
// folder at its turn's start (one read of a missing folder when nothing
// waits) and delivers it to itself. The task's History decides whether it
// still needs delivering, so a copy in another home never delivers twice.

// PendingDir is the folder of items waiting for name in jcode home.
func PendingDir(home, name string) string {
	return filepath.Join(home, "hug-inbox", "pending", safeName(strings.ToLower(name)))
}

// AddPending writes it into name's pending folder in home.
func AddPending(home, name string, it InboxItem) error {
	name = strings.ToLower(strings.TrimSpace(name))
	if name == "" || strings.ContainsAny(name, `/\`) {
		return fmt.Errorf("bad coordinator name %q", name)
	}
	dir := PendingDir(home, name)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	b, _ := json.Marshal(it)
	n := inboxName(it)
	tmp := filepath.Join(dir, "."+n+".tmp")
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, filepath.Join(dir, n))
}

// Pending lists the items waiting for name in home with their files.
func Pending(home, name string) (items []InboxItem, files []string) {
	dir := PendingDir(home, name)
	ents, err := os.ReadDir(dir)
	if err != nil {
		return nil, nil
	}
	for _, e := range ents {
		if strings.HasPrefix(e.Name(), ".") || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		b, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			continue
		}
		var it InboxItem
		if json.Unmarshal(b, &it) != nil || it.Task == "" || it.Board == "" {
			continue
		}
		items = append(items, it)
		files = append(files, filepath.Join(dir, e.Name()))
	}
	return items, files
}

// SessionName is the name in a jcode session id (session_<name>_<ms>_<hex>),
// "" if it is not one.
func SessionName(sid string) string {
	rest, ok := strings.CutPrefix(sid, "session_")
	if !ok {
		return ""
	}
	name, _, ok := strings.Cut(rest, "_")
	if !ok {
		return ""
	}
	return name
}

// Inbox: what hug delivered to a session and waits to see it receive. One
// file per item, <JCODE_HOME>/hug-inbox/<session>/<task>-<kind>-<n>.json;
// hug's hook in that session reads the folder (one read of a missing
// folder when nothing waits) and acknowledges an item once its message is
// in the session's transcript.

// InboxItem is one delivered item waiting for its receipt.
type InboxItem struct {
	Board string `json:"board"` // the board's repository (main worktree)
	Task  string `json:"task"`
	Kind  string `json:"kind"` // request, answer, outcome or review
	N     int    `json:"n"`
	To    string `json:"to"`    // who it is for: the agent doing the task, the coordinator or the reviewer
	Token string `json:"token"` // in the message, nowhere else
	// Missed: the coordinator's copy of an item the agent doing the task
	// could not take live (5yh7hr), the line saying so.
	Missed string `json:"missed,omitempty"`
}

// InboxDir is the inbox folder of session in jcode home.
func InboxDir(home, session string) string { return filepath.Join(home, "hug-inbox", session) }

// AddInbox writes it into session's inbox.
func AddInbox(home, session string, it InboxItem) error {
	if session == "" || strings.ContainsAny(session, `/\`) {
		return fmt.Errorf("bad jcode session id %q", session)
	}
	dir := InboxDir(home, session)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	b, _ := json.Marshal(it)
	name := inboxName(it)
	tmp := filepath.Join(dir, "."+name+".tmp")
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, filepath.Join(dir, name))
}

// RemoveInbox takes it out of session's inbox.
func RemoveInbox(home, session string, it InboxItem) {
	os.Remove(filepath.Join(InboxDir(home, session), inboxName(it)))
}

func inboxName(it InboxItem) string {
	return fmt.Sprintf("%s-%s-%d.json", safeName(it.Task), it.Kind, it.N)
}

func safeName(s string) string {
	return strings.Map(func(r rune) rune {
		if r == '/' || r == '\\' || r == '.' {
			return '_'
		}
		return r
	}, s)
}
