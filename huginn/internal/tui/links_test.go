package tui

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/veridit/hug/huginn/internal/task"
)

// linkModel opens a task whose description and notes name two other tasks
// as task:ID, one of them twice, plus one id that matches no task.
func linkModel(t *testing.T) (model, task.Task, task.Task, task.Task) {
	t.Helper()
	s := &task.Store{Root: t.TempDir()}
	alpha, err := s.New(task.NewOptions{Title: "Alpha", Goal: "g", Acceptance: []string{"a"}})
	if err != nil {
		t.Fatal(err)
	}
	beta, err := s.New(task.NewOptions{Title: "Beta", Goal: "g", Acceptance: []string{"b"}})
	if err != nil {
		t.Fatal(err)
	}
	main, err := s.New(task.NewOptions{Title: "Main", Goal: "Builds on task:" + alpha.Task.ID + " and Task:" + beta.Task.ID + ".", Acceptance: []string{"m"}, Top: true})
	if err != nil {
		t.Fatal(err)
	}
	d, _ := s.Get(main.Task.ID)
	b := strings.Replace(string(d.Bytes()), "## Notes\n", "## Notes\nsee task:"+alpha.Task.ID+" again, and task:zzzzzz\n", 1)
	if err := task.AtomicWrite(filepath.Join(s.Root, main.Task.Path), []byte(b)); err != nil {
		t.Fatal(err)
	}
	m := newModel(s, Options{User: "tester"})
	m.width, m.height = 100, 30
	m.resize()
	m.setView("list")
	m.move(0, true)
	m = press(m, "enter")
	if tk, _ := m.selected(); tk.Title != "Main" {
		t.Fatalf("opened %q, want Main", tk.Title)
	}
	return m, main.Task, alpha.Task, beta.Task
}

// xxvm6t check 4: a task:ID in the description or notes can be followed to
// that task, and Esc comes back.
func TestFollowTaskRefAndBack(t *testing.T) {
	m, main, alpha, beta := linkModel(t)
	links := m.links(main)
	if len(links) != 3 || links[0].ID != alpha.ID || links[1].ID != beta.ID || !links[2].Missing {
		t.Fatalf("links: %+v", links)
	}
	// Tab steps the task ids: from the checks, the first link (4bwcgp).
	m = press(m, "tab")
	if m.detailPane != focusLinks {
		t.Fatalf("Tab from checks: focus %d, want links", m.detailPane)
	}
	text := detailPlain(m)
	if !strings.Contains(text, "▸ LINKS") || !strings.Contains(text, "> ↳ "+alpha.ID+" Alpha") {
		t.Fatalf("links section not focused on Alpha:\n%s", text)
	}
	if !strings.Contains(ansi.Strip(m.View()), "Enter follow") {
		t.Fatalf("key bar lacks Enter follow:\n%s", ansi.Strip(m.View()))
	}
	// j steps to Beta; Enter opens it; Esc comes back to Main.
	m = press(m, "j", "enter")
	if tk, _ := m.selected(); tk.ID != beta.ID {
		t.Fatalf("Enter on the second link opened %s, want %s", tk.ID, beta.ID)
	}
	bar := ansi.Strip(strings.Split(m.View(), "\n")[0]) // the breadcrumb (e3sevr)
	if !strings.Contains(bar, main.ID+" › "+beta.ID) {
		t.Fatalf("trail after following a link: %q", bar)
	}
	m = press(m, "esc")
	if tk, _ := m.selected(); tk.ID != main.ID || !m.detail {
		t.Fatalf("Esc after following: open %s detail %v, want %s", tk.ID, m.detail, main.ID)
	}
	// A reference to no task says so and stays.
	m.detailPane, m.link = focusLinks, 2 // zzzzzz: no Tab stop (4bwcgp), j/k still reach it
	m = press(m, "enter")
	if tk, _ := m.selected(); tk.ID != main.ID || !strings.Contains(m.message, "zzzzzz") {
		t.Fatalf("missing link: open %s message %q", tk.ID, m.message)
	}
}

// xxvm6t check 1: no view shows an id in brackets; rows, the detail header
// and the trail show the bare id.
func TestIDsBare(t *testing.T) {
	m, main, alpha, _ := linkModel(t)
	t.Setenv("NO_COLOR", "1")
	m.styles = newTheme()
	m.detailPane = focusPage
	m.syncDetail()
	views := map[string]string{"detail": ansi.Strip(m.View())}
	for _, v := range []string{"overview", "list", "board"} {
		m.setView(v)
		views[v] = ansi.Strip(m.View())
	}
	for name, v := range views {
		for _, id := range []string{main.ID, alpha.ID} {
			if strings.Contains(v, "["+id+"]") || strings.Contains(v, "#"+id) {
				t.Errorf("%s shows %s in brackets or with a sigil:\n%s", name, id, v)
			}
			if !strings.Contains(v, id) {
				t.Errorf("%s does not show %s:\n%s", name, id, v)
			}
		}
	}
}
