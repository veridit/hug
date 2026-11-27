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
		Actions: []recall.BoardAction{{Subject: "TUI: human names (7js5hw)"}, edit},
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
	other.Actions = []recall.BoardAction{{Subject: "setup: desktop apps (48hx44)"}, edit}
	if line, _ := e.BoardCue("jcode:s1", "continue", "", now.Add(62*time.Minute), src(other)); !strings.Contains(line, "48hx44") {
		t.Fatalf("another task after the hour: %q", line)
	}
	if line, _ := e.BoardCue("jcode:s1", "continue", "", now.Add(25*time.Hour), src(in)); line == "" {
		t.Fatal("no line after a day")
	}
}

// rzfrgp: reading a task with hug task show does not make it the task the
// take line names. guppy 2026-11-16 20:16 read jrj1vq (a dependency of
// ez50rb), edited code with no task doing, and was told 'this looks like
// jrj1vq'; both judgings called it wrong.
func TestBoardTakeNotFromShow(t *testing.T) {
	e := boardEngine(t)
	now := time.Date(2026, 11, 16, 20, 16, 10, 0, time.UTC)
	in := recall.BoardInput{
		Tasks:   []recall.BoardTask{{ID: "jrj1vq", Title: "When-then memories: triggers stated at write time", State: "todo"}},
		Self:    recall.BoardSelf{Name: "guppy", Model: "claude-opus-5-5"},
		Actions: []recall.BoardAction{{Hug: []recall.BoardHug{{Verb: "show", ID: "jrj1vq"}}}, edit},
	}
	if line, _ := e.BoardCue("guppy", "continue", "", now, src(in)); strings.Contains(line, "jrj1vq") {
		t.Fatalf("take line names a task the session only read: %q", line)
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

// 4vxqp4: a task the message names only as context (quoted, a dependency,
// a task not to start) gets no missing-section line; named as the work, or
// acted on, it does. Messages from the replay (hog, evergreen, maple).
func TestBoardMissingSectionNotForAMention(t *testing.T) {
	now := time.Date(2026, 11, 16, 10, 9, 34, 0, time.UTC)
	section := func(id string) recall.BoardFinding {
		return recall.BoardFinding{TaskID: id, TaskState: "todo", Problem: "missing section Purpose (why)", Fix: "hug task set " + id + " --purpose \"...\""}
	}
	in := func(as ...recall.BoardAction) recall.BoardInput {
		return recall.BoardInput{
			Tasks: []recall.BoardTask{{ID: "s6ff5k", State: "todo"}, {ID: "mv4vye", State: "doing", Assignee: []string{"hog"}},
				{ID: "n8zmjw", State: "doing", Assignee: []string{"hibiscus"}}, {ID: "p39w4a", State: "todo"}},
			Findings: func() []recall.BoardFinding {
				return []recall.BoardFinding{section("s6ff5k"), section("n8zmjw"), section("p39w4a")}
			},
			Self:    recall.BoardSelf{Name: "hog"},
			Actions: as,
		}
	}
	cases := []struct {
		name, msg string
		as        []recall.BoardAction
		want      string // task id the line names; "" for none
	}{
		{"quoted prompt", `Your one task is hug task mv4vye. Do not start any other task. Confirm prompts ("Confirm edit s6ff5k Title?") are bland.`, nil, ""},
		{"quoted output", "Today: `hug task lint` prints `next: hug task take s6ff5k` (unrelated).", nil, ""},
		{"forbidden", "Work on mv4vye. Do not start s6ff5k or any other task.", nil, ""},
		{"dependency", "Read `hug task show p39w4a`. Don't take it: p39w4a depends on n8zmjw.", nil, "p39w4a"},
		{"assigned", "Your ticket: s6ff5k. Read `hug task show s6ff5k`.", nil, "s6ff5k"},
		{"quoted but acted on", `Fix the prompt "Confirm edit s6ff5k Title?".`, []recall.BoardAction{{Hug: []recall.BoardHug{{Verb: "note", ID: "s6ff5k"}}}}, "s6ff5k"},
		{"read another's doing task", "back", []recall.BoardAction{{Hug: []recall.BoardHug{{Verb: "show", ID: "n8zmjw"}}}}, ""},
		{"read an open todo", "back", []recall.BoardAction{{Hug: []recall.BoardHug{{Verb: "show", ID: "s6ff5k"}}}}, "s6ff5k"},
	}
	for _, c := range cases {
		e := boardEngine(t)
		line, _ := e.BoardCue("hog", c.msg, "start", now, src(in(c.as...)))
		if c.want == "" && line != "" || c.want != "" && !strings.Contains(line, c.want+": missing section") {
			t.Errorf("%s: got %q, want a line on %q", c.name, line, c.want)
		}
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

// agh2h7 (bg1rd8 check 2): a waiting task whose every check holds is its
// owner's to close; with every owner gone (by stale-owner's rule, 9ahvgt:
// closed over an hour ago, or silent for a day), the board's coordinator
// is told instead, and nobody else. 44w1ev: wyvern's, all 5 checks
// ticked, a day waiting.
func TestBoardChecksHoldOwnerElseCoordinator(t *testing.T) {
	now := time.Date(2026, 11, 18, 12, 0, 0, 0, time.UTC)
	f := recall.BoardFinding{TaskID: "44w1ev", TaskState: "doing",
		Problem: "every check holds, but doing/waiting on landing slot for 25 hours: 5 of 5 checks are ticked or struck and 3 of 3 children are done; wyvern: is it done?",
		Fix:     "hug task done 44w1ev"}
	in := func(self, coordinator string, ownerGone bool) recall.BoardInput {
		return recall.BoardInput{
			Tasks:       []recall.BoardTask{{ID: "44w1ev", State: "doing", Assignee: []string{"wyvern"}, Checks: 5, Open: 0}},
			Coordinator: coordinator,
			Findings:    func() []recall.BoardFinding { return []recall.BoardFinding{f} },
			Self:        recall.BoardSelf{Name: self},
			Gone: func(who string, at time.Time) (bool, bool) {
				switch who {
				case "wyvern":
					return ownerGone, true
				case "crab", "kestrel":
					return false, true
				}
				return false, false
			},
		}
	}
	const gone, live = true, false
	cases := []struct {
		name, self, coordinator string
		owner                   bool
		want                    string // "" none, "owner", "coordinator"
	}{
		{"the owner", "wyvern", "crab", live, "owner"},
		{"the owner, after a long absence", "wyvern", "crab", gone, "owner"},
		{"a passer-by", "kestrel", "crab", gone, ""},
		{"the coordinator, owner live", "crab", "crab", live, ""},
		{"the coordinator, owner gone", "crab", "crab", gone, "coordinator"},
		{"no coordinator on the board", "kestrel", "", gone, ""},
	}
	for _, c := range cases {
		e := boardEngine(t)
		line, kind := e.BoardCue(c.self, "back", "return", now, src(in(c.self, c.coordinator, c.owner)))
		switch c.want {
		case "":
			if line != "" {
				t.Errorf("%s: got %q, want no line", c.name, line)
			}
		case "owner":
			// The owner's own done line says it already (mineAll); the lint
			// line is the same fix.
			if kind != "lint" && kind != "done" || !strings.Contains(line, "44w1ev") || strings.Contains(line, "coordinator") || !strings.HasSuffix(line, "hug task done 44w1ev") {
				t.Errorf("%s: got %q (%s)", c.name, line, kind)
			}
		case "coordinator":
			if kind != "lint" || !strings.Contains(line, "44w1ev: every check holds") || !strings.Contains(line, "Its owner wyvern has no live session, so it is yours as coordinator") || !strings.HasSuffix(line, "fix: hug task done 44w1ev") {
				t.Errorf("%s: got %q (%s)", c.name, line, kind)
			}
		}
	}
	// An owner the harness does not know (a person) is never gone: no line
	// to the coordinator on its task.
	person := in("crab", "crab", gone)
	person.Tasks[0].Assignee = []string{"jhf"}
	if line, _ := boardEngine(t).BoardCue("crab", "back", "return", now, src(person)); line != "" {
		t.Errorf("a person's task to the coordinator: %q", line)
	}
	// Said once: the next break within LintRepeat is quiet.
	e := boardEngine(t)
	if line, _ := e.BoardCue("crab", "back", "return", now, src(in("crab", "crab", gone))); line == "" {
		t.Fatal("first break: no line")
	}
	if line, _ := e.BoardCue("crab", "back", "return", now.Add(7*time.Hour), src(in("crab", "crab", gone))); line != "" {
		t.Errorf("said again: %q", line)
	}
}

// rk5q7r: open reflections older than 3 days reach the board's coordinator
// at a break, as a count with hug task reflections --by-instruction; never
// another session, never mid-work, never young ones, at most once a day.
func TestBoardOldReflectionsToCoordinator(t *testing.T) {
	now := time.Date(2026, 11, 21, 10, 0, 0, 0, time.Local)
	dates := []string{"2026-11-17", "2026-11-17", "2026-11-18", "2026-11-20"} // 2 older than 3 days
	in := func(self, coordinator string) recall.BoardInput {
		return recall.BoardInput{
			Coordinator: coordinator,
			Findings:    func() []recall.BoardFinding { return nil },
			Self:        recall.BoardSelf{Name: self},
			Reflections: func() []string { return dates },
		}
	}
	want := "Task board (return): 2 open reflections are older than 3 days; read them grouped by the instruction they name: hug task reflections --by-instruction"
	e := boardEngine(t)
	if line, kind := e.BoardCue("crab", "back", "return", now, src(in("crab", "crab"))); line != want || kind != "reflect" {
		t.Fatalf("coordinator at a break: %q (%s)", line, kind)
	}
	if line, _ := e.BoardCue("crab", "back", "return", now.Add(7*time.Hour), src(in("crab", "crab"))); line != "" {
		t.Errorf("said again within a day: %q", line)
	}
	if line, _ := e.BoardCue("crab", "back", "return", now.Add(25*time.Hour), src(in("crab", "crab"))); !strings.Contains(line, "3 open reflections") {
		t.Errorf("next day (11-18 now old too): %q", line)
	}
	for _, c := range []struct{ name, self, coordinator, brk string }{
		{"another session", "kestrel", "crab", "return"},
		{"no coordinator", "crab", "", "return"},
		{"mid-work", "crab", "crab", ""},
	} {
		if line, _ := boardEngine(t).BoardCue(c.self, "back", c.brk, now, src(in(c.self, c.coordinator))); line != "" {
			t.Errorf("%s: %q", c.name, line)
		}
	}
	young := in("crab", "crab")
	young.Reflections = func() []string { return []string{"2026-11-18", "2026-11-20"} }
	if line, _ := boardEngine(t).BoardCue("crab", "back", "return", now, src(young)); line != "" {
		t.Errorf("only young reflections: %q", line)
	}
	// A lint finding for the coordinator comes first; the reflections wait.
	lint := in("crab", "crab")
	lint.Tasks = []recall.BoardTask{{ID: "abc123", State: "todo"}}
	lint.Findings = func() []recall.BoardFinding {
		return []recall.BoardFinding{{TaskID: "abc123", TaskState: "todo", Problem: "title is longer than 80 characters", Fix: "hug task set abc123 --title \"...\""}}
	}
	if line, _ := boardEngine(t).BoardCue("crab", "back", "return", now, src(lint)); !strings.Contains(line, "abc123: title") {
		t.Errorf("lint first: %q", line)
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

// hbgybn check 3: a person's open request is said once to the task's
// assignee, with its text and the resolve command; a task nobody holds
// routes to the coordinator, or with none set to every session (each once);
// another session does not hear a held task's request.
func TestBoardRequestRouting(t *testing.T) {
	now := time.Date(2026, 11, 18, 1, 0, 0, 0, time.UTC)
	req := []recall.BoardRequest{{N: 1, By: "jhf", At: "2026-11-18T00:58", Text: "make it per board\nand split the TUI part"}}
	held := recall.BoardTask{ID: "6ephqk", Title: "tags", State: "doing", Assignee: []string{"nautilus"}, Checks: 2, Open: 1, Requests: req}
	free := recall.BoardTask{ID: "x1t1t1", Title: "rows", State: "todo", Requests: []recall.BoardRequest{{N: 2, By: "jhf", At: "2026-11-18T00:59", Text: "shorter title"}}}
	in := func(self, coordinator string, ts ...recall.BoardTask) recall.BoardInput {
		return recall.BoardInput{Tasks: ts, Self: recall.BoardSelf{Name: self}, Coordinator: coordinator}
	}
	e := boardEngine(t)
	line, kind := e.BoardCue("s", "go on", "", now, src(in("nautilus", "crab", held)))
	want := `Task board: jhf asks on 6ephqk: "make it per board and split the TUI part" (request 1). Act on it, then: hug task resolve 6ephqk 1 "what changed"`
	if kind != "request" || line != want {
		t.Fatalf("assignee: got %q %q, want %q", kind, line, want)
	}
	// Said once.
	if line, kind = e.BoardCue("s", "go on", "", now.Add(time.Minute), src(in("nautilus", "crab", held))); kind == "request" {
		t.Fatalf("said twice: %q", line)
	}
	// Held by nautilus: the coordinator does not hear it.
	if line, _ = boardEngine(t).BoardCue("c", "go on", "", now, src(in("crab", "crab", held))); line != "" {
		t.Fatalf("coordinator heard a held task's request: %q", line)
	}
	// Nobody holds it: the coordinator hears it, another session does not.
	if line, kind = boardEngine(t).BoardCue("c", "go on", "", now, src(in("crab", "crab", free))); kind != "request" || !strings.Contains(line, "hug task resolve x1t1t1 2") {
		t.Fatalf("coordinator, unowned: %q %q", kind, line)
	}
	if line, _ = boardEngine(t).BoardCue("w", "go on", "", now, src(in("wyvern", "crab", free))); line != "" {
		t.Fatalf("a non-coordinator heard an unowned request: %q", line)
	}
	// No coordinator: every session hears it, once each.
	for _, who := range []string{"wyvern", "crab"} {
		ew := boardEngine(t)
		if line, kind = ew.BoardCue(who, "go on", "", now, src(in(who, "", free))); kind != "request" {
			t.Fatalf("%s, no coordinator: %q %q", who, kind, line)
		}
		if line, kind = ew.BoardCue(who, "go on", "", now.Add(time.Minute), src(in(who, "", free))); kind == "request" {
			t.Fatalf("%s heard it twice: %q", who, line)
		}
	}
	// Without the session's name, nothing is routed.
	if line, _ = boardEngine(t).BoardCue("n", "go on", "", now, src(in("", "", free))); line != "" {
		t.Fatalf("nameless session heard: %q", line)
	}
}

// 41hhgn check 11 (cue half): the board's coordinator hears once what the
// asker did after a person's answer; the asker itself, or a board with no
// coordinator, hears nothing.
func TestBoardAnswerOutcomeToCoordinator(t *testing.T) {
	now := time.Date(2026, 11, 18, 1, 0, 0, 0, time.UTC)
	tk := recall.BoardTask{ID: "41hhgn", Title: "asked", State: "doing", Assignee: []string{"nautilus"}, Checks: 2, Open: 1,
		Outcome: &recall.BoardOutcome{Stage: "asked", By: "jhf", For: "nautilus", At: "2026-11-18T00:59", Text: `hug task note 41hhgn "rows now list the open ones first"`}}
	in := func(self, coordinator string) recall.BoardInput {
		return recall.BoardInput{Tasks: []recall.BoardTask{tk}, Self: recall.BoardSelf{Name: self}, Coordinator: coordinator}
	}
	e := boardEngine(t)
	line, kind := e.BoardCue("c", "go on", "", now, src(in("crab", "crab")))
	want := `Task board: nautilus acted on jhf's answer on 41hhgn (asked): hug task note 41hhgn "rows now list the open ones first". hug task show 41hhgn`
	if kind != "answered" || line != want {
		t.Fatalf("coordinator: got %q %q, want %q", kind, line, want)
	}
	if line, kind = e.BoardCue("c", "go on", "", now.Add(time.Minute), src(in("crab", "crab"))); kind == "answered" {
		t.Fatalf("said twice: %q", line)
	}
	if line, _ = boardEngine(t).BoardCue("n", "go on", "", now, src(in("nautilus", "nautilus"))); strings.Contains(line, "acted on") {
		t.Fatalf("the asker heard its own outcome: %q", line)
	}
	if line, _ = boardEngine(t).BoardCue("w", "go on", "", now, src(in("wyvern", "crab"))); strings.Contains(line, "acted on") {
		t.Fatalf("a non-coordinator heard it: %q", line)
	}
	if line, _ = boardEngine(t).BoardCue("c", "go on", "", now, src(in("crab", ""))); strings.Contains(line, "acted on") {
		t.Fatalf("heard with no coordinator set: %q", line)
	}
}

// erv680: a step another role waits on (a review, ship it) reaches the
// session it is for, once per task, not held back by the hourly rate: two
// reviews in one hour are both said; another session hears neither.
func TestBoardStepForAnotherRole(t *testing.T) {
	now := time.Date(2026, 11, 27, 10, 0, 0, 0, time.UTC)
	step := func(id, kind string) recall.BoardTask {
		cue := "wolf asks you to review " + id + " (x): read it, then answer looks right, or what to change: hug task answer " + id + ` "looks right"`
		if kind == "ship" {
			cue = "crab approved " + id + " (x): ship it, as your AGENTS.md says; once it has shipped: hug task done " + id
		}
		return recall.BoardTask{ID: id, Title: "x", State: "doing", Assignee: []string{"wolf"}, Checks: 1, Open: 0,
			Step: recall.BoardStep{Kind: kind, For: "crab", Cue: cue}}
	}
	in := recall.BoardInput{Tasks: []recall.BoardTask{step("aaaaaa", "review"), step("bbbbbb", "review"), step("cccccc", "ship")}, Coordinator: "crab", Self: recall.BoardSelf{Name: "crab"}}
	e := boardEngine(t)
	var said []string
	for i := 0; i < 4; i++ {
		line, kind := e.BoardCue("s", "go on", "", now.Add(time.Duration(i)*time.Minute), src(in))
		if line != "" {
			said = append(said, kind+" "+line)
		}
	}
	if len(said) != 3 || !strings.Contains(said[0], "review Task board: wolf asks you to review aaaaaa") || !strings.Contains(said[1], "review Task board: wolf asks you to review bbbbbb") || !strings.HasPrefix(said[2], "ship Task board: crab approved cccccc") {
		t.Fatalf("crab heard %q", said)
	}
	other := in
	other.Self = recall.BoardSelf{Name: "wolf"}
	if line, kind := boardEngine(t).BoardCue("s", "go on", "", now, src(other)); kind == "review" || kind == "ship" {
		t.Fatalf("wolf heard crab's step: %q %q", kind, line)
	}
}
