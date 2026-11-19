package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/muesli/termenv"
)

// sortLine is the plain line of the sort picker that names field (by its
// row text, "rank", "priority" ...).
func sortLine(t *testing.T, m model, field string) string {
	t.Helper()
	for _, l := range strings.Split(ansi.Strip(m.sortText()), "\n") {
		if len(l) > 5 && l[3:5] == "  " && strings.HasPrefix(l[5:], field) {
			return l
		}
	}
	t.Fatalf("no %q line in the sort picker:\n%s", field, ansi.Strip(m.sortText()))
	return ""
}

// highlighted is the row text the sort picker marks with "> ", "" for none.
func highlighted(m model) string {
	var rows []string
	for _, l := range strings.Split(ansi.Strip(m.sortText()), "\n") {
		if strings.HasPrefix(l, "> ") {
			rows = append(rows, strings.Fields(l[2:])[1])
		}
	}
	return strings.Join(rows, ",")
}

// 17w9r6 check 1: in the sort picker j/k (and ↑↓, ctrl-n/p) move a
// highlight over the rows, as in the filter picker, and Space or Enter on
// it does what the row's letter does; Enter with nothing new applies.
func TestSortPickerHighlightMoves(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	m := manyModel(t)
	s := press(m, "s")
	if got := highlighted(s); got != "rank" {
		t.Fatalf("on opening the highlight is on %q, want rank (the first row, as in the filter picker)", got)
	}
	for _, step := range []struct{ key, want string }{
		{"j", "current"}, {"j", "priority"}, {"ctrl+n", "type"}, {"down", "state"},
		{"k", "type"}, {"up", "priority"}, {"ctrl+p", "current"}, {"k", "rank"}, {"k", "title"}, {"j", "rank"},
	} {
		s = press(s, step.key)
		if got := highlighted(s); got != step.want {
			t.Fatalf("after %s the highlight is on %q, want %q", step.key, got, step.want)
		}
	}
	if s.prompt != "sort" {
		t.Fatalf("moving closed the picker: prompt %q", s.prompt)
	}
}

func TestSortPickerSpaceAndEnterPick(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	m := manyModel(t)
	// Space on a field adds it, Space again flips it, as its letter does.
	s := press(m, "s", "j", "j", " ")
	if got := sortLabel(s.sortDraft); got != "priority ↓" || s.prompt != "sort" {
		t.Fatalf("Space on priority: draft %q prompt %q", got, s.prompt)
	}
	if s = press(s, " "); sortLabel(s.sortDraft) != "priority ↑" {
		t.Fatalf("Space again did not flip: %q", sortLabel(s.sortDraft))
	}
	// The picked field reads its place and direction; the highlight is
	// "> " alone, the filter picker's cue.
	if got := sortLine(t, press(s, "j"), "priority"); !strings.HasPrefix(got, "  p  priority  1 ↑") {
		t.Fatalf("a picked field off the highlight reads %q", got)
	}
	// Enter on a field not in the order adds it and stays open.
	s = press(s, "j", "enter")
	if got := sortLabel(s.sortDraft); got != "priority ↑ · type" || s.prompt != "sort" {
		t.Fatalf("Enter on type: draft %q prompt %q", got, s.prompt)
	}
	// Enter with nothing new applies.
	applied := press(s, "enter")
	if applied.prompt != "" || sortLabel(applied.sort) != "priority ↑ · type" {
		t.Fatalf("Enter with nothing new: prompt %q sort %q", applied.prompt, sortLabel(applied.sort))
	}
	// Enter or Space on r is r: rank at once.
	for _, k := range []string{"enter", " "} {
		r := press(applied, "s", k)
		if r.prompt != "" || len(r.sort) != 0 {
			t.Fatalf("%q on rank: prompt %q sort %q", k, r.prompt, sortLabel(r.sort))
		}
	}
	// Letters still pick at once and take the highlight to their row, so
	// Enter after them applies, as before.
	l := press(m, "s", "n")
	if sortLabel(l.sortDraft) != "title" || highlighted(l) != "title" {
		t.Fatalf("n: draft %q highlight %q", sortLabel(l.sortDraft), highlighted(l))
	}
	if l = press(l, "enter"); l.prompt != "" || sortLabel(l.sort) != "title" {
		t.Fatalf("n Enter: prompt %q sort %q", l.prompt, sortLabel(l.sort))
	}
}

// 17w9r6 check 2: letters still pick a field at once in both pickers, and
// the footers and ? help list the same movement keys for both (golden at
// 100x30, plain and in colour).
func TestSortAndFilterPickersShareKeys(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	m := manyModel(t)
	m.width, m.height = 100, 30
	m.resize()
	f, s := press(m, "f"), press(m, "s")
	if got := press(f, "s"); got.pickerField != "state" {
		t.Fatalf("s in the filter picker picked %q", got.pickerField)
	}
	if got := press(s, "p"); sortLabel(got.sortDraft) != "priority ↓" || got.prompt != "sort" {
		t.Fatalf("p in the sort picker: draft %q prompt %q", sortLabel(got.sortDraft), got.prompt)
	}
	move := func(mm model) (bar string, help []string) {
		for _, r := range mm.contextHelp().rows {
			help = append(help, r.key)
		}
		return ansi.Strip(mm.keyBar()), help
	}
	fb, fh := move(f)
	sb, sh := move(s)
	for _, k := range []string{"j/k ↑↓"} {
		if !strings.Contains(fb, k) || !strings.Contains(sb, k) {
			t.Fatalf("footers differ on %q:\nfilter: %s\nsort:   %s", k, fb, sb)
		}
		if !contains(fh, k) || !contains(sh, k) {
			t.Fatalf("help differs on %q:\nfilter: %q\nsort:   %q", k, fh, sh)
		}
	}
	if !contains(sh, "Space") {
		t.Fatalf("sort help lacks Space: %q", sh)
	}
	var shots []string
	for _, c := range []struct {
		name string
		m    model
	}{
		{"filter", f}, {"sort", s}, {"sort p, then j j", press(s, "p", "j", "j")},
		{"filter ?", press(f, "?")}, {"sort ?", press(s, "?")},
	} {
		shots = append(shots, "== "+c.name+" at 100x30", ansi.Strip(c.m.View()))
	}
	goldenFile(t, "testdata/header/pickers-100x30.txt", strings.Join(shots, "\n")+"\n")

	// In colour: the highlight is the filter picker's cue in both.
	c := manyModel(t)
	c.styles = newTheme()
	c.styles.renderer.SetColorProfile(termenv.TrueColor)
	c.styles.renderer.SetHasDarkBackground(true)
	c.width, c.height = 100, 30
	c.resize()
	fc := strings.Split(press(c, "f").filterText(), "\n")[2]
	var hl string
	for _, l := range strings.Split(press(c, "s", "j", "j").sortText(), "\n") {
		if strings.HasPrefix(ansi.Strip(l), "> p") {
			hl = l
		}
	}
	if prefix := func(l string) string { return l[:strings.Index(l, ">")] }; hl == "" || prefix(fc) != prefix(hl) {
		t.Fatalf("the highlight is not the filter picker's cue:\nfilter %q\nsort   %q", fc, hl)
	}
	shot := "== sort p, then j j at 100x30 truecolor\n" + press(c, "s", "p", "j", "j").View() + "\n"
	goldenFile(t, "testdata/header/sort-highlight-100x30.txt", strings.ReplaceAll(shot, "\x1b", `\e`))
}
