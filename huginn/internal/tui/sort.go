package tui

import (
	"slices"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/veridit/hug/huginn/internal/task"
)

// The sort builder (mkzcqr): ": sort" opens it; each field letter adds a
// field (then by ...), the same letter again flips that field, Backspace
// removes the last one, Enter applies, Esc keeps the order as it was.
// Without a sort, rows are in rank order, the only order moves change.

// sortField is one field the builder offers.
type sortField struct {
	letter, name, hint string
	desc               bool // the direction a fresh pick starts with
}

var sortFields = []sortField{
	{"p", "priority", "critical first", true},
	{"t", "type", "feature, bug, chore, spike, docs", false},
	{"s", "status", "todo, doing, depends, done, dropped", false},
	{"u", "updated", "newest first", true},
	{"c", "created", "newest first", true},
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
	for _, f := range sortFields {
		if f.name == name {
			return f
		}
	}
	return sortField{}
}

// sortLabel is an order as the header writes it: "priority ↓ · type". A
// field in its usual direction for a newest- or critical-first field is ↓;
// flipped it is ↑; a-to-z fields show no arrow until flipped (↓).
func sortLabel(keys []sortKey) string {
	if len(keys) == 0 {
		return "rank order"
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

// updateSort handles a key while the builder is open.
func (m model) updateSort(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	k := msg.String()
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
	case "status":
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
	case "status":
		return order(boardStatuses, m.place(a)) - order(boardStatuses, m.place(b))
	}
	// Dates in ISO form, and names, compare as text.
	return strings.Compare(strings.ToLower(sortValue(field, a)), strings.ToLower(sortValue(field, b)))
}

// sortText is the builder screen.
func (m model) sortText() string {
	lines := []string{
		m.styles.accent.Render("SORT · a letter adds a field, the same letter flips it · ⌫ removes the last · Enter applies · Esc keeps the order"),
		m.styles.normal.Bold(true).Render("sort: " + sortLabel(m.sortDraft)),
		"",
	}
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
