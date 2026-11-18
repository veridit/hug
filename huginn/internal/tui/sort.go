package tui

import (
	"slices"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/veridit/hug/huginn/internal/task"
)

// The sort picker (mkzcqr, ahejza): s opens it (and ": sort"); each field
// letter adds a field (then by ...), the same letter again flips that
// field, Backspace removes the last one, Enter applies, Esc keeps the
// order as it was. c is the marked column, as o is (Jørgen 23:39:44: "o
// is the same as sc"); created is d.
// Without a sort, rows are in rank order, the only order moves change.

// sortField is one field the builder offers.
type sortField struct {
	letter, name, hint string
	desc               bool // the direction a fresh pick starts with
}

var sortFields = []sortField{
	{"p", "priority", "critical first", true},
	{"t", "type", "feature, bug, chore, spike, docs", false},
	{"s", "state", "todo, doing, depends, done, dropped", false},
	{"u", "updated", "newest first", true},
	{"d", "created", "date created, newest first", true},
	{"g", "group", "a to z", false},
	{"a", "assignee", "a to z, unassigned last", false},
	{"n", "title", "a to z", false},
}

// sortKey is one field of an order and its direction.
type sortKey struct {
	field string
	desc  bool
}

func fieldByLetter(letter string) (sortField, bool) {
	for _, f := range sortFields {
		if f.letter == letter {
			return f, true
		}
	}
	return sortField{}, false
}
func fieldByName(name string) sortField {
	for _, f := range append(sortFields, columnFields...) {
		if f.name == name {
			return f
		}
	}
	return sortField{}
}

// columnFields are orders only the column cursor sets (bejd89): they have
// no builder letter. checks orders by the share ticked, stage by the
// stage's place in the tree.
var columnFields = []sortField{
	{"", "checks", "least done first", false},
	{"", "id", "a to z", false},
	{"", "stage", "in the tree's order", false},
}

// sortLabel is an order as the header writes it: "priority ↓ · type". A
// field in its usual direction for a newest- or critical-first field is ↓;
// flipped it is ↑; a-to-z fields show no arrow until flipped (↓).
func sortLabel(keys []sortKey) string {
	if len(keys) == 0 {
		return "rank" // as the filter line says it (ahejza)
	}
	var parts []string
	for _, k := range keys {
		s := k.field
		switch {
		case k.desc:
			s += " ↓"
		case fieldByName(k.field).desc:
			s += " ↑"
		}
		parts = append(parts, s)
	}
	return strings.Join(parts, " · ")
}

// sortRefusal is the answer to a move while rows are not in rank order.
const sortRefusal = "moving needs rank order: : sort rank"

// openSort starts building an order from nothing.
func (m *model) openSort() {
	m.prompt = "sort"
	m.sortDraft = nil
	m.input.Blur()
}

// currentColumn is the marked column's field, the one c in the picker
// and o order by; ok is false with no column marked.
func (m model) currentColumn() (sortField, bool) {
	if m.column < 0 || m.column >= len(gridColumns) {
		return sortField{}, false
	}
	return fieldByName(gridColumns[m.column].field), true
}

// updateSort handles a key while the builder is open.
func (m model) updateSort(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	k := msg.String()
	if k == "c" {
		// sc is o (ahejza check 8): the marked column, at once; again
		// flips it, then off, as o does.
		if _, ok := m.currentColumn(); !ok {
			m.message = "sort: c orders by the marked column; mark one with H/L first"
			return m, nil
		}
		m.prompt = ""
		m.orderByColumn()
		return m, nil
	}
	switch k {
	case "enter":
		m.applySort(m.sortDraft)
		m.prompt = ""
		return m, nil
	case "backspace":
		if n := len(m.sortDraft); n > 0 {
			m.sortDraft = m.sortDraft[:n-1]
		}
		return m, nil
	}
	f, ok := fieldByLetter(k)
	if !ok {
		m.message = "sort: a field letter, ⌫ removes the last, Enter applies, Esc keeps the order"
		return m, nil
	}
	m.message = ""
	for i := range m.sortDraft {
		if m.sortDraft[i].field == f.name {
			m.sortDraft[i].desc = !m.sortDraft[i].desc
			return m, nil
		}
	}
	m.sortDraft = append(m.sortDraft, sortKey{f.name, f.desc})
	return m, nil
}

// applySort orders the rows by keys, rank order when there are none.
func (m *model) applySort(keys []sortKey) {
	sel := m.selectionID()
	m.sort = slices.Clone(keys)
	if len(m.sort) > 0 {
		m.yanked = "" // a paste needs rank order
	}
	m.applyFilter()
	for i, t := range m.visible {
		if t.ID == sel {
			m.cursor = i
		}
	}
	m.clamp()
	m.message = "sort: " + sortLabel(m.sort)
}

// sortVisible orders visible by the active sort; ties keep rank order,
// which visible is already in. A task without the field's value comes
// after every task with one, in both directions.
func (m *model) sortVisible() {
	if len(m.sort) == 0 {
		return
	}
	m.visibleGen = nextVisibleGen() // the rows move: sections are found again (46ek1f)
	sort.SliceStable(m.visible, func(i, j int) bool {
		a, b := m.visible[i], m.visible[j]
		for _, k := range m.sort {
			ea, eb := sortValue(k.field, a) == "", sortValue(k.field, b) == ""
			if ea != eb {
				return eb
			}
			if ea {
				continue
			}
			if c := m.compareBy(k.field, a, b); c != 0 {
				if k.desc {
					return c > 0
				}
				return c < 0
			}
		}
		return false
	})
}

// sortValue is the text a task has for field, "" when it has none.
func sortValue(field string, t task.Task) string {
	switch field {
	case "priority":
		return t.Priority
	case "type":
		return t.Type
	case "state":
		return "set" // every task has a place
	case "updated":
		return strings.Trim(strings.TrimSpace(t.Updated), `"'`)
	case "created":
		return strings.Trim(strings.TrimSpace(t.Created), `"'`)
	case "group":
		return t.Group
	case "assignee":
		return strings.Join(t.Assignee, ", ")
	case "title":
		return t.Title
	case "id":
		return t.ID
	case "checks":
		if len(t.Acceptance) == 0 {
			return ""
		}
		return "set"
	case "stage":
		return "set"
	}
	return ""
}

// compareBy compares two tasks that both have a value for field,
// ascending: least important, first type, todo, oldest, a to z.
func (m model) compareBy(field string, a, b task.Task) int {
	order := func(values []string, v string) int {
		if i := slices.Index(values, v); i >= 0 {
			return i
		}
		return len(values)
	}
	switch field {
	case "priority":
		// Free labels sit below low, a to z among themselves.
		known := []string{"low", "medium", "high", "critical"}
		pa, pb := order(known, a.Priority), order(known, b.Priority)
		if pa == len(known) && pb == len(known) {
			return -strings.Compare(a.Priority, b.Priority)
		}
		if pa == len(known) {
			pa = -1
		}
		if pb == len(known) {
			pb = -1
		}
		return pa - pb
	case "type":
		return order(task.Types, a.Type) - order(task.Types, b.Type)
	case "state":
		return order(boardStates, m.place(a)) - order(boardStates, m.place(b))
	case "checks":
		// Share ticked, as the pie shows it: 1/4 before 3/4.
		da, ta := checkCount(a)
		db, tb := checkCount(b)
		return da*tb - db*ta
	case "stage":
		wa, wb := m.stageOf(a), m.stageOf(b)
		if wa == "" {
			wa = a.State
		}
		if wb == "" {
			wb = b.State
		}
		stages := append(append(append([]string{}, task.StageNames("todo")...), task.StageNames("doing")...), "done", "dropped")
		if c := order(stages, wa) - order(stages, wb); c != 0 {
			return c
		}
		return strings.Compare(wa, wb)
	}
	// Dates in ISO form, and names, compare as text.
	return strings.Compare(strings.ToLower(sortValue(field, a)), strings.ToLower(sortValue(field, b)))
}

// sortText is the builder screen.
func (m model) sortText() string {
	// The filter picker's look (ahejza check 3): heading, then one line
	// per field with its letter and a dim hint.
	lines := []string{
		m.styles.accent.Render("SORT · a letter adds a field (then by), again flips it · ⌫ last · Enter applies · Esc keeps the order"),
		m.styles.normal.Bold(true).Render("sort: " + sortLabel(m.sortDraft)),
	}
	current := "  c  current column " + m.styles.dim.Render("mark one with H/L first")
	if f, ok := m.currentColumn(); ok {
		current = "  c  current column (" + f.name + ") " + m.styles.dim.Render("same as o")
	}
	lines = append(lines, current)
	for _, f := range sortFields {
		line := "  " + f.letter + "  " + fit(f.name, 10) + m.styles.dim.Render(f.hint)
		for _, k := range m.sortDraft {
			if k.field == f.name {
				line = m.styles.selected.Render(">" + line[1:])
			}
		}
		lines = append(lines, line)
	}
	return strings.Join(lines, "\n")
}
