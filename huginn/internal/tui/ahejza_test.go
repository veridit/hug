package tui

import (
	"strconv"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

// ahejza check 1: the line under the view line reads 'filter: none · sort:
// rank' with f and s marked, and shows chips and the order once set.
func TestFilterLineNamesFilterAndSort(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	var golden []string
	for _, w := range []int{80, 140} {
		m := press(manyModel(t), "F") // 'filter: none': the list's default cleared (7zg9mf)
		m.width, m.height = w, 24
		m.resize()
		line := strings.TrimRight(strings.Split(ansi.Strip(m.View()), "\n")[2], " ")
		if line != "filter: none · sort: rank" {
			t.Fatalf("at %d the filter line reads %q", w, line)
		}
		raw := m.filterBar()
		key := m.styles.accent.Underline(true)
		for _, k := range []string{"f", "s"} {
			if !strings.Contains(raw, key.Render(k)) {
				t.Fatalf("at %d %q is not marked: %q", w, k, raw)
			}
		}
		set := commandLine(m, "doing")
		set = press(set, "s", "p", "u", "enter")
		got := strings.TrimRight(strings.Split(ansi.Strip(set.View()), "\n")[2], " ")
		if !strings.HasPrefix(got, "filter: [state:doing ×]") || !strings.HasSuffix(got, "· sort: priority ↓ · updated ↓") {
			t.Fatalf("at %d with a chip and an order: %q", w, got)
		}
		golden = append(golden, "== "+strconv.Itoa(w), line, got)
	}
	goldenFile(t, "testdata/header/filter-line.txt", strings.Join(golden, "\n")+"\n")
}

// ahejza check 2: s opens the sort picker: field letters with hints, picks
// chain as 'then by', the same letter flips, Backspace drops the last,
// Enter applies, Esc leaves the order unchanged.
func TestSortPickerOnS(t *testing.T) {
	m := manyModel(t)
	m.width, m.height = 140, 40
	m.resize()
	s := press(m, "s")
	if s.prompt != "sort" {
		t.Fatalf("s opened %q, want the sort picker", s.prompt)
	}
	panel := ansi.Strip(s.View())
	for _, want := range []string{"SORT · a letter adds a field (then by)", "p  priority  critical first", "d  created   date created, newest first", "c  current column"} {
		if !strings.Contains(panel, want) {
			t.Fatalf("the sort picker lacks %q:\n%s", want, panel)
		}
	}
	s = press(s, "p", "t")
	if got := sortLabel(s.sortDraft); got != "priority ↓ · type" {
		t.Fatalf("p t: %q", got)
	}
	if s = press(s, "t"); sortLabel(s.sortDraft) != "priority ↓ · type ↓" {
		t.Fatalf("t again did not flip: %q", sortLabel(s.sortDraft))
	}
	if s = press(s, "backspace"); sortLabel(s.sortDraft) != "priority ↓" {
		t.Fatalf("⌫: %q", sortLabel(s.sortDraft))
	}
	applied := press(s, "enter")
	if applied.prompt != "" || sortLabel(applied.sort) != "priority ↓" {
		t.Fatalf("Enter: prompt %q sort %q", applied.prompt, sortLabel(applied.sort))
	}
	kept := press(applied, "s", "n", "esc")
	if kept.prompt != "" || sortLabel(kept.sort) != "priority ↓" || strings.Join(listIDs(kept), " ") != strings.Join(listIDs(applied), " ") {
		t.Fatalf("Esc changed the order: %q", sortLabel(kept.sort))
	}
	// From a task too: s sorts the list it came from.
	if in := press(m, "enter", "s"); in.prompt != "sort" {
		t.Fatalf("s in a task opened %q", in.prompt)
	}
}

// ahejza check 3: f still opens the filter picker, and both panels share
// one look: an accent heading, the order lines ('filter: …' / 'now: …'
// and 'new: …', nch9d4), then a letter, a name and a dim hint per field.
func TestFilterAndSortPickersLookAlike(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	m := manyModel(t)
	f := press(m, "f")
	if f.prompt != "filter" {
		t.Fatalf("f opened %q", f.prompt)
	}
	fp := strings.Split(ansi.Strip(f.filterText()), "\n")
	sp := strings.Split(ansi.Strip(press(m, "s").sortText()), "\n")
	if !strings.HasPrefix(fp[0], "FILTER · ") || !strings.HasPrefix(sp[0], "SORT · ") {
		t.Fatalf("headings: %q / %q", fp[0], sp[0])
	}
	if fp[1] != "filter: state:!done" || sp[1] != "now: rank (board order)" { // the list's default (7zg9mf)
		t.Fatalf("second lines: %q / %q", fp[1], sp[1])
	}
	if !strings.HasPrefix(fp[2][1:], " s  state     todo, doing") || !strings.HasPrefix(sp[5], "  p  priority  critical first") {
		t.Fatalf("field lines: %q / %q", fp[2], sp[5])
	}
	goldenFile(t, "testdata/header/pickers.txt", strings.Join(fp, "\n")+"\n--\n"+strings.Join(sp, "\n")+"\n")
}

// ahejza check 4: keys 1-4 do nothing and no hint names them; s no longer
// opens ': state '.
func TestDigitsNoLongerFilter(t *testing.T) {
	for _, k := range []string{"1", "2", "3", "4"} {
		m := manyModel(t)
		before := state(m)
		if got := press(m, k); state(got) != before {
			t.Fatalf("%s changed something: filter %+v view %s", k, got.filter, got.view)
		}
	}
	m := manyModel(t)
	m.width, m.height = 140, 60
	m.resize()
	screen := ansi.Strip(m.View())
	help := ansi.Strip(press(m, "?").helpText())
	for _, text := range []string{screen, help} {
		if strings.Contains(text, "1 todo") || strings.Contains(text, "1 / 2 / 3 / 4") {
			t.Fatalf("a hint still names the digits:\n%s", text)
		}
	}
	if got := press(m, "s"); got.prompt == ":" || strings.Contains(got.input.Value(), "state") {
		t.Fatalf("s opened the state palette: prompt %q input %q", got.prompt, got.input.Value())
	}
}

// ahejza check 5: help lists f filter and s sort with their picker keys.
func TestHelpListsFilterAndSortPickers(t *testing.T) {
	m := manyModel(t)
	m.width, m.height = 140, 60
	m.resize()
	help := strings.Join(strings.Fields(ansi.Strip(press(m, "?").helpText())), " ")
	for _, want := range []string{"f filter picker: s t p g l a e", "s sort picker: c column, p t s u d g a n; again flips", "s r or : sort rank goes back"} {
		if !strings.Contains(help, want) {
			t.Fatalf("help lacks %q:\n%s", want, help)
		}
	}
	sortHelp := strings.Join(strings.Fields(ansi.Strip(press(m, "s", "?").helpText())), " ")
	for _, want := range []string{"IN THE SORT PICKER", "r rank: the default board order", "c the marked column, as o", "d created"} {
		if !strings.Contains(sortHelp, want) {
			t.Fatalf("sort picker help lacks %q:\n%s", want, sortHelp)
		}
	}
}

// ahejza check 8: o and sc both sort by the marked column, and repeating
// either flips it; in the s picker created is d and c reads 'c current
// column (NAME)'.
func TestSortCIsTheMarkedColumn(t *testing.T) {
	m := manyModel(t)
	m.width, m.height = 140, 40
	m.resize()
	if got := press(m, "s", "c"); got.prompt != "sort" || !strings.Contains(got.message, "mark one with H/L") {
		t.Fatalf("sc with no column: prompt %q message %q", got.prompt, got.message)
	}
	m = press(m, "L", "L", "L") // id, title, stage (911j70)
	col, ok := m.currentColumn()
	if !ok {
		t.Fatal("H/L marked no column")
	}
	if panel := ansi.Strip(press(m, "s").View()); !strings.Contains(panel, "c  current column ("+col.name+")") {
		t.Fatalf("the picker does not name the marked column %q:\n%s", col.name, panel)
	}
	for _, way := range [][]string{{"o"}, {"s", "c"}} {
		one := press(m, way...)
		if one.prompt != "" || len(one.sort) != 1 || one.sort[0].field != col.name || one.sort[0].desc {
			t.Fatalf("%v: prompt %q sort %+v", way, one.prompt, one.sort)
		}
		two := press(one, way...)
		if len(two.sort) != 1 || !two.sort[0].desc {
			t.Fatalf("%v again did not flip: %+v", way, two.sort)
		}
		// o then sc: the same order, so sc flips what o set.
		if mixed := press(press(m, "o"), "s", "c"); len(mixed.sort) != 1 || !mixed.sort[0].desc {
			t.Fatalf("o then sc did not flip: %+v", mixed.sort)
		}
	}
	if d := press(m, "s", "d", "enter"); len(d.sort) != 1 || d.sort[0].field != "created" {
		t.Fatalf("s d: %+v", d.sort)
	}
}
