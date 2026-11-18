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
