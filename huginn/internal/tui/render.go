package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/muesli/termenv"
	"github.com/veridit/hug/huginn/internal/task"
)

type theme struct {
	renderer                      *lipgloss.Renderer
	normal, dim, accent, selected lipgloss.Style
	// One colour per meaning (zhvkmj, crab 02:52): forYou (amber) is
	// only what waits on the reader: asked of you, '? N for you', ASKED,
	// a confirm. problem (red) is only what is wrong: broken
	// lint, its ! and LINT, a file that cannot be read. A wait on a thing
	// and advice are dim.
	forYou, problem lipgloss.Style
	// focus marks where focus is in a task: one heading or the task id.
	// ask is every confirm line: a question the reader cannot miss.
	focus, ask lipgloss.Style
	// cursor is the one thing on screen that moves (qpp30y, puppy's
	// mock/grid spec): where typing goes, an underlined bold accent that
	// blinks at bubbles' 530 ms. The answer and request box use it.
	cursor lipgloss.Style
	// The view line, one cue per meaning (r80svs, Jørgen 2026-11-19
	// 11:09Z, 11:11Z): bold is a key you can press, colour the step of
	// the key sequence, underline the state. viewPrefixKey is v, step 1,
	// bold in the accent; viewKey the second step's letters, o l b !,
	// bold in violet. viewCurrent is the view you are in: its whole word
	// underlined on the line's dim (its key letter too, viewKey
	// underlined). viewPicked is v while it waits for its second key: the
	// key you picked, a steady underline under it, no fill.
	viewPrefixKey, viewKey, viewCurrent, viewPicked lipgloss.Style
	// noColor is NO_COLOR at start: colour is off, and the view line
	// still writes its bold and underlines (r80svs check 3; NO_COLOR asks
	// for no colour, not no attributes). A renderer without colour for
	// another reason (no terminal) stays plain text.
	noColor bool
	// marked is the mark in a marked row's gutter, and only that (rxr4z8,
	// crab 20:31: one cue, no tint).
	marked lipgloss.Style
	// column tints the column the cursor marks (e3sevr); columnCursor is
	// that column's cell on the cursor row.
	column, columnCursor lipgloss.Style
	// The lists as a spreadsheet (911j70, order B): idCell is a row's
	// header, the id, normal weight in the accent; rowHeader the cursor
	// row's id, on the selection tint; colHead a column heading, dim;
	// colHeadSelected the marked column's heading, on the same tint;
	// rowCursor the > before the current cell, the sign of qpp30y/kg07bt
	// in the accent, never underlined (an underline is the view line's
	// active view); owner the owner cell.
	idCell, rowHeader, colHead, colHeadSelected, rowCursor, owner lipgloss.Style
	progress                                                      lipgloss.Style // a task partly ticked (d8yx7y)
	status, kind, priority                                        map[string]lipgloss.Style
}

// The colours of the theme, dark then light, one meaning each: the table
// is in header-spec.md 'Colours' (zhvkmj, crab 03:01).
var (
	forYouColour  = [2]string{"#ffb454", "#995200"} // waits on you
	problemColour = [2]string{"#ff8585", "#ae2525"} // something is wrong: broken lint, a file that cannot be read
	// dependsColour: a task waits on another's checks (zhvkmj check 6,
	// Jørgen 19:52: depends "seems like dependencies is a problem, it
	// isn't"): a calm indigo of its own, not red, not amber, not dim.
	dependsColour = [2]string{"#6b6bd6", "#2e2e78"}
	// stepTwoColour is the second key of a two-key trie, o l b ! after v
	// (911j70, puppy's mock/grid).
	stepTwoColour = [2]string{"#c9b3ff", "#6a3fb0"}
	// typeColours follow task.Types: feature, bug, chore, spike, docs.
	typeColours = [][2]string{{"#bc9dff", "#7143a8"}, {"#ff9ecb", "#a8327f"}, {"#b8bccb", "#586474"}, {"#c4b48a", "#6b5a2a"}, {"#7bd4c2", "#146f61"}}
	// priorityColours: critical, high, medium, low.
	priorityColours = [][2]string{{"#ff7ad9", "#a3246f"}, {"#d8cc85", "#5f6b1a"}, {"#a7b4c4", "#586474"}, {"#8795a8", "#586474"}}
)

func newTheme() theme {
	r := lipgloss.NewRenderer(os.Stdout)
	noColor := os.Getenv("NO_COLOR") != ""
	if noColor {
		r.SetColorProfile(termenv.Ascii)
	}
	base := r.NewStyle()
	color := func(dark, light string) lipgloss.Style {
		return base.Foreground(lipgloss.AdaptiveColor{Dark: dark, Light: light})
	}
	t := theme{renderer: r, normal: base, dim: color("#8795a8", "#586474"), accent: color("#00bfff", "#006c9c").Bold(true),
		selected: base.Background(lipgloss.AdaptiveColor{Dark: "#16364a", Light: "#d8eef8"}).Bold(true), forYou: color(forYouColour[0], forYouColour[1]), problem: color(problemColour[0], problemColour[1]),
		focus:  base.Background(lipgloss.AdaptiveColor{Dark: "#00bfff", Light: "#006c9c"}).Foreground(lipgloss.AdaptiveColor{Dark: "#0b1620", Light: "#ffffff"}).Bold(true),
		ask:    base.Background(lipgloss.AdaptiveColor{Dark: "#ffb454", Light: "#995200"}).Foreground(lipgloss.AdaptiveColor{Dark: "#1a1200", Light: "#ffffff"}).Bold(true),
		status: map[string]lipgloss.Style{}, kind: map[string]lipgloss.Style{}, priority: map[string]lipgloss.Style{}}
	t.progress = progressStyle(color)
	t.cursor = color("#00bfff", "#006c9c").Underline(true).Bold(true)
	t.viewPrefixKey = color("#00bfff", "#006c9c").Bold(true)
	t.marked = color("#00bfff", "#006c9c").Faint(true) // dim accent, never red
	t.viewKey = color(stepTwoColour[0], stepTwoColour[1]).Bold(true)
	t.viewCurrent = t.dim.Underline(true)
	t.viewPicked = t.viewPrefixKey.Underline(true)
	t.noColor = noColor
	// A quiet tint down the marked column, and underlined, so it shows
	// without colour too (NO_COLOR); on the cursor row the row's tint, bold.
	t.column = base.Background(lipgloss.AdaptiveColor{Dark: "#1c2a36", Light: "#eef4f8"}).Underline(true)
	t.columnCursor = t.selected.Underline(true)
	// One selection tint, in two places only: the cursor row's id and the
	// marked column's heading (911j70). The current cell has the > alone.
	selTint := lipgloss.AdaptiveColor{Dark: "#2b5a78", Light: "#bfe3f5"}
	selText := lipgloss.AdaptiveColor{Dark: "#ffffff", Light: "#0b1620"}
	t.idCell = color("#00bfff", "#006c9c")
	t.rowHeader = base.Background(selTint).Foreground(selText)
	t.colHead = t.dim
	t.colHeadSelected = base.Background(selTint).Foreground(selText)
	t.rowCursor = color("#00bfff", "#006c9c").Bold(true)
	t.owner = color("#9fb3c8", "#3d4a5a")
	for i, s := range []string{"todo", "doing", "done", "depends", "dropped"} {
		t.status[s] = color([]string{"#75bfff", "#00bfff", "#8ac987", dependsColour[0], "#8795a8"}[i], []string{"#215da1", "#006c9c", "#31732c", dependsColour[1], "#586474"}[i])
	}
	// Types and priorities are no problem and wait on no one: none takes
	// the problem red or the for-you amber, and no two types look alike
	// (zhvkmj, crab 03:01; TestOneColourPerMeaning).
	for i, s := range task.Types {
		t.kind[s] = color(typeColours[i][0], typeColours[i][1])
	}
	for i, s := range []string{"critical", "high", "medium", "low"} {
		t.priority[s] = color(priorityColours[i][0], priorityColours[i][1])
	}
	return t
}
func safe(s string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsControl(r) && r != '\n' && r != '\t' {
			return -1
		}
		return r
	}, strings.ToValidUTF8(s, "�"))
}
func oneLine(s string) string { return strings.Join(strings.Fields(safe(s)), " ") }
func fit(s string, width int) string {
	if width <= 0 {
		return ""
	}
	s = ansi.Truncate(s, width, "…")
	return s + strings.Repeat(" ", max(0, width-lipgloss.Width(s)))
}
func wrap(s string, width int) string {
	return ansi.Hardwrap(ansi.Wrap(strings.ReplaceAll(s, "\t", "    "), max(1, width), ""), max(1, width), false)
}
func (m model) bodyHeight() int { return max(1, m.height-3-m.headLines()-m.markedBoxExtra()) }

// viewHeight is the body the view has to itself: all of it, or what the
// palette leaves above it (05r3pg), so the row it acts on stays in sight.
func (m model) viewHeight() int {
	if m.prompt == ":" && !m.help {
		return max(1, m.bodyHeight()-m.paletteHeight())
	}
	return m.bodyHeight()
}

// headLines is how many lines the breadcrumb and global bands take: three
// over a list (breadcrumb, views with counts, filters), one in a task, where
// only the breadcrumb draws (e3sevr, header-spec.md). A palette or picker
// opened from a task lists, so it keeps three.
func (m model) headLines() int {
	if m.taskScreen() {
		return 1
	}
	return 3
}

// taskScreen says whether the screen shows the open task (or one of its
// files, or a prompt that writes to it), not a list, help or a picker.
func (m model) taskScreen() bool {
	if !m.detail || m.help {
		return false
	}
	switch m.prompt {
	case "#", "filter", "sort":
		return false
	}
	return true
}

// brand opens the header: 'HUG · <project>' (0sw0dd, Jørgen 2026-11-16),
// the project being hug.toml [board] project, else the checkout's folder
// name; plain 'HUG' when neither is known.
func (m model) brand() string {
	name := strings.TrimSpace(m.project)
	if name == "" && m.opts.Checkout != "" {
		name = filepath.Base(filepath.Clean(m.opts.Checkout))
	}
	if name = oneLine(name); name == "" || name == "." || name == string(filepath.Separator) {
		return "HUG"
	}
	return "HUG · " + name
}

// breadcrumb is a task's one header band: HUG and the project, the view u
// goes back to, each task followed to get here, then this task's id; "? N
// for you" on the right (header-spec.md).
func (m model) breadcrumb() string {
	left := m.styles.accent.Render(m.brand()+" ·") + " " + m.detailTrail()
	right := ""
	if n := m.forYouCount(); n > 0 {
		right = m.styles.forYou.Bold(true).Render(fmt.Sprintf("? %d for you", n))
	}
	if s := m.spinner(); s != "" {
		right = strings.TrimSpace(right + m.styles.dim.Render(" · "+s))
	}
	gap := m.width - lipgloss.Width(left) - lipgloss.Width(right)
	if right == "" || gap < 2 {
		return fit(left, m.width)
	}
	return left + strings.Repeat(" ", gap) + right
}

// lintMark is marker's text without colour: "!", "·" or " ".
func (m model) lintMark(path string) string { return ansi.Strip(m.marker(path)) }

func (m model) marker(path string) string {
	if m.broken[path] {
		return m.styles.problem.Render("!")
	}
	if m.linted[path] {
		return m.styles.dim.Render("·") // advice only (pbh9xq)
	}
	return " "
}
func styled(styles map[string]lipgloss.Style, value string, width int) string {
	return styles[value].Render(fit(oneLine(value), width))
}
func (m model) taskRow(t task.Task, width int, selected, compact bool) string {
	return m.taskRowMark(t, width, selected, compact, "")
}

// scrollMark puts ↑ on the first shown row when rows are hidden above and ↓
// on the last when rows are hidden below; the header's "a–b shown" gives the
// count. The arrows use the pointer gutter, so they cost no lines.
func scrollMark(n, start, end, total int) string {
	switch {
	case n == start && start > 0:
		return "↑"
	case n == end-1 && end < total:
		return "↓"
	}
	return ""
}
func (m model) taskRowMark(t task.Task, width int, selected, compact bool, mark string) string {
	return m.taskRowCols(t, width, selected, compact, mark, m.rowLayout(width, compact, 0, 0))
}

// taskRowCols draws t in the cells c, width wide: one line, never more
// (102bc0). The row is a spreadsheet row (911j70, order B, Jørgen
// 2026-11-20 08:26Z): ID TITLE STAGE ✓ TYPE PRIORITY OWNER, every cell in
// its own colour, normal weight, on the plain background. The cursor row
// has no bar: its id, the row header, takes the selection tint, and the >
// stands right before the current cell, the marked column's, or in the
// gutter before the id when no column is marked. Nothing is underlined:
// an underline is the view line's mark of the active view.
func (m model) taskRowCols(t task.Task, width int, selected, compact bool, mark string, c rowCols) string {
	// Gutter: the pointer, the grab or yank mark, or a scroll arrow, and
	// nothing else (e3sevr, Jørgen 23:43: ? collided with ↓). The lint
	// mark ! has a cell of its own right before the id, so ids line up
	// whether a row has it or not: "↓ !89y0b7".
	cur := -1
	if selected {
		cur = m.cursorColumn(compact, c)
	}
	pointer := " "
	marked := m.isMarked(t.ID) // rxr4z8: the gutter only, nothing else on the row
	switch {
	case selected && m.grab.active && (t.Path == m.grab.path || contains(m.grab.block, t.Path)):
		pointer = "≡" // grabbed: it moves with j/k (3xcesq)
	case selected && cur < 0 && marked:
		pointer = m.styles.rowCursor.Render(cursorOnMark) // the cursor on a marked row: one sign (check 7)
	case selected && cur < 0:
		pointer = m.styles.rowCursor.Render(">")
	case m.grab.active && contains(m.grab.block, t.Path):
		pointer = m.styles.marked.Render("≡") // the rest of a grabbed block
	case m.yanked != "" && t.ID == m.yanked:
		pointer = m.styles.accent.Render("y") // yanked: P/p paste it
	case marked:
		pointer = m.styles.marked.Render(markSign)
	case mark != "":
		pointer = m.styles.dim.Render(mark)
	}
	closed := t.State == "done" || t.State == "dropped"
	// soft is a cell's style: dim on a closed task, whose id, stage and
	// pie keep their colours (zhvkmj: the one scheme of the count line).
	soft := func(style lipgloss.Style) lipgloss.Style {
		if closed {
			return m.styles.dim
		}
		return style
	}
	id := fit(oneLine(t.ID), idCell-1)
	if selected {
		// The row header: the id, tinted, as a spreadsheet marks the
		// selected row's header (911j70); its padding stays plain.
		bare := strings.TrimRight(id, " ")
		id = m.styles.rowHeader.Render(bare) + strings.Repeat(" ", len(id)-len(bare))
	} else {
		id = m.styles.idCell.Render(id)
	}
	cells := map[int]string{0: id, 1: m.titleCell(t, c.title-1, soft)}
	// The stage's reason replaces the title (6ephqk check 5); the word is
	// the STAGE cell, or for a closed task its state.
	word, _ := m.rowStage(t)
	if word == "" {
		word = t.State // done and dropped have no stage: the list shows the state
	}
	if c.status && !compact {
		if closed {
			cells[2] = styled(m.styles.status, t.State, stageCell-1)
		} else {
			cells[2] = m.stageWord(t, word, stageCell-1)
		}
	}
	if c.checks {
		cells[3] = m.progressCell(t, m.progressWidth()-1)
	}
	if compact {
		// Board columns are narrow: type and priority are one coloured
		// letter each, spelled out in the legend under the board.
		if c.kind {
			cells[4] = soft(m.styles.kind[t.Type]).Render(letter(t.Type))
		}
		if c.priority {
			cells[5] = soft(m.styles.priority[t.Priority]).Render(letter(t.Priority))
		}
	} else {
		if c.kind {
			cells[4] = soft(m.styles.kind[t.Type]).Render(fit(oneLine(t.Type), kindCell-1))
		}
		if c.priority {
			cells[5] = soft(m.styles.priority[t.Priority]).Render(fit(oneLine(t.Priority), priorityCell-1))
		}
	}
	if c.owner > 0 {
		cells[6] = soft(m.styles.owner).Render(fit(m.rowOwner(t), c.owner))
	}
	row := pointer + " " + m.marker(t.Path)
	for _, g := range m.rowCells(compact, c) {
		gap := g.at - lipgloss.Width(row)
		if gap > 0 && g.col == cur {
			row += strings.Repeat(" ", gap-1) + m.styles.rowCursor.Render(">")
		} else {
			row += strings.Repeat(" ", max(0, gap))
		}
		row += cells[g.col]
	}
	return fit(row, width)
}

// titleCell is t's TITLE cell, width wide: its tags, dim, as a prefix
// ("design · Design: imported data pass", 911j70 check 5), then where it
// belongs and its text. The text is what is cut (check 5); the tags only
// at tagMax cells, or where the text would keep less than titleMinCompact
// (a narrow board column).
func (m model) titleCell(t task.Task, width int, soft func(lipgloss.Style) lipgloss.Style) string {
	if width <= 0 {
		return ""
	}
	prefix := ""
	if tags := m.tagText(t); tags != "" {
		if room := min(tagMax, width-titleMinCompact-3); room >= 4 {
			prefix = m.styles.dim.Render(ansi.Truncate(tags, room, "…") + " · ")
		}
	}
	// The text: the stage's reason in place of the title, or the title
	// (6ephqk check 5). A task with files: ⎘ and the count before it
	// (zgr01t).
	_, text := m.rowStage(t)
	title := soft(m.styles.normal).Render(oneLine(t.Title))
	if text != "" {
		title = soft(m.styles.normal).Render(text)
	}
	// Taken with the owner's active task (byj479): said on its row, so
	// two in IN PROGRESS read as one piece of work, never a slip.
	if w := task.TakenWith(t); w != "" {
		title += m.styles.dim.Render(" · with " + oneLine(w))
	}
	if mark := m.requestCount(t); mark != "" {
		title = soft(m.styles.accent).Render(mark) + title // an open request: an agent's to act on (hbgybn)
	}
	if mark := m.fileCount(t); mark != "" {
		title = m.styles.dim.Render(mark) + title
	}
	// Group and epic before the text, dim, no line of their own (kx2ndm):
	// "later·↳gem59b Launch posts". Before, so a long title cannot push
	// them out; at most 24 cells, and the text keeps titleMinCompact (a
	// board column is narrow).
	rest := width - lipgloss.Width(prefix)
	if where := m.placeCell(t); where != "" {
		if room := min(24, rest-titleMinCompact-1); room >= 4 {
			title = m.styles.dim.Render(ansi.Truncate(where, room, "…")) + " " + title
		}
	}
	// A search that found the task by an imported id says which (vzghj8).
	// Only when nothing else matched: the title already shows why.
	if _, words := parseSearch(m.query); !textMatches(t, strings.ToLower(words)) {
		if alias := matchedAlias(t, strings.ToLower(words)); alias != "" {
			title += m.styles.dim.Render(" · " + alias)
		}
	}
	return prefix + fit(title, rest)
}

// letter is a board cell's one letter for value, upper case.
func letter(value string) string {
	v := oneLine(value)
	if v == "" {
		return " "
	}
	return strings.ToUpper(v[:1])
}

// initial is the value's first letter, upper case, in the value's colour.
func initial(styles map[string]lipgloss.Style, value string) string {
	v := oneLine(value)
	if v == "" {
		return " "
	}
	return styles[value].Render(strings.ToUpper(v[:1]))
}

// legend spells out the board's one-letter type and priority cells. Its two
// words start with the column labels T and P, emphasised, so the reader sees
// which column each half explains (2026-11-16, user).
func (m model) legend() string {
	var parts []string
	for _, k := range task.Types {
		parts = append(parts, initial(m.styles.kind, k)+" "+m.styles.dim.Render(k))
	}
	kinds := strings.Join(parts, " ")
	parts = nil
	for _, p := range []string{"critical", "high", "medium", "low"} {
		parts = append(parts, initial(m.styles.priority, p)+" "+m.styles.dim.Render(p))
	}
	priorities := strings.Join(parts, " ")
	label := func(word string) string {
		return m.styles.normal.Bold(true).Underline(true).Render(word[:1]) + m.styles.dim.Render(word[1:]+": ")
	}
	full := label("Type") + kinds + m.styles.dim.Render(" · ") + label("Priority") + priorities
	if ansi.StringWidth(full) > m.width {
		// Narrow terminals shorten the words to their letters, never a
		// value's meaning.
		full = label("T") + kinds + m.styles.dim.Render(" · ") + label("P") + priorities
	}
	return fit(full, m.width)
}
func (m model) row(t task.Task) string {
	selected, ok := m.selected()
	return m.taskRow(t, m.width, ok && selected.Path == t.Path, false)
}
func (m model) columns(width int, compact bool) string {
	return m.columnsOwner(width, compact, m.rowLayout(width, compact, 0, 0))
}

// columnsOwner is the column header for rows laid out as c: each heading
// over its cell (rowCells), upper case and dim; the marked column's on the
// selection tint (911j70), and the column the rows are ordered by with ▲
// or ▼ (bejd89).
func (m model) columnsOwner(width int, compact bool, c rowCols) string {
	marked := m.markedColumn()
	s := strings.Repeat(" ", rowHead)
	for _, g := range m.rowCells(compact, c) {
		label := strings.ToUpper(gridColumns[g.col].label)
		if compact && (g.col == 4 || g.col == 5) {
			label = label[:1] // T and P, spelled out in the legend
		}
		s += strings.Repeat(" ", max(0, g.at-lipgloss.Width(s)))
		s += m.headerCell(g.col, label, g.width, g.col == marked)
	}
	return fit(s, width)
}

func (m model) listText() string {
	indices := m.paneRows(m.pane)
	c := m.sectionLayout(m.pane) // the list is one section (1bbe96)
	head := m.columnsOwner(m.width, false, c)
	if same := m.sectionSame(m.pane); same.any() {
		// The list has no section title: the column header says it,
		// after TITLE, which gives the room.
		// When TITLE and the label do not both fit, the label is cut,
		// never the column names.
		label := m.sameLabel(same, m.visible[m.rawPane(m.pane)[0]])
		short := c
		short.title = max(len("TITLE"), c.title-lipgloss.Width(label))
		room := c.title - short.title
		head = m.columnsOwner(m.width-room, false, short) + ansi.Truncate(label, room, "…")
	}
	lines := []string{head}
	if len(indices) == 0 {
		return strings.Join(append(lines, "No tasks match. :clear resets filters; N creates a task."), "\n")
	}
	cur := m.rowCursor(m.pane)
	start, end := m.window(indices, cur, 0, max(1, m.viewHeight()-2), m.cursor)
	for n, i := range indices[start:end] {
		if isHead(i) {
			lines = append(lines, m.headText(i, m.pane, m.width, i == cur))
			continue
		}
		t := m.visible[i]
		lines = append(lines, m.taskRowCols(t, m.width, i == m.cursor, false, scrollMark(start+n, start, end, len(indices)), c))
		if i == m.cursor {
			if note := m.noteRow(t, m.width, true); note != "" {
				lines = append(lines, note) // what it waits on, under the selected row (89y0b7, tje4sx)
			}
		}
	}
	a, b, _ := shownTasks(indices, start, end)
	lines = append(lines, m.styles.dim.Render(fmt.Sprintf("Rows %d–%d of %d · ! lint finding · ↳ epic · depends = a dependency is not done", a, b, m.paneTotal(m.pane))))
	return strings.Join(lines, "\n")
}

// ownerWidth is the IN PROGRESS and WAITING owner column (one width, so
// the two sections line up): 16 cells for a name, and on
// a wide terminal as many as the longest owner with its branch needs, up
// to 30 (x1t1t1). A board without branches looks as it did.
func (m model) ownerWidth() int {
	w := 16
	if m.width < 120 {
		return w
	}
	for _, i := range append(append(m.paneIndices(sectionAsked), m.paneIndices(sectionDoing)...), m.paneIndices(sectionWaiting)...) {
		w = max(w, min(30, lipgloss.Width(m.rowOwner(m.visible[i]))))
	}
	return w
}

// rowOwner is who works on t and, when it is not the main branch, where:
// "alice · feat/x" (x1t1t1). The name comes first, so a narrow column
// cuts the branch, never the owner.
func (m model) rowOwner(t task.Task) string {
	owner := strings.Join(t.Assignee, ", ")
	if owner == "" || owner == t.Model {
		owner = t.ModelLabel() // the model holds it: with its effort (jr55s8)
	}
	if owner == "" {
		return "unassigned"
	}
	if b := oneLine(t.Branch); b != "" && b != m.opts.MainBranch {
		owner += " · " + b
	}
	return oneLine(owner)
}

func (m model) overviewText() string {
	names := overviewSections
	rows := m.overviewRows()
	var lines []string
	for p, name := range names {
		indices := m.paneRows(p)
		pointer := "  "
		if p == m.pane {
			pointer = "> "
		}
		label := fmt.Sprintf("%s%s  [%d]", pointer, name, m.paneTotal(p))
		if p == sectionLint {
			broken := m.brokenCount()
			label += fmt.Sprintf(" · %d broken · %d advice", broken, len(m.findings)-broken)
		}
		per := rows[p]
		cursor := m.rowCursor(p)
		start, end := m.window(indices, cursor, m.scrollOf(p), per, m.selIn(p))
		if len(indices) == 0 {
			// Nothing to navigate to: None goes on the title line (h9zghj).
			label += " · None"
		} else {
			label += shownLabel(indices, start, end)
		}
		if n := m.marksIn(p); n > 0 {
			label += fmt.Sprintf(" · %d marked", n) // a mark out of sight is not forgotten (rxr4z8)
		}
		// A column the same on every task of the section is said once,
		// here, not drawn on each row (1bbe96).
		same := m.sectionSame(p)
		if !same.any() {
			label = m.sectionStyle(p).Render(fit(label, m.width))
		} else {
			tail := m.sameLabel(same, m.visible[m.rawPane(p)[0]])
			label = m.sectionStyle(p).Render(label) + tail + m.sectionStyle(p).Render(strings.Repeat(" ", max(0, m.width-lipgloss.Width(label)-lipgloss.Width(tail))))
			label = fit(label, m.width)
		}
		lines = append(lines, label) // focus: its > and its bold colour, no bar (911j70)
		// IN PROGRESS and WAITING rows share one layout with the owner cell
		// (102bc0, tje4sx), without the columns the section shares.
		doing := m.sectionLayout(p)
		if p == m.pane && m.overviewHeading() && p != sectionLint && len(indices) > 0 {
			lines = append(lines, m.columnsOwner(m.width, false, doing))
		}
		for n, i := range indices[start:end] {
			if isHead(i) {
				lines = append(lines, m.headText(i, p, m.width, p == m.pane && i == cursor))
				continue
			}
			mark := scrollMark(start+n, start, end, len(indices))
			t := m.visible[i]
			sel := p == m.pane && i == m.cursor
			// Owners get a reserved column, the last, never lost behind a
			// long title (x1t1t1).
			lines = append(lines, m.taskRowCols(t, m.width, sel, false, mark, doing))
			if sel {
				if note := m.noteRow(t, m.width, true); note != "" {
					lines = append(lines, note) // what it waits on, under the selected row (89y0b7, tje4sx)
				}
			}
		}
		if sectionBlank(p, len(indices), len(names)) {
			// A blank parts a section with rows from the next; an empty
			// one is its title alone ("· None"), and the last needs none
			// (41hhgn: six sections fit 80x24).
			lines = append(lines, "")
		}
	}
	return strings.Join(lines, "\n")
}

// sectionBlank says whether a blank line follows overview section p: one
// with rows, not the last. An empty one is its title alone ("· None"):
// six sections fit 80x24 (41hhgn).
func sectionBlank(p, rows, sections int) bool { return rows > 0 && p < sections-1 }

// sectionChrome is the overview's lines that are not rows: each title and
// the blanks between sections.
func (m model) sectionChrome() int {
	n := len(overviewSections)
	lines := 0
	for p := range n {
		lines++
		if sectionBlank(p, len(m.paneRows(p)), n) {
			lines++
		}
	}
	return lines
}

// sectionState is the status whose colour an overview section's title
// takes (zhvkmj): one scheme for the count line, the status word, the
// board columns and these titles. LINT takes the colour of the ! mark.
var sectionState = map[int]string{sectionDoing: "doing", sectionNext: "todo", sectionRecent: "done"}

// sectionStyle is the title style of overview section p: ASKED amber,
// what waits on you; LINT the problem red of its ! mark; WAITING dim,
// where inactive work parks (zhvkmj, crab 02:52).
func (m model) sectionStyle(p int) lipgloss.Style {
	if s, ok := sectionState[p]; ok {
		return m.styles.status[s].Bold(true)
	}
	switch p {
	case sectionAsked:
		return m.styles.forYou.Bold(true)
	case sectionLint:
		return m.styles.problem.Bold(true)
	}
	return m.styles.dim.Bold(true)
}

// overviewRows shares the overview's rows (h9zghj): every section the
// cursor is not in shows at least min(count, recentDone) rows, an empty one
// none (None sits on its title line), and the focused section takes all the
// rows it needs. Rows still left go to IN PROGRESS, then NEXT UP (z6kz20).
// When even the minimum does not fit, RECENTLY DONE gives way first,
// then the largest of the others, down to one row each (zaec08).
func (m model) overviewRows() []int {
	n := len(overviewSections)
	want, got := make([]int, n), make([]int, n)
	budget := m.bodyHeight() - m.sectionChrome()
	if m.overviewHeading() {
		budget-- // the focused section's column header (bejd89)
	}
	others := 0
	// Counted in lines: the selected row takes two when it waits (89y0b7,
	// tje4sx), every other row one. least is the
	// lines of a section's first recentDone rows; floor its first row.
	least, floor := make([]int, n), make([]int, n)
	for p := range want {
		idx := m.paneRows(p)
		least[p] = m.linesOf(idx[:min(len(idx), recentDone)], m.selIn(p))
		// want is only compared with what fits (the budget, or least):
		// summed that far, not over a 2,000-row section (46ek1f).
		want[p] = m.linesUpTo(idx, m.selIn(p), max(budget, least[p]))
		if len(idx) > 0 {
			floor[p] = m.heightOf(idx[0], m.selIn(p))
		}
		if p != m.pane {
			got[p] = least[p]
			others += got[p]
		}
	}
	// The focused section never gets less than an unfocused one would.
	need := 0
	if m.pane >= 0 && m.pane < n {
		need = least[m.pane]
	}
	for others+need > budget {
		shrink := -1
		if got[sectionRecent] > floor[sectionRecent] && sectionRecent != m.pane {
			shrink = sectionRecent
		} else {
			for p := range got {
				if p != m.pane && got[p] > floor[p] && (shrink < 0 || got[p] > got[shrink]) {
					shrink = p
				}
			}
		}
		if shrink < 0 {
			break
		}
		got[shrink]--
		others--
	}
	if m.pane >= 0 && m.pane < n {
		got[m.pane] = min(want[m.pane], max(need, budget-others))
	}
	// Rows the focused section does not need are not left blank (z6kz20):
	// they go to IN PROGRESS up to all its tasks, then NEXT UP, then the
	// rest in screen order.
	spare := budget - others
	if m.pane >= 0 && m.pane < n {
		spare -= got[m.pane]
	}
	for _, p := range spareOrder {
		if spare <= 0 {
			break
		}
		if p == m.pane || p >= n {
			continue
		}
		more := min(spare, want[p]-got[p])
		got[p] += more
		spare -= more
	}
	// A section draws whole rows (89y0b7: one or two lines each), so a line
	// it cannot fill would stay blank while rows elsewhere are hidden. Pass
	// such lines on in spareOrder, focus first, until none moves.
	order := append([]int{m.pane}, spareOrder...)
	for moved := true; moved; {
		moved = false
		for p := range got {
			used := m.drawnLines(p, got[p])
			if used >= got[p] {
				continue
			}
			for _, q := range order {
				if q < 0 || q >= n || q == p || got[q] >= want[q] {
					continue
				}
				// Only when the extra line lets q draw one more row.
				if m.drawnLines(q, got[q]+got[p]-used) > m.drawnLines(q, got[q]) {
					got[q] += got[p] - used
					got[p] = used
					moved = true
					break
				}
			}
		}
	}
	return got
}

// drawnLines is how many of lines section p fills with whole rows, where
// it is scrolled to now.
func (m model) drawnLines(p, lines int) int {
	idx := m.paneRows(p)
	start, end := m.window(idx, m.rowCursor(p), m.scrollOf(p), lines, m.selIn(p))
	return m.linesOf(idx[start:end], m.selIn(p))
}

// spareOrder is who gets the overview's spare rows first (z6kz20).
var spareOrder = []int{sectionAsked, sectionDoing, sectionWaiting, sectionNext, sectionRecent, sectionLint}

// scrollOf is the first row section p of the current view showed last.
func (m model) scrollOf(p int) int {
	if v := memoView(m.view); v >= 0 && p >= 0 && p < len(m.memo[v]) {
		return m.memo[v][p].start
	}
	return 0
}

// memoCursor is the row section p had selected when the cursor left it,
// as an index into visible, or -1.
func (m model) memoCursor(p int) int {
	v := memoView(m.view)
	if v < 0 || p < 0 || p >= maxSections {
		return -1
	}
	path := m.memo[v][p].path
	if path == "" {
		return -1
	}
	for i := range m.visible {
		if m.visible[i].Path == path {
			return i
		}
	}
	return -1
}

// sectionRows is how many rows section p of the current view shows now.
func (m model) sectionRows(p int) int {
	if m.view == "overview" {
		return m.overviewRows()[p]
	}
	return max(1, m.viewHeight()-4) // a board column
}
func (m model) boardText() string {
	count := min(len(boardStates), max(1, (m.width+2)/40))
	first := min(max(0, m.pane-count+1), len(boardStates)-count)
	width := max(1, (m.width-2*(count-1))/count)
	cols := make([][]string, count)
	rows := max(1, m.viewHeight()-4)
	for c := 0; c < count; c++ {
		p := first + c
		indices := m.paneRows(p)
		pointer := "  "
		if p == m.pane {
			pointer = "> "
		}
		label := fmt.Sprintf("%s%s [%d]", pointer, strings.ToUpper(boardStates[p]), m.paneTotal(p))
		if len(indices) == 0 {
			label += " · None" // one line, as in the overview (h9zghj)
		}
		label = m.styles.status[boardStates[p]].Bold(true).Render(fit(label, width)) // focus: its >, no bar (911j70)
		heading := m.columns(width, true)
		if p != m.pane {
			unmarked := m
			unmarked.column = -1 // only the focused column's heading shows the marked column
			heading = unmarked.columns(width, true)
		}
		if len(indices) == 0 {
			heading = fit("", width)
		}
		cols[c] = append(cols[c], label, heading)
		cursor := m.rowCursor(p)
		start, end := m.window(indices, cursor, m.scrollOf(p), rows, m.selIn(p))
		for n, i := range indices[start:end] {
			if isHead(i) {
				cols[c] = append(cols[c], m.headText(i, p, width, p == m.pane && i == cursor))
				continue
			}
			sel := p == m.pane && i == m.cursor
			cols[c] = append(cols[c], m.taskRowMark(m.visible[i], width, sel, true, scrollMark(start+n, start, end, len(indices))))
			if sel {
				if note := m.noteRow(m.visible[i], width, true); note != "" {
					cols[c] = append(cols[c], note)
				}
			}
		}
		for len(cols[c]) < rows+2 {
			cols[c] = append(cols[c], fit("", width))
		}
	}
	var lines []string
	for n := range cols[0] {
		line := ""
		for c := range cols {
			if c > 0 {
				line += "  "
			}
			line += cols[c][n]
		}
		lines = append(lines, line)
	}
	lines = append(lines, m.legend())
	lines = append(lines, m.styles.dim.Render(fmt.Sprintf("Columns %d–%d of %d · h/l reveals other columns", first+1, first+count, len(boardStates))))
	return strings.Join(lines, "\n")
}
func (m model) paletteText() string {
	return strings.Join(m.paletteLines(m.paletteHeight()), "\n")
}

// paletteHeight is how many lines the palette takes at the bottom of the
// body (05r3pg check 1): at least a third of the screen (eight lines at
// the least), more when its rows need it, but the view keeps a third of
// the body and ten lines, so what the palette acts on stays above.
func (m model) paletteHeight() int {
	body := m.bodyHeight()
	least := max(8, m.height/3)
	most := body - max(10, body/3)
	return max(1, min(body-3, max(least, min(m.paletteNeed(), most))))
}

// paletteNeed is the lines every row of the palette takes: the rule, the
// input, the rows and their headings.
func (m model) paletteNeed() int {
	need, last := 2, "\x00"
	for _, c := range m.completions() {
		if c.group != last {
			need++
			if c.group == "CHECK" {
				need++ // TASK id above it
			}
			last = c.group
		}
		need++
	}
	return need
}

// paletteLines draws the palette in n lines (05r3pg): a rule, the input,
// then the rows under their headings, TASK id (and CHECK n) first, then
// GLOBAL; each row is its key, its name and what it does. The rows scroll
// with the pick; the headings say which group a row is in.
func (m model) paletteLines(n int) []string {
	rule := m.styles.dim.Render(strings.Repeat("─", m.width))
	lines := []string{rule, m.input.View()}
	cs := m.completions()
	if len(cs) == 0 {
		lines = append(lines, m.styles.dim.Render("  no choices: Esc goes up a level; F1 shows help"))
	}
	subject, _ := m.subject()
	heading := func(group string) string {
		switch group {
		case "TASK":
			return "TASK " + oneLine(subject.ID) + " " + oneLine(subject.Title)
		case "CHECK":
			text := ""
			if a, ok := m.focusedCheck(subject); ok {
				text = " " + oneLine(cleanMarkdown(a.Text))
			}
			return fmt.Sprintf("  CHECK %d%s", m.acceptance+1, text)
		}
		return "GLOBAL"
	}
	// Every row with its heading before it, as drawn; the pick's line is
	// kept in view.
	type row struct {
		text string
		pick int // the completion's index, -1 for a heading
	}
	var rows []row
	keyWidth := 5
	if !m.detail {
		keyWidth = 3
	}
	for _, c := range cs {
		keyWidth = max(keyWidth, ansi.StringWidth(c.key))
	}
	nameWidth := min(30, max(12, m.width/4))
	last, taskHead := "\x00", false
	for i, c := range cs {
		// CHECK n is a branch of TASK id (a tree, 05r3pg principle): the
		// task's heading comes first, the check's under it, indented, and
		// its rows under that.
		if (c.group == "CHECK" || c.group == "TASK") && !taskHead {
			rows = append(rows, row{m.styles.accent.Render(fit(heading("TASK"), m.width)), -1})
			taskHead = true
			last = "TASK"
		}
		if c.group != last {
			if c.group != "TASK" {
				rows = append(rows, row{m.styles.accent.Render(fit(heading(c.group), m.width)), -1})
			}
			last = c.group
		}
		indent := "  "
		if c.group == "CHECK" {
			indent = "    "
		}
		name, desc := oneLine(c.value), oneLine(c.description)
		if c.branch {
			name = strings.TrimRight(name, " ") + " ›"
		}
		if c.field != "" {
			desc = fmt.Sprintf("%d tasks", m.valueCount(c.field, c.fieldValue))
		}
		key := fit(c.key, keyWidth)
		width := nameWidth - (len(indent) - 2) // the names line up across groups
		line := indent + key + " " + fit(name, width) + " " + desc
		if i == m.completion {
			rows = append(rows, row{m.styles.selected.Render(fit(">"+line[1:], m.width)), i})
			continue
		}
		rows = append(rows, row{fit(m.styles.accent.Render(indent+key)+" "+fit(name, width)+" "+m.styles.dim.Render(desc), m.width), i})
	}
	space := max(0, n-len(lines))
	at := 0
	for j, r := range rows {
		if r.pick == m.completion {
			at = j
		}
	}
	start := max(0, at-space+1)
	for j := start; j < len(rows) && len(lines) < n; j++ {
		lines = append(lines, rows[j].text)
	}
	for len(lines) < n {
		lines = append(lines, "")
	}
	return lines
}

// paletteFooter says what the palette offers, in place of the view's
// footer (05r3pg check 5): how many choices, and what Enter does.
func (m model) paletteFooter() string {
	cs := m.completions()
	subject, ok := m.subject()
	for _, c := range cs {
		if c.group != "" && ok {
			return fmt.Sprintf("Palette · %d choices, %s's first · type to narrow · › opens a level", len(cs), oneLine(subject.ID))
		}
	}
	return fmt.Sprintf("Palette · %d choices · type to narrow · › opens a level", len(cs))
}

// View is the frame, with the shown image drawn over its rows where the
// terminal can (2qndsd): imageOverlay's codes go before the first line
// and after the last, taking no cells.
func (m model) View() string {
	frame := m.frameView()
	before, after := m.imageOverlay()
	if before == "" && after == "" {
		return frame
	}
	return before + frame + after
}

// frameView is the screen as text: what S captures.
func (m model) frameView() string {
	mode := m.view
	if m.detail {
		mode = "detail"
	}
	if m.prompt == "#" {
		mode = "jump"
	}
	if m.prompt == "filter" {
		mode = "filter"
	}
	if m.prompt == "sort" {
		mode = "sort"
	}
	if m.help {
		mode = "help"
	}
	// The warning count is broken findings only; advice never raises it
	// (pbh9xq).
	header := m.styles.accent.Render(m.brand()+" · "+mode) + m.styles.dim.Render(fmt.Sprintf("   %d/%d tasks · ! %d broken", len(m.visible), len(m.all), m.brokenCount()))
	if n := len(m.findings) - m.brokenCount(); n > 0 {
		header += m.styles.dim.Render(fmt.Sprintf(" · %d advice", n))
	}
	if s := m.spinner(); s != "" {
		header += m.styles.dim.Render(" · " + s) // a load past 200 ms (vnyz8z)
	}
	if n := m.forYouCount(); n > 0 {
		header += m.styles.dim.Render(" · ") + m.styles.forYou.Bold(true).Render(fmt.Sprintf("? %d for you", n))
	}
	// The active order is on the filter line, 'sort: …' (ahejza), the
	// line that says what shapes the rows; a task (no global band) has no
	// rows to order.
	if g := m.groupingLabel(); g != "" {
		header += m.styles.dim.Render(" · " + g) // kx2ndm check 2
	}
	content := ""
	switch {
	case m.help:
		content = m.helpViewport.View()
	case m.prompt == "#":
		content = m.jumpText()
	case m.prompt == "filter":
		content = m.filterText()
	case m.prompt == "sort":
		content = m.sortText()
	case m.detail && m.glance:
		// A shown file is all the screen holds (z82qsy check 4, Jørgen's
		// 5z4zt4 request 2): the crumb names it on top, its attachment note
		// under that, then a rule; the file; a rule; then the message line
		// when there is a message, and the keys (yp3c09). A screenshot's
		// own lines read as inside the frame.
		rule := m.styles.dim.Render(strings.Repeat("─", m.width))
		var top []string
		if note := m.glanceNote(); note != "" {
			top = append(top, m.styles.dim.Render(fit(note, m.width)))
		}
		top = append(top, rule)
		view := m.viewport
		view.Height = m.glanceHeight() // the rows as drawn now (yp3c09)
		body := strings.Split(view.View(), "\n")
		for len(body) < view.Height {
			body = append(body, "")
		}
		content = strings.Join(append(append(top, body...), rule), "\n")
	case m.detail:
		if t, ok := m.selected(); ok {
			content = strings.Join(m.detailHeader(t), "\n") + "\n" + m.styles.dim.Render(strings.Repeat("─", m.width)) + "\n" + m.viewport.View()
		}
	case m.view == "overview":
		content = m.overviewText()
	case m.view == "board":
		content = m.boardText()
	default:
		content = m.listText()
	}
	// The global band: the view line with the counts on it (qh9ep1),
	// then the filters.
	lines := []string{header, m.viewLine(m.stateCounts()), m.filterBar()}
	if m.taskScreen() {
		lines = []string{m.breadcrumb()} // the global band goes in a task
	}
	body := strings.Split(content, "\n")
	foot, spare := m.footRows()
	rows := m.bodyHeight() + spare
	// The palette takes the bottom rows of the body; what it acts on
	// stays drawn above it (05r3pg check 1).
	if m.prompt == ":" && !m.help {
		n := m.paletteHeight()
		keep := max(0, rows-n)
		if len(body) > keep {
			body = body[:keep]
		}
		for len(body) < keep {
			body = append(body, "")
		}
		body = append(body, m.paletteLines(n)...)
	}
	for n := 0; n < rows; n++ {
		line := ""
		if n < len(body) {
			line = body[n]
		}
		lines = append(lines, line)
	}
	lines = append(append(lines, foot...), m.keyBar())
	// Final cell-aware boundary also protects malformed fields, Unicode, and tiny resizes.
	if len(lines) > m.height {
		lines = lines[:m.height]
		lines[len(lines)-1] = m.keyBar()
	}
	for i, line := range lines {
		lines[i] = fit(line, m.width)
	}
	return strings.Join(lines, "\n")
}

// fileFrame says whether the screen is the file view's frame (z82qsy):
// a file shown, no help or list prompt over it.
func (m model) fileFrame() bool {
	return m.detail && m.glance && !m.help && m.prompt != "#" && m.prompt != "filter" && m.prompt != "sort"
}

// footRows are the lines between the body and the key bar, and how many
// rows the body gets over bodyHeight. Everywhere else that is the message
// line and the input line, blank or not. Under the file frame only the
// ones that say something are drawn, and a blank one is a row more of the
// file (yp3c09, Jørgen 2026-11-19: two empty rows above the keys while
// the file went on below).
func (m model) footRows() (foot []string, spare int) {
	if m.markedBox() {
		// The box for the marked under the list (ec6z6j check 5), then
		// the message line.
		msg := ""
		if m.message != "" {
			first, _, _ := strings.Cut(m.message, "\n")
			msg = m.renderMarks(oneLine(first), m.styles.dim)
		}
		return append(m.markedBoxLines(), msg), 0
	}
	footer, input := m.footLines()
	if !m.fileFrame() {
		return []string{footer, input}, 0
	}
	for _, l := range []string{footer, input} {
		if strings.TrimSpace(ansi.Strip(l)) == "" {
			spare++
			continue
		}
		foot = append(foot, l)
	}
	return foot, spare
}

// footLines are the message line (the footer: a message, else what the
// screen is) and the input line (a fix:, a prompt or a confirm) under
// the body.
func (m model) footLines() (footerLine, inputLine string) {
	footer := m.message
	if footer == "" && m.columnHinted && !m.detail && !m.help && m.prompt == "" && m.confirm == "" {
		footer = m.columnHint() // rows only: a task has no column (qtj177)
	}
	messageLines := strings.Split(m.message, "\n")
	if len(messageLines) > 1 {
		footer = messageLines[0]
	}
	if footer == "" {
		filters := []string{}
		for _, f := range []struct{ k, v string }{{"state", m.filter.State}, {"type", m.filter.Type}, {"group", m.filter.Group}, {"tag", m.filter.Label}, {"find", m.query}} {
			if f.v != "" {
				filters = append(filters, f.k+"="+oneLine(f.v))
			}
		}
		footer = "All tasks · : filters and commands"
		if len(filters) > 0 {
			footer = strings.Join(filters, " · ") + " · Esc clears"
		}
		if m.view == "overview" {
			// The list explains ! and ↳ in its last row; the overview has
			// no such row, so its footer does (kx2ndm, Jørgen 2026-11-16:
			// compact, no extra line).
			footer += " · ! lint finding · ↳ epic"
		}
		if m.detail {
			footer = fmt.Sprintf("Detail · %.0f%% · Esc back", m.viewport.ScrollPercent()*100)
			if pos := m.detailPosition(); pos != "" {
				footer = fmt.Sprintf("Task %s · %.0f%% · %sext · %srevious · Esc back", pos, m.viewport.ScrollPercent()*100, mk("n"), mk("p"))
			}
			if m.glance {
				// The file view's keys are on the bar under it; this line
				// is only for a message (z82qsy check 4).
				footer = ""
			}
		}
		if m.grab.active {
			rows := m.indices()
			footer = fmt.Sprintf("Moving %s · row %d of %d in %s (was %d) · nothing written until Enter", m.grab.id, m.rowOf(rows, m.grab.path)+1, len(rows), m.sectionName(), m.grab.start+1)
			if n := len(m.grab.block); n > 1 {
				footer = fmt.Sprintf("Moving %d marked · rows %d–%d of %d in %s · nothing written until Enter", n, m.rowOf(rows, m.grab.path)+1, m.rowOf(rows, m.grab.path)+n, len(rows), m.sectionName())
			}
		} else if len(m.marks) > 0 && !m.detail && m.prompt == "" && m.confirm == "" {
			footer = m.markedFooter() // the action bar (rxr4z8 check 2)
		}
		if m.prompt == ":" && !m.help {
			footer = m.paletteFooter()
		}
		if m.picking && m.prompt == "" {
			footer = pickFooter() // hjbfzd: Enter attaches the screen
		}
	}
	footerLine = m.renderMarks(oneLine(footer), m.styles.dim)
	if len(messageLines) > 1 {
		for _, line := range messageLines[1:] {
			if strings.HasPrefix(line, "fix: ") {
				inputLine = safe(line)
				break
			}
		}
	}
	if notice, bar, ok := m.editNoticeLines(); ok && m.confirm == "" && m.prompt == "" && !m.help && m.message == "" {
		footerLine = notice
		inputLine = bar
	}
	if m.confirm != "" {
		inputLine = m.confirmLine()
	} else if m.prompt == capturePrompt || m.prompt == captureTitlePrompt {
		inputLine = m.captureLine() // the keys s and c beside the field (hjbfzd)
	} else if m.prompt != "" && m.prompt != ":" && m.prompt != "#" && m.prompt != "filter" && m.prompt != "sort" && !m.boxPrompt() {
		inputLine = m.input.View()
	}
	return footerLine, inputLine
}

// editNoticeLines are the two lines shown after an edit that left the task
// with a finding: what is wrong (task, line, problem) in the warning colour,
// then a bar in the confirm style with the fix and the key that reopens the
// editor, which survives truncation (rx74bp, 2026-11-16: the only sign of a
// broken edit was "editor closed; re-linted").
func (m model) editNoticeLines() (string, string, bool) {
	t, fs, ok := m.noticeTask()
	if !ok {
		return "", "", false
	}
	f := fs[0]
	what := "⚠ " + oneLine(t.ID)
	if f.Line > 0 {
		what += fmt.Sprintf(" line %d", f.Line)
	}
	what += ": " + oneLine(f.Problem)
	if len(fs) > 1 {
		what += fmt.Sprintf(" (+%d more)", len(fs)-1)
	}
	keys := " · " + mk("e") + "dit again · Esc dismiss"
	fix := "▶ fix: " + oneLine(f.FixLine())
	// The marks are drawn before measuring: they take no cells.
	keysWidth := ansi.StringWidth(plainMarks(keys))
	fix = ansi.Truncate(fix, max(1, m.width-keysWidth), "…")
	pad := strings.Repeat(" ", max(0, m.width-ansi.StringWidth(fix)-keysWidth))
	line := m.styles.ask.Render(fix) + m.renderMarks(keys, m.styles.ask) + m.styles.ask.Render(pad)
	return m.styles.problem.Render(fit(what, m.width)), line, true
}

// confirmLine is the question every confirm asks, in one style that cannot be
// missed: a full-width bold bar in the warning colour, starting "▶ " so it
// stands out without colour too, and ending in the question mark and the
// answer keys, which survive truncation (2026-11-16, user: a plain prompt
// went unnoticed).
func (m model) confirmLine() string {
	question := "Confirm " + actionNames[m.confirm] + " "
	if len(m.bulk) > 0 {
		question = m.bulkQuestion() // the marked (rxr4z8)
	}
	switch {
	case len(m.bulk) > 0:
	case m.confirm == runKey:
		// The command itself, as the CLI takes it (05r3pg): what runs is
		// what an agent would type.
		question = "Run " + oneLine(shellLine(m.pendingArgs))
	case m.confirm == "n":
		question += oneLine(m.pendingValue)
	case m.confirm == "t":
		if t, ok := m.taskAt(m.confirmPath); ok {
			question += oneLine(t.ID) + " " + oneLine(t.Title)
			if m.pendingValue == "give back" {
				stage := task.Stage(task.Task{State: "todo", Fields: t.Fields, Goal: t.Goal, Acceptance: t.Acceptance, Dependencies: t.Dependencies}, m.all)
				question = "Give back " + oneLine(t.ID) + " to todo (" + stage + ")"
			} else if m.taskStage(t) == "review" {
				question = "Take " + oneLine(t.ID) + " for review as " + oneLine(m.me())
			}
		}
	case m.confirm == "x":
		for _, t := range m.all {
			if t.Path == m.confirmPath {
				question = "Drop " + oneLine(t.ID) + ": " + oneLine(m.pendingValue)
				break
			}
		}
	case m.confirm == "e":
		for _, t := range m.all {
			if t.Path == m.confirmPath {
				question = "Edit " + oneLine(t.ID) + " in " + oneLine(m.pendingValue)
				break
			}
		}
	case m.confirm == "-":
		for _, t := range m.all {
			if t.Path == m.confirmPath && len(t.Acceptance) > 0 {
				n := min(m.acceptance, len(t.Acceptance)-1)
				question = fmt.Sprintf("Restore struck check: ☒ %d. %s", n+1, oneLine(cleanMarkdown(t.Acceptance[n].Text)))
				break
			}
		}
	case m.confirm == " ":
		// Say exactly what Space will do: tick or untick which item, quoted
		// as the ticket shows it, box and number after the colon.
		for _, t := range m.all {
			if t.Path == m.confirmPath && len(t.Acceptance) > 0 {
				n := min(m.acceptance, len(t.Acceptance)-1)
				verb, box := "Tick", "☐"
				if t.Acceptance[n].Checked {
					verb, box = "Untick", "☑"
				}
				question = fmt.Sprintf("%s: %s %d. %s", verb, box, n+1, oneLine(cleanMarkdown(t.Acceptance[n].Text)))
				break
			}
		}
	default:
		if t, ok := m.taskAt(m.confirmPath); ok {
			question += oneLine(t.ID) + " " + oneLine(t.Title)
		}
	}
	answer := "? Enter yes · Esc no"
	question = strings.TrimRight(question, " ")
	line := "▶ " + ansi.Truncate(question, max(1, m.width-2-ansi.StringWidth(answer)), "…") + answer
	return m.styles.ask.Render(fit(line, m.width))
}

// detailTrail is the path to the open task, after the view tabs: the view it
// was opened from, any tasks followed through dependencies, then the task.
// With page focus the task id carries the focus highlight; with a section
// focused, that heading does instead.
func (m model) detailTrail() string {
	t, ok := m.selected()
	if !ok {
		return ""
	}
	// The view and the tasks followed to get here, dim, then this task's
	// id in its colour as the last crumb (header-spec.md, Jørgen 00:02).
	// The pane it was opened from follows the view (24403z check 5): the
	// section, the column or the filter that n, p and Esc follow. Not
	// while a file is shown: the file is the last crumb and needs the room.
	parts := []string{m.styles.dim.Render(m.view)}
	if pane := m.paneCrumb(); pane != "" && !m.glance {
		parts = append(parts, m.styles.dim.Render(oneLine(pane)))
	}
	for _, path := range m.detailStack {
		for _, v := range m.all {
			if v.Path == path {
				parts = append(parts, m.styles.dim.Render(oneLine(v.ID)))
			}
		}
	}
	parts = append(parts, m.styles.accent.Render(oneLine(t.ID))+m.nowCrumb(t))
	if m.detail && m.glance && m.glanceHead != "" {
		// A shown file is the last crumb: path · size · n of m (z82qsy),
		// and a zoomed picture's zoom (33v71x).
		head := m.glanceHead
		if m.picZoom > 0 && m.zoomable() {
			head += " · zoom " + zoomPercent(m.picZoom)
		}
		parts = append(parts, m.styles.accent.Render(head))
	}
	return strings.Join(parts, m.styles.dim.Render(" › "))
}

// brokenCount is the number of shown findings of severity broken.
func (m model) brokenCount() int {
	n := 0
	for _, f := range m.findings {
		if f.Severity() == task.Broken {
			n++
		}
	}
	return n
}
