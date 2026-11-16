package tui

import (
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
	"github.com/muesli/termenv"
	"github.com/veridit/hug/huginn/internal/task"
)

// focusModel opens, from the list, a task with three checks, one dependency
// and a description long enough to scroll. A second task without
// dependencies and a third, the dependency, sit after it in the list.
func focusModel(t *testing.T) (model, task.Task) {
	t.Helper()
	s := &task.Store{Root: t.TempDir()}
	dep, err := s.New(task.NewOptions{Title: "Dependency", Goal: "g", Acceptance: []string{"d"}})
	if err != nil {
		t.Fatal(err)
	}
	long := strings.Repeat("A long paragraph that needs scrolling.\n\n", 30)
	main, err := s.New(task.NewOptions{Title: "Main", Goal: long, Acceptance: []string{"one", "two", "three"}, Deps: []string{dep.Task.ID}, Top: true})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.New(task.NewOptions{Title: "Plain", Goal: "g", Acceptance: []string{"only"}, After: main.Task.ID}); err != nil {
		t.Fatal(err)
	}
	m := newModel(s, Options{User: "tester"})
	m.width, m.height = 100, 30
	m.resize()
	m.setView("list")
	m.move(0, true)
	m = press(m, "enter")
	if tk, _ := m.selected(); tk.Title != "Main" {
		t.Fatalf("opened %q, want Main", tk.Title)
	}
	return m, main.Task
}

func detailPlain(m model) string {
	tk, _ := m.selected()
	return ansi.Strip(m.detailText(tk))
}

// Check 1: no heading, key bar or help row writes keys as "[ / ]" or "[/]",
// which reads as the / key; every key the key bar shows does what it says.
func TestFocusHintsAreHonest(t *testing.T) {
	m, _ := focusModel(t)
	bad := regexp.MustCompile(`\[ ?/ ?\]|\{ ?/ ?\}`)
	for _, pane := range []int{focusAcceptance, focusDeps, focusPage} {
		m.detailPane = pane
		m.syncDetail()
		for _, w := range []int{140, 80} {
			m.width = w
			m.resize()
			if s := ansi.Strip(m.View()); bad.MatchString(s) {
				t.Fatalf("pane %d width %d shows a key as [ / ]:\n%s", pane, w, s)
			}
		}
	}
	m.width = 140
	m.resize()
	m = press(m, "?")
	if s := ansi.Strip(m.helpText()); bad.MatchString(s) {
		t.Fatalf("help shows a key as [ / ]:\n%s", s)
	}
	m = press(m, "esc")

	// Press every key the bar shows, in every focus, and see it act.
	for _, pane := range []int{focusAcceptance, focusDeps, focusPage} {
		for _, b := range func() []string {
			mm := m
			mm.detailPane = pane
			var out []string
			for _, h := range mm.detailHints(false) {
				out = append(out, h.Help().Key+"="+h.Help().Desc)
			}
			return out
		}() {
			fresh, _ := focusModel(t)
			fresh.detailPane = pane
			fresh.syncDetail()
			before := fresh
			name, desc, _ := strings.Cut(b, "=")
			var after model
			var cmd tea.Cmd
			switch name {
			case "j/k":
				after = press(fresh, "j")
				switch desc {
				case "check":
					if after.acceptance != before.acceptance+1 {
						t.Fatalf("j did not step to the next check: %d", after.acceptance)
					}
				case "dependency":
					// One dependency: j steps past the end and scrolls.
					if after.dependency != 0 || after.viewport.YOffset == before.viewport.YOffset {
						t.Fatalf("j on the last dependency did not scroll")
					}
				case "scroll":
					if after.viewport.YOffset != before.viewport.YOffset+1 || after.acceptance != 0 {
						t.Fatalf("j did not scroll the page: %d", after.viewport.YOffset)
					}
				default:
					t.Fatalf("untested j/k hint %q", desc)
				}
			case "Space":
				if after = press(fresh, " "); after.confirm != " " {
					t.Fatalf("Space shown but did not ask to tick: %q", after.message)
				}
			case "Enter":
				if after = press(fresh, "enter"); len(after.detailStack) != 1 {
					t.Fatal("Enter shown but did not follow the dependency")
				}
			case "n/p":
				after = press(fresh, "n")
				if tk, _ := after.selected(); tk.Title != "Plain" {
					t.Fatalf("n went to %q", tk.Title)
				}
				if tk, _ := press(after, "p").selected(); tk.Title != "Main" {
					t.Fatalf("p went to %q", tk.Title)
				}
			case "Tab":
				if after = press(fresh, "tab"); after.detailPane == pane {
					t.Fatal("Tab shown but focus did not move")
				}
			case "Esc":
				if after = press(fresh, "esc"); after.detail {
					t.Fatal("Esc did not leave the task")
				}
			case ":":
				if after = press(fresh, ":"); after.prompt != ":" {
					t.Fatal(": did not open the palette")
				}
			case "?":
				if after = press(fresh, "?"); !after.help {
					t.Fatal("? did not open help")
				}
			case "q":
				if _, cmd = key(fresh, "q"); cmd == nil {
					t.Fatal("q did not quit")
				}
			default:
				t.Fatalf("key bar shows %q with no test", name)
			}
		}
	}

	// The headings' own hints: "[ ]" steps, Space ticks, Enter follows.
	m.detailPane = focusAcceptance
	if !strings.Contains(detailPlain(m), "ACCEPTANCE · j/k or [ ] step · Space ticks") {
		t.Fatalf("acceptance heading:\n%s", detailPlain(m))
	}
	if got := press(m, "]"); got.acceptance != 1 {
		t.Fatalf("] did not step: %d", got.acceptance)
	}
	if got := press(m, "]", "["); got.acceptance != 0 {
		t.Fatalf("[ did not step back: %d", got.acceptance)
	}
	deps := press(m, "tab")
	if !strings.Contains(detailPlain(deps), "DEPENDENCIES · j/k or [ ] step · Enter follows") {
		t.Fatalf("dependency heading:\n%s", detailPlain(deps))
	}
}

// Check 2: Tab and Shift-Tab cycle page, acceptance and dependencies, skip a
// section that is empty, and only the focused heading is highlighted.
func TestFocusTabCyclesAndHighlightsOneHeading(t *testing.T) {
	m, _ := focusModel(t)
	if m.detailPane != focusAcceptance {
		t.Fatalf("a task with checks opens on them: %d", m.detailPane)
	}
	want := []int{focusDeps, focusPage, focusAcceptance}
	for _, w := range want {
		m = press(m, "tab")
		if m.detailPane != w {
			t.Fatalf("Tab: focus %d, want %d", m.detailPane, w)
		}
	}
	for _, w := range []int{focusPage, focusDeps, focusAcceptance} {
		m = press(m, "shift+tab")
		if m.detailPane != w {
			t.Fatalf("Shift-Tab: focus %d, want %d", m.detailPane, w)
		}
	}
	// Exactly one place shows focus: one heading, or the task id.
	for _, pane := range []int{focusAcceptance, focusDeps, focusPage} {
		m.detailPane = pane
		text := detailPlain(m)
		heads := strings.Count(text, "▸ ")
		trail := strings.Contains(ansi.Strip(strings.Split(m.View(), "\n")[1]), "[")
		switch pane {
		case focusAcceptance:
			if heads != 1 || !strings.Contains(text, "▸ ACCEPTANCE") || trail {
				t.Fatalf("acceptance focus: %d marked headings, trail marked %v\n%s", heads, trail, text)
			}
		case focusDeps:
			if heads != 1 || !strings.Contains(text, "▸ DEPENDENCIES") || trail {
				t.Fatalf("dependency focus: %d marked headings, trail marked %v\n%s", heads, trail, text)
			}
		case focusPage:
			if heads != 0 || !trail {
				t.Fatalf("page focus: %d marked headings, trail marked %v\n%s", heads, trail, text)
			}
		}
	}
	// In colour the focused heading carries the focus style; others do not.
	m.styles.renderer.SetColorProfile(termenv.TrueColor)
	m.detailPane = focusAcceptance
	tk, _ := m.selected()
	coloured := m.detailText(tk)
	if !strings.Contains(coloured, m.sectionHeading("Acceptance", "j/k or [ ] step · Space ticks", true)) || strings.Contains(coloured, m.sectionHeading("Dependencies", "j/k or [ ] step · Enter follows", true)) {
		t.Fatal("focus style is not on the acceptance heading alone")
	}
	if m.sectionHeading("Acceptance", "", true) == m.sectionHeading("Acceptance", "", false) {
		t.Fatal("focused and unfocused headings look the same")
	}

	// A task without dependencies cycles acceptance and page only.
	plain := press(m, "n")
	if tk, _ := plain.selected(); tk.Title != "Plain" {
		t.Fatalf("n opened %q", tk.Title)
	}
	for _, w := range []int{focusPage, focusAcceptance, focusPage} {
		plain = press(plain, "tab")
		if plain.detailPane != w {
			t.Fatalf("no dependencies: Tab focus %d, want %d", plain.detailPane, w)
		}
	}
	// A task without checks opens on the page, and Tab goes to its
	// dependencies and back.
	d, _ := m.store.Get(tk.ID)
	raw := regexp.MustCompile(`(?m)^- \[[ x]\] .*\n`).ReplaceAllString(string(d.Bytes()), "")
	if err := task.AtomicWrite(filepath.Join(m.store.Root, tk.Path), []byte(raw)); err != nil {
		t.Fatal(err)
	}
	m.refresh(tk.Path)
	m = press(m, "esc", "enter")
	if got, _ := m.selected(); len(got.Acceptance) != 0 || m.detailPane != focusPage {
		t.Fatalf("no checks: %d checks, focus %d", len(got.Acceptance), m.detailPane)
	}
	if m = press(m, "tab"); m.detailPane != focusDeps {
		t.Fatalf("no checks: Tab focus %d", m.detailPane)
	}
	if m = press(m, "tab"); m.detailPane != focusPage {
		t.Fatalf("no checks: second Tab focus %d", m.detailPane)
	}
}

// Check 3: with page focus j/k scroll the page and no check or dependency
// shows the pointer or tint; with acceptance focus j/k step through checks.
func TestFocusPageScrollsAcceptanceSteps(t *testing.T) {
	m, _ := focusModel(t)
	m = press(m, "j", "j")
	if m.acceptance != 2 {
		t.Fatalf("acceptance focus: j j -> check %d", m.acceptance)
	}
	m = press(m, "k")
	if m.acceptance != 1 {
		t.Fatalf("acceptance focus: k -> check %d", m.acceptance)
	}
	m = press(m, "tab", "tab") // dependencies, then page
	if m.detailPane != focusPage {
		t.Fatalf("focus %d, want page", m.detailPane)
	}
	for _, line := range strings.Split(detailPlain(m), "\n") {
		if strings.HasPrefix(line, ">") {
			t.Fatalf("page focus shows a pointer: %q", line)
		}
	}
	m.styles.renderer.SetColorProfile(termenv.TrueColor)
	tk, _ := m.selected()
	if tint := m.styles.selected.Render("x"); strings.Contains(m.detailText(tk), strings.TrimSuffix(tint, "x\x1b[0m")) {
		t.Fatal("page focus shows the selected tint")
	}
	before := m.acceptance
	m = press(m, "<")
	top := m.viewport.YOffset
	m = press(m, "j", "j", "down")
	if m.viewport.YOffset != top+3 || m.acceptance != before {
		t.Fatalf("page focus: j j down -> offset %d from %d, check %d", m.viewport.YOffset, top, m.acceptance)
	}
	m = press(m, "k", "up")
	if m.viewport.YOffset != top+1 {
		t.Fatalf("page focus: k up -> offset %d from %d", m.viewport.YOffset, top)
	}
	// Space ticks only the check marked >: with page focus it explains.
	if got := press(m, " "); got.confirm != "" || !strings.Contains(got.message, "Tab focuses the checks") {
		t.Fatalf("Space on the page: confirm %q message %q", got.confirm, got.message)
	}
}

// Check 4: in a task the tab bar marks no view; it shows the trail instead.
func TestFocusTabBarShowsTrailInTask(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	m, main := focusModel(t)
	m.styles = newTheme()
	m.resize()
	bar := func(m model) string { return strings.Split(ansi.Strip(m.View()), "\n")[1] }
	for _, v := range []string{"[overview]", "[list]", "[board]", "[!lint]"} {
		if strings.Contains(bar(m), v) {
			t.Fatalf("tab bar marks %s inside a task: %q", v, bar(m))
		}
	}
	if !strings.Contains(bar(m), "list › "+main.ID) {
		t.Fatalf("tab bar lacks the trail: %q", bar(m))
	}
	page := press(m, "tab", "tab")
	if !strings.Contains(bar(page), "list › ["+main.ID+"]") {
		t.Fatalf("page focus should bracket the task id: %q", bar(page))
	}
	followed := press(m, "tab", "enter")
	if tk, _ := followed.selected(); !strings.Contains(bar(followed), "list › "+main.ID+" › "+tk.ID) {
		t.Fatalf("trail after following a dependency: %q", bar(followed))
	}
	if back := press(m, "esc"); !strings.Contains(bar(back), "[list]") {
		t.Fatalf("leaving the task marks its view again: %q", bar(back))
	}
	if n := len(strings.Split(m.View(), "\n")); n != m.height {
		t.Fatalf("detail view has %d lines, want %d", n, m.height)
	}
}

// Check 5: every confirm prompt is one prominent question: it starts with ▶,
// ends with the question mark and its answer keys, and in colour its style
// differs from plain text.
func TestFocusConfirmPromptsStandOut(t *testing.T) {
	t.Setenv("VISUAL", "")
	t.Setenv("EDITOR", "subl --wait")
	for k := range actionNames {
		m, main := focusModel(t)
		m.styles.renderer.SetColorProfile(termenv.TrueColor)
		switch k {
		case "n":
			m = press(m, "N", "Brand new", "enter")
		case "J", "K":
			// One step moves at once; the opposite key undoes it (3xcesq).
			if m = press(m, k); m.confirm != "" {
				t.Fatalf("%s asked: %q", k, m.confirmLine())
			}
			continue
		default:
			m = press(m, k)
		}
		if m.confirm != k {
			t.Fatalf("%q: no confirm (message %q)", k, m.message)
		}
		lines := strings.Split(m.View(), "\n")
		line := lines[len(lines)-2]
		plain := strings.TrimRight(ansi.Strip(line), " ")
		if !strings.HasPrefix(plain, "▶ ") || !strings.HasSuffix(plain, "? Enter yes · Esc no") {
			t.Fatalf("%q: not a marked question: %q", k, plain)
		}
		if ansi.StringWidth(ansi.Strip(line)) != m.width {
			t.Fatalf("%q: confirm bar is not full width: %d", k, ansi.StringWidth(ansi.Strip(line)))
		}
		if line == m.styles.normal.Render(ansi.Strip(line)) || !strings.Contains(line, "48;2;") || !strings.Contains(line, "\x1b[1") {
			t.Fatalf("%q: confirm line is not styled apart from plain text: %q", k, line)
		}
		switch k {
		case " ":
			if !strings.Contains(plain, "Tick: ☐ 1. one") {
				t.Fatalf("tick wording: %q", plain)
			}
		case "n":
			if !strings.Contains(plain, "Confirm new Brand new?") {
				t.Fatalf("new wording: %q", plain)
			}
		case "e":
			if !strings.Contains(plain, "Edit "+main.ID+" in subl --wait?") {
				t.Fatalf("edit wording: %q", plain)
			}
		default:
			if !strings.Contains(plain, "Confirm "+actionNames[k]+" "+main.ID+" Main?") {
				t.Fatalf("%q wording: %q", k, plain)
			}
		}
	}
	// A long title is cut, never the question or its answer keys.
	m, _ := focusModel(t)
	if help := ansi.Strip(m.helpText()); !strings.Contains(help, "$VISUAL, else $EDITOR, git core.editor, vi") {
		t.Fatalf("help does not say how the editor is chosen:\n%s", help)
	}
	m.width = 40
	m.resize()
	m = press(m, "t")
	lines := strings.Split(ansi.Strip(m.View()), "\n")
	if got := strings.TrimRight(lines[len(lines)-2], " "); !strings.HasSuffix(got, "? Enter yes · Esc no") || !strings.Contains(got, "…") {
		t.Fatalf("narrow confirm lost its question: %q", got)
	}
}

// The detail header shows no empty badge for an unset priority.
func TestDetailHeaderHasNoEmptyBadge(t *testing.T) {
	m, main := focusModel(t)
	main.Priority = ""
	for _, line := range m.detailHeader(main) {
		if strings.Contains(ansi.Strip(line), "[]") {
			t.Fatalf("empty badge: %q", ansi.Strip(line))
		}
	}
}

// The board legend names its columns: "Type: … · Priority: …", with T and
// P emphasised; narrow boards shorten to "T: … · P: …".
func TestBoardLegendNamesColumns(t *testing.T) {
	m, _ := focusModel(t)
	m.width = 140
	if got := ansi.Strip(m.legend()); !strings.HasPrefix(got, "Type: ") || !strings.Contains(got, " · Priority: C critical") {
		t.Fatalf("wide legend: %q", got)
	}
	m.width = 60
	if got := ansi.Strip(m.legend()); !strings.HasPrefix(got, "T: ") || !strings.Contains(got, " · P: C critical") {
		t.Fatalf("narrow legend: %q", got)
	}
	m.styles.renderer.SetColorProfile(termenv.TrueColor)
	m.width = 140
	if got := m.legend(); !strings.Contains(got, m.styles.normal.Bold(true).Underline(true).Render("T")) {
		t.Fatalf("T not emphasised: %q", got)
	}
}

// n moves the list selection down like j; p only pastes (3xcesq); N creates a task.
func TestNAndPMoveInListsAndNCreates(t *testing.T) {
	m, _ := focusModel(t)
	m = press(m, "esc")
	start := m.cursor
	if m = press(m, "n"); m.cursor == start {
		t.Fatal("n did not move down")
	}
	if m = press(m, "p"); m.cursor == start || m.message != "nothing yanked: y on a task first" {
		t.Fatalf("p in a list must only paste: cursor %d message %q", m.cursor, m.message)
	}
	if m = press(m, "N"); m.prompt != "new title" {
		t.Fatalf("N opened %q", m.prompt)
	}
}
