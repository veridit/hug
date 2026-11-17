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
		body := "---\nid: \"" + id + "\"\ntitle: \"" + title + "\"\n" + front + "\ntype: \"feature\"\npriority: \"high\"\nlabels: []\nassignee: " + who + "\ncreated: \"2026-11-01\"\nupdated: \"2026-11-01\"\n---\n\n## Goal\ng\n"
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
	write("aaaaaa", "Status colors: rows and sections share one subtle scheme", "status: \"doing\"\nstage: \"active\"\ndoing_since: \"2026-11-17T08:00\"")
	write("bbbbbb", "Release, then hug setup in seid-lang", "status: \"doing\"\nstage: \"review\"\ndoing_since: \"2026-11-17T08:00\"\nreview_by: \"Jørgen\"\nreview_since: \"2026-11-17T09:48\"")
	write("cccccc", "Warm cache under 50 ms", "status: \"doing\"\nstage: \"waiting\"\ndoing_since: \"2026-11-17T08:00\"\nwaiting_on: \"CI run 44063552584\"\nwaiting_since: \"2026-11-17T09:41\"")
	write("dddddd", "One set of variable names", "status: \"doing\"\nstage: \"depends\"\ndoing_since: \"2026-11-17T08:00\"", "cccccc", "eeeeee")
	write("eeeeee", "Tags listed per board", "status: \"todo\"\nstage: \"design\"\ndesign_question: \"should type leave the rows, or stay as a column?\"")
	write("ffffff", "Depend on a group or epic", "status: \"todo\"\nstage: \"ready\"")
	write("gggggg", "Answer a question in place", "status: \"todo\"\nstage: \"deferred\"\ndeferred_reason: \"after the release\"")
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
	for _, want := range []string{
		`aaaaaa   feature high     active    Status colors: rows and sections share one subtle scheme`,
		`bbbbbb   feature high     review:   Jørgen reviewing · 5 min`,
		`cccccc   feature high     waiting:  CI run 44063552584 · 12 min`,
		`dddddd   feature high     depends   ▱▱ 0/2  One set of variable names`,
		`eeeeee   feature high     design:   should type leave the rows, or stay as a column?`,
		`ffffff   feature high     ready     Depend on a group or epic`,
		`gggggg   feature high     deferred: after the release`,
	} {
		if !strings.Contains(view, want) {
			t.Errorf("overview lacks %q", want)
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
