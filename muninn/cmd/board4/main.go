// Command board4 replays real agent sessions through the board nudge
// (qawmnb, muninn/internal/recall/boardcue.go) with the board as it was at
// each turn, rebuilt from the hug/tasks history. It writes one JSON line per
// fired nudge and a summary per session. Scratch only: its state database
// must be under $JCODE_SCRATCH_DIR/board4.
//
//	go run ./muninn/cmd/board4 -db $JCODE_SCRATCH_DIR/board4/run.db \
//	  -board ~/Projects/hug/.hug/tasks -sessions ~/.jcode/sessions \
//	  -ids session_crab_...,session_vole_... > fired.jsonl
//
// Each session's user turns are replayed in order through the same engine
// the hook runs (BoardCue with the turn's time); the session's tool calls
// between turns are its actions; the board is the newest hug/tasks commit at
// or before the turn. Rate state is fresh per run.
package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/veridit/hug/huginn/board"
	"github.com/veridit/hug/muninn/internal/harness"
	"github.com/veridit/hug/muninn/internal/recall"
	"github.com/veridit/hug/muninn/internal/store"
	"github.com/veridit/hug/muninn/run"
)

type commit struct {
	hash string
	at   time.Time
}

// history replays the board branch forward in time.
type history struct {
	dir     string
	commits []commit
	next    int
	files   *board.Files
	cat     *exec.Cmd
	catIn   io.WriteCloser
	catOut  *bufio.Reader
}

func newHistory(dir string) (*history, error) {
	out, err := exec.Command("git", "-C", dir, "log", "--reverse", "--format=%H %ct", "hug/tasks").Output()
	if err != nil {
		return nil, err
	}
	h := &history{dir: dir, files: board.NewFiles()}
	for _, l := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		f := strings.Fields(l)
		if len(f) != 2 {
			continue
		}
		s, _ := strconv.ParseInt(f[1], 10, 64)
		h.commits = append(h.commits, commit{f[0], time.Unix(s, 0)})
	}
	return h, nil
}

// at advances the board to the newest commit at or before t (time only
// moves forward: sessions are replayed interleaved in time order).
func (h *history) at(t time.Time) (*board.Snapshot, bool) {
	prev := ""
	if h.next > 0 {
		prev = h.commits[h.next-1].hash
	}
	moved := false
	for h.next < len(h.commits) && !h.commits[h.next].at.After(t) {
		h.next++
		moved = true
	}
	if h.next == 0 {
		return nil, false
	}
	if moved {
		cur := h.commits[h.next-1].hash
		h.apply(prev, cur)
	}
	return h.files.Snapshot(), true
}

func (h *history) apply(from, to string) {
	var out []byte
	if from == "" {
		out, _ = exec.Command("git", "-C", h.dir, "ls-tree", "-r", "--name-only", "-z", to).Output()
		for _, p := range strings.Split(strings.TrimRight(string(out), "\x00"), "\x00") {
			h.load(to, p)
		}
		return
	}
	out, _ = exec.Command("git", "-C", h.dir, "diff", "--name-status", "--no-renames", "-z", from, to).Output()
	parts := strings.Split(strings.TrimRight(string(out), "\x00"), "\x00")
	for i := 0; i+1 < len(parts); i += 2 {
		if parts[i] == "D" {
			h.files.Delete(parts[i+1])
		} else {
			h.load(to, parts[i+1])
		}
	}
}

func (h *history) load(rev, path string) {
	if !strings.HasSuffix(path, ".md") {
		return
	}
	if h.cat == nil {
		c := exec.Command("git", "-C", h.dir, "cat-file", "--batch")
		in, _ := c.StdinPipe()
		out, _ := c.StdoutPipe()
		if c.Start() != nil {
			return
		}
		h.cat, h.catIn, h.catOut = c, in, bufio.NewReaderSize(out, 1<<20)
	}
	fmt.Fprintf(h.catIn, "%s:%s\n", rev, path)
	hdr, err := h.catOut.ReadString('\n')
	if err != nil {
		return
	}
	f := strings.Fields(hdr)
	if len(f) != 3 {
		return // "<obj> missing"
	}
	n, _ := strconv.Atoi(f[2])
	b := make([]byte, n+1)
	if _, err := io.ReadFull(h.catOut, b); err != nil {
		return
	}
	h.files.Set(path, b[:n])
}

// turn is one user message and the tool calls the session made before it.
type turn struct {
	at      time.Time
	msg     string
	actions []harness.Action
}

type message struct {
	ID        string            `json:"id"`
	Role      string            `json:"role"`
	Display   string            `json:"display_role"`
	Timestamp string            `json:"timestamp"`
	Content   []json.RawMessage `json:"content"`
}

func loadSession(dir, id string) ([]message, string, string, error) {
	var snap struct {
		Messages   []message `json:"messages"`
		WorkingDir string    `json:"working_dir"`
		Model      string    `json:"model"`
	}
	b, err := os.ReadFile(filepath.Join(dir, id+".json"))
	if err != nil {
		return nil, "", "", err
	}
	if err := json.Unmarshal(b, &snap); err != nil {
		return nil, "", "", err
	}
	msgs := snap.Messages
	seen := map[string]bool{}
	for _, m := range msgs {
		seen[m.ID] = true
	}
	if f, err := os.Open(filepath.Join(dir, id+".journal.jsonl")); err == nil {
		sc := bufio.NewScanner(f)
		sc.Buffer(make([]byte, 0, 1<<20), 64<<20)
		for sc.Scan() {
			var e struct {
				Append []message `json:"append_messages"`
			}
			if json.Unmarshal(sc.Bytes(), &e) == nil {
				for _, m := range e.Append {
					if !seen[m.ID] {
						seen[m.ID] = true
						msgs = append(msgs, m)
					}
				}
			}
		}
		f.Close()
	}
	return msgs, snap.WorkingDir, snap.Model, nil
}

func text(m message) string {
	var parts []string
	for _, c := range m.Content {
		var b struct {
			Type, Text string
		}
		if json.Unmarshal(c, &b) == nil && (b.Type == "text" || b.Type == "") && strings.TrimSpace(b.Text) != "" {
			parts = append(parts, b.Text)
		}
	}
	return strings.TrimSpace(strings.Join(parts, " "))
}

func turns(msgs []message) []turn {
	var out []turn
	var pending []harness.Action
	for _, m := range msgs {
		at, _ := time.Parse(time.RFC3339Nano, m.Timestamp)
		switch m.Role {
		case "assistant":
			for _, c := range m.Content {
				var b struct {
					Type  string          `json:"type"`
					Name  string          `json:"name"`
					Input json.RawMessage `json:"input"`
				}
				if json.Unmarshal(c, &b) == nil && b.Type == "tool_use" {
					pending = append(pending, harness.ParseToolCall(at, b.Name, b.Input))
				}
			}
		case "user":
			t := text(m)
			if t == "" || m.Display == "system" || recall.IsOwnOutput(t) || strings.HasPrefix(t, "[Attached image") {
				continue
			}
			out = append(out, turn{at: at, msg: t, actions: pending})
			pending = nil
		}
	}
	return out
}

type fired struct {
	Session string `json:"session"`
	Turn    int    `json:"turn"`
	At      string `json:"at"`
	Kind    string `json:"kind"`
	Line    string `json:"line"`
	Msg     string `json:"msg"`
	// Did is what the session did since the previous turn (commit subjects,
	// edited files, hug commands), for the blind judge.
	Did []string `json:"did"`
	// Board is the named task's state on the board at the turn.
	Board string `json:"board"`
	// Ran is the first hug command the session ran within 30 minutes that
	// the line asked for (take/done ID), or "".
	Ran string `json:"ran"`
	// Closed: for a done line, whether the board shows the task done (or
	// dropped) within 24 hours; the line was true when it is.
	Closed bool `json:"closed,omitempty"`
	// KeptEditing: for take/work lines, whether the session changed code
	// again in the next 30 minutes (the work the line was about went on).
	KeptEditing bool `json:"kept_editing,omitempty"`
}

type summary struct {
	Session    string  `json:"session"`
	Turns      int     `json:"turns"`
	BoardTurns int     `json:"board_turns"`
	Hours      float64 `json:"hours"`
	Fired      int     `json:"fired"`
	ByKind     map[string]int
	P50ms      float64 `json:"p50_ms"`
	P95ms      float64 `json:"p95_ms"`
}

func main() {
	dbpath := flag.String("db", "", "scratch state database")
	boardDir := flag.String("board", "", "the board checkout (.hug/tasks) whose history to replay")
	sessions := flag.String("sessions", "", "jcode sessions directory")
	ids := flag.String("ids", "", "comma-separated session ids")
	from := flag.String("from", "", "only turns at or after (RFC3339)")
	until := flag.String("until", "", "only turns before (RFC3339)")
	worktree := flag.String("worktree", "", "the session's worktree (default: snapshot working_dir)")
	summaryOut := flag.String("summary", "", "write per-session summaries (JSON lines) here")
	onlyCwd := flag.String("cwd", "", "skip sessions whose working_dir is not this folder (or inside it)")
	flag.Parse()
	root := filepath.Join(os.Getenv("JCODE_SCRATCH_DIR"), "board4")
	abs, _ := filepath.Abs(*dbpath)
	if os.Getenv("JCODE_SCRATCH_DIR") == "" || !strings.HasPrefix(abs, root+string(filepath.Separator)) {
		fmt.Fprintln(os.Stderr, "database must be under $JCODE_SCRATCH_DIR/board4")
		fmt.Fprintln(os.Stderr, "fix: -db $JCODE_SCRATCH_DIR/board4/run.db")
		os.Exit(2)
	}
	os.Remove(abs)
	db, err := store.Open(abs)
	if err != nil {
		panic(err)
	}
	defer db.Close()
	eng := &recall.Engine{DB: db}
	h, err := newHistory(*boardDir)
	if err != nil {
		panic(err)
	}
	var lo, hi time.Time
	if *from != "" {
		lo, _ = time.Parse(time.RFC3339, *from)
	}
	if *until != "" {
		hi, _ = time.Parse(time.RFC3339, *until)
	}
	enc := json.NewEncoder(os.Stdout)
	var sums []summary
	// Sessions are replayed interleaved in time, as they ran: the board and
	// the cross-session lint rotation see one clock.
	type sess struct {
		id, wd, model, name string
		ts                  []turn
		sum                 summary
		first, last         time.Time
		lat                 []float64
		out                 []fired
	}
	type event struct {
		s *sess
		i int
	}
	var all []*sess
	var events []event
	for _, id := range strings.Split(*ids, ",") {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		msgs, wd, model, err := loadSession(*sessions, id)
		if err != nil {
			fmt.Fprintln(os.Stderr, id, err)
			continue
		}
		if *onlyCwd != "" && wd != *onlyCwd && !strings.HasPrefix(wd, *onlyCwd+string(filepath.Separator)) {
			continue
		}
		if *worktree != "" {
			wd = *worktree
		}
		name := ""
		if n, _, ok := strings.Cut(strings.TrimPrefix(id, "session_"), "_"); ok {
			name = n
		}
		x := &sess{id: id, wd: wd, model: model, name: name, ts: turns(msgs), sum: summary{Session: id, ByKind: map[string]int{}}}
		all = append(all, x)
		for i, t := range x.ts {
			if !lo.IsZero() && t.at.Before(lo) || !hi.IsZero() && !t.at.Before(hi) {
				continue
			}
			x.sum.Turns++
			events = append(events, event{x, i})
		}
	}
	sort.SliceStable(events, func(a, b int) bool { return events[a].s.ts[events[a].i].at.Before(events[b].s.ts[events[b].i].at) })
	for _, ev := range events {
		x, i := ev.s, ev.i
		t := x.ts[i]
		id, wd, name, model := x.id, x.wd, x.name, x.model
		snap, ok := h.at(t.at)
		if !ok {
			continue
		}
		x.sum.BoardTurns++
		if x.first.IsZero() {
			x.first = t.at
		}
		x.last = t.at
		actions := t.actions
		src := func(since time.Time, wantLint bool) *recall.BoardInput {
			in := &recall.BoardInput{Self: recall.BoardSelf{Name: name, Model: model, Branch: "main", Worktree: wd}}
			for _, y := range snap.Tasks {
				in.Tasks = append(in.Tasks, run.BoardTask(y))
			}
			for _, a := range actions {
				b := recall.BoardAction{Edits: a.EditsIn(wd), Subject: a.Subject}
				for _, c := range a.Hug {
					b.Hug = append(b.Hug, recall.BoardHug{Verb: c.Verb, ID: c.ID, N: c.N})
				}
				in.Actions = append(in.Actions, b)
			}
			if wantLint {
				fs, _ := snap.Findings()
				var bf []recall.BoardFinding
				for _, f := range fs {
					bf = append(bf, recall.BoardFinding{TaskID: f.TaskID, TaskStatus: f.TaskStatus, Problem: f.Problem, Fix: f.Fix})
				}
				in.Findings = func() []recall.BoardFinding { return bf }
			}
			return in
		}
		t0 := time.Now()
		brk := ""
		msg := t.msg
		if recall.IsHarnessNotice(msg) {
			msg = ""
		} else {
			brk = eng.NoteTurn("jcode:"+id, t.msg, t.at)
		}
		line, kind := eng.BoardCue("jcode:"+id, msg, brk, t.at, src)
		x.lat = append(x.lat, float64(time.Since(t0).Microseconds())/1000)
		if line != "" {
			x.sum.Fired++
			x.sum.ByKind[kind]++
			fr := fired{Session: id, Turn: i, At: t.at.Format(time.RFC3339), Kind: kind, Line: line, Msg: recall.Truncate(t.msg, 600), Ran: ranAfter(x.ts, i, line), Did: did(t.actions, wd)}
			fr.Board = boardState(snap, lineID(line), name)
			if kind == "done" {
				fr.Closed = h.closedBy(lineID(line), t.at.Add(24*time.Hour))
			}
			if kind == "take" || kind == "work" {
				fr.KeptEditing = keptEditing(x.ts, i, wd)
			}
			x.out = append(x.out, fr)
		}
	}
	for _, x := range all {
		if !x.first.IsZero() {
			x.sum.Hours = x.last.Sub(x.first).Hours()
		}
		sort.Float64s(x.lat)
		if n := len(x.lat); n > 0 {
			x.sum.P50ms, x.sum.P95ms = x.lat[n/2], x.lat[(n*95)/100]
		}
		for _, f := range x.out {
			enc.Encode(f)
		}
		sums = append(sums, x.sum)
	}
	if *summaryOut != "" {
		var buf bytes.Buffer
		e := json.NewEncoder(&buf)
		for _, s := range sums {
			e.Encode(s)
		}
		os.WriteFile(*summaryOut, buf.Bytes(), 0o600)
	}
}

// did lists what the actions changed, for the judge: commit subjects,
// edited files (relative to wd) and hug task commands, newest last, at most
// 25 entries.
func did(as []harness.Action, wd string) []string {
	var out []string
	for _, a := range as {
		if a.Commit {
			out = append(out, "commit: "+a.Subject)
		}
		for _, p := range a.Paths {
			if r, err := filepath.Rel(wd, p); err == nil && !strings.HasPrefix(r, "..") {
				p = r
			}
			out = append(out, "edit: "+p)
		}
		for _, c := range a.Hug {
			s := "hug task " + c.Verb
			if c.ID != "" {
				s += " " + c.ID
			}
			if c.N > 0 {
				s += " " + strconv.Itoa(c.N)
			}
			out = append(out, s)
		}
	}
	if len(out) > 25 {
		out = append([]string{fmt.Sprintf("(%d earlier actions left out)", len(out)-25)}, out[len(out)-25:]...)
	}
	return out
}

// boardState describes the board for the judge: the named task, and which
// tasks are doing and for whom.
func boardState(s *board.Snapshot, id, self string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "this session is %q. ", self)
	var doing []string
	for _, t := range s.Tasks {
		if t.ID == id {
			fmt.Fprintf(&b, "task %s %q is %s, assignee %v, %d of %d checks ticked. ", t.ID, t.Title, t.Status, t.Assignee, t.Checks-t.Open, t.Checks)
		}
		if t.Status == "doing" {
			doing = append(doing, fmt.Sprintf("%s (%v)", t.ID, t.Assignee))
		}
	}
	fmt.Fprintf(&b, "doing on the board: %s", strings.Join(doing, ", "))
	if len(doing) == 0 {
		b.WriteString("none")
	}
	return b.String()
}

// lineID is the task id a line's command names (hug task done ID).
func lineID(line string) string {
	if j := strings.LastIndex(line, "hug task "); j >= 0 {
		if f := strings.Fields(line[j:]); len(f) >= 4 {
			return f[3]
		}
	}
	return ""
}

// closedBy reports whether the board's history shows task id done or
// dropped in a commit at or before t.
func (h *history) closedBy(id string, t time.Time) bool {
	if id == "" {
		return false
	}
	out, _ := exec.Command("git", "-C", h.dir, "log", "--format=%ct", "--until="+t.Format(time.RFC3339), "-G", `^status: "?(done|dropped)`, "hug/tasks", "--", "*"+id+"*").Output()
	return strings.TrimSpace(string(out)) != ""
}

func keptEditing(ts []turn, i int, wd string) bool {
	end := ts[i].at.Add(30 * time.Minute)
	for k := i + 1; k < len(ts) && !ts[k].at.After(end); k++ {
		for _, a := range ts[k].actions {
			if a.EditsIn(wd) {
				return true
			}
		}
	}
	return false
}

// ranAfter reports the hug command the line asked for when the session ran
// it in the 30 minutes after turn i.
func ranAfter(ts []turn, i int, line string) string {
	want := ""
	if j := strings.LastIndex(line, "hug task "); j >= 0 {
		f := strings.Fields(line[j:])
		if len(f) >= 4 && (f[2] == "take" || f[2] == "done") {
			want = f[2] + " " + f[3]
		}
	}
	if want == "" {
		return ""
	}
	end := ts[i].at.Add(30 * time.Minute)
	for k := i + 1; k < len(ts) && !ts[k].at.After(end); k++ {
		for _, a := range ts[k].actions {
			for _, c := range a.Hug {
				if c.Verb+" "+c.ID == want {
					return want
				}
			}
		}
	}
	return ""
}
