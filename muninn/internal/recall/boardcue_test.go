package recall_test

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/veridit/hug/muninn/internal/recall"
	"github.com/veridit/hug/muninn/internal/store"
)

func boardEngine(t *testing.T) *recall.Engine {
	t.Helper()
	db, err := store.Open(filepath.Join(t.TempDir(), "m.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	return &recall.Engine{DB: db}
}

func src(in recall.BoardInput) recall.BoardSource {
	return func(time.Time, bool) *recall.BoardInput { c := in; return &c }
}

var edit = recall.BoardAction{Edits: true}

func TestBoardTakeNamedTask(t *testing.T) {
	e := boardEngine(t)
	now := time.Date(2026, 11, 16, 1, 0, 0, 0, time.UTC)
	in := recall.BoardInput{
		Tasks:   []recall.BoardTask{{ID: "7js5hw", Title: "TUI: human names for places", State: "todo", Checks: 2, Open: 2}},
		Self:    recall.BoardSelf{Name: "crab", Model: "claude-opus-5-5"},
		Actions: []recall.BoardAction{{Hug: []recall.BoardHug{{Verb: "show", ID: "7js5hw"}}}, edit},
	}
	line, kind := e.BoardCue("jcode:s1", "continue", "", now, src(in))
	if kind != "take" || !strings.Contains(line, "hug task take 7js5hw --model claude-opus-5-5") {
		t.Fatalf("kind=%q line=%q", kind, line)
	}
	// Once per hour per kind, and the same line about the same task once a day.
	if line, _ := e.BoardCue("jcode:s1", "continue", "", now.Add(30*time.Minute), src(in)); line != "" {
		t.Fatalf("second within the hour: %q", line)
	}
	if line, _ := e.BoardCue("jcode:s1", "continue", "", now.Add(61*time.Minute), src(in)); line != "" {
		t.Fatalf("same line within the day: %q", line)
	}
	other := in
	other.Tasks = append([]recall.BoardTask{{ID: "48hx44", Title: "hug setup wires desktop apps", State: "todo"}}, in.Tasks...)
	other.Actions = []recall.BoardAction{{Hug: []recall.BoardHug{{Verb: "show", ID: "48hx44"}}}, edit}
	if line, _ := e.BoardCue("jcode:s1", "continue", "", now.Add(62*time.Minute), src(other)); !strings.Contains(line, "48hx44") {
		t.Fatalf("another task after the hour: %q", line)
	}
	if line, _ := e.BoardCue("jcode:s1", "continue", "", now.Add(25*time.Hour), src(in)); line == "" {
		t.Fatal("no line after a day")
	}
}

func TestBoardSilentWhenTaskIsMine(t *testing.T) {
	e := boardEngine(t)
	now := time.Now()
	for _, self := range []recall.BoardSelf{
		{Name: "guppy"}, // assignee
		{Linked: true, Worktree: "/r/wt-a", Branch: "feat/a"}, // worktree (x1t1t1)
		{Worktree: "/r", Branch: "feat/a"},                    // non-main branch
	} {
		in := recall.BoardInput{
			Tasks:   []recall.BoardTask{{ID: "qawmnb", Title: "Recall nudges", State: "doing", Assignee: []string{"guppy"}, Branch: "feat/a", Worktree: "/r/wt-a", Checks: 3, Open: 2}},
			Self:    self,
			Actions: []recall.BoardAction{edit, edit},
		}
		for i := 0; i < 3; i++ {
			if line, _ := e.BoardCue("s-"+self.Name+self.Branch+self.Worktree, "go on", "", now.Add(time.Duration(i)*2*time.Hour), src(in)); line != "" {
				t.Fatalf("%+v: %q", self, line)
			}
		}
	}
}

func TestBoardMainWorktreeIsNotOwnership(t *testing.T) {
	// Every worker shares the main checkout on main: a doing task taken there
	// is not this session's (crab: never hide a real mismatch).
	e := boardEngine(t)
	in := recall.BoardInput{
		Tasks:   []recall.BoardTask{{ID: "x1t1t1", Title: "Worktrees", State: "doing", Assignee: []string{"dragon"}, Branch: "main", Worktree: "/r"}},
		Self:    recall.BoardSelf{Name: "guppy", Branch: "main", Worktree: "/r"},
		Actions: []recall.BoardAction{edit},
	}
	now := time.Now()
	if line, _ := e.BoardCue("s", "go", "", now, src(in)); line != "" {
		t.Fatalf("first edit turn without an id must stay silent: %q", line)
	}
	line, kind := e.BoardCue("s", "go", "", now.Add(time.Minute), src(in))
	if kind != "take" || !strings.Contains(line, "hug task next") {
		t.Fatalf("second edit turn: kind=%q %q", kind, line)
	}
}

func TestBoardOwnTakeCounts(t *testing.T) {
	// A session that ran hug task take ID owns ID even when the board names
	// someone else (pre-et85h1 assignees were the git user).
	e := boardEngine(t)
	now := time.Now()
	in := recall.BoardInput{
		Tasks:   []recall.BoardTask{{ID: "5g86n6", Title: "Layout", State: "doing", Assignee: []string{"Jørgen H. Fjeld"}, Checks: 6, Open: 3}},
		Actions: []recall.BoardAction{{Hug: []recall.BoardHug{{Verb: "take", ID: "5g86n6"}}}, edit},
	}
	if line, _ := e.BoardCue("s", "go", "", now, src(in)); line != "" {
		t.Fatalf("own take: %q", line)
	}
	in.Actions = []recall.BoardAction{edit}
	if line, _ := e.BoardCue("s", "go", "", now.Add(time.Minute), src(in)); line != "" {
		t.Fatalf("own take, later turn: %q", line)
	}
}

func TestBoardDoneWhenEveryCheckHolds(t *testing.T) {
	e := boardEngine(t)
	in := recall.BoardInput{
		Tasks: []recall.BoardTask{{ID: "6tj2b8", Title: "lint codes", State: "doing", Assignee: []string{"crab"}, Checks: 3, Open: 0}},
		Self:  recall.BoardSelf{Name: "crab"},
	}
	line, kind := e.BoardCue("s", "next", "", time.Now(), src(in))
	if kind != "done" || !strings.HasSuffix(line, "hug task done 6tj2b8") {
		t.Fatalf("kind=%q %q", kind, line)
	}
}

func TestBoardWorkOnTodo(t *testing.T) {
	e := boardEngine(t)
	in := recall.BoardInput{
		Tasks:   []recall.BoardTask{{ID: "npda3r", Title: "pattern", State: "todo", Checks: 3, Open: 3}},
		Self:    recall.BoardSelf{Name: "crab", Model: "gpt-6.1-sol"},
		Actions: []recall.BoardAction{{Hug: []recall.BoardHug{{Verb: "check", ID: "npda3r", N: 1}}}},
	}
	line, kind := e.BoardCue("s", "go", "", time.Now(), src(in))
	if kind != "work" || !strings.Contains(line, "hug task take npda3r --model gpt-6.1-sol") {
		t.Fatalf("kind=%q %q", kind, line)
	}
}

func TestBoardChatOnlySilent(t *testing.T) {
	e := boardEngine(t)
	in := recall.BoardInput{Tasks: []recall.BoardTask{{ID: "aaaaaa", Title: "x", State: "todo"}}, Self: recall.BoardSelf{Name: "crab"}}
	for i := 0; i < 5; i++ {
		if line, _ := e.BoardCue("s", "what do you think?", "", time.Now().Add(time.Duration(i)*time.Hour), src(in)); line != "" {
			t.Fatalf("chat turn: %q", line)
		}
	}
}

func TestBoardLintOnlyAtBreak(t *testing.T) {
	e := boardEngine(t)
	now := time.Now()
	in := recall.BoardInput{
		Tasks: []recall.BoardTask{{ID: "aaaaaa", State: "done"}, {ID: "bbbbbb", State: "todo"}, {ID: "cccccc", State: "doing"}},
		Findings: func() []recall.BoardFinding {
			return []recall.BoardFinding{
				{TaskID: "bbbbbb", TaskState: "todo", Problem: "no acceptance items", Fix: "hug task edit bbbbbb"},
				{TaskID: "cccccc", TaskState: "doing", Problem: "missing Goal", Fix: "hug task edit cccccc"},
			}
		},
		Self: recall.BoardSelf{Name: "crab"},
	}
	if line, _ := e.BoardCue("s", "go", "", now, src(in)); line != "" {
		t.Fatalf("no break: %q", line)
	}
	line, kind := e.BoardCue("s", "thanks", "wrap-up", now, src(in))
	if kind != "lint" || !strings.Contains(line, "cccccc") || !strings.HasSuffix(line, "fix: hug task edit cccccc") {
		t.Fatalf("doing first: kind=%q %q", kind, line)
	}
	if line, _ := e.BoardCue("s", "thanks", "return", now.Add(5*time.Hour), src(in)); line != "" {
		t.Fatalf("within 6 h: %q", line)
	}
	// A touched task comes first.
	in.Actions = []recall.BoardAction{{Hug: []recall.BoardHug{{Verb: "show", ID: "bbbbbb"}}}}
	line, _ = e.BoardCue("s", "thanks", "return", now.Add(7*time.Hour), src(in))
	if !strings.Contains(line, "bbbbbb") {
		t.Fatalf("touched first: %q", line)
	}
}

func TestBoardLintRotatesAcrossSessions(t *testing.T) {
	// One finding goes to one session; the next session gets the next one.
	e := boardEngine(t)
	now := time.Now()
	in := recall.BoardInput{
		Tasks: []recall.BoardTask{{ID: "q70jjy", State: "todo"}, {ID: "jy00xt", State: "todo"}},
		Findings: func() []recall.BoardFinding {
			return []recall.BoardFinding{
				{TaskID: "q70jjy", TaskState: "todo", Problem: "sections out of order", Fix: "hug upgrade"},
				{TaskID: "q70jjy", TaskState: "todo", Problem: "title is 138 characters (max 80)", Fix: "hug task set q70jjy --title \"...\""},
				{TaskID: "jy00xt", TaskState: "todo", Problem: "sections out of order", Fix: "hug task edit jy00xt"},
			}
		},
	}
	a, _ := e.BoardCue("a", "back", "return", now, src(in))
	b, _ := e.BoardCue("b", "back", "return", now.Add(time.Minute), src(in))
	c, _ := e.BoardCue("c", "back", "return", now.Add(2*time.Minute), src(in))
	if !strings.Contains(a, "q70jjy") || !strings.Contains(b, "jy00xt") || c != "" {
		t.Fatalf("a=%q\nb=%q\nc=%q", a, b, c)
	}
	if d, _ := e.BoardCue("d", "back", "return", now.Add(8*24*time.Hour), src(in)); d == "" {
		t.Fatal("finding never handed out again")
	}
}

// x325pp check 2: a turn that tells the agent to stop gets no board line.
func TestBoardSilentOnStopOrder(t *testing.T) {
	in := recall.BoardInput{
		Tasks: []recall.BoardTask{{ID: "6tj2b8", Title: "lint codes", State: "doing", Assignee: []string{"crab"}, Checks: 3, Open: 0}},
		Findings: func() []recall.BoardFinding {
			return []recall.BoardFinding{{TaskID: "p39w4a", TaskState: "todo", Problem: "title too long", Fix: "hug task set p39w4a --title \"...\""}}
		},
		Self: recall.BoardSelf{Name: "crab"},
	}
	for _, msg := range []string{
		"orangutan (foreman): STOP NOW. Do not edit any file, do not run any command, do not commit. Then end your session.",
		"Received and verified. Please stop now. The King will rule on the questions.",
		"Stand down; another engineer owns this lane.",
	} {
		e := boardEngine(t)
		if line, _ := e.BoardCue("s", msg, "return", time.Now(), src(in)); line != "" {
			t.Fatalf("%q got %q", msg, line)
		}
	}
	// A scope rule is not a stop order: the done line still comes.
	e := boardEngine(t)
	if line, kind := e.BoardCue("s", "Do not start any other task. When 6tj2b8 is finished, report and stop.", "", time.Now(), src(in)); kind != "done" {
		t.Fatalf("scope rule silenced the board: %q", line)
	}
}

// A finding withheld on a stop turn is not spent: the next break of the
// same session, or the next session, still gets it (tpnd3n; held-out
// replay: skunk was told to stop at the turn w96jpg's title line came due).
func TestBoardStopTurnKeepsFinding(t *testing.T) {
	now := time.Date(2026, 11, 11, 18, 13, 45, 0, time.UTC)
	in := recall.BoardInput{
		Tasks: []recall.BoardTask{{ID: "w96jpg", State: "todo"}},
		Findings: func() []recall.BoardFinding {
			return []recall.BoardFinding{{TaskID: "w96jpg", TaskState: "todo", Problem: "title is 151 characters (max 80)", Fix: "hug task set w96jpg --title \"...\""}}
		},
		Self: recall.BoardSelf{Name: "skunk"},
	}
	e := boardEngine(t)
	if line, _ := e.BoardCue("skunk", "Received. Please stop now.", "return", now, src(in)); line != "" {
		t.Fatalf("stop turn: %q", line)
	}
	if line, _ := e.BoardCue("skunk", "back: next item", "return", now.Add(time.Minute), src(in)); !strings.Contains(line, "w96jpg") {
		t.Fatalf("next break, same session: %q", line)
	}
	e = boardEngine(t)
	if line, _ := e.BoardCue("skunk", "Stand down.", "return", now, src(in)); line != "" {
		t.Fatalf("stop turn: %q", line)
	}
	in.Self.Name = "octopus"
	if line, _ := e.BoardCue("octopus", "back", "return", now.Add(time.Minute), src(in)); !strings.Contains(line, "w96jpg") {
		t.Fatalf("next session: %q", line)
	}
}

// A missing Purpose or Principle goes only to a session that touched the
// task; a passer-by gets the title finding instead (tpnd3n).
func TestBoardMissingSectionOnlyToItsSession(t *testing.T) {
	now := time.Date(2026, 11, 16, 10, 9, 34, 0, time.UTC)
	section := recall.BoardFinding{TaskID: "s6ff5k", TaskState: "todo", Problem: "missing section Purpose (why)", Fix: "hug task set s6ff5k --purpose \"...\""}
	title := recall.BoardFinding{TaskID: "q70jjy", TaskState: "todo", Problem: "title is 138 characters (max 80): move detail into the Goal", Fix: "hug task set q70jjy --title \"...\""}
	fs := []recall.BoardFinding{section}
	in := recall.BoardInput{
		Tasks:    []recall.BoardTask{{ID: "s6ff5k", State: "todo"}, {ID: "q70jjy", State: "todo"}},
		Findings: func() []recall.BoardFinding { return fs },
		Self:     recall.BoardSelf{Name: "hog"},
	}
	e := boardEngine(t)
	if line, _ := e.BoardCue("hog", "back", "return", now, src(in)); line != "" {
		t.Fatalf("section finding to a passer-by: %q", line)
	}
	fs = []recall.BoardFinding{section, title}
	if line, _ := e.BoardCue("hog", "back", "return", now.Add(7*time.Hour), src(in)); !strings.Contains(line, "q70jjy") {
		t.Fatalf("title finding to a passer-by: %q", line)
	}
	// The session named the task: its section is its to fill.
	fs = []recall.BoardFinding{section}
	if line, _ := e.BoardCue("hog", "now s6ff5k", "return", now.Add(14*time.Hour), src(in)); !strings.Contains(line, "s6ff5k") {
		t.Fatalf("section finding to the task's session: %q", line)
	}
}

// at3qep check 2: a question hug cannot settle reaches the ticket's owner
// at its next break, and nobody else: kestrel took 0hkaa5, it landed, no
// check is ticked.
func TestBoardQuestionReachesOwnerOnly(t *testing.T) {
	now := time.Date(2026, 11, 17, 10, 0, 0, 0, time.UTC)
	q := recall.BoardFinding{TaskID: "0hkaa5", TaskState: "doing",
		Problem: "landed, but no check is ticked: commit 61bfb21 naming 0hkaa5 reached main (its note says so) and 0 of 6 checks are ticked or struck; kestrel: which checks hold?",
		Fix:     "hug task check 0hkaa5 1"}
	in := func(self string) recall.BoardInput {
		return recall.BoardInput{
			Tasks:    []recall.BoardTask{{ID: "0hkaa5", State: "doing", Assignee: []string{"kestrel"}, Checks: 6, Open: 6}},
			Findings: func() []recall.BoardFinding { return []recall.BoardFinding{q} },
			Self:     recall.BoardSelf{Name: self},
		}
	}
	e := boardEngine(t)
	if line, _ := e.BoardCue("wyvern", "back", "return", now, src(in("wyvern"))); line != "" {
		t.Fatalf("question to a passer-by: %q", line)
	}
	line, kind := e.BoardCue("kestrel", "back", "return", now, src(in("kestrel")))
	if kind != "lint" || !strings.Contains(line, "0hkaa5: landed, but no check is ticked") || !strings.HasSuffix(line, "fix: hug task check 0hkaa5 1") {
		t.Fatalf("question to its owner: %q (%s)", line, kind)
	}
}

func TestBoardOffWhenNoSource(t *testing.T) {
	e := boardEngine(t)
	if line, _ := e.BoardCue("s", "go", "start", time.Now(), nil); line != "" {
		t.Fatal(line)
	}
}

func TestBoardOnHarnessNotice(t *testing.T) {
	// An autonomous night has only "[auto]" turns; the board still speaks.
	e := boardEngine(t)
	in := recall.BoardInput{
		Tasks: []recall.BoardTask{{ID: "6tj2b8", Title: "lint codes", State: "doing", Assignee: []string{"crab"}, Checks: 3, Open: 0}},
		Self:  recall.BoardSelf{Name: "crab"},
	}
	r, err := e.Recall(t.Context(), recall.Request{Session: "jcode:s", Message: "[auto] Do more validation", Board: src(in)})
	if err != nil {
		t.Fatal(err)
	}
	if r.Skipped != "harness notice" || r.BoardKind != "done" || !strings.HasPrefix(r.Block, recall.Marker+" Task board:") {
		t.Fatalf("%+v", r)
	}
}

// v3k6cp: a session working a doing task that the board assigns to another
// name gets one owner line with the exact fix; its own assignment, or a
// task it only looks at, gets none.
func TestBoardOwnerLine(t *testing.T) {
	now := time.Date(2026, 11, 17, 9, 44, 0, 0, time.UTC)
	task := recall.BoardTask{ID: "44w1ev", Title: "Scale", State: "doing", Assignee: []string{"humpback"}, Checks: 5, Open: 3}
	for name, c := range map[string]struct {
		in   recall.BoardInput
		want string
	}{
		"checks another's task": {recall.BoardInput{Tasks: []recall.BoardTask{task}, Self: recall.BoardSelf{Name: "wyvern"},
			Actions: []recall.BoardAction{{Hug: []recall.BoardHug{{Verb: "check", ID: "44w1ev", N: 1}}}}},
			"Task board: you are working on 44w1ev but the board says humpback. If it is yours now: hug task set 44w1ev --assignee wyvern"},
		"notes it while editing": {recall.BoardInput{Tasks: []recall.BoardTask{task}, Self: recall.BoardSelf{Name: "wyvern"},
			Actions: []recall.BoardAction{{Hug: []recall.BoardHug{{Verb: "note", ID: "44w1ev"}}}, edit}}, "owner"},
		"works in its worktree": {recall.BoardInput{Tasks: []recall.BoardTask{{ID: "44w1ev", State: "doing", Assignee: []string{"humpback"}, Worktree: "/s/hug-44w1ev", Checks: 5, Open: 3}},
			Self: recall.BoardSelf{Name: "wyvern", Worktree: "/s/hug-44w1ev", Linked: true}, Actions: []recall.BoardAction{edit}}, "owner"},
		"only shows it":   {recall.BoardInput{Tasks: []recall.BoardTask{task}, Self: recall.BoardSelf{Name: "wyvern"}, Actions: []recall.BoardAction{{Hug: []recall.BoardHug{{Verb: "show", ID: "44w1ev"}}}}}, ""},
		"notes, no edits": {recall.BoardInput{Tasks: []recall.BoardTask{task}, Self: recall.BoardSelf{Name: "crab"}, Actions: []recall.BoardAction{{Hug: []recall.BoardHug{{Verb: "note", ID: "44w1ev"}}}}}, ""},
		"its own":         {recall.BoardInput{Tasks: []recall.BoardTask{task}, Self: recall.BoardSelf{Name: "humpback"}, Actions: []recall.BoardAction{{Hug: []recall.BoardHug{{Verb: "check", ID: "44w1ev", N: 1}}}}}, ""},
		"no session name": {recall.BoardInput{Tasks: []recall.BoardTask{task}, Actions: []recall.BoardAction{{Hug: []recall.BoardHug{{Verb: "check", ID: "44w1ev", N: 1}}}}}, ""},
	} {
		e := boardEngine(t)
		line, kind := e.BoardCue("s", "go on", "", now, src(c.in))
		switch {
		case c.want == "" && line != "":
			t.Errorf("%s: got %q", name, line)
		case c.want == "owner" && kind != "owner":
			t.Errorf("%s: kind %q line %q", name, kind, line)
		case c.want != "" && c.want != "owner" && line != c.want:
			t.Errorf("%s:\n got %q\nwant %q", name, line, c.want)
		}
	}
	// Once per task per day, as the other kinds.
	e := boardEngine(t)
	in := recall.BoardInput{Tasks: []recall.BoardTask{task}, Self: recall.BoardSelf{Name: "wyvern"}, Actions: []recall.BoardAction{{Hug: []recall.BoardHug{{Verb: "check", ID: "44w1ev", N: 1}}}}}
	if _, k := e.BoardCue("s", "go", "", now, src(in)); k != "owner" {
		t.Fatal(k)
	}
	if l, _ := e.BoardCue("s", "go", "", now.Add(2*time.Hour), src(in)); l != "" {
		t.Fatalf("repeated: %q", l)
	}
}

// tpnd3n check 1: while the session's own doing task has a finding, that
// one comes first and nothing unrelated is handed out; once it is fixed,
// an unrelated finding may follow at the next break.
func TestBoardUnrelatedLintWaitsForOwnTask(t *testing.T) {
	e := boardEngine(t)
	now := time.Date(2026, 11, 17, 10, 0, 0, 0, time.UTC)
	own := recall.BoardFinding{TaskID: "own111", TaskState: "doing", Problem: "missing section Purpose", Fix: "hug task set own111 --purpose \"...\""}
	other := recall.BoardFinding{TaskID: "oth222", TaskState: "doing", Problem: "title is 120 characters (max 80)", Fix: "hug task set oth222 --title \"...\""}
	fs := []recall.BoardFinding{other, own}
	in := recall.BoardInput{
		Tasks:    []recall.BoardTask{{ID: "own111", State: "doing", Assignee: []string{"crab"}, Checks: 2, Open: 1}, {ID: "oth222", State: "doing", Assignee: []string{"vole"}}},
		Findings: func() []recall.BoardFinding { return fs },
		Self:     recall.BoardSelf{Name: "crab"},
	}
	line, kind := e.BoardCue("s", "back", "return", now, src(in))
	if kind != "lint" || !strings.Contains(line, "own111") {
		t.Fatalf("own task's finding first: %q %q", kind, line)
	}
	// Own finding still open (handed out, not yet fixed): nothing unrelated.
	if line, _ := e.BoardCue("s", "back", "return", now.Add(7*time.Hour), src(in)); line != "" {
		t.Fatalf("unrelated finding while own task has one: %q", line)
	}
	// Fixed: the unrelated one follows at the next break.
	fs = []recall.BoardFinding{other}
	if line, _ := e.BoardCue("s", "back", "return", now.Add(14*time.Hour), src(in)); !strings.Contains(line, "oth222") {
		t.Fatalf("unrelated after own fixed: %q", line)
	}
	// A session with no own task gets the unrelated finding as before.
	e2 := boardEngine(t)
	free := in
	free.Tasks = []recall.BoardTask{{ID: "oth222", State: "doing", Assignee: []string{"vole"}}}
	free.Findings = func() []recall.BoardFinding { return []recall.BoardFinding{other} }
	if line, _ := e2.BoardCue("s", "back", "return", now, src(free)); !strings.Contains(line, "oth222") {
		t.Fatalf("no own task: %q", line)
	}
}

// An unread answer to the session's own question is the first board line,
// once; an answer for another session is not shown (3jtfw9 check 3).
func TestBoardAnswerForAsker(t *testing.T) {
	now := time.Date(2026, 11, 17, 13, 31, 0, 0, time.UTC)
	ans := &recall.BoardAnswer{Stage: "design", Question: "one key or two?", Text: "one key: a", By: "jhf", For: "nautilus", At: "2026-11-17T13:30"}
	in := recall.BoardInput{
		Tasks: []recall.BoardTask{
			{ID: "3jtfw9", Title: "answer in place", State: "todo", Answer: ans},
			{ID: "6tj2b8", Title: "lint codes", State: "doing", Assignee: []string{"nautilus"}, Checks: 2, Open: 0},
		},
		Self: recall.BoardSelf{Name: "Nautilus"},
	}
	e := boardEngine(t)
	line, kind := e.BoardCue("s", "go on", "", now, src(in))
	want := "jhf answered your question on 3jtfw9: one key: a (design: one key or two?). hug task show 3jtfw9"
	if kind != "answer" || line != want {
		t.Fatalf("got %q %q, want %q", kind, line, want)
	}
	// Said once: the next turn gets the done line it held back.
	if line, kind := e.BoardCue("s", "go on", "", now.Add(time.Minute), src(in)); kind != "done" {
		t.Fatalf("second turn: %q %q", kind, line)
	}
	// A later answer on the same task is new, and not held by the hour.
	in.Tasks[0].Answer = &recall.BoardAnswer{Stage: "design", Question: "and the key name?", Text: "a", By: "jhf", For: "nautilus", At: "2026-11-17T13:40"}
	if line, kind := e.BoardCue("s", "go on", "", now.Add(10*time.Minute), src(in)); kind != "answer" {
		t.Fatalf("later answer: %q %q", kind, line)
	}
	// Another session does not see it.
	other := in
	other.Self = recall.BoardSelf{Name: "crab"}
	other.Tasks = in.Tasks[:1]
	if line, _ := boardEngine(t).BoardCue("s2", "go on", "", now, src(other)); line != "" {
		t.Fatalf("answer for nautilus shown to crab: %q", line)
	}
}
