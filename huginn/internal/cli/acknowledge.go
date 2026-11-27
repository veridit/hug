package cli

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/veridit/hug/huginn/internal/harness"
	"github.com/veridit/hug/huginn/internal/task"
)

// Acknowledge is the coordinator's half of delivery (gmzd87): hug's hook,
// running in jcode session sid of jcode home home, at its turn's start and
// after each tool call. For each item hug delivered to this session
// (home/hug-inbox/sid/), once the session's own transcript holds the
// message's token, it writes "ITEM acknowledged by NAME" on the task and
// takes the item out. Received means in the session's context: written
// to it is not enough. With nothing delivered, it costs one ReadDir of a
// missing folder.
func Acknowledge(ctx context.Context, home, sid string) (acked []string, err error) {
	dir := harness.InboxDir(home, sid)
	ents, err := os.ReadDir(dir)
	if err != nil || len(ents) == 0 {
		return nil, nil
	}
	var items []harness.InboxItem
	var paths []string
	var since time.Time
	for _, e := range ents {
		if strings.HasPrefix(e.Name(), ".") || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		b, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			continue
		}
		var it harness.InboxItem
		if json.Unmarshal(b, &it) != nil || it.Token == "" {
			continue
		}
		items = append(items, it)
		paths = append(paths, filepath.Join(dir, e.Name()))
		if fi, err := e.Info(); err == nil && (since.IsZero() || fi.ModTime().Before(since)) {
			since = fi.ModTime()
		}
	}
	if len(items) == 0 {
		return nil, nil
	}
	seen := transcriptHas(filepath.Join(home, "sessions", sid), items, since)
	for i, it := range items {
		if !seen[it.Token] {
			continue
		}
		a := &App{Cwd: it.Board, In: strings.NewReader(""), Out: io.Discard, Err: io.Discard}
		s, err := a.open(ctx)
		if err != nil {
			return acked, err
		}
		release := s.turn()
		s.s.Invalidate()
		_, err = s.s.Acknowledge(it.Task, task.Item{Kind: it.Kind, N: it.N}, it.To)
		if err == nil {
			err = s.commit("hug task " + it.Kind + " " + strconv.Itoa(it.N) + " on " + it.Task + " acknowledged by " + it.To)
		}
		release()
		s.c.Close()
		if err != nil {
			return acked, err
		}
		os.Remove(paths[i])
		acked = append(acked, it.Task+" "+it.Kind+" "+strconv.Itoa(it.N))
	}
	return acked, nil
}

// transcriptHas reports which items' tokens the session's transcript holds
// in a message to it: the journal's appended messages, then the snapshot
// (the compacted past) if any token is still unseen and the snapshot was
// written after since (the oldest delivery).
func transcriptHas(base string, items []harness.InboxItem, since time.Time) map[string]bool {
	seen := map[string]bool{}
	want := func() bool { return len(seen) < len(items) }
	look := func(line []byte) {
		hit := false
		for _, it := range items {
			if !seen[it.Token] && bytes.Contains(line, []byte(it.Token)) {
				hit = true
			}
		}
		if !hit {
			return
		}
		// Only a message to the session counts: a user message's text
		// (a turn or a soft interrupt), never a tool's result or the
		// session's own words that happen to quote the token.
		var entry struct {
			Append []struct {
				Role    string `json:"role"`
				Content []struct {
					Type string `json:"type"`
					Text string `json:"text"`
				} `json:"content"`
			} `json:"append_messages"`
			Messages []struct {
				Role    string `json:"role"`
				Content []struct {
					Type string `json:"type"`
					Text string `json:"text"`
				} `json:"content"`
			} `json:"messages"`
		}
		if json.Unmarshal(line, &entry) != nil {
			return
		}
		for _, m := range append(entry.Append, entry.Messages...) {
			if m.Role != "user" {
				continue
			}
			for _, c := range m.Content {
				if c.Type != "text" {
					continue
				}
				for _, it := range items {
					if strings.Contains(c.Text, it.Token) {
						seen[it.Token] = true
					}
				}
			}
		}
	}
	scan := func(path string, tail int64) {
		f, err := os.Open(path)
		if err != nil {
			return
		}
		defer f.Close()
		if fi, err := f.Stat(); err == nil && tail > 0 && fi.Size() > tail {
			f.Seek(fi.Size()-tail, io.SeekStart)
		}
		sc := bufio.NewScanner(f)
		sc.Buffer(make([]byte, 0, 1<<20), 256<<20)
		for sc.Scan() && want() {
			look(sc.Bytes())
		}
	}
	scan(base+".journal.jsonl", 8<<20)
	// The snapshot (one large JSON) holds a delivered message only if a
	// compaction rewrote it after the delivery: read it only then.
	if fi, err := os.Stat(base + ".json"); want() && err == nil && fi.ModTime().After(since) {
		scan(base+".json", 0)
	}
	return seen
}

// DeliverPending is a session's name picking up, at its next session,
// what could not reach it live (bbxdd7; 5yh7hr: the agent doing a task
// too, not only the coordinator): hug's hook in jcode session sid (JCODE_HOME home)
// at its turn's start. For each item waiting for the session's name
// (home/hug-inbox/pending/NAME/), still pending and not delivered on its
// task, it hands the item to this session through the daemon, as
// Deliver does, and records "delivered to NAME (sid)" on the task: the
// same inbox, so Acknowledge then marks it received from the transcript.
// It runs under the board's turn and re-reads History there, so one item
// is delivered once however many sessions or homes hold a copy. With
// nothing waiting, it costs one ReadDir of a missing folder. getenv is the
// hook's environment, for the daemon's socket.
func DeliverPending(ctx context.Context, home, sid string, getenv func(string) string) (delivered []string, err error) {
	name := harness.SessionName(sid)
	if name == "" {
		return nil, nil
	}
	items, files := harness.Pending(home, name)
	for i, it := range items {
		done, err := deliverPendingItem(ctx, home, sid, getenv, it)
		if err != nil {
			return delivered, err
		}
		if done != "" {
			delivered = append(delivered, done)
		}
		// Delivered now, or already by another session: nothing is owed.
		// Not delivered (the daemon refused): it stays for the next start.
		if done != "" || !stillOwed(ctx, it) {
			os.Remove(files[i])
		}
	}
	return delivered, nil
}

// deliverPendingItem delivers one pending item to sid, under the board's
// turn; "ID kind N" when it did, "" when nothing was owed or the daemon
// did not take it.
func deliverPendingItem(ctx context.Context, home, sid string, getenv func(string) string, it harness.InboxItem) (string, error) {
	a := &App{Cwd: it.Board, In: strings.NewReader(""), Out: io.Discard, Err: io.Discard}
	s, err := a.open(ctx)
	if err != nil {
		return "", err
	}
	defer s.c.Close()
	release := s.turn()
	defer release()
	s.s.Invalidate()
	d, err := s.s.Get(it.Task)
	if err != nil {
		return "", nil // the task is gone: nothing to deliver
	}
	item := task.Item{Kind: it.Kind, N: it.N}
	if !d.OwedTo(item, it.To) {
		return "", nil
	}
	by, text := "", ""
	var answer *task.Answer
	var targets []targetRef
	switch it.Kind {
	case "request":
		for _, q := range d.Task.Requests {
			if q.N == it.N {
				by, text = q.By, q.Text
			}
		}
		// The same message as a delivery at once (ec6z6j): a request for
		// several tasks names them.
		if ts, e := s.s.Tasks(); e == nil {
			targets = requestTargets(ts, d.Task, it.N)
		}
	case "answer":
		if answer = d.AnswerN(it.N); answer != nil {
			by, text = answer.By, answer.Text
		}
	}
	it.Token = newToken(d.Task.ID, item)
	ss := harness.Session{ID: sid, Home: home}
	msg := itemMessage(it.Board, d.Task, item, by, text, answer, it.Token, targets, it.To, it.Missed)
	took, _, err := harness.DeliverTo(ss, getenv, msg, it)
	if err != nil || !took {
		return "", err
	}
	if _, err = s.s.Delivered(d.Task.ID, item, it.To, sid, ""); err != nil {
		return "", err
	}
	if err = s.commit("hug task " + item.Name() + " on " + d.Task.ID + " delivered to " + it.To + " at its session's start"); err != nil {
		return "", err
	}
	return d.Task.ID + " " + item.Name(), nil
}

// stillOwed reports whether it is still owed to it.To on its task
// (Document.OwedTo; a read outside the turn: only to drop a stale
// pending file).
func stillOwed(ctx context.Context, it harness.InboxItem) bool {
	a := &App{Cwd: it.Board, In: strings.NewReader(""), Out: io.Discard, Err: io.Discard}
	s, err := a.open(ctx)
	if err != nil {
		return true
	}
	defer s.c.Close()
	d, err := s.s.Get(it.Task)
	if err != nil {
		return false
	}
	return d.OwedTo(task.Item{Kind: it.Kind, N: it.N}, it.To)
}
