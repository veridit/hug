package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
	"github.com/muesli/termenv"
	"github.com/veridit/huginn/internal/task"
	"github.com/veridit/huginn/internal/tui/testfixture"
)

func manyModel(t *testing.T) model {
	t.Helper()
	root := t.TempDir()
	if err := testfixture.Seed(root); err != nil {
		t.Fatal(err)
	}
	m := newModel(&task.Store{Root: root}, Options{User: "tester"})
	m.width = 80
	m.height = 24
	m.resize()
	m.setView("list")
	m.move(0, true)
	return m
}

func TestUnknownWindowSizeKeepsUsableLayout(t *testing.T) {
	m := manyModel(t)
	for _, size := range []tea.WindowSizeMsg{{Width: 0, Height: 0}, {Width: 0, Height: 24}, {Width: 80, Height: 0}, {Width: -1, Height: -1}} {
		next, _ := m.Update(size)
		m = next.(model)
		if m.width != 80 || m.height != 24 || !strings.Contains(ansi.Strip(m.View()), "HUGINN · list") {
			t.Fatalf("invalid size %v replaced the usable layout: %dx%d", size, m.width, m.height)
		}
	}
	next, _ := m.Update(tea.WindowSizeMsg{Width: 140, Height: 40})
	m = next.(model)
	if m.width != 140 || m.height != 40 {
		t.Fatalf("valid resize ignored: %dx%d", m.width, m.height)
	}
}

func TestKeyTableListNavigation(t *testing.T) {
	for _, down := range []string{"j", "down"} {
		t.Run(down, func(t *testing.T) {
			m := press(manyModel(t), down)
			if m.cursor != 1 {
				t.Fatal(m.cursor)
			}
			for _, up := range []string{"k", "up"} {
				n := press(m, up)
				if n.cursor != 0 {
					t.Fatal(up, n.cursor)
				}
			}
		})
	}
	for _, bottom := range []string{"G", ">", "end"} {
		for _, top := range [][]string{{"g", "g"}, {"gg"}, {"<"}, {"home"}} {
			m := press(manyModel(t), bottom)
			if m.cursor != len(m.visible)-1 {
				t.Fatal(bottom, m.cursor)
			}
			m = press(m, top...)
			if m.cursor != 0 || m.prompt != "" {
				t.Fatal(top, m.cursor, m.prompt)
			}
		}
	}
	m := press(manyModel(t), "G", "g", "j", "g")
	if !m.pendingG || m.cursor != 59 {
		t.Fatal("g prefix was not reset")
	}
	for _, tc := range []struct {
		down, up string
		amount   int
	}{{"ctrl+d", "ctrl+u", 8}, {"pgdown", "pgup", 17}} {
		m := press(manyModel(t), tc.down)
		if m.cursor != tc.amount {
			t.Fatal(tc, m.cursor)
		}
		m = press(m, tc.up)
		if m.cursor != 0 {
			t.Fatal(tc, m.cursor)
		}
	}
}
func TestPaneKeysAndBoardSelection(t *testing.T) {
	for _, right := range []string{"l", "right", "tab"} {
		for _, left := range []string{"h", "left", "shift+tab"} {
			m := manyModel(t)
			m.setView("overview")
			m = press(m, right)
			if m.pane != 1 {
				t.Fatal(right, m.pane)
			}
			m = press(m, left)
			if m.pane != 0 {
				t.Fatal(left, m.pane)
			}
			m.setView("board")
			m = press(m, right, "G")
			indices := m.indices()
			if len(indices) == 0 || m.cursor != indices[len(indices)-1] {
				t.Fatal(m.View())
			}
			selected, _ := m.selected()
			if m.boardStatus(selected) != "doing" {
				t.Fatal("invisible board selection", m.View())
			}
			m = press(m, "gg", left)
			if m.pane != 0 {
				t.Fatal(left, m.pane)
			}
		}
	}
	m := manyModel(t)
	m = press(m, "tab")
	if m.view != "board" {
		t.Fatal(m.view)
	}
	m = press(m, "shift+tab")
	if m.view != "board" || m.pane != 4 {
		t.Fatal(m.view, m.pane)
	}
}
func TestDetailAndHelpScrollKeys(t *testing.T) {
	for _, overlay := range []string{"enter", "?"} {
		for _, bottom := range []string{"G", ">", "end"} {
			for _, top := range [][]string{{"g", "g"}, {"gg"}, {"<"}, {"home"}} {
				m := press(manyModel(t), overlay, bottom)
				viewport := m.viewport
				if overlay == "?" {
					viewport = m.helpViewport
				}
				if !viewport.AtBottom() {
					t.Fatal(overlay, bottom, m.View())
				}
				m = press(m, top...)
				viewport = m.viewport
				if overlay == "?" {
					viewport = m.helpViewport
				}
				if !viewport.AtTop() {
					t.Fatal(overlay, top)
				}
			}
		}
		for _, pair := range [][2]string{{"j", "k"}, {"down", "up"}, {"ctrl+d", "ctrl+u"}, {"pgdown", "pgup"}} {
			m := press(manyModel(t), overlay, pair[0])
			v := m.viewport
			if overlay == "?" {
				v = m.helpViewport
			}
			if v.YOffset == 0 {
				t.Fatal(overlay, pair)
			}
			m = press(m, pair[1])
			v = m.viewport
			if overlay == "?" {
				v = m.helpViewport
			}
			if v.YOffset != 0 {
				t.Fatal(overlay, pair, v.YOffset)
			}
		}
	}
}
func TestEscAtEveryLevel(t *testing.T) {
	m := manyModel(t)
	m = press(m, "enter", ":", "type b", "?", "esc")
	if m.help || m.prompt != ":" || !m.detail {
		t.Fatal("help did not close one level")
	}
	m = press(m, "esc")
	if m.prompt != "" || !m.detail {
		t.Fatal("palette did not close one level")
	}
	m = press(m, "t", "esc")
	if m.confirm != "" || !m.detail {
		t.Fatal("confirmation did not close one level")
	}
	m = press(m, "esc")
	if m.detail || m.view != "list" {
		t.Fatal("detail did not close one level")
	}
	m = press(m, "f", "t", "esc")
	if m.prompt != "filter" || m.pickerField != "" {
		t.Fatal("picker value did not return to fields")
	}
	m = press(m, "esc")
	if m.prompt != "" {
		t.Fatal("picker fields did not close")
	}
	m = press(m, "/", "feature", "enter", "esc")
	if m.query != "" || m.view != "list" {
		t.Fatal("search was not cleared")
	}
	m = press(m, "1", "esc")
	if m.filter.Status != "" || m.view != "list" {
		t.Fatal("chips not cleared")
	}
	m = press(m, "#", "000", "esc")
	if m.prompt != "" || m.view != "list" {
		t.Fatal("jump did not close")
	}
	m = press(m, "esc")
	if m.view != "overview" {
		t.Fatal("did not return to overview")
	}
	_, cmd := key(m, "esc")
	if cmd != nil {
		t.Fatal("Esc quit from overview")
	}
}
func TestInputsNeverSwallowHelpOrQuit(t *testing.T) {
	for _, prefix := range [][]string{{":"}, {"/"}, {"#"}, {"n"}, {"f", "g"}} {
		m := press(manyModel(t), prefix...)
		m = press(m, "a?q")
		if !m.help || strings.Contains(m.input.Value(), "?") {
			t.Fatal(prefix, m.input.Value())
		}
		before := m.input.Value()
		m = press(m, "q", "esc")
		if m.help || m.input.Value() != before {
			t.Fatal("help leaked input")
		}
		m, _ = key(m, "q")
		if !strings.HasSuffix(m.input.Value(), "q") {
			t.Fatal("q not typed in focused input", prefix, m.input.Value())
		}
	}
	_, cmd := key(manyModel(t), "ctrl+c")
	if cmd == nil {
		t.Fatal("ctrl-c quit missing")
	}
}
func TestPaletteCompletionAndCommands(t *testing.T) {
	for _, complete := range []string{"tab", "enter"} {
		m := press(manyModel(t), ":", "bo", complete)
		if m.input.Value() != "board" || m.prompt != ":" {
			t.Fatal(complete, m.input.Value())
		}
		m = press(m, "enter")
		if m.view != "board" || m.prompt != "" {
			t.Fatal(m.View())
		}
	}
	m := press(manyModel(t), ":", "b", "ctrl+n", "ctrl+p", "down", "up")
	if m.completion != 0 {
		t.Fatal(m.completion)
	}
	for _, tc := range []struct{ query, want string }{{"type b", "type bug"}, {"status bl", "status blocked"}, {"priority h", "priority high"}, {"group comp", "group compiler"}, {"label term", "label terminal"}, {"owner al", "owner alice"}, {"goto 000001", "goto 000001"}} {
		m := press(manyModel(t), ":", tc.query, "tab")
		if m.input.Value() != tc.want {
			t.Fatalf("%s => %s", tc.query, m.input.Value())
		}
		m = press(m, "enter")
		if m.prompt != "" {
			t.Fatal(m.View())
		}
	}
	for _, view := range []string{"overview", "list", "board", "lint"} {
		m := commandLine(manyModel(t), view)
		if m.view != view {
			t.Fatal(view, m.view)
		}
	}
	for _, status := range []string{"todo", "doing", "done", "dropped", "blocked"} {
		m := commandLine(manyModel(t), status)
		if m.filter.Status != status || len(m.chips) != 1 {
			t.Fatal(status, m.View())
		}
	}
	m = commandLine(manyModel(t), "help")
	if !m.help {
		t.Fatal("help command")
	}
	m = commandLine(manyModel(t), "type absent")
	if m.prompt != ":" || !strings.Contains(m.message, "unknown type") {
		t.Fatal(m.View())
	}
	m = press(manyModel(t), ":", "nonexistent", "enter")
	if !strings.Contains(m.message, "no completions") {
		t.Fatal(m.message)
	}
}
func TestQuickFilterChipsCountsAndShortcuts(t *testing.T) {
	for _, tc := range []struct{ letter, field, value string }{{"s", "status", "todo"}, {"t", "type", "bug"}, {"p", "priority", "high"}, {"g", "group", "compiler"}, {"l", "label", "terminal"}, {"o", "owner", "alice"}} {
		m := press(manyModel(t), "f", tc.letter, tc.value)
		if len(m.filterChoices()) != 1 || m.valueCount(tc.field, tc.value) == 0 {
			t.Fatal(tc, m.View())
		}
		m = press(m, "enter")
		if len(m.chips) != 1 || m.chips[0] != (filterChip{tc.field, tc.value}) {
			t.Fatal(tc, m.View())
		}
		if !strings.Contains(m.View(), "["+tc.field+":"+tc.value+" ×]") {
			t.Fatal(m.View())
		}
		m = press(m, "backspace")
		if len(m.chips) != 0 || len(m.visible) != 60 {
			t.Fatal(m.View())
		}
	}
	m := press(manyModel(t), "f", "j", "k", "enter", "down", "up", "enter")
	if len(m.chips) != 1 {
		t.Fatal(m.View())
	}
	for i, status := range []string{"todo", "doing", "done", "blocked"} {
		m := press(manyModel(t), string(rune('1'+i)))
		if m.filter.Status != status || len(m.chips) != 1 {
			t.Fatal(i, m.View())
		}
	}
	m = commandLine(manyModel(t), "status doing")
	m = commandLine(m, "type bug")
	m = press(m, "/", "task", "enter")
	if len(m.chips) != 2 || m.query != "task" || len(m.visible) == 0 {
		t.Fatal(m.View())
	}
	m = press(m, "backspace")
	if m.filter.Status != "doing" || m.filter.Type != "" || m.query != "task" {
		t.Fatal(m.View())
	}
	for _, clear := range []string{"F", "c"} {
		n := press(m, clear)
		if n.filter != (task.Filter{}) || n.query != "" || len(n.chips) != 0 {
			t.Fatal(clear, n.View())
		}
	}
	m = commandLine(m, "clear")
	if len(m.visible) != 60 {
		t.Fatal(m.View())
	}
}
func TestJumpPrefixesAndImportedAliases(t *testing.T) {
	for _, query := range []string{"000003", "00000", "TASK-1785", "1785"} {
		m := press(manyModel(t), "#", query)
		if len(m.jumpChoices()) == 0 {
			t.Fatal(query)
		}
		m = press(m, "enter")
		if !m.detail || m.prompt != "" {
			t.Fatal(query, m.View())
		}
		selected, _ := m.selected()
		if query != "00000" && selected.ID != "000003" {
			t.Fatal(query, selected)
		}
	}
	m := press(manyModel(t), "#", "00000", "ctrl+n", "ctrl+p", "down", "up", "tab")
	if m.input.Value() != "000001" {
		t.Fatal(m.input.Value())
	}
	m = press(manyModel(t), "#", "zz-no-id", "enter")
	if m.detail || m.message != "no matching id or alias" {
		t.Fatal(m.View())
	}
}
func TestEveryEditRequiresConfirmation(t *testing.T) {
	for _, k := range []string{"t", "d", "x", "J", "K", " ", "e"} {
		m := fixture(t)
		before, _ := os.ReadFile(filepath.Join(m.store.Root, m.all[0].Path))
		commits := 0
		m.opts.Commit = func(string) error { commits++; return nil }
		m = press(m, k)
		if m.confirm != k {
			t.Fatal(k, "no confirmation")
		}
		after, _ := os.ReadFile(filepath.Join(m.store.Root, m.all[0].Path))
		if string(before) != string(after) || commits != 0 {
			t.Fatal(k, "mutated before confirmation")
		}
		m = press(m, "esc")
		if m.confirm != "" || commits != 0 {
			t.Fatal(k, "cancel failed")
		}
	}
	m := press(fixture(t), "n", "Never created", "enter", "esc")
	if len(m.all) != 2 {
		t.Fatal("new mutated before confirmation")
	}
}
func TestMarkdownDetailFixedHeaderAndEmptySections(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	m := press(manyModel(t), "#", "1785", "enter")
	m.width = 140
	m.height = 40
	m.resize()
	v := ansi.Strip(m.View())
	for _, want := range []string{"[done]", "[000003]", "⚠ Acceptance requires at least one item", "compiler-lang", "rendered code", "no acceptance items yet", "no dependencies"} {
		if !strings.Contains(v, want) {
			t.Fatalf("missing %s\n%s", want, v)
		}
	}
	for _, bad := range []string{"<!--", "SECTION:", "**compiler-lang**", "`huginn`", "## Notes", ".md:1:"} {
		if strings.Contains(v, bad) {
			t.Fatal("raw storage/markdown", bad, v)
		}
	}
	if strings.Count(v, "Acceptance requires at least one item") != 1 {
		t.Fatal(v)
	}
	header := m.detailHeader(m.visible[m.cursor])
	m = press(m, "G")
	for i, line := range header {
		if strings.TrimSpace(strings.Split(m.View(), "\n")[4+i]) != strings.TrimSpace(line) {
			t.Fatal("detail header scrolled away")
		}
	}
}
func TestDependencyFollowAndBack(t *testing.T) {
	m := fixture(t)
	a, b := m.all[0], m.all[1]
	d, _ := m.store.Get(a.ID)
	raw := strings.Replace(string(d.Bytes()), "## Depends on\n", "## Depends on\n- "+b.ID+"\n", 1)
	if err := task.AtomicWrite(filepath.Join(m.store.Root, a.Path), []byte(raw)); err != nil {
		t.Fatal(err)
	}
	m.refresh(a.Path)
	m = press(m, "enter", "tab", "enter")
	got, _ := m.selected()
	if got.ID != b.ID || len(m.detailStack) != 1 {
		t.Fatal(m.View())
	}
	m = press(m, "esc")
	got, _ = m.selected()
	if got.ID != a.ID || !m.detail {
		t.Fatal(m.View())
	}
	m = press(m, "esc")
	if m.detail {
		t.Fatal("detail did not close")
	}
}
func TestColorAndNoColorLayout(t *testing.T) {
	m := fixture(t)
	m.styles.renderer.SetColorProfile(termenv.TrueColor)
	colored := m.row(m.visible[0])
	if !strings.Contains(colored, "38;2;0;191;255") {
		t.Fatal("missing accent", colored)
	}
	m.styles.renderer.SetColorProfile(termenv.Ascii)
	if plain := m.View(); strings.Contains(plain, "\x1b[") {
		t.Fatal("NO_COLOR styles leaked", plain)
	}
}

func TestCompletedRowsKeepIDAndLintColors(t *testing.T) {
	m := manyModel(t)
	m.styles.renderer.SetColorProfile(termenv.TrueColor)
	m.styles.renderer.SetHasDarkBackground(true)
	for _, status := range []string{"done", "dropped"} {
		task := m.all[2]
		task.Status = status
		row := m.taskRow(task, 140, false, false)
		warning := m.styles.warning.Render("!")
		if !strings.Contains(row, "38;2;0;191;255") || !strings.Contains(warning, "\x1b[") || !strings.Contains(row, warning) {
			t.Fatal("completed row lost id or warning colour", row)
		}
		if ansi.StringWidth(row) != 140 {
			t.Fatal("completed row no longer aligns", ansi.StringWidth(row))
		}
	}
	m.setView("overview")
	if header := strings.Split(m.overviewText(), "\n")[0]; !strings.Contains(header, "48;2;") {
		t.Fatal("selected overview pane lacks a background highlight", header)
	}
	darkAccent := m.styles.accent.Render("badge")
	m.styles.renderer.SetHasDarkBackground(false)
	if light := m.styles.accent.Render("badge"); light == darkAccent || !strings.Contains(light, "\x1b[") {
		t.Fatal("light-terminal accent did not adapt", light)
	}
}

func TestJumpRetainsFiltersAndDetailShortcuts(t *testing.T) {
	m := press(manyModel(t), "2", "#", "1785", "enter")
	selected, _ := m.selected()
	if selected.ID != "000003" || m.filter.Status != "doing" {
		t.Fatal("jump lost filter context", m.View())
	}
	m = press(m, "F")
	if !m.detail || len(m.chips) != 0 || len(m.visible) != 60 {
		t.Fatal("clear from detail failed", m.View())
	}
	m = press(m, "1")
	if m.detail || m.filter.Status != "todo" {
		t.Fatal(m.View())
	}
	m = press(m, "enter", "backspace")
	if !m.detail || m.filter.Status != "" {
		t.Fatal(m.View())
	}
}

func TestEmptyMetadataFilters(t *testing.T) {
	m := fixture(t)
	for _, tc := range []struct{ field, value string }{{"group", "(root)"}, {"owner", "(unassigned)"}} {
		if !contains(m.values(tc.field), tc.value) {
			t.Fatal(tc, m.values(tc.field))
		}
		m = commandLine(m, tc.field+" "+tc.value)
		if len(m.visible) == 0 {
			t.Fatal(tc, m.View())
		}
		m = press(m, "F")
	}
	for i := range m.all {
		m.all[i].Priority = "high"
	}
	m.all[0].Priority = ""
	m.setFilter("priority", "(none)")
	if len(m.visible) != 1 || m.visible[0].Priority != "" {
		t.Fatal(m.View())
	}
}

func TestFullHelpFits140AndRepairStaysVisible(t *testing.T) {
	m := fixture(t)
	m.width, m.height = 140, 40
	m.resize()
	m = press(m, "?")
	for _, text := range []string{"ctrl-d", "Shift-Tab", "Backspace", "$EDITOR", "confirm / cancel", "priority <p>", "owner <o>"} {
		if !strings.Contains(m.View(), text) {
			t.Fatal("help omitted "+text, m.View())
		}
	}
	m = press(m, "esc")
	id := m.visible[0].ID
	m = act(m, "d")
	if !strings.Contains(m.View(), "fix: huginn check "+id+" 1") {
		t.Fatal(m.View())
	}
}
