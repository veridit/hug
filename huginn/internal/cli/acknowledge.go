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
	}
	if len(items) == 0 {
		return nil, nil
	}
	seen := transcriptHas(filepath.Join(home, "sessions", sid), items)
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
// (the compacted past) if any token is still unseen.
func transcriptHas(base string, items []harness.InboxItem) map[string]bool {
	seen := map[string]bool{}
	want := func() bool { return len(seen) < len(items) }
	look := func(line []byte) {
		for _, it := range items {
			if !seen[it.Token] && bytes.Contains(line, []byte(it.Token)) && bytes.Contains(line, []byte(`"role":"user"`)) {
				seen[it.Token] = true
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
	if want() {
		scan(base+".json", 0)
	}
	return seen
}
