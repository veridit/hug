package tui

import (
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/veridit/hug/huginn/internal/task"
)

// paletteBoard is 05r3pg's case: Alpha (todo, two checks, the one opened)
// and Beta, a second row for the list's subject.
func paletteBoard(t *testing.T) (model, map[string]string) {
	t.Helper()
	s := &task.Store{Root: t.TempDir()}
	ids := map[string]string{}
	for _, title := range []string{"Alpha", "Beta"} {
		r, err := s.New(task.NewOptions{Title: title, Goal: "Goal of " + title, Acceptance: []string{"first check of " + title, "second check of " + title}})
		if err != nil {
			t.Fatal(err)
		}
		ids[title] = r.Task.ID
	}
	m := newModel(s, Options{User: "tester"})
	m.width, m.height = 80, 24
	m.resize()
	m.setView("list")
	return m, ids
}

// lineOf is the first screen line holding s, or -1.
func lineOf(lines []string, s string) int {
	for i, l := range lines {
		if strings.Contains(l, s) {
			return i
		}
	}
	return -1
}

// inputLine is the palette's input, the line that starts with ": ", or -1.
func inputLine(lines []string) int {
	for i, l := range lines {
		if strings.HasPrefix(l, ": ") {
			return i
		}
	}
	return -1
}

// 05r3pg check 1: with a task open, : keeps the task drawn above and puts
// the palette in the bottom part of the screen; the band stays the task's.
func TestPaletteKeepsTheTaskAbove(t *testing.T) {
	m, ids := paletteBoard(t)
	m = openID(t, m, ids["Alpha"])
	m = press(m, ":")
	lines := strings.Split(screenOf(m), "\n")
	if len(lines) != 24 {
		t.Fatalf("screen has %d lines, want 24", len(lines))
	}
	if !strings.Contains(lines[0], "HUG ·") || !strings.Contains(lines[0], ids["Alpha"]) {
		t.Fatalf("the task's breadcrumb is gone with the palette open:\n%s", strings.Join(lines, "\n"))
	}
	if at := lineOf(lines, "Alpha"); at < 1 || at > 8 {
		t.Fatalf("the task's title is not drawn above the palette (line %d):\n%s", at, strings.Join(lines, "\n"))
	}
	in := -1
	for i, l := range lines {
		if strings.HasPrefix(l, ": ") {
			in = i
		}
	}
	if in < 24/2 {
		t.Fatalf("the palette's input is on line %d, want the bottom part (>= 12):\n%s", in, strings.Join(lines, "\n"))
	}
	if strings.Contains(lines[0], "palette") {
		t.Fatalf("the band names the palette, not what it acts on:\n%s", lines[0])
	}
}

// 05r3pg check 2: the first group is headed by the open task's id and
// lists its actions; edit acts on that task without asking which. In a
// list the subject is the selected row.
func TestPaletteContextEditsThisTask(t *testing.T) {
	m, ids := paletteBoard(t)
	m = openID(t, m, ids["Alpha"])
	m = press(m, ":")
	lines := strings.Split(screenOf(m), "\n")
	in := inputLine(lines)
	head := lineOf(lines, "TASK "+ids["Alpha"])
	if head < 0 || head < in {
		t.Fatalf("no TASK %s heading under the input:\n%s", ids["Alpha"], strings.Join(lines, "\n"))
	}
	for _, l := range lines[in+1 : head] {
		if strings.TrimSpace(l) != "" {
			t.Fatalf("a row comes before the TASK heading: %q", l)
		}
	}
	if !regexp.MustCompile(`\be +edit\b`).MatchString(strings.Join(lines[head:], "\n")) {
		t.Fatalf("the TASK group lacks 'e edit':\n%s", strings.Join(lines, "\n"))
	}
	m = press(m, "edit", "enter")
	if m.confirm != "e" || m.prompt != "" {
		t.Fatalf("edit did not ask to edit: confirm %q prompt %q message %q", m.confirm, m.prompt, m.message)
	}
	if tk, _ := m.taskAt(m.confirmPath); tk.ID != ids["Alpha"] {
		t.Fatalf("edit asks about %s, want %s", tk.ID, ids["Alpha"])
	}
	if !strings.Contains(screenOf(m), "Edit "+ids["Alpha"]+" in") {
		t.Fatalf("the confirm does not name the task:\n%s", screenOf(m))
	}
	// A list: the subject is the selected row, Beta.
	l, lids := paletteBoard(t)
	l = selectRow(t, l, lids["Beta"])
	l = press(l, ":")
	if !strings.Contains(screenOf(l), "TASK "+lids["Beta"]) {
		t.Fatalf("the list's palette is not headed by the selected row:\n%s", screenOf(l))
	}
	l = press(l, "edit", "enter")
	if tk, _ := l.taskAt(l.confirmPath); l.confirm != "e" || tk.ID != lids["Beta"] {
		t.Fatalf("edit in a list: confirm %q on %s, want e on %s", l.confirm, tk.ID, lids["Beta"])
	}
}

// 05r3pg check 3: with a check focused, tick/untick and promote for that
// check are offered, under its own heading, and act on that check.
func TestPaletteOffersTheFocusedCheck(t *testing.T) {
	m, ids := paletteBoard(t)
	m = openID(t, m, ids["Alpha"])
	if m.detailPane != focusAcceptance {
		t.Fatalf("the checks are not focused on open (pane %d)", m.detailPane)
	}
	m = press(m, "j", ":")
	text := screenOf(m)
	for _, want := range []string{"CHECK 2", "tick", "promote", "strike"} {
		if !strings.Contains(text, want) {
			t.Fatalf("the palette lacks %q with check 2 focused:\n%s", want, text)
		}
	}
	if !regexp.MustCompile(`Space +tick`).MatchString(text) {
		t.Fatalf("tick does not show its key Space:\n%s", text)
	}
	m = press(m, "tick", "enter")
	if m.confirm != " " || m.acceptance != 1 || !strings.Contains(screenOf(m), "Tick: ☐ 2. second check of Alpha") {
		t.Fatalf("tick did not ask to tick check 2: confirm %q check %d\n%s", m.confirm, m.acceptance+1, screenOf(m))
	}
	// Without a focused check there is no CHECK group.
	m = press(m, "esc", "tab")
	for m.detailPane == focusAcceptance {
		m = press(m, "tab")
	}
	if text := screenOf(press(m, ":")); strings.Contains(text, "CHECK ") {
		t.Fatalf("a CHECK group with no check focused:\n%s", text)
	}
}

// 05r3pg check 4: the global commands follow under their own heading, as
// a tree: filter is one branch, views are one branch.
func TestPaletteGlobalTreeFollows(t *testing.T) {
	m, ids := paletteBoard(t)
	m.height = 60
	m.resize()
	m = openID(t, m, ids["Alpha"])
	m = press(m, ":")
	lines := strings.Split(screenOf(m), "\n")
	task, global := lineOf(lines, "TASK "+ids["Alpha"]), lineOf(lines, "GLOBAL")
	if task < 0 || global < task {
		t.Fatalf("GLOBAL does not follow TASK (%d, %d):\n%s", task, global, strings.Join(lines, "\n"))
	}
	below := strings.Join(lines[global:], "\n")
	for _, want := range []string{"filter ›", "view ›", "sort", "goto ›", "clear"} {
		if !strings.Contains(below, want) {
			t.Fatalf("GLOBAL lacks %q:\n%s", want, below)
		}
	}
	if strings.Count(below, "filter") != 1 || strings.Contains(below, "Filter todo") {
		t.Fatalf("filter is not one branch:\n%s", below)
	}
	m = press(m, "view ")
	text := screenOf(m)
	for _, want := range []string{"overview", "list", "board", "lint"} {
		if !strings.Contains(text, "view "+want) {
			t.Fatalf("the view branch lacks %s:\n%s", want, text)
		}
	}
}

// 05r3pg check 5: rows show their direct key, typing filters across both
// groups, and the footer describes the palette, not the view under it.
func TestPaletteRowsShowKeysAndFooter(t *testing.T) {
	m, ids := paletteBoard(t)
	m.width, m.height = 140, 60
	m.resize()
	m = openID(t, m, ids["Alpha"])
	m = press(m, ":")
	text := screenOf(m)
	for _, want := range []string{`t +take`, `x +drop`, `e +edit`, `r +request`, `v +view ›`, `f +filter ›`, `s +sort`, `# +goto ›`} {
		if !regexp.MustCompile(want).MatchString(text) {
			t.Fatalf("no row %q:\n%s", want, text)
		}
	}
	lines := strings.Split(text, "\n")
	footer := lines[len(lines)-3]
	if !strings.Contains(footer, "Palette") || strings.Contains(footer, "Esc back") || strings.Contains(footer, "Detail") {
		t.Fatalf("the footer does not describe the palette: %q", footer)
	}
	m = press(m, "s")
	text = screenOf(m)
	if !strings.Contains(text, "TASK "+ids["Alpha"]) || !strings.Contains(text, "GLOBAL") || !strings.Contains(text, "set ›") || !strings.Contains(text, "sort") {
		t.Fatalf("typing s does not filter both groups:\n%s", text)
	}
	if strings.Contains(text, "edit") || strings.Contains(text, "filter ›") {
		t.Fatalf("typing s keeps rows that do not start with s:\n%s", text)
	}
}

// Esc backs out one level, then closes (05r3pg principle).
func TestPaletteEscBacksOutOneLevel(t *testing.T) {
	m, ids := paletteBoard(t)
	m = openID(t, m, ids["Alpha"])
	m = press(m, ":", "set type ")
	for _, want := range []string{"set ", ""} {
		m = press(m, "esc")
		if m.prompt != ":" || m.input.Value() != want {
			t.Fatalf("Esc: prompt %q input %q, want the palette at %q", m.prompt, m.input.Value(), want)
		}
	}
	m = press(m, "esc")
	if m.prompt != "" || !m.detail {
		t.Fatalf("the last Esc did not close the palette only: prompt %q detail %v", m.prompt, m.detail)
	}
}

// 05r3pg check 6: goldens of the palette in a task and in a list, at 80x24
// and 140x40 (reviewed).
func TestPaletteGolden(t *testing.T) {
	m, ids := paletteBoard(t)
	var out []string
	for _, size := range [][2]int{{80, 24}, {140, 40}} {
		g := m
		g.width, g.height = size[0], size[1]
		g.resize()
		d := openID(t, g, ids["Alpha"])
		d = press(d, "j", ":")
		out = append(out, "== task, check 2 focused, "+strconv.Itoa(size[0])+"x"+strconv.Itoa(size[1]), screenOf(d))
		l := selectRow(t, g, ids["Beta"])
		l = press(l, ":")
		out = append(out, "== list, Beta selected, "+strconv.Itoa(size[0])+"x"+strconv.Itoa(size[1]), screenOf(l))
	}
	text := strings.Join(out, "\n")
	text = strings.NewReplacer(idsReplacer(ids)...).Replace(text)
	text = regexp.MustCompile(`\d{4}-\d\d-\d\d(T\d\d:\d\d)?`).ReplaceAllStringFunc(text, func(d string) string { return strings.Repeat("#", len(d)) })
	goldenFile(t, filepath.Join("testdata", "palette", "context.txt"), text)
}
