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
	old := now
	now = func() time.Time { return time.Date(2026, 11, 17, 9, 53, 0, 0, time.Local) }
	t.Cleanup(func() { now = old })
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

// 6ephqk check 5: rows have no state column and read pie, id, type,
// priority, stage, text; a stage with extra data shows "stage: reason" in
// place of the title, a bare stage the title, depends a bar of
// dependencies done then the title; one line per row (goldens at 80 and
// 140).
func TestRowsShowStage(t *testing.T) {
	m := stageBoard(t)
	view := ansi.Strip(viewAt(m, "overview", 140))
	// 89y0b7: line 1 is the task with its title; what it waits on goes on
	// line 2, under the id, the reason right after the colon.
	for _, want := range []string{
		`aaaaaa   feature high     active    Status colors: rows and sections share one subtle scheme`,
		`bbbbbb   feature high     review    Release, then hug setup in seid-lang`,
		"└─ in review by Jørgen",
		`cccccc   feature high     waiting   Warm cache under 50 ms`,
		"└─ waiting on CI run 44063552584",
		`dddddd   feature high     depends   One set of variable names`,
		"└─ depends on cccccc, eeeeee  ▱▱ 0/2",
		`eeeeee   feature high     design    Tags listed per board`,
		"└─ question for jhf: should type leave the rows, or stay as a column?",
		`ffffff   feature high     ready     Depend on a group or epic`,
		`gggggg   feature high     deferred  Answer a question in place`,
		"└─ deferred: after the release",
	} {
		if !strings.Contains(view, want) {
			t.Errorf("overview lacks %q", want)
		}
	}
	// Active and ready rows stay one line.
	for _, id := range []string{"aaaaaa", "ffffff"} {
		lines := strings.Split(view, "\n")
		for i, l := range lines {
			if strings.Contains(l, id) && i+1 < len(lines) && strings.Contains(lines[i+1], "└─") {
				t.Errorf("%s has a second line: %q", id, lines[i+1])
			}
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
		for _, v := range []string{"overview", "list"} {
			screen := ansi.Strip(viewAt(m, v, w))
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

// 6ephqk check 8, 89y0b7: a task waiting on the TUI user says "you" on
// its second line, is marked ? and sorts to the top of its section, and
// the header says "? N for you" (Jørgen 11:42, crab 12:38 and 15:28:
// review_by, design_for, or waiting_for).
func TestWaitingOnYou(t *testing.T) {
	m := stageBoard(t)
	// jhf: the design question (design_for) is theirs; nothing else.
	m.opts.User = "jhf"
	view := ansi.Strip(viewAt(m, "overview", 140))
	if !strings.Contains(view, "? 1 for you") {
		t.Fatalf("header lacks ? 1 for you:\n%s", view)
	}
	next := view[strings.Index(view, "NEXT UP"):strings.Index(view, "RECENTLY DONE")]
	rows := strings.Split(next, "\n")[1:]
	if !strings.HasPrefix(rows[0], "? ") || !strings.Contains(rows[0], "eeeeee   feature high     design    Tags listed per board") ||
		!strings.Contains(rows[1], "└─ question for you: should type leave the rows") {
		t.Fatalf("the design question for jhf is not first in NEXT UP, marked ?:\n%s", next)
	}
	// Jørgen: the review he is doing.
	m.opts.User = "Jørgen"
	m.refresh("")
	view = ansi.Strip(viewAt(m, "overview", 140))
	doing := view[strings.Index(view, "IN PROGRESS"):strings.Index(view, "NEXT UP")]
	rows = strings.Split(doing, "\n")[1:]
	if !strings.Contains(view, "? 1 for you") || !strings.HasPrefix(rows[0], "? ") || !strings.Contains(rows[0], "bbbbbb   feature high     review    Release, then hug setup") ||
		!strings.Contains(rows[1], "└─ in review by you") || !strings.HasSuffix(strings.TrimRight(rows[1], " "), "since 09:48") {
		t.Fatalf("Jørgen's review is not first in IN PROGRESS, marked ?:\n%s", view)
	}
	// Someone with nothing waiting: no count, no mark.
	m.opts.User = "carol"
	m.refresh("")
	view = ansi.Strip(viewAt(m, "overview", 140))
	if strings.Contains(view, "for you") || regexp.MustCompile(`(?m)^\? `).MatchString(view) {
		t.Fatalf("carol sees something for her:\n%s", view)
	}
	// Waiting on a person: --on carol writes waiting_for. cccccc waits already;
	// a new wait replaces what it waits on. (Resuming it first would make
	// nautilus's second active ticket next to aaaaaa: n2hq8p refuses that.)
	s := m.store
	if _, e := s.Resume("cccccc"); e == nil {
		t.Fatal("resume gave nautilus two active tickets")
	}
	if _, e := s.Wait("cccccc", "approve the look", "nautilus", "carol"); e != nil {
		t.Fatal(e)
	}
	m.refresh("")
	view = ansi.Strip(viewAt(m, "overview", 140))
	if !strings.Contains(view, "? 1 for you") || !regexp.MustCompile(`(?m)^\? .*cccccc   feature high     waiting   Warm cache under 50 ms.*\n\s+└─ waiting on you: approve the look\s+since \d\d:\d\d`).MatchString(view) {
		t.Fatalf("carol's wait:\n%s", view)
	}
	// Anyone else sees her name, never initials typed in the text.
	m.opts.User = "dave"
	m.refresh("")
	if view = ansi.Strip(viewAt(m, "overview", 140)); !strings.Contains(view, "└─ waiting on carol: approve the look") {
		t.Fatalf("dave's view of carol's wait:\n%s", view)
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
		view := ansi.Strip(viewAt(m, "overview", 140))
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
	if view := ansi.Strip(viewAt(m, "overview", 140)); m.me() != "nautilus" || strings.Contains(view, "for you") || !strings.Contains(view, "└─ question for jhf:") {
		t.Fatalf("an agent with jhf's email took his question (me %q):\n%s", m.me(), view)
	}
}
