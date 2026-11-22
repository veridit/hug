package tui

import (
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/x/ansi"
	"github.com/veridit/hug/huginn/internal/task"
)

// stageBoard is a board with a task in every stage that shows a reason,
// written as hug upgrade will write them, with fixed ids.
func stageBoard(t *testing.T) model {
	t.Helper()
	// Both clocks: a wait stamped by the store reads "since HH:MM" on any
	// day the test runs (it failed after midnight).
	old, oldTask := now, task.Now
	now = func() time.Time { return time.Date(2026, 11, 17, 9, 53, 0, 0, time.Local) }
	task.Now = func() time.Time { return time.Date(2026, 11, 17, 9, 50, 0, 0, time.Local) }
	t.Cleanup(func() { now, task.Now = old, oldTask })
	s := &task.Store{Root: t.TempDir()}
	write := func(id, title, front string, deps ...string) {
		who := "[]"
		if strings.Contains(front, "\"doing\"") {
			who = "[\"nautilus\"]"
		}
		body := "---\nid: \"" + id + "\"\ntitle: \"" + title + "\"\n" + front + "\ntype: \"feature\"\npriority: \"high\"\nlabels: []\nassignee: " + who + "\ncreated: \"2026-11-01\"\nupdated: \"2026-11-01\"\n---\n\n## Goal\ng\n\n## Purpose\np\n\n## Principle\nq\n"
		if len(deps) > 0 {
			body += "\n## Depends on\n"
			for _, d := range deps {
				body += "- " + d + "\n"
			}
		}
		body += "\n## Acceptance\n- [x] a\n- [ ] b\n"
		if e := os.WriteFile(filepath.Join(s.Root, "0"+id[:1]+"."+id+".t.md"), []byte(body), 0o644); e != nil {
			t.Fatal(e)
		}
	}
	write("aaaaaa", "Status colors: rows and sections share one subtle scheme", "state: \"doing\"\nstage: \"active\"\ndoing_since: \"2026-11-17T08:00\"")
	write("bbbbbb", "Release, then hug setup in seid-lang", "state: \"doing\"\nstage: \"review\"\ndoing_since: \"2026-11-17T08:00\"\nreview_by: \"Jørgen\"\nreview_since: \"2026-11-17T09:48\"")
	write("cccccc", "Warm cache under 50 ms", "state: \"doing\"\nstage: \"waiting\"\ndoing_since: \"2026-11-17T08:00\"\nwaiting_on: \"CI run 44063552584\"\nwaiting_since: \"2026-11-17T09:41\"")
	write("dddddd", "One set of variable names", "state: \"doing\"\nstage: \"depends\"\ndoing_since: \"2026-11-17T08:00\"", "cccccc", "eeeeee")
	write("eeeeee", "Tags listed per board", "state: \"todo\"\nstage: \"design\"\ndesign_question: \"should type leave the rows, or stay as a column?\"\ndesign_for: \"jhf\"")
	write("ffffff", "Depend on a group or epic", "state: \"todo\"\nstage: \"ready\"")
	write("gggggg", "Answer a question in place", "state: \"todo\"\nstage: \"deferred\"\ndeferred_reason: \"after the release\"")
	m := newModel(s, Options{User: "tester"})
	return m
}

// focusAt draws the overview w wide with the cursor on id, in the section
// that holds it: the row under the cursor opens its second line (tje4sx).
func focusAt(t *testing.T, m model, w int, id string) string {
	t.Helper()
	m.width, m.height = w, 30
	m.resize()
	m.setView("overview")
	for p := range overviewSections {
		for _, i := range m.paneIndices(p) {
			if m.visible[i].ID == id {
				m.pane, m.cursor = p, i
				return ansi.Strip(m.View())
			}
		}
	}
	t.Fatalf("%s is in no overview section", id)
	return ""
}

// section is the text of overview section name, its title line first.
func section(view, name string) string {
	at := strings.Index(view, name)
	if at < 0 {
		return ""
	}
	var out []string
	for _, line := range strings.Split(view[at:], "\n") {
		// The section ends at its end rule (v68n82), or a blank line.
		if strings.TrimSpace(line) == "" || endRule.MatchString(strings.TrimRight(line, " ")) {
			break
		}
		out = append(out, line)
	}
	return strings.Join(out, "\n")
}

// 6ephqk check 5, tje4sx: rows have no state column and read pie, id,
// type, priority, stage, title, one line each; the row under the cursor
// opens a second line that says what it waits on, under the id, the
// reason right after the colon (89y0b7). IN PROGRESS holds only active
// work, WAITING what waits on a person or a thing, todo or doing, and
// doing tasks held by a dependency; a todo task that depends stays in
// NEXT UP (goldens at 80 and 140).
func TestRowsShowStage(t *testing.T) {
	m := stageBoard(t)
	view := ansi.Strip(viewAt(m, "overview", 140))
	for _, want := range []string{
		// IN PROGRESS holds one task, so it keeps its columns; WAITING and
		// NEXT UP share type and priority, said once in the heading
		// (1bbe96).
		// 911j70: ID TITLE STAGE ✓ TYPE PRIORITY OWNER.
		`aaaaaa   Status colors: rows and sections share one subtle scheme +active    ◑ 1/2 feature high`,
		`bbbbbb   Release, then hug setup in seid-lang +review `,
		`cccccc   Warm cache under 50 ms +waiting `,
		`dddddd   One set of variable names +depends `,
		`eeeeee   Tags listed per board +design `,
		`ffffff   Depend on a group or epic +ready `,
		`gggggg   Answer a question in place +deferred `,
	} {
		if !regexp.MustCompile(want).MatchString(view) {
			t.Errorf("overview lacks %q", want)
		}
	}
	// Check 1: who is where.
	for name, ids := range map[string][]string{
		"IN PROGRESS": {"aaaaaa"},
		"WAITING":     {"bbbbbb", "cccccc", "dddddd", "eeeeee"},
		"NEXT UP":     {"ffffff", "gggggg"},
	} {
		got := section(view, name)
		for _, id := range ids {
			if !strings.Contains(got, id) {
				t.Errorf("%s lacks %s:\n%s", name, id, got)
			}
		}
		if n := strings.Count(got, "\n"); n != len(ids) {
			t.Errorf("%s has %d rows, want %d (one line each, nothing else):\n%s", name, n, len(ids), got)
		}
	}
	if !regexp.MustCompile(`(?m)^─── WAITING  \[4\] · all feature · all high ─+\s*$`).MatchString(view) {
		t.Errorf("WAITING header is not 'WAITING  [4]' (what asks you is in ASKED, 41hhgn):\n%s", view)
	}
	// Check 4: no row the cursor is not on has a second line.
	if strings.Contains(view, "└─") {
		t.Errorf("an unfocused row has a second line:\n%s", view)
	}
	// The row under the cursor opens it, right under itself.
	for id, note := range map[string]string{
		"bbbbbb": "└─ in review by Jørgen",
		"cccccc": "└─ waiting on CI run 44063552584",
		"dddddd": "└─ depends on cccccc, eeeeee  ▱▱ 0/2",
		"eeeeee": "└─ question for jhf: should type leave the rows, or stay as a column?",
		"gggggg": "└─ deferred: after the release",
	} {
		focused := focusAt(t, m, 140, id)
		if !regexp.MustCompile(`(?m)^> .*`+id+`.*\n\s+`+regexp.QuoteMeta(note)).MatchString(focused) || strings.Count(focused, "└─") != 1 {
			t.Errorf("focused %s does not open %q, alone:\n%s", id, note, focused)
		}
	}
	if regexp.MustCompile(`(?m)^\S*\s*[○◔◑◕●].*\b(doing|todo)\b\s{2,}`).MatchString(view) {
		t.Errorf("a row still shows its state:\n%s", view)
	}
	if t.Failed() {
		t.Fatal(view)
	}
	var got []string
	for _, w := range []int{80, 140} {
		for _, v := range []string{"overview", "WAITING focused", "list"} {
			var screen string
			if v == "WAITING focused" {
				screen = focusAt(t, m, w, "cccccc")
			} else {
				screen = ansi.Strip(viewAt(m, v, w))
			}
			for _, line := range strings.Split(screen, "\n") {
				if n := ansi.StringWidth(line); n > w {
					t.Fatalf("at %d a line is %d wide: %q", w, n, line)
				}
			}
			got = append(got, "== "+v+" at "+strconv.Itoa(w), screen)
		}
	}
	text := strings.Join(got, "\n")
	path := filepath.Join("testdata", "rows", "stages.txt")
	if os.Getenv("HUG_UPDATE_GOLDEN") == "1" {
		if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("%v\nfix: HUG_UPDATE_GOLDEN=1 go test -run TestRowsShowStage ./huginn/internal/tui/", err)
	}
	if text != strings.ReplaceAll(string(want), "\r\n", "\n") {
		t.Fatalf("stage rows differ from %s:\n%s", path, text)
	}
}

// The depends bar: done of total, at most 5 cells (Jørgen 11:44; the look
// is one function, dependsCell).
func TestDependsBar(t *testing.T) {
	for _, c := range []struct {
		done, total int
		want        string
	}{{0, 1, "▱ 0/1"}, {2, 3, "▰▰▱ 2/3"}, {3, 3, "▰▰▰ 3/3"}, {4, 10, "▰▰▱▱▱ 4/10"}, {0, 0, ""}} {
		if got := dependsBar(c.done, c.total); got != c.want {
			t.Errorf("dependsBar(%d, %d) = %q, want %q", c.done, c.total, got, c.want)
		}
	}
}

// 6ephqk check 8, 89y0b7, tje4sx check 2: a task waiting on the TUI user
// sorts to the top of WAITING and is the only row marked ?; the header
// says "? N for you" (crab 20:10: the one count seen in every view), the
// WAITING title only counts its rows. Focused, its second line says "you"
// (Jørgen 11:42, crab 12:38 and 15:28: review_by, design_for, or
// waiting_for).
func TestWaitingOnYou(t *testing.T) {
	m := stageBoard(t)
	marked := regexp.MustCompile(`(?m)^\? `) // the ? glyph is gone (e3sevr, 41hhgn): none may show
	// jhf: the design question (design_for) is theirs; nothing else.
	m.opts.User = "jhf"
	m.refresh("") // sections are found per load: who "you" is is read then
	view := ansi.Strip(viewAt(m, "overview", 140))
	// What asks you is in ASKED at the top, its count the header's
	// (41hhgn check 10); WAITING keeps the rest.
	rows := strings.Split(section(view, "ASKED"), "\n")
	if !strings.Contains(view, "? 1 for you") || !strings.HasPrefix(rows[0], "ASKED · for you  [1]") || strings.Count(section(view, "ASKED"), "feature") != 1 {
		t.Fatalf("header lacks ? 1 for you, or ASKED is not 'ASKED · for you  [1]' with one row:\n%s", view)
	}
	if !regexp.MustCompile(`eeeeee   Tags listed per board +design +◑ 1/2 feature high`).MatchString(rows[1]) || len(marked.FindAllString(view, -1)) != 0 { // 911j70: order B
		t.Fatalf("the design question for jhf is not in ASKED, or a row still carries ?:\n%s", view)
	}
	if strings.Contains(section(view, "NEXT UP"), "eeeeee") || strings.Contains(section(view, "WAITING"), "eeeeee") {
		t.Fatalf("the design question is still in NEXT UP or WAITING:\n%s", view)
	}
	if focused := focusAt(t, m, 140, "eeeeee"); !strings.Contains(focused, "└─ question for you: should type leave the rows") {
		t.Fatalf("focused, jhf's question does not say you:\n%s", focused)
	}
	// Jørgen: the review he is doing.
	m.opts.User = "Jørgen"
	m.refresh("")
	view = ansi.Strip(viewAt(m, "overview", 140))
	rows = strings.Split(section(view, "ASKED"), "\n")
	if !strings.Contains(view, "? 1 for you") || !regexp.MustCompile(`bbbbbb   Release, then hug setup in seid-lang +review +◑ 1/2 feature high`).MatchString(rows[1]) || len(marked.FindAllString(view, -1)) != 0 {
		t.Fatalf("Jørgen's review is not in ASKED, or a row still carries ?:\n%s", view)
	}
	focused := focusAt(t, m, 140, "bbbbbb")
	if note := regexp.MustCompile(`(?m)^\s+└─ in review by you\s+since 09:48\s*$`); !note.MatchString(focused) {
		t.Fatalf("focused, Jørgen's review does not say you since 09:48:\n%s", focused)
	}
	// Someone with nothing waiting: no count, no mark.
	m.opts.User = "carol"
	m.refresh("")
	view = ansi.Strip(viewAt(m, "overview", 140))
	if strings.Contains(view, "? 1 for you") || strings.Contains(view, "ASKED · for you  [1]") || marked.MatchString(view) {
		t.Fatalf("carol sees something for her:\n%s", view)
	}
	// Waiting on a person: --on carol writes waiting_for. cccccc waits already;
	// a new wait replaces what it waits on. (Resuming it first would make
	// nautilus's second active ticket next to aaaaaa: n2hq8p refuses that.)
	s := m.store
	if _, e := s.Resume("cccccc"); e == nil {
		t.Fatal("resume gave nautilus two active tickets")
	}
	if _, e := s.Ask("cccccc", task.Question{What: "approve the look", For: "carol"}, "nautilus"); e != nil {
		t.Fatal(e)
	}
	m.refresh("")
	view = ansi.Strip(viewAt(m, "overview", 140))
	rows = strings.Split(section(view, "ASKED"), "\n")
	if !strings.Contains(view, "? 1 for you") || !regexp.MustCompile(`cccccc   Warm cache under 50 ms +asked +◑ 1/2 feature high`).MatchString(rows[1]) {
		t.Fatalf("carol's question is not in ASKED:\n%s", view)
	}
	if focused := focusAt(t, m, 140, "cccccc"); !regexp.MustCompile(`(?m)^> .*cccccc.*\n\s+└─ asked: approve the look · nautilus\s+since \d\d:\d\d`).MatchString(focused) {
		t.Fatalf("carol's question, focused:\n%s", focused)
	}
	// Anyone else sees it in WAITING with her name, never initials typed
	// in the text.
	m.opts.User = "dave"
	m.refresh("")
	if view = focusAt(t, m, 140, "cccccc"); !strings.Contains(section(view, "WAITING"), "cccccc") || !strings.Contains(view, "└─ asked carol: approve the look · nautilus") {
		t.Fatalf("dave's view of carol's question:\n%s", view)
	}
}

// 89y0b7 (Jørgen 15:31): a person is found by git email in hug.toml
// [people] and named by the short name; with no entry, by the email
// before the @. An agent (Options.User set) is always its session name,
// whatever git email it commits with.
func TestYouByGitEmail(t *testing.T) {
	m := stageBoard(t)
	if e := os.WriteFile(filepath.Join(m.store.Root, "hug.toml"), []byte("[people.jhf]\nname = \"Jørgen H. Fjeld\"\nemail = [\"jorgen@veridit.no\", \"12345+jhf@users.noreply.github.com\"]\n"), 0o644); e != nil {
		t.Fatal(e)
	}
	for _, email := range []string{"jorgen@veridit.no", "12345+JHF@users.noreply.github.com"} {
		m.opts.User, m.opts.Email = "", email
		m.refresh("")
		if m.me() != "jhf" {
			t.Fatalf("%s: me() = %q, want jhf", email, m.me())
		}
		view := focusAt(t, m, 140, "eeeeee")
		if !strings.Contains(view, "? 1 for you") || !strings.Contains(view, "└─ question for you: should type") {
			t.Fatalf("%s: jhf's design question is not his:\n%s", email, view)
		}
	}
	// No entry: the email before the @.
	m.opts.Email = "carol@example.org"
	if m.refresh(""); m.me() != "carol" {
		t.Fatalf("unlisted email: me() = %q", m.me())
	}
	// An agent committing with Jørgen's email is the agent.
	m.opts.User, m.opts.Email = "nautilus", "jorgen@veridit.no"
	m.refresh("")
	if view := focusAt(t, m, 140, "eeeeee"); m.me() != "nautilus" || strings.Contains(view, "? 1 for you") || strings.Contains(view, "question for you") || !strings.Contains(view, "└─ question for jhf:") {
		t.Fatalf("an agent with jhf's email took his question (me %q):\n%s", m.me(), view)
	}
}
