package tui

import (
	"fmt"
	"slices"
	"strings"

	"github.com/charmbracelet/x/ansi"
	"github.com/veridit/hug/huginn/internal/task"
)

// Grouping (kx2ndm): rows flat, by group (the folder a task lives in) or
// by epic. Grouped, every section's rows come under one heading per value,
// in the order its first task has in the section, so the top group leads:
//
//	  ▾ later  [3]
//	> ○ 1/2  gem59b  feature  medium  ready  Launch posts
//	  ▸ ops  [2]
//
// A heading is a row the cursor stops on; Space or Enter on it folds or
// unfolds it, and a folded heading keeps its count. No group is special by
// name (Jørgen 2026-11-17: groups are folders with no meaning; "not now"
// is the deferred stage), so nothing starts folded. A fold holds for its
// value wherever it shows: fold "later" and it is folded in every section.
//
// Headings are not tasks: paneIndices stays the tasks (folded ones left
// out), paneRows adds the headings, encoded as headRow(h) = -(h+2) so
// that -1 stays "no row".

// groupings are the orders g (or : grouping) cycles through; tag groups
// by area (txjkrm).
var groupings = []string{"", "group", "epic", "tag"}

// groupedRows is one section's rows under its headings.
type groupedRows struct {
	rows  []int // task rows as indices into visible, headings as headRow(h)
	tasks []int // the task rows, without the folded ones
	heads []groupHead
}

// groupHead is one heading: the grouping's value and how many tasks it has.
type groupHead struct {
	key    string // foldKey: the grouping and the value, "group:later"
	value  string // "" for tasks with no group or no epic
	count  int
	folded bool
}

func headRow(h int) int          { return -(h + 2) }
func headOf(row int) int         { return -row - 2 }
func isHead(row int) bool        { return row < -1 }
func foldKey(g, v string) string { return g + ":" + v }

// groupValue is the value t is grouped by: its group, or its epic's id
// (an epic with no parent leads its own children), or its tag.
func (m model) groupValue(t task.Task) string {
	if m.grouping == "tag" {
		return m.leadTag(t)
	}
	if m.grouping == "epic" {
		if e := m.epicOf[t.Path]; e != "" {
			return e
		}
		if m.epics[strings.ToLower(t.ID)] {
			return t.ID
		}
		return ""
	}
	return oneLine(t.Group)
}

// leadTag is the one tag t is grouped under (txjkrm): a row has one
// heading, so a task with several tags goes under the one hug.toml lists
// first (the board's order of areas). With none of its tags listed, its
// own first tag; with no tags, "" (the "no tag" heading).
func (m model) leadTag(t task.Task) string {
	lead, at := "", -1
	for _, tg := range t.Labels {
		if i, ok := m.tagOrder[tg]; ok && (at == -1 || i < at) {
			lead, at = tg, i
		}
	}
	if lead == "" && len(t.Labels) > 0 {
		lead = t.Labels[0]
	}
	return oneLine(lead)
}

// buildGroups puts raw, a section's rows in their order, under headings.
func (m model) buildGroups(raw []int) groupedRows {
	var order []string
	members := map[string][]int{}
	for _, i := range raw {
		v := m.groupValue(m.visible[i])
		if _, ok := members[v]; !ok {
			order = append(order, v)
		}
		members[v] = append(members[v], i)
	}
	g := groupedRows{rows: []int{}, tasks: []int{}}
	for _, v := range order {
		key := foldKey(m.grouping, v)
		h := groupHead{key: key, value: v, count: len(members[v]), folded: m.folded[key]}
		g.heads = append(g.heads, h)
		g.rows = append(g.rows, headRow(len(g.heads)-1))
		if !h.folded {
			g.rows = append(g.rows, members[v]...)
			g.tasks = append(g.tasks, members[v]...)
		}
	}
	return g
}

// grouped is section p's rows under headings, built once per set of rows.
func (m model) grouped(p int) groupedRows {
	build := func(p int) groupedRows { return m.buildGroups(m.rawPane(p)) }
	if m.panes == nil {
		return build(p)
	}
	return m.panes.groupsOf(m.visibleGen, m.view, p, build)
}

// paneRows is what section p draws, top to bottom: its tasks, and when
// grouped the headings between them.
func (m model) paneRows(p int) []int {
	if m.grouping == "" {
		return m.rawPane(p)
	}
	return m.grouped(p).rows
}

// paneTotal is how many tasks section p has, folded ones too.
func (m model) paneTotal(p int) int { return len(m.rawPane(p)) }

// headUnder is the heading row under the cursor in section p, or -1.
func (m model) headUnder(p int) int {
	if m.head == "" || p != m.pane || m.grouping == "" {
		return -1
	}
	for h, x := range m.grouped(p).heads {
		if x.key == m.head {
			return headRow(h)
		}
	}
	return -1
}

// rowCursor is the row section p marks: the heading or task under the
// cursor in the focused section, the remembered task elsewhere.
func (m model) rowCursor(p int) int {
	if r := m.headUnder(p); r != -1 {
		return r
	}
	if p == m.pane {
		return m.cursor
	}
	return m.memoCursor(p)
}

// moveRows is move over headings and tasks alike (grouped rows).
func (m *model) moveRows(n int, absolute bool) {
	rows := m.paneRows(m.pane)
	if len(rows) == 0 {
		return
	}
	pos := slices.Index(rows, m.rowCursor(m.pane))
	if pos < 0 && !absolute {
		pos, n = 0, 0
	}
	if absolute {
		pos = n
	} else {
		pos += n
	}
	pos = max(0, min(pos, len(rows)-1))
	if r := rows[pos]; isHead(r) {
		m.head = m.grouped(m.pane).heads[headOf(r)].key
		m.cursor = -1
	} else {
		m.head, m.cursor = "", r
	}
	m.clamp()
}

// toggleFold folds or unfolds the heading under the cursor. The map is
// copied, never written in place: model copies share it.
func (m *model) toggleFold() {
	m.setFold(m.head, !m.folded[m.head])
	m.clamp()
}

// unfold opens the heading t is under, so a task jumped to is shown.
func (m *model) unfold(t task.Task) {
	m.setFold(foldKey(m.grouping, m.groupValue(t)), false)
}

// setFold folds or unfolds heading key.
func (m *model) setFold(key string, fold bool) {
	folded := make(map[string]bool, len(m.folded)+1)
	for k, v := range m.folded {
		folded[k] = v
	}
	if fold {
		folded[key] = true
	} else {
		delete(folded, key)
	}
	m.folded = folded
	m.visibleGen = nextVisibleGen()
}

// setGrouping groups rows by g ("", "group", "epic", "tag"); "next"
// cycles flat → group → epic → tag. The selected task stays selected.
func (m *model) setGrouping(g string) bool {
	switch g {
	case "next":
		at := slices.Index(groupings, m.grouping)
		g = groupings[(at+1)%len(groupings)]
	case "flat":
		g = ""
	case "", "group", "epic", "tag":
	default:
		m.message = "unknown grouping: " + g + "\nfix: : grouping flat, : grouping group, : grouping epic or : grouping tag"
		return false
	}
	t, ok := m.rowTask()
	m.grouping, m.head = g, ""
	m.visibleGen = nextVisibleGen()
	m.offset = 0
	if !ok || !m.follow(t.ID) {
		m.clamp()
	}
	m.message = "rows flat · " + mk("g") + "roup them"
	if g != "" {
		m.message = "rows grouped by " + g + ": Space on a heading folds it · " + mk("g") + "rouping next"
	}
	return true
}

// groupingLabel names the grouping for the header: "grouped by epic".
func (m model) groupingLabel() string {
	if m.grouping == "" {
		return ""
	}
	return "grouped by " + m.grouping
}

// headLabel is a heading's text: the group, or the epic's id and title.
func (m model) headLabel(v string) string {
	if m.grouping == "epic" {
		if v == "" {
			return "no epic"
		}
		return "↳" + v + " " + m.epicTitle[strings.ToLower(v)]
	}
	if m.grouping == "tag" && v == "" {
		return "no tag"
	}
	if v == "" {
		return "no group"
	}
	return v
}

// headText draws heading row r of section p, width cells wide: ▾ open,
// ▸ folded, the count always.
func (m model) headText(r, p, width int, selected bool) string {
	h := m.grouped(p).heads[headOf(r)]
	arrow := "▾"
	if h.folded {
		arrow = "▸"
	}
	pointer := " "
	if selected {
		pointer = m.sign(">") // the cursor's sign, no bar (911j70); it pulses (q3sh69)
	}
	count := fmt.Sprintf("  [%d]", h.count)
	label := ansi.Truncate(oneLine(m.headLabel(h.value)), max(0, width-2-2-len(count)), "…")
	return fit(pointer+" "+m.styles.dim.Render(arrow)+" "+m.styles.normal.Bold(true).Render(label)+m.styles.dim.Render(count), width)
}

// shownTasks is the "a–b shown" of rows start to end: tasks counted,
// headings not. ok is false when no task row is shown.
func shownTasks(rows []int, start, end int) (a, b int, ok bool) {
	for _, r := range rows[:start] {
		if !isHead(r) {
			a++
		}
	}
	b = a
	for _, r := range rows[start:end] {
		if !isHead(r) {
			b++
		}
	}
	return a + 1, b, b > a
}

// shownLabel is the section title's " · a–b shown" when rows are hidden.
func shownLabel(rows []int, start, end int) string {
	if end-start >= len(rows) {
		return ""
	}
	a, b, ok := shownTasks(rows, start, end)
	if !ok {
		return " · scrolled"
	}
	return shownText(a-1, b)
}

// placeCell is where t belongs, before its text: its group and its epic,
// "later·↳gem59b", or "" when it has neither (kx2ndm). What the rows are
// grouped by is in the heading, so the cell leaves it out.
func (m model) placeCell(t task.Task) string {
	var parts []string
	if g := oneLine(t.Group); g != "" && m.grouping != "group" {
		parts = append(parts, g)
	}
	if e := m.epicOf[t.Path]; e != "" && m.grouping != "epic" {
		parts = append(parts, "↳"+e)
	}
	return strings.Join(parts, "·")
}
