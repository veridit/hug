package tui

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/veridit/muninn/internal/search"
	"github.com/veridit/muninn/internal/store"
)

type chip struct{ field, value string }
type filterValue struct {
	value string
	count int
}

func (m model) matches(u store.Unit, except string) bool {
	for _, c := range m.chips {
		if c.field == except {
			continue
		}
		switch c.field {
		case "place":
			if u.Scope != c.value {
				return false
			}
		case "category":
			if u.Category != c.value {
				return false
			}
		case "date":
			date := u.Updated.Local().Format("2006-01-02")
			want := c.value
			if want == "today" {
				want = time.Now().Format("2006-01-02")
			}
			if want == "yesterday" {
				want = time.Now().AddDate(0, 0, -1).Format("2006-01-02")
			}
			if date != want {
				return false
			}
		}
	}
	return true
}
func (m model) filtered(rows []search.Result) []search.Result {
	var out []search.Result
	for _, r := range rows {
		if _, ok := m.binding.Present(r.Unit); ok && m.matches(r.Unit, "") {
			out = append(out, r)
		}
	}
	return out
}
func (m *model) addChip(field, value string) {
	for i, c := range m.chips {
		if c.field == field {
			m.chips = append(m.chips[:i], m.chips[i+1:]...)
			break
		}
	}
	if value != "all" {
		m.chips = append(m.chips, chip{field, value})
	}
	if m.screen != listing {
		m.showList("Filtered memories", m.allResults())
	}
	m.cursor = 0
}
func (m model) allResults() []search.Result {
	var rows []search.Result
	for _, u := range m.snapshot.units {
		rows = append(rows, search.Result{Unit: u})
	}
	return rows
}
func (m *model) clearFilters() {
	m.chips = nil
	m.filter = ""
	m.searchSeq++
	m.input.SetValue("")
	m.results = m.allResults()
	m.cursor = 0
	if m.screen == listing {
		m.title = "Recent"
	}
}
func (m *model) shortcutPlace(n string) {
	if n == "4" {
		m.addChip("place", "all")
		return
	}
	alias := map[string]string{"1": "here", "2": "client", "3": "global"}[n]
	for _, p := range m.binding.Places() {
		if p.Name == alias {
			m.addChip("place", p.Scope)
			return
		}
	}
	if alias == "client" {
		m.status = "No client place in this session"
		return
	}
	// Places collapses aliases with the same destination.
	p, err := m.binding.Placement("", alias)
	if err != nil {
		m.status = err.Error()
		return
	}
	m.addChip("place", p.Scope)
}
func (m *model) filterChoices() []filterValue {
	counts := map[string]int{}
	for _, u := range m.snapshot.units {
		if !m.matches(u, m.filterField) {
			continue
		}
		if m.filter != "" {
			found := false
			for _, r := range m.results {
				if r.Unit.SID == u.SID {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}
		switch m.filterField {
		case "place":
			counts[u.Scope]++
		case "category":
			counts[u.Category]++
		case "date":
			counts[u.Updated.Local().Format("2006-01-02")]++
		}
	}
	var values []filterValue
	switch m.filterField {
	case "place":
		for _, p := range m.snapshot.places {
			values = append(values, filterValue{p.Scope, counts[p.Scope]})
		}
	case "category":
		for _, c := range categories {
			values = append(values, filterValue{c, counts[c]})
		}
	case "date":
		for _, d := range []string{"today", "yesterday"} {
			date := time.Now()
			if d == "yesterday" {
				date = date.AddDate(0, 0, -1)
			}
			values = append(values, filterValue{d, counts[date.Format("2006-01-02")]})
		}
		var dates []string
		for date := range counts {
			dates = append(dates, date)
		}
		sort.Sort(sort.Reverse(sort.StringSlice(dates)))
		for _, date := range dates {
			values = append(values, filterValue{date, counts[date]})
		}
	}
	needle := strings.ToLower(m.input.Value())
	var out []filterValue
	for _, v := range values {
		label := v.value
		if m.filterField == "place" {
			label = m.placeName(v.value) + " " + v.value
		}
		if strings.Contains(strings.ToLower(label), needle) {
			out = append(out, v)
		}
	}
	return out
}
func (m *model) jumpChoices() {
	prefix := m.input.Value()
	m.jumpIDs = nil
	for _, u := range m.snapshot.units {
		if strings.HasPrefix(u.SID, prefix) || strings.HasPrefix(strconv.FormatInt(u.ID, 10), prefix) {
			m.jumpIDs = append(m.jumpIDs, u.SID)
		}
	}
	m.completion = 0
}
func (m model) updatePicker(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	k := msg.String()
	if m.focus == filterInput {
		if m.filterField == "" {
			switch k {
			case "p":
				m.filterField = "place"
			case "c":
				m.filterField = "category"
			case "d":
				m.filterField = "date"
			default:
				return m, nil
			}
			m.input.SetValue("")
			m.completion = 0
			return m, m.input.Focus()
		}
		choices := m.filterChoices()
		switch k {
		case "j", "down", "ctrl+n":
			m.completion = min(max(0, len(choices)-1), m.completion+1)
			return m, nil
		case "k", "up", "ctrl+p":
			m.completion = max(0, m.completion-1)
			return m, nil
		case "enter":
			if len(choices) > 0 {
				v := choices[min(m.completion, len(choices)-1)]
				m.addChip(m.filterField, v.value)
				m.focus = noInput
				m.input.Blur()
			}
			return m, nil
		}
		before := m.input.Value()
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		if before != m.input.Value() {
			m.completion = 0
		}
		return m, cmd
	}
	switch k {
	case "down", "ctrl+n":
		m.completion = min(max(0, len(m.jumpIDs)-1), m.completion+1)
		return m, nil
	case "up", "ctrl+p":
		m.completion = max(0, m.completion-1)
		return m, nil
	case "tab":
		if len(m.jumpIDs) > 0 {
			m.input.SetValue(m.jumpIDs[m.completion])
			m.input.CursorEnd()
			m.jumpChoices()
		}
		return m, nil
	case "enter":
		ref := m.input.Value()
		if len(m.jumpIDs) > 0 {
			ref = m.jumpIDs[min(m.completion, len(m.jumpIDs)-1)]
		}
		if ref == "" {
			m.status = "Type a visible id or prefix"
			return m, nil
		}
		m.focus = noInput
		m.input.Blur()
		m.openUnit(ref)
		return m, nil
	}
	before := m.input.Value()
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	if before != m.input.Value() {
		m.jumpChoices()
	}
	return m, cmd
}
func (m model) filterBar() string {
	if len(m.chips) == 0 {
		return dim("Filters: none · f pick · 1 here · 2 client · 3 global · 4 all")
	}
	var labels []string
	for _, c := range m.chips {
		v := c.value
		if c.field == "place" {
			v = m.placeName(v)
		}
		label := "[" + c.field + ":" + flat(v) + "]"
		if c.field == "category" {
			labels = append(labels, colourCategory(label))
		} else {
			labels = append(labels, colourPlace(label))
		}
	}
	return strings.Join(labels, " ") + dim(" · ⌫ remove · F clear")
}
func (m model) pickerView(h int) string {
	if m.focus == jumpInput {
		lines := []string{m.input.View(), dim("Type a visible id or number · ↑↓ selects · Enter opens"), ""}
		start := max(0, m.completion-7)
		for i := start; i < len(m.jumpIDs) && i < start+8; i++ {
			id := m.jumpIDs[i]
			text := ""
			for _, u := range m.snapshot.units {
				if u.SID == id {
					text = flat(u.Body)
					break
				}
			}
			lines = append(lines, selectedLine(dim(pad(id, 8))+" "+text, i == m.completion, m.width-2))
		}
		return box("Jump to memory", lines, m.width, h, true)
	}
	if m.filterField == "" {
		return box("Quick filter · choose a field", []string{"", "  p  place", "  c  category", "  d  date", "", "Press a field letter, then choose a value with counts."}, m.width, h, true)
	}
	lines := []string{m.input.View(), dim("Type to narrow · j/k or ↑↓ selects · Enter adds a chip"), ""}
	choices := m.filterChoices()
	start := max(0, m.completion-7)
	for i := start; i < len(choices) && i < start+8; i++ {
		v := choices[i]
		label := v.value
		if m.filterField == "place" {
			label = m.placeName(v.value)
		}
		lines = append(lines, selectedLine(pad(flat(label), max(10, min(40, m.width-14)))+fmt.Sprintf(" %5d", v.count), i == m.completion, m.width-2))
	}
	return box("Quick filter · "+m.filterField, lines, m.width, h, true)
}
