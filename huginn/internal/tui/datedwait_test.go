package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/x/ansi"
	"github.com/veridit/hug/huginn/internal/task"
)

// datedBoard has three waits for 2026-11-24 (a wait --until, an ask
// --after, and an ask --after as v0.7.55 wrote it, 47sxn2's fields), a
// wait on CI and an active task; the clocks are 2026-11-20 13:05.
func datedBoard(t *testing.T) model {
	t.Helper()
	old, oldTask := now, task.Now
	now = func() time.Time { return time.Date(2026, 11, 20, 13, 5, 0, 0, time.Local) }
	task.Now = func() time.Time { return time.Date(2026, 11, 18, 5, 8, 0, 0, time.Local) }
	t.Cleanup(func() { now, task.Now = old, oldTask })
	s := &task.Store{Root: t.TempDir()}
	write := func(id, title, front string) {
		body := "---\nid: \"" + id + "\"\ntitle: \"" + title + "\"\n" + front + "\ntype: \"feature\"\npriority: \"high\"\nlabels: []\ncreated: \"2026-11-01\"\nupdated: \"2026-11-01\"\n---\n\n## Goal\ng\n\n## Purpose\np\n\n## Principle\nq\n\n## Acceptance\n- [x] a\n- [ ] b\n"
		if e := os.WriteFile(filepath.Join(s.Root, "0"+id[:1]+"."+id+".t.md"), []byte(body), 0o644); e != nil {
			t.Fatal(e)
		}
	}
	doing := "state: \"doing\"\nstage: \"active\"\ndoing_since: \"2026-11-17T08:00\""
	write("aaaaaa", "Look again after a week of use", doing+"\nassignee: [\"kestrel\"]")
	write("bbbbbb", "Ask after a week of use", doing+"\nassignee: [\"crab\"]")
	write("cccccc", "The board keeps itself true", "state: \"doing\"\nstage: \"waiting\"\nassignee: [\"crab\"]\ndoing_since: \"2026-11-17T18:10\"\nwaiting_since: \"2026-11-18T05:08\"\nwaiting_until: \"2026-11-24\"\nwaiting_ask_what: \"a week of use: has he had to remind an agent to update a ticket?\"\nwaiting_ask_for: \"jhf\"\nwaiting_asked_by: \"crab\"\nwaiting_ask_check: \"2\"\nwaiting_on: \"the date 2026-11-24, then ask jhf\"")
	write("dddddd", "Warm cache under 50 ms", "state: \"doing\"\nstage: \"waiting\"\nassignee: [\"nautilus\"]\ndoing_since: \"2026-11-17T08:00\"\nwaiting_on: \"CI run 44063552584\"\nwaiting_since: \"2026-11-20T12:41\"")
	write("eeeeee", "Status colors", doing+"\nassignee: [\"wyvern\"]")
	if _, e := s.WaitUntil("aaaaaa", "a week of use", "2026-11-24", "kestrel"); e != nil {
		t.Fatal(e)
	}
	if _, e := s.AskAfter("bbbbbb", task.Question{What: "has he had to remind an agent?", For: "jhf", Check: 2}, "2026-11-24", "crab"); e != nil {
		t.Fatal(e)
	}
	return newModel(s, Options{User: "tester"})
}

// atex3d (z2vmte check 5, Jørgen 2026-11-20 13:05): a wait for a date
// leads with the date in the row's STAGE cell ("til 11-24", crab 23:07Z:
// the cell keeps its 9 cells), the focused row's second line and the
// detail header: "waiting until 2026-11-24 · on WHY · since 18 Nov", or
// "· then ask NAME" for a held question, never "on the date" or a second
// "until". A wait with no date is as before.
func TestDatedWaitLeadsWithTheDate(t *testing.T) {
	m := datedBoard(t)
	m.width, m.height = 120, 30
	m.resize()
	m.setView("list")
	at := func(id string) model {
		t.Helper()
		for i, tk := range m.visible {
			if tk.ID == id {
				c := m
				c.cursor = i
				return c
			}
		}
		t.Fatalf("%s not listed", id)
		return m
	}
	row := func(v, id string) string {
		for _, l := range strings.Split(v, "\n") {
			if strings.Contains(l, id) {
				return l
			}
		}
		return ""
	}
	list := ansi.Strip(m.View())
	for _, id := range []string{"aaaaaa", "bbbbbb", "cccccc"} {
		if r := row(list, id); !strings.Contains(r, " til 11-24 ") {
			t.Errorf("%s row's stage cell is not 'til 11-24':\n%s", id, r)
		}
	}
	if r := row(list, "dddddd"); !strings.Contains(r, " waiting ") || strings.Contains(r, "til") {
		t.Errorf("a wait with no date keeps 'waiting':\n%s", r)
	}
	lines := map[string]string{
		"aaaaaa": "└─ waiting until 2026-11-24 · on a week of use",
		"bbbbbb": "└─ waiting until 2026-11-24 · then ask jhf",
		"cccccc": "└─ waiting until 2026-11-24 · then ask jhf",
		"dddddd": "└─ waiting on CI run 44063552584",
	}
	var shots []string
	for _, id := range []string{"aaaaaa", "bbbbbb", "cccccc", "dddddd"} {
		c := at(id)
		v := ansi.Strip(c.View())
		if !strings.Contains(v, lines[id]) {
			t.Errorf("%s focused row lacks %q:\n%s", id, lines[id], v)
		}
		d := press(c, "enter")
		dv := ansi.Strip(d.View())
		since := " · since 18 Nov"
		if id == "dddddd" {
			since = " · since 12:41"
		}
		if !strings.Contains(dv, lines[id]+since) {
			t.Errorf("%s detail header lacks %q:\n%s", id, lines[id]+since, dv)
		}
		for _, bad := range []string{"on the date", "· until"} {
			if strings.Contains(v, bad) || strings.Contains(dv, bad) {
				t.Errorf("%s says %q:\n%s\n%s", id, bad, v, dv)
			}
		}
		if id == "cccccc" {
			shots = append(shots, v, dv)
		}
	}
	golden(t, filepath.Join("testdata", "rows", "dated-wait-120.txt"), "TestDatedWaitLeadsWithTheDate", strings.Join(shots, "\n")+"\n")
}

// atex3d: only a wait says its date in the STAGE cell; a task in another
// stage with a waiting_until left over keeps its own word.
func TestOnlyAWaitSaysItsDateInTheCell(t *testing.T) {
	m := datedBoard(t)
	// The stage comes from the fields (task.Stage): waiting_on makes a
	// wait, review_by a review; waiting_until alone holds nothing.
	for _, c := range []struct {
		fields map[string]string
		want   string
	}{
		{map[string]string{"waiting_on": "a week of use"}, "til 11-24"},
		{map[string]string{}, "active"},
		{map[string]string{"review_by": "crab"}, "review"},
	} {
		c.fields["waiting_until"] = "2026-11-24"
		tk := task.Task{ID: "xxxxxx", State: "doing", Fields: c.fields}
		if got, _ := m.rowStage(tk); got != c.want {
			t.Errorf("%v: cell %q, want %q", c.fields, got, c.want)
		}
	}
}
