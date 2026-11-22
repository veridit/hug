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
			case "n":
				if after = press(fresh, "n"); func() bool { tk, _ := after.selected(); return tk.Title != "Plain" }() {
					t.Fatal("n did not go to the next task")
				}
			case "p":
				after = press(fresh, "n")
				if tk, _ := press(after, "p").selected(); tk.Title != "Main" {
					t.Fatalf("p went to %q", tk.Title)
				}
			case "h/l Tab":
				// One ring (e3sevr): l and Tab land on the same stop, h
				// and Shift-Tab too.
				l, tab := press(fresh, "l"), press(fresh, "tab")
				if l.detailPane == pane && l.dependency == fresh.dependency || l.detailPane != tab.detailPane || l.dependency != tab.dependency || l.idAt != tab.idAt {
					t.Fatalf("l and Tab differ or do not move: l %d/%d tab %d/%d", l.detailPane, l.dependency, tab.detailPane, tab.dependency)
				}
				if h, st := press(fresh, "h"), press(fresh, "shift+tab"); h.detailPane != st.detailPane || h.dependency != st.dependency {
					t.Fatalf("h and Shift-Tab differ: %d %d", h.detailPane, st.detailPane)
				}
				after = l
			case "Esc":
				if after = press(fresh, "esc"); after.detail {
					t.Fatal("Esc did not leave the task")
				}
			case "u":
				if after = press(fresh, "u"); after.detail {
					t.Fatal("u did not go up out of the task")
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
			case "J/K":
				if after = press(fresh, "J"); after.viewport.YOffset <= before.viewport.YOffset || !after.detail {
					t.Fatalf("J shown as half page but did not scroll: %d", after.viewport.YOffset)
				}
			case "e", "t", "x", "d":
				// On the task itself each writer has its own hint (e3sevr
				// check 9) and asks, prompts or says why not.
				if after = press(fresh, name); after.confirm == "" && after.prompt == "" && after.message == "" {
					t.Fatalf("%s shown on the task but did nothing", name)
				}
			case "w":
				if after = press(fresh, "w"); after.prompt != waitPrompt && after.message == "" {
					t.Fatalf("w shown but no wait prompt: %q", after.message)
				}
			case "y":
				if after = press(fresh, "y"); after.yanked == "" {
					t.Fatalf("y shown but nothing yanked: %q", after.message)
				}
			case "t/d/x/e":
				// Each writes only after a confirm or a prompt (x1ra0t).
				for _, w := range []string{"t", "d", "x", "e"} {
					after = press(fresh, w)
					if after.confirm == "" && after.prompt == "" && after.message == "" {
						t.Fatalf("%s changed something without asking or saying why not", w)
					}
				}
			case "r":
				if after = press(fresh, "r"); after.prompt != requestPrompt {
					t.Fatalf("r shown but opened no request box: %q", after.message)
				}
			case "a":
				if after = press(fresh, "a"); after.prompt == "" && after.message == "" {
					t.Fatal("a shown but opened no answer input")
				}
			case "S":
				// S copies at once and asks where it goes before it attaches
				// (5z4zt4, hjbfzd).
				if after = press(fresh, "S"); after.prompt != capturePrompt {
					t.Fatalf("S did not ask where to attach: %q", after.message)
				}
			case "/":
				if after = press(fresh, "/"); after.prompt != findPrompt {
					t.Fatalf("/ shown but opened no find: %q", after.prompt)
				}
			case "N":
				if after = press(fresh, "N"); after.prompt != "new title" {
					t.Fatalf("N did not open the new title prompt: %q", after.prompt)
				}
			case "-":
				if after = press(fresh, "-"); after.prompt != strikePrompt && after.confirm != "-" && !strings.Contains(after.message, "ticked") {
					t.Fatalf("- did not ask to strike out: prompt %q message %q", after.prompt, after.message)
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
	deps := press(m, "l")
	if !strings.Contains(detailPlain(deps), "DEPENDENCIES · j/k or [ ] step · Enter follows") {
		t.Fatalf("dependency heading:\n%s", detailPlain(deps))
	}
}

// Check 2: l and h (and Tab, Shift-Tab: e3sevr) cycle the ring, page,
// acceptance and each dependency, skip a section that is empty, and only
// the focused heading is highlighted.
func TestFocusTabCyclesAndHighlightsOneHeading(t *testing.T) {
	m, _ := focusModel(t)
	if m.detailPane != focusAcceptance {
		t.Fatalf("a task with checks opens on them: %d", m.detailPane)
	}
	want := []int{focusDeps, focusPage, focusAcceptance}
	for _, w := range want {
		m = press(m, "l")
		if m.detailPane != w {
			t.Fatalf("l: focus %d, want %d", m.detailPane, w)
		}
	}
	for _, w := range []int{focusPage, focusDeps, focusAcceptance} {
		m = press(m, "h")
		if m.detailPane != w {
			t.Fatalf("h: focus %d, want %d", m.detailPane, w)
		}
	}
	// Exactly one place shows focus: one heading, or the task id.
	for _, pane := range []int{focusAcceptance, focusDeps, focusPage} {
		m.detailPane = pane
		text := detailPlain(m)
		heads := strings.Count(text, "▸ ")
		trail := strings.Contains(ansi.Strip(strings.Join(m.detailHeader(m.visible[m.cursor]), "\n")), "▸ Main")
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
	if !strings.Contains(coloured, m.sectionHeading("Acceptance", "j/k or [ ] step · Space ticks · - strikes out", true)) || strings.Contains(coloured, m.sectionHeading("Dependencies", "j/k or [ ] step · Enter follows", true)) {
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
		plain = press(plain, "l")
		if plain.detailPane != w {
			t.Fatalf("no dependencies: l focus %d, want %d", plain.detailPane, w)
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
	if m = press(m, "l"); m.detailPane != focusDeps {
		t.Fatalf("no checks: l focus %d", m.detailPane)
	}
	if m = press(m, "l"); m.detailPane != focusPage {
		t.Fatalf("no checks: second l focus %d", m.detailPane)
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
	m = press(m, "l", "l") // dependencies, then page
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
	if got := press(m, " "); got.confirm != "" || !strings.Contains(got.message, "l focuses the checks") {
		t.Fatalf("Space on the page: confirm %q message %q", got.confirm, got.message)
	}
}

// Check 4 (e3sevr check 12, header-spec.md): in a task the global band
// is gone; line 1 is the breadcrumb, the view u goes back to, then each
// task followed, then this task's id; the subject (title) carries ▸ on
// page focus.
func TestFocusTabBarShowsTrailInTask(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	m, main := focusModel(t)
	m.styles = newTheme()
	m.resize()
	crumb := func(m model) string { return strings.TrimRight(strings.Split(ansi.Strip(m.View()), "\n")[0], " ") }
	for _, v := range []string{"view: ", "!lint", "/3 tasks", "todo 3", "filter: none"} {
		if strings.Contains(ansi.Strip(m.View()), v) {
			t.Fatalf("the global band shows %q inside a task:\n%s", v, ansi.Strip(m.View()))
		}
	}
	if got := crumb(m); got != "HUG · list › state:!done › "+main.ID { // the list's default chip (7zg9mf)
		t.Fatalf("breadcrumb %q", got)
	}
	page := press(m, "l", "l")
	if !strings.Contains(ansi.Strip(page.View()), "▸ Main") || strings.Contains(crumb(page), "▸") {
		t.Fatalf("page focus: ▸ before the subject, not in the breadcrumb:\n%s", ansi.Strip(page.View()))
	}
	followed := press(m, "l", "enter")
	if tk, _ := followed.selected(); crumb(followed) != "HUG · list › state:!done › "+main.ID+" › "+tk.ID {
		t.Fatalf("breadcrumb after following a dependency: %q", crumb(followed))
	}
	if back := press(m, "esc"); !strings.Contains(ansi.Strip(back.View()), "view: ") {
		t.Fatalf("leaving the task draws the global band again:\n%s", ansi.Strip(back.View()))
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
		case "d":
			// done only from doing (6ephqk); Main depends on an open
			// task, so the doing task here is its dependency.
			dep := m.all[len(m.all)-1]
			for _, tk := range m.all {
				if tk.Title == "Dependency" {
					dep = tk
				}
			}
			if _, err := m.store.Take(dep.ID, "tester"); err != nil {
				t.Fatal(err)
			}
			m.refresh(dep.Path)
			m = press(m, "esc")
			m = selectRow(t, m, dep.ID)
			m = press(m, k)
		case "x":
			m = press(m, k, "superseded", "enter") // the reason first
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
		if m.pageFramed() {
			// In a task the confirm sits on the page's bottom rule, which
			// stays on the keys (4ddcwa).
			if rule := ansi.Strip(line); !strings.HasPrefix(rule, "↓── ") && !strings.HasPrefix(rule, "─── ") {
				t.Fatalf("%q: the confirm is not right above the bottom rule: %q", k, rule)
			}
			line = lines[len(lines)-3]
		}
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
		case "d":
			if !strings.Contains(plain, "Confirm done ") || !strings.Contains(plain, " Dependency?") {
				t.Fatalf("done wording: %q", plain)
			}
		case "x":
			if !strings.Contains(plain, "Drop "+main.ID+": superseded?") {
				t.Fatalf("drop wording: %q", plain)
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
	confirmRow := len(lines) - 2
	if m.pageFramed() {
		confirmRow-- // over the page's bottom rule (4ddcwa)
	}
	if got := strings.TrimRight(lines[confirmRow], " "); !strings.HasSuffix(got, "? Enter yes · Esc no") || !strings.Contains(got, "…") {
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
