package tui

import (
	"strconv"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

// noneDone says whether no done task is among the rows shown.
func noneDone(m model) bool {
	for _, t := range m.visible {
		if t.State == "done" {
			return false
		}
	}
	return true
}

// 7zg9mf check 2 (the TUI half): ': state !done' and ': type !bug' keep
// every task but those; the chip shows the !, and so does a section the
// filter empties.
func TestNegatedFilterCommand(t *testing.T) {
	m := manyModel(t)
	m = press(m, "F") // the list's own default aside (check 6)
	all := len(m.all)
	m = commandLine(m, "state !done")
	if m.filter.State != "!done" || len(m.visible) == 0 || len(m.visible) == all || !noneDone(m) {
		t.Fatalf(": state !done: filter %q, %d of %d shown, none done %v", m.filter.State, len(m.visible), all, noneDone(m))
	}
	if bar := ansi.Strip(m.filterBar()); !strings.Contains(bar, "[state:!done ×]") {
		t.Fatalf("the filter line lacks the !: %s", bar)
	}
	m = commandLine(m, "type !bug")
	for _, v := range m.visible {
		if v.State == "done" || v.Type == "bug" {
			t.Fatalf(": type !bug kept %s (%s, %s)", v.ID, v.State, v.Type)
		}
	}
	if bar := ansi.Strip(m.filterBar()); !strings.Contains(bar, "[state:!done ×] [type:!bug ×]") {
		t.Fatalf("filter line: %s", bar)
	}
	// Every state but todo, doing, done, dropped and depends is none.
	m = press(m, "F")
	m.setView("overview")
	m = commandLine(m, "state !todo")
	if s := ansi.Strip(m.View()); !strings.Contains(s, "· None with state:!todo") {
		t.Fatalf("an emptied section does not name the negated filter:\n%s", s)
	}
	// A value hug does not know is refused, with the ! kept.
	m = press(m, "F")
	m = commandLine(m, "state !don")
	if m.filter.State != "" || !strings.Contains(m.message, "unknown state: !don") {
		t.Fatalf(": state !don: filter %q, message %q", m.filter.State, m.message)
	}
}

// 7zg9mf check 3: in the f picker, ! flips the highlighted value to its
// negation, before choosing the field, before the value, or on the value
// highlighted; ! again flips it back. The rows and counts say which.
func TestFilterPickerBangFlips(t *testing.T) {
	for _, tc := range []struct {
		name string
		keys []string
		want string
	}{
		{"before the field", []string{"F", "f", "!", "s", "done", "enter"}, "!done"},
		{"before the value", []string{"F", "f", "s", "!", "done", "enter"}, "!done"},
		{"on the value", []string{"F", "f", "s", "done", "!", "enter"}, "!done"},
		{"twice", []string{"F", "f", "s", "done", "!", "!", "enter"}, "done"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m := press(manyModel(t), tc.keys...)
			if m.prompt != "" || m.filter.State != tc.want {
				t.Fatalf("%v: prompt %q, state filter %q, want %q\n%s", tc.keys, m.prompt, m.filter.State, tc.want, ansi.Strip(m.View()))
			}
			if tc.want == "!done" && !noneDone(m) {
				t.Fatal("!done kept a done task")
			}
		})
	}
	// While flipped, the highlighted row reads !done with the count it
	// would leave, and the input keeps no ! as text.
	m := press(manyModel(t), "F", "f", "s", "done")
	done := m.valueCount("state", "done")
	m = press(m, "!")
	if m.input.Value() != "done" {
		t.Fatalf("! went into the text: %q", m.input.Value())
	}
	text := ansi.Strip(m.filterText())
	if !strings.Contains(text, "> !done") || !strings.Contains(text, " "+strconv.Itoa(len(m.all)-done)) {
		t.Fatalf("flipped picker row:\n%s", text)
	}
	if bar := ansi.Strip(m.keyBar()); !strings.Contains(bar, "! not") {
		t.Fatalf("picker footer lacks '! not': %s", bar)
	}
	// The picker opens without it: ! then Esc out, and the next f picks
	// the value itself.
	m = press(manyModel(t), "F", "f", "!", "esc", "f", "s", "done", "enter")
	if m.filter.State != "done" {
		t.Fatalf("a new f kept the last picker's !: state %q", m.filter.State)
	}
}

// 7zg9mf check 6: the list opens on 'state !done', a chip like any other;
// F or ⌫ on it clears it to 'filter: none' and it stays cleared; Esc does
// not clear it. The overview and the board are not filtered by it.
func TestListOpensOnNotDone(t *testing.T) {
	m := manyModel(t) // in the list
	bar := ansi.Strip(m.filterBar())
	if m.filter.State != "!done" || !strings.Contains(bar, "filter: [state:!done ×]") || !noneDone(m) {
		t.Fatalf("the list opened with filter %q: %s", m.filter.State, bar)
	}
	// A task opened from the list and Esc: the chip stays.
	m = press(m, "enter", "esc")
	if m.view != "list" || m.filter.State != "!done" {
		t.Fatalf("Esc from a task: view %s filter %q", m.view, m.filter.State)
	}
	// Esc in the list goes up to the overview, which the default does not
	// filter; back in the list it is there again.
	m = press(m, "esc")
	if m.view != "overview" || m.filter.State != "" || len(m.visible) != len(m.all) || !strings.Contains(ansi.Strip(m.filterBar()), "filter: none") {
		t.Fatalf("overview after Esc: view %s filter %q, %d of %d", m.view, m.filter.State, len(m.visible), len(m.all))
	}
	m = press(m, "v", "b")
	if m.filter.State != "" || len(m.visible) != len(m.all) {
		t.Fatalf("board: filter %q, %d of %d", m.filter.State, len(m.visible), len(m.all))
	}
	m = press(m, "v", "l")
	if m.filter.State != "!done" {
		t.Fatalf("back in the list: filter %q", m.filter.State)
	}
	// F clears it; it stays cleared in the list for the session.
	m = press(m, "F")
	if m.filter.State != "" || !strings.Contains(ansi.Strip(m.filterBar()), "filter: none") || len(m.visible) != len(m.all) {
		t.Fatalf("F: filter %q: %s", m.filter.State, ansi.Strip(m.filterBar()))
	}
	m = press(m, "v", "o", "v", "l")
	if m.filter.State != "" {
		t.Fatalf("after F, the list came back with %q", m.filter.State)
	}
	// ⌫ on it, in a fresh session, does the same, and it stays cleared.
	m = press(manyModel(t), "backspace")
	if m.filter.State != "" || !strings.Contains(ansi.Strip(m.filterBar()), "filter: none") {
		t.Fatalf("⌫: filter %q", m.filter.State)
	}
	m = press(m, "v", "o", "v", "l")
	if m.filter.State != "" {
		t.Fatalf("after ⌫, the list came back with %q", m.filter.State)
	}
	// A filter of the reader's own in the list is kept across views, as
	// any chip (5cjy88); the default is not.
	m = manyModel(t)
	m = commandLine(m, "type bug")
	m = press(m, "v", "o")
	if m.filter.Type != "bug" || m.filter.State != "" {
		t.Fatalf("overview after ': type bug' in the list: type %q state %q", m.filter.Type, m.filter.State)
	}
	// A state chip of the reader's own replaces the default and is kept.
	m = manyModel(t)
	m = commandLine(m, "state todo")
	m = press(m, "v", "o")
	if m.filter.State != "todo" {
		t.Fatalf("overview after ': state todo' in the list: state %q", m.filter.State)
	}
}

// The reload (R) keeps the list's default as it was (7zg9mf): up, cleared
// with F, or replaced by a state chip of the reader's own.
func TestReloadKeepsListDefault(t *testing.T) {
	for _, tc := range []struct {
		name  string
		keys  []string
		state string
	}{
		{"default", nil, "!done"},
		{"cleared", []string{"F"}, ""},
		{"own", []string{":", "state todo", "enter"}, "todo"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m := press(manyModel(t), tc.keys...)
			fresh := newModel(m.store, m.opts)
			fresh.resume(m.handoff())
			if fresh.view != "list" || fresh.filter.State != tc.state || len(fresh.chips) != len(m.chips) {
				t.Fatalf("resumed: view %s state %q chips %v; want list, %q, %v", fresh.view, fresh.filter.State, fresh.chips, tc.state, m.chips)
			}
			fresh = press(fresh, "v", "o")
			if tc.name != "own" && fresh.filter.State != "" {
				t.Fatalf("the overview after the reload shows %q", fresh.filter.State)
			}
		})
	}
}

// / takes the same mark (0sw0dd's field:value): '/ state:!done' finds
// every task but the done ones; '/ state:!' is a word, not a term.
func TestNegatedSearchTerm(t *testing.T) {
	m := press(manyModel(t), "F", "/", "state:!done", "enter")
	if len(m.visible) == 0 || len(m.visible) == len(m.all) || !noneDone(m) {
		t.Fatalf("/ state:!done: %d of %d shown, none done %v", len(m.visible), len(m.all), noneDone(m))
	}
	if terms, words := parseSearch("state:! x"); len(terms) != 0 || words != "state:! x" {
		t.Fatalf("parseSearch(state:! x) = %v %q", terms, words)
	}
}
