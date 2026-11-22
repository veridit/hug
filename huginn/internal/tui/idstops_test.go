package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/veridit/hug/huginn/internal/task"
)

// idBoard is jr55s8's case (4bwcgp, Jørgen 19:43): Main is under the
// epic Epic, waits "on after <Blocker> (crab 17:52)", depends on Dep,
// names task:Linked in its goal and Noted bare in a note, with a word
// that is no task (zzzzzz) in both. Epic has Main and Other as children.
func idBoard(t *testing.T) (model, map[string]string) {
	t.Helper()
	s := &task.Store{Root: t.TempDir()}
	ids := map[string]string{}
	add := func(title string, o task.NewOptions) {
		o.Title, o.Acceptance = title, []string{"a"}
		if o.Goal == "" {
			o.Goal = "g"
		}
		r, err := s.New(o)
		if err != nil {
			t.Fatal(err)
		}
		ids[title] = r.Task.ID
	}
	for _, title := range []string{"Epic", "Other", "Blocker", "Dep", "Linked", "Noted"} {
		add(title, task.NewOptions{})
	}
	add("Main", task.NewOptions{Goal: "Builds on task:" + ids["Linked"] + " and task:zzzzzz.", Deps: []string{ids["Dep"]}, Top: true})
	for _, child := range []string{"Main", "Other"} {
		if _, err := s.Set(ids[child], "parent", ids["Epic"]); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := s.Note(ids["Main"], "waits on "+ids["Noted"]+" and zzzzzz first"); err != nil {
		t.Fatal(err)
	}
	// Dep is done, so Main may be taken and wait.
	d, _ := s.Get(ids["Dep"])
	raw := strings.Replace(string(d.Bytes()), "- [ ] a", "- [x] a", 1)
	if err := task.AtomicWrite(filepath.Join(s.Root, d.Task.Path), []byte(raw)); err != nil {
		t.Fatal(err)
	}
	if _, err := s.TakeAs(ids["Dep"], "kestrel", "m"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Done(ids["Dep"]); err != nil {
		t.Fatal(err)
	}
	if _, err := s.TakeAs(ids["Main"], "wyvern", "m"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Wait(ids["Main"], "after "+ids["Blocker"]+" (crab 17:52)", "wyvern"); err != nil {
		t.Fatal(err)
	}
	m := newModel(s, Options{User: "tester"})
	m.width, m.height = 120, 40
	m.resize()
	return m, ids
}

func openID(t *testing.T, m model, id string) model {
	t.Helper()
	m.setView("list")
	m = selectRow(t, m, id)
	m = press(m, "enter")
	if tk, _ := m.selected(); tk.ID != id || !m.detail {
		t.Fatalf("Enter opened %s (detail %v), want %s", tk.ID, m.detail, id)
	}
	return m
}

// screenOf is the whole screen, plain.
func screenOf(m model) string { return ansi.Strip(m.View()) }

// 4bwcgp check 1 (e3sevr: one ring): in a detail whose waiting_on names a
// task id, Shift-Tab (h) from the page reaches that id, Enter opens it, Esc
// returns to the first detail (golden).
func TestTabReachesWaitedOnID(t *testing.T) {
	m, ids := idBoard(t)
	m = openID(t, m, ids["Main"])
	m = press(m, "shift+tab", "shift+tab") // checks -> page -> the stage's id
	screen := screenOf(m)
	if !strings.Contains(screen, "└─ waiting on after ▸"+ids["Blocker"]+" (crab 17:52)") {
		t.Fatalf("Shift-Tab from the page does not mark the id it waits on:\n%s", screen)
	}
	if !strings.Contains(screen, "h/l Tab next") || !strings.Contains(screen, "Enter open") {
		t.Fatalf("key bar lacks h/l Tab next, Enter open:\n%s", screen)
	}
	var golden []string
	for _, w := range []int{80, 140} {
		g := m
		g.width = w
		g.resize()
		g.syncDetail()
		golden = append(golden, "== waiting-on id marked at "+strconv.Itoa(w), screenOf(g))
	}
	m = press(m, "enter")
	if tk, _ := m.selected(); tk.ID != ids["Blocker"] {
		t.Fatalf("Enter opened %s, want %s", tk.ID, ids["Blocker"])
	}
	m = press(m, "esc")
	if tk, _ := m.selected(); tk.ID != ids["Main"] || !m.detail {
		t.Fatalf("Esc: open %s detail %v, want %s", tk.ID, m.detail, ids["Main"])
	}
	text := strings.Join(golden, "\n")
	text = strings.NewReplacer(idsReplacer(ids)...).Replace(text)
	// The store stamps with the real clock: dates and times are masked.
	text = regexp.MustCompile(`\d{4}-\d\d-\d\d(T\d\d:\d\d)?`).ReplaceAllStringFunc(text, func(d string) string { return strings.Repeat("#", len(d)) })
	text = regexp.MustCompile(`since \d\d:\d\d`).ReplaceAllString(text, "since ##:##")
	path := filepath.Join("testdata", "detail", "tab-ids.txt")
	if os.Getenv("HUG_UPDATE_GOLDEN") == "1" {
		_ = os.MkdirAll(filepath.Dir(path), 0o755)
		if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("%v\nfix: HUG_UPDATE_GOLDEN=1 go test -run TestTabReachesWaitedOnID ./huginn/internal/tui/", err)
	}
	if text != strings.ReplaceAll(string(want), "\r\n", "\n") {
		t.Fatalf("differs from %s:\n%s", path, text)
	}
}

// idsReplacer names the random ids by title, so the golden is stable.
func idsReplacer(ids map[string]string) []string {
	var out []string
	for title, id := range ids {
		name := strings.ToUpper(title + "______")[:6] // as wide as an id
		out = append(out, id, name)
	}
	return out
}

// 4bwcgp check 2, e3sevr check 6: parent, the stage's id, the page, the
// checks, dependencies, links and notes are one ring, in reading order,
// around the end; Tab and l go forward, Shift-Tab and h back; an id that
// resolves to no task is not a stop. An epic's children are stops too.
// The Notes are a long section, folded (8a0ybb): its heading is a stop,
// and the id in it one once Enter has opened it; here they are open.
func TestTabCyclesEveryIDInReadingOrder(t *testing.T) {
	m, ids := idBoard(t)
	m = openID(t, m, ids["Main"])
	m = m.withUnfolded("notes")
	m.syncDetail()
	tk, _ := m.selected()
	var got []string
	for _, s := range m.idStops(tk) {
		got = append(got, s.where+":"+s.target.ID)
	}
	want := []string{"parent:" + ids["Epic"], "stage:" + ids["Blocker"], "deps:" + ids["Dep"], "links:" + ids["Linked"], "notes:" + ids["Noted"]}
	if strings.Join(got, " ") != strings.Join(want, " ") {
		t.Fatalf("stops %v, want %v (zzzzzz is no task, so no stop)", got, want)
	}
	// The ring as h/l and Tab see it: the header's ids, the page, the
	// checks, then the ids in the body.
	var ring []string
	for _, s := range m.focusRing(tk) {
		ring = append(ring, fmt.Sprint(s.pane))
	}
	if want := fmt.Sprint(focusIDs, focusIDs, focusPage, focusAcceptance, focusDeps, focusLinks, focusFold, focusIDs, focusFold); strings.Join(ring, " ") != want {
		t.Fatalf("ring %v, want %s", ring, want)
	}
	// From the checks (where Main opens) Tab visits the dependency, the
	// link, the NOTES heading, the note's id, the HISTORY heading, then
	// wraps to the epic and the stage's id.
	marks := []string{"> ↳ " + ids["Dep"], "> ↳ " + ids["Linked"], "waits on ▸" + ids["Noted"] + " and zzzzzz", "epic:▸" + ids["Epic"], "waiting on after ▸" + ids["Blocker"]}
	for i, mark := range marks {
		m = press(m, "tab")
		if i == 2 {
			if !strings.Contains(detailPlain(m), "▸ NOTES (1) · Enter folds") {
				t.Fatalf("Tab 3: not on the open NOTES heading:\n%s", detailPlain(m))
			}
			m = press(m, "tab")
		}
		if i == 3 {
			if !strings.Contains(detailPlain(m), "▸ HISTORY (") {
				t.Fatalf("Tab 5: not on the HISTORY heading:\n%s", detailPlain(m))
			}
			m = press(m, "tab")
		}
		m.height = 60 // the whole detail on screen
		m.resize()
		m.syncDetail()
		screen := screenOf(m)
		if !strings.Contains(screen, mark) || strings.Count(screen, "▸")-strings.Count(screen, "▸ ") > 1 {
			t.Fatalf("Tab %d: no single mark %q:\n%s", i+1, mark, screen)
		}
	}
	for i, mark := range []string{marks[3], marks[2], marks[1]} {
		m = press(m, []string{"shift+tab", "h"}[i%2])
		if i > 0 {
			m = press(m, "shift+tab") // past the HISTORY, then the NOTES, heading
		}
		if screen := screenOf(m) + "\n" + detailPlain(m); !strings.Contains(screen, mark) {
			t.Fatalf("Shift-Tab %d: no mark %q:\n%s", i+1, mark, screen)
		}
	}
	// Enter on each kind opens it; Esc comes back. A page opens with the
	// Notes folded: l reaches NOTES, Enter opens it, l its id.
	for i, c := range []struct {
		wantID string
		keys   []string
	}{
		{ids["Dep"], []string{"l"}},
		{ids["Linked"], []string{"l", "l"}},
		{ids["Noted"], []string{"l", "l", "l", "enter", "l"}},
		{ids["Epic"], []string{"l", "l", "l", "l", "l"}},
		{ids["Blocker"], []string{"l", "l", "l", "l", "l", "l"}},
	} {
		wantID := c.wantID
		m = openID(t, press(m, "esc", "esc"), ids["Main"])
		m = press(m, c.keys...)
		m = press(m, "enter")
		if tk, _ := m.selected(); tk.ID != wantID {
			t.Fatalf("stop %d: Enter opened %s, want %s", i+1, tk.ID, wantID)
		}
		if m = press(m, "esc"); func() bool { tk, _ := m.selected(); return tk.ID != ids["Main"] }() {
			t.Fatalf("stop %d: Esc did not come back", i+1)
		}
	}
	// The epic lists its children; Tab reaches each.
	m = openID(t, press(m, "esc", "esc"), ids["Epic"])
	if text := detailPlain(m); !strings.Contains(text, "CHILDREN (2)") || !strings.Contains(text, ids["Main"]+" Main") || !strings.Contains(text, ids["Other"]+" Other") {
		t.Fatalf("epic detail lacks its children:\n%s", text)
	}
	m = press(m, "tab") // from the checks: the first child
	first := detailPlain(m)
	m = press(m, "tab")
	second := detailPlain(m)
	if !strings.Contains(first, "> ↳ "+ids["Main"]) || !strings.Contains(second, "> ↳ "+ids["Other"]) {
		t.Fatalf("Tab does not step the children:\n%s\n---\n%s", first, second)
	}
	m = press(m, "enter")
	if tk, _ := m.selected(); tk.ID != ids["Other"] {
		t.Fatalf("Enter on a child opened %s", tk.ID)
	}
	m = openID(t, press(m, "esc", "esc"), ids["Other"])
	tk, _ = m.selected()
	if len(m.idStops(tk)) != 1 { // its epic
		t.Fatalf("Other: stops %v", m.idStops(tk))
	}
	// A task with no ids still has its page and checks on the ring.
	m = openID(t, press(m, "esc"), ids["Linked"])
	if before := m.detailPane; press(m, "tab").detailPane == before {
		t.Fatalf("Tab with no ids did not move: %q", m.message)
	}
}
