package tui

import (
	"fmt"
	"hash/fnv"
	"os"
	"strings"
	"unicode"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/veridit/hug/muninn/internal/search"
)

// Shared huginn/muninn palette. Categories and places have their own stable
// meanings, but accent, dim text and selection use identical adaptive tokens.
var accent = lipgloss.AdaptiveColor{Dark: "#00bfff", Light: "#006c9c"}
var muted = lipgloss.AdaptiveColor{Dark: "#8795a8", Light: "#586474"}
var selection = lipgloss.AdaptiveColor{Dark: "#16364a", Light: "#d8eef8"}
var typeColours = []lipgloss.AdaptiveColor{{Dark: "#75bfff", Light: "#215da1"}, {Dark: "#bc9dff", Light: "#7143a8"}, {Dark: "#8ac987", Light: "#31732c"}, {Dark: "#ffb454", Light: "#995200"}, {Dark: "#ff8585", Light: "#ae2525"}}

func paint(s string, c lipgloss.TerminalColor, bold bool) string {
	if os.Getenv("NO_COLOR") != "" {
		return s
	}
	return lipgloss.NewStyle().Foreground(c).Bold(bold).Render(s)
}
func dim(s string) string { return paint(s, muted, false) }
func bold(s string) string {
	if os.Getenv("NO_COLOR") != "" {
		return s
	}
	return lipgloss.NewStyle().Bold(true).Render(s)
}
func colourCategory(s string) string {
	i := 0
	value := strings.TrimPrefix(strings.Trim(s, "[]"), "category:")
	switch value {
	case "preference":
		i = 1
	case "decision":
		i = 2
	case "correction":
		i = 3
	}
	return paint(s, typeColours[i], false)
}
func colourPlace(s string) string {
	return paint(s, placeColour(s), false)
}
func placeColour(s string) lipgloss.TerminalColor {
	h := fnv.New32a()
	value := strings.TrimPrefix(strings.Trim(s, "[]"), "place:")
	_, _ = h.Write([]byte(value))
	return typeColours[h.Sum32()%uint32(len(typeColours))]
}
func safe(s string) string {
	return strings.Map(func(r rune) rune {
		if r == '\n' || r == '\t' {
			return r
		}
		if unicode.IsControl(r) {
			return -1
		}
		return r
	}, ansi.Strip(s))
}
func flat(s string) string { return strings.Join(strings.Fields(safe(s)), " ") }
func fit(s string, w int) string {
	if w < 1 {
		return ""
	}
	return ansi.Truncate(s, w, "…")
}
func pad(s string, w int) string {
	s = fit(s, w)
	return s + strings.Repeat(" ", max(0, w-ansi.StringWidth(s)))
}
func wrapText(s string, w int) string {
	return ansi.Hardwrap(ansi.Wrap(safe(s), max(1, w), ""), max(1, w), false)
}
func selectedLine(s string, selected bool, w int) string {
	marker := "  "
	if selected {
		marker = "› "
		if os.Getenv("NO_COLOR") == "" {
			return lipgloss.NewStyle().Background(selection).Bold(true).Render(pad(marker+s, w))
		}
	}
	return fit(marker+s, w)
}
func box(title string, lines []string, w, h int, focused bool) string {
	w = max(4, w)
	h = max(3, h)
	inner := w - 2
	top := "┌ " + fit(title, max(1, w-5)) + " "
	top += strings.Repeat("─", max(0, w-1-ansi.StringWidth(top))) + "┐"
	bottom := "└" + strings.Repeat("─", inner) + "┘"
	border := func(s string) string {
		if focused {
			return paint(s, accent, false)
		}
		return dim(s)
	}
	out := []string{border(top)}
	for i := 0; i < h-2; i++ {
		line := ""
		if i < len(lines) {
			line = lines[i]
		}
		out = append(out, border("│")+pad(line, inner)+border("│"))
	}
	out = append(out, border(bottom))
	return strings.Join(out, "\n")
}
func frame(lines string, w, h int) string {
	src := strings.Split(lines, "\n")
	out := make([]string, h)
	for i := range out {
		if i < len(src) {
			out[i] = fit(src[i], w)
		} else {
			out[i] = ""
		}
	}
	return strings.Join(out, "\n")
}
func (m model) View() string {
	w, h := m.width, m.height
	cwd, _ := os.Getwd()
	home, _ := os.UserHomeDir()
	if home != "" {
		cwd = strings.Replace(cwd, home, "~", 1)
	}
	left := "muninn · " + flat(cwd)
	right := fmt.Sprintf("%d memories · %d places", len(m.snapshot.units), len(m.snapshot.places))
	header := pad(paint(left, accent, true), max(0, w-ansi.StringWidth(right)-1)) + " " + dim(right)
	if w < 80 {
		header = fit(paint("muninn · "+m.title, accent, true), w)
	}
	contentH := max(3, h-4)
	var content string
	switch {
	case m.binding.Err != nil:
		content = box("Binding unavailable", strings.Split(wrapText(m.binding.Err.Error(), w-4), "\n"), w, contentH, true)
	case m.help:
		content = m.helpView(contentH)
	case m.focus == jumpInput || m.focus == filterInput:
		content = m.pickerView(contentH)
	case m.focus == correctionInput:
		content = box("Correction · "+m.pending.unit.SID, strings.Split("Original text and date stay in history. Ctrl-s reviews the new correction.\n\n"+m.editor.View(), "\n"), w, contentH, true)
	case m.focus == moveInput:
		var lines []string
		lines = append(lines, "Choose a destination. Enter reviews the move.", "")
		visible := max(1, contentH-4)
		start := windowStart(m.pending.choice, visible)
		for i := start; i < len(m.pending.places) && i < start+visible; i++ {
			p := m.pending.places[i]
			lines = append(lines, selectedLine(colourPlace(p.Name)+" · "+dim(p.Scope), i == m.pending.choice, w-2))
		}
		content = box("Move · "+m.pending.unit.SID, lines, w, contentH, true)
	case m.focus == confirmInput:
		lines := []string{"", m.confirmText(), "", "y confirms · n or Esc cancels", ""}
		if m.pending.kind == "e" {
			lines = append(lines, strings.Split(wrapText(m.pending.text, w-6), "\n")...)
		}
		content = box("Confirm owner action", lines, w, contentH, true)
	case m.focus == paletteInput:
		lines := []string{m.input.View(), dim("Tab/Enter completes · next Enter runs · ↑↓ or ctrl-n/p selects"), ""}
		start := max(0, m.completion-7)
		for i := start; i < len(m.completions) && i < start+8; i++ {
			lines = append(lines, selectedLine(m.completions[i], i == m.completion, w-2))
		}
		if len(m.completions) == 0 {
			lines = append(lines, dim("No completion. Enter runs the command with these arguments."))
		}
		content = box("Command palette", lines, w, contentH, true)
	case m.screen == overview:
		content = m.overviewView(contentH)
	case m.screen == detail:
		content = m.detailView(contentH)
	case m.screen == review:
		content = m.reviewView(contentH)
	case m.screen == preview:
		content = box("Recall preview", strings.Split(m.viewport.View(), "\n"), w, contentH, true)
	default:
		lines := []string{}
		if m.focus == findInput {
			lines = append(lines, m.input.View())
		} else if m.filter != "" {
			lines = append(lines, dim("/ "+flat(m.filter)+" · filter kept"))
		}
		lines = append(lines, m.listHeader(w-2))
		lines = append(lines, m.listLines(m.filtered(m.results), m.cursor, w-2, max(1, contentH-len(lines)-2))...)
		content = box(m.title, lines, w, contentH, true)
	}
	status := m.binding.Name + " · " + m.status
	if m.screen == detail {
		status = fmt.Sprintf("%s · text %d%% · %d related · %s", m.binding.Name, int(m.viewport.ScrollPercent()*100), len(m.related), m.status)
	}
	return frame(header, w, 1) + "\n" + fit(m.filterBar(), w) + "\n" + frame(content, w, contentH) + "\n" + fit(dim(flat(status)), w) + "\n" + fit(m.keyBar(), w)
}
func windowStart(cursor, n int) int {
	if n < 1 {
		return 0
	}
	return max(0, cursor-n+1)
}
func (m model) overviewView(h int) string {
	w := m.width
	topH := max(5, (h*3)/5)
	bottomH := h - topH
	leftW := max(24, min(34, w/4))
	rightW := w - leftW
	var places, recent, told, needs []string
	start := windowStart(m.selections[0], topH-2)
	for i := start; i < len(m.snapshot.places) && i < start+topH-2; i++ {
		p := m.snapshot.places[i]
		name := p.Name
		if p.Name == "here" {
			name += "  " + m.snapshot.hereLabel
		}
		line := pad(colourPlace(fit(name, leftW-10)), leftW-10) + fmt.Sprintf(" %5d", p.count)
		places = append(places, selectedLine(line, m.pane == 0 && i == m.selections[0], leftW-2))
	}
	if len(places) == 0 {
		places = []string{dim("  No visible places")}
	}
	start = windowStart(m.selections[1], topH-2)
	visible := m.visibleResults()
	for i := start; i < len(visible) && i < start+topH-2; i++ {
		u := visible[i].Unit
		line := dim(u.Updated.Local().Format("01-02 15:04")) + " " + pad(colourCategory(u.Category), 10) + " " + flat(u.Body)
		recent = append(recent, selectedLine(line, m.pane == 1 && i == m.selections[1], rightW-2))
	}
	if len(recent) == 0 {
		recent = []string{dim("  No memories visible here yet."), "  Add one with muninn remember."}
	}
	toldW := max(30, w*3/5)
	needsW := w - toldW
	start = windowStart(m.selections[2], bottomH-2)
	for i := start; i < len(m.snapshot.told) && i < start+bottomH-2; i++ {
		t := m.snapshot.told[i]
		suffix := ""
		if !t.unit.Active {
			suffix = " (retired)"
		}
		line := dim(t.at.Local().Format("01-02 15:04")) + " " + pad(flat(t.session), 10) + " " + dim(pad(t.unit.SID, 7)) + " " + flat(t.unit.Body) + suffix
		told = append(told, selectedLine(line, m.pane == 2 && i == m.selections[2], toldW-2))
	}
	if len(told) == 0 {
		told = []string{dim("  No recall injections recorded here yet.")}
	}
	duplicateMemories := 0
	for _, c := range m.snapshot.dupes.Duplicates {
		duplicateMemories += len(c.Retire)
	}
	dupe := fmt.Sprintf("%d look like duplicates", duplicateMemories)
	if m.snapshot.dupePath == "" {
		dupe = "Run muninn tidy dupes"
	}
	move := fmt.Sprintf("%d may be in the wrong place", len(m.snapshot.moves.Entries))
	if m.snapshot.movePath == "" {
		move = "Run muninn tidy scan"
	}
	needs = append(needs, selectedLine(dupe, m.pane == 3 && m.selections[3] == 0, needsW-2), selectedLine(move, m.pane == 3 && m.selections[3] == 1, needsW-2), "", dim("  Enter reviews · no automatic changes"))
	if bottomH < 3 {
		bottomH = 3
		topH = max(3, h-bottomH)
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, box("Places", places, leftW, topH, m.pane == 0), box("Recent", recent, rightW, topH, m.pane == 1)) + "\n" + lipgloss.JoinHorizontal(lipgloss.Top, box("Told agents lately", told, toldW, bottomH, m.pane == 2), box("Needs a look", needs, needsW, bottomH, m.pane == 3))
}
func (m model) columnWidths(w int) (int, int, int, int, int) {
	id, date, place, cat, score := 7, 10, 12, 10, 0
	if w >= 110 {
		place = 18
		id = 8
	}
	if m.filter != "" {
		score = 6
	}
	if w < 70 {
		place = 8
		date = 5
		cat = 5
	}
	return id, date, place, cat, score
}
func (m model) listHeader(w int) string {
	id, date, place, cat, score := m.columnWidths(w)
	line := "  " + pad("id", id) + " " + pad("date", date) + " " + pad("place", place) + " " + pad("category", cat) + " "
	if score > 0 {
		line += pad("score", score) + " "
	}
	return dim(fit(line+"text", w))
}
func (m model) listLines(rows []search.Result, cursor, w, h int) []string {
	id, date, place, cat, score := m.columnWidths(w)
	var lines []string
	start := windowStart(cursor, h)
	for i := start; i < len(rows) && i < start+h; i++ {
		r := rows[i]
		u, ok := m.binding.Present(r.Unit)
		if !ok {
			continue
		}
		dateText := u.Updated.Local().Format("2006-01-02")
		if date == 5 {
			dateText = u.Updated.Local().Format("01-02")
		}
		line := dim(pad(u.SID, id)) + " " + dim(pad(dateText, date)) + " " + pad(colourPlace(flat(m.placeName(u.Scope))), place) + " " + pad(colourCategory(u.Category), cat) + " "
		if score > 0 {
			line += dim(fmt.Sprintf("%*.2f", score, r.Total)) + " "
		}
		text := flat(u.Body)
		if m.explain {
			text += " · " + r.Explain()
		}
		lines = append(lines, selectedLine(line+text, i == cursor, w))
	}
	if len(lines) == 0 {
		return []string{dim("  No matching memories. / finds · :recent shows visible memories.")}
	}
	return lines
}
func (m model) detailText() string {
	u := m.unit
	lines := []string{paint("Text", accent, true), "", renderMarkdown(u.Body, m.viewport.Width-2), "", paint("Origin/session", accent, true)}
	lines = append(lines, strings.Split(wrapText(m.metadata.origin, m.viewport.Width-2), "\n")...)
	if m.explain {
		for _, r := range m.results {
			if r.Unit.SID == u.SID {
				lines = append(lines, "", paint("Search explanation", accent, true), wrapText(r.Explain(), m.viewport.Width-2))
				break
			}
		}
	}
	return strings.Join(lines, "\n")
}
func (m model) detailView(h int) string {
	relatedH := max(4, min(8, h/3))
	textH := h - relatedH
	title := m.unit.Title
	if title == "" {
		title = "Memory " + m.unit.SID
	}
	lines := []string{bold(flat(title)), m.detailBadges(), dim(strings.Repeat("─", max(1, m.width-4)))}
	lines = append(lines, strings.Split(m.viewport.View(), "\n")...)
	related := m.listLines(m.related, m.cursor, m.width-2, max(1, relatedH-3))
	related = append([]string{m.listHeader(m.width - 2)}, related...)
	return box("Memory · "+m.unit.SID+" · text (Tab switches)", lines, m.width, textH, m.pane == 0) + "\n" + box(fmt.Sprintf("Related · %d", len(m.related)), related, m.width, relatedH, m.pane == 1)
}
func (m model) confirmText() string {
	p := m.pending
	switch p.kind {
	case "x":
		return "Retire " + p.unit.SID + "? The memory is kept in history."
	case "m":
		return "Move " + p.unit.SID + " to " + p.target + "?"
	case "e":
		return "Create a dated correction superseding " + p.unit.SID + "?"
	}
	return "Confirm?"
}
func (m model) reviewView(h int) string {
	if m.reviewIndex >= m.reviewCount() {
		return box("Needs a look", []string{"Review complete.", "Esc returns. Skipped entries were not changed."}, m.width, h, true)
	}
	lines := m.reviewLines()
	start := min(m.reviewOffset, max(0, len(lines)-1))
	return box(m.title, lines[start:], m.width, h, true)
}
func (m model) reviewLines() []string {
	if m.reviewIndex >= m.reviewCount() {
		return nil
	}
	var lines []string
	if m.reviewKind == "dupes" {
		c := m.snapshot.dupes.Duplicates[m.reviewIndex]
		lines = append(lines, fmt.Sprintf("Cluster %d / %d · y merges into keeper · n skips · Esc stops", m.reviewIndex+1, m.reviewCount()), "", "KEEP "+c.Keeper.SID)
		if u, err := m.binding.Resolve(m.a, c.Keeper.SID); err == nil {
			lines = append(lines, strings.Split(renderMarkdown(u.Body, m.width-6), "\n")...)
		}
		for _, r := range c.Retire {
			lines = append(lines, "", "RETIRE "+r.SID)
			if u, err := m.binding.Resolve(m.a, r.SID); err == nil {
				lines = append(lines, strings.Split(renderMarkdown(u.Body, m.width-6), "\n")...)
			}
		}
		lines = append(lines, "", "Approval preserves keeper text and combines strength. Existing tidy apply creates a backup.")
	} else {
		e := m.snapshot.moves.Entries[m.reviewIndex]
		lines = append(lines, fmt.Sprintf("Move %d / %d · y approves · n skips · Esc stops", m.reviewIndex+1, m.reviewCount()), "", e.SID+" · "+e.Source+" → "+e.Target, "", "Current memory:")
		if u, err := m.binding.Resolve(m.a, e.SID); err == nil {
			lines = append(lines, strings.Split(renderMarkdown(u.Body, m.width-6), "\n")...)
		}
		lines = append(lines, "", "Only this entry is applied, through muninn tidy apply with its backup.")
	}
	return lines
}
func binding(keys []string, label string) key.Binding {
	return key.NewBinding(key.WithKeys(keys...), key.WithHelp(strings.Join(keys, "/"), label))
}
func (m model) keyBar() string {
	h := help.New()
	h.Width = m.width
	h.ShortSeparator = " · "
	if m.width < 110 {
		h.ShortSeparator = " "
	}
	h.Ellipsis = "…"
	if os.Getenv("NO_COLOR") != "" {
		h.Styles.ShortKey = lipgloss.NewStyle()
		h.Styles.ShortDesc = lipgloss.NewStyle()
		h.Styles.ShortSeparator = lipgloss.NewStyle()
		h.Styles.Ellipsis = lipgloss.NewStyle()
	}
	var keys []key.Binding
	switch {
	case m.help:
		keys = []key.Binding{binding([]string{"?", "esc"}, "close help")}
		if m.focus == noInput && !m.busy {
			keys = append(keys, binding([]string{"q"}, "quit"))
		}
	case m.busy:
		keys = []key.Binding{binding([]string{"ctrl+c"}, "quit after current write")}
	case m.focus == confirmInput:
		keys = []key.Binding{binding([]string{"y"}, "confirm"), binding([]string{"n", "esc"}, "cancel"), binding([]string{"?"}, "help")}
	case m.focus == correctionInput:
		keys = []key.Binding{binding([]string{"ctrl+s"}, "review correction"), binding([]string{"esc"}, "cancel"), binding([]string{"?"}, "help")}
	case m.focus == moveInput:
		keys = []key.Binding{binding([]string{"j", "k"}, "choose place"), binding([]string{"enter"}, "review move"), binding([]string{"esc"}, "cancel"), binding([]string{"?"}, "help")}
	case m.focus == paletteInput:
		keys = []key.Binding{binding([]string{"tab", "enter"}, "complete/run"), binding([]string{"↑", "↓"}, "completion"), binding([]string{"esc"}, "close"), binding([]string{"?"}, "help")}
	case m.focus == jumpInput:
		keys = []key.Binding{binding([]string{"enter"}, "open id"), binding([]string{"tab"}, "complete"), binding([]string{"↑", "↓"}, "select"), binding([]string{"esc"}, "close"), binding([]string{"?"}, "help")}
	case m.focus == filterInput:
		keys = []key.Binding{binding([]string{"p", "c", "d"}, "field"), binding([]string{"j", "k"}, "value"), binding([]string{"enter"}, "add chip"), binding([]string{"esc"}, "back"), binding([]string{"?"}, "help")}
	case m.focus == findInput:
		keys = []key.Binding{binding([]string{"enter"}, "keep filter"), binding([]string{"↑", "↓"}, "move"), binding([]string{"esc"}, "clear"), binding([]string{"?"}, "help")}
	case m.screen == review:
		keys = []key.Binding{binding([]string{"j", "k"}, "scroll"), binding([]string{"enter"}, "full memory"), binding([]string{"y"}, "approve"), binding([]string{"n"}, "skip"), binding([]string{"esc"}, "stop"), binding([]string{"?"}, "help"), binding([]string{"q"}, "quit")}
	case m.screen == preview:
		keys = []key.Binding{binding([]string{"j", "k"}, "scroll"), binding([]string{"gg", "G"}, "top/bottom"), binding([]string{"ctrl-d", "ctrl-u"}, "page"), binding([]string{"esc"}, "back"), binding([]string{"?"}, "help"), binding([]string{"q"}, "quit")}
	default:
		keys = []key.Binding{binding([]string{"j", "k"}, "move"), binding([]string{"h", "l"}, "pane"), binding([]string{"enter"}, "open"), binding([]string{"/"}, "find"), binding([]string{":"}, "command"), binding([]string{"?"}, "help"), binding([]string{"esc"}, "back"), binding([]string{"q"}, "quit")}
		if m.width >= 120 {
			if _, ok := m.selected(); ok {
				keys = append(keys, binding([]string{"e"}, "correct"), binding([]string{"m"}, "move"), binding([]string{"x"}, "retire"))
			}
		}
		if m.width < 110 {
			keys = []key.Binding{binding([]string{"j", "k"}, "row"), binding([]string{"h", "l"}, "pane"), binding([]string{"↵"}, "open"), binding([]string{"/"}, "find"), binding([]string{"#"}, "id"), binding([]string{"f"}, "filter"), binding([]string{":"}, "cmd"), binding([]string{"?"}, "help"), binding([]string{"esc"}, "back"), binding([]string{"q"}, "quit")}
		}
	}
	return h.ShortHelpView(keys)
}
func (m model) helpView(h int) string {
	lines := []string{"Movement", "  j/k or ↓/↑             next / previous row (scroll full text)", "  gg / G · < / >         top / bottom · Home / End also work", "  ctrl-d / ctrl-u        half page down / up", "  PgDn / PgUp            full page down / up", "  h/l · Tab / Shift-Tab  previous / next pane (or related memories)", "  Enter                  open detail (in find: first keep filter)", "  Esc                    back one level, clear filter or close input", "Find and commands", "  /                      live search · Enter keeps · Esc clears", "  :                      palette · Tab/Enter completes, next Enter runs", "  ctrl-n/p · ↓/↑         select palette completion", "  ctrl-e / ctrl-r        toggle score explanation / related list", "Owner actions (each confirmed)", "  e · ctrl-s             new correction · review before y confirms", "  m / x                  choose move destination / retire · y confirms", "  y / n                  approve / skip a tidy entry · Esc stops", "  ?                      help, even while typing · ?/Esc closes", "  q / ctrl-c             quit (q only outside input)", "Commands: overview, recent, place <p>, told, search <q>, category <c>,", "          today, yesterday, dupes, moves, goto <id>, clear, recall <q>"}
	if h < 24 {
		lines = []string{"j/k ↓/↑ row · gg/G </> Home/End top/bottom", "ctrl-d/u half page · PgDn/Up full page", "h/l ←/→ Tab/Shift-Tab pane · Enter opens", "Esc closes input, clears filter, detail, then overview", "/ live find · Enter keeps · Esc clears", "# id/prefix jump · Enter opens · Tab completes", "f picker: p place, c category, d date · j/k + Enter", "1/2/3/4 here/client/global/all · ⌫ chip · F clears", ": palette · Tab/Enter completes, next Enter runs", "ctrl-n/p ↑/↓ completion · ctrl-e explain · ctrl-r related", "e correction · ctrl-s reviews · m move · x retire", "y confirms owner action · n/Esc cancels", "Tidy: y approve · n skip · Esc stop (backed up)", "q quit outside input · ctrl-c quit · ?/Esc closes help", "Commands: overview recent place <p> told search <q>", "category <c> date <d> today yesterday dupes moves", "goto <id> clear recall <q> (read-only preview)"}
	} else {
		lines = append(lines, "  #                      id/prefix jump · Enter opens · Tab completes", "  f                      picker: p place, c category, d date · counts", "  1/2/3/4 · ⌫ · F        here/client/global/all · remove chip · clear filters", "  Filters combine with / search · :date today|yesterday|YYYY-MM-DD")
	}
	return box("Help · Vim keys · Esc always goes back", lines, m.width, h, true)
}
