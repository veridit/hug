package tui

import (
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"
	"unicode"

	bkey "github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/x/ansi"
	"github.com/muesli/termenv"
	"github.com/veridit/hug/huginn/internal/task"
)

// e3sevr check 7: column 1 holds only the pointer, the grab or yank mark
// and the scroll arrows; the lint mark has its own cell right before the
// id, so ids line up and an arrow never shares a cell with a mark. The
// golden has a waiting-on-you task as the last row shown, under ↓: since
// 41hhgn those are in ASKED, cut short with the cursor elsewhere.
func TestGutterHoldsOnlyPointerAndArrows(t *testing.T) {
	m := stageBoard(t)
	m.opts.User = "jhf" // eeeeee's design question is for jhf
	// Two more questions for jhf: three in ASKED, more than a short
	// screen shows.
	if _, err := m.store.Ask("cccccc", task.Question{What: "approve the look", For: "jhf", Check: 2}, "nautilus"); err != nil {
		t.Fatal(err)
	}
	if _, err := m.store.Ask("aaaaaa", task.Question{What: "which colour for asked", For: "jhf"}, "nautilus"); err != nil {
		t.Fatal(err)
	}
	m.refresh("")
	var golden []string
	for _, w := range []int{80, 140} {
		g := m
		g.width, g.height = w, 18 // short: ASKED is cut, ↓ on its last row; every title fits
		g.resize()
		g.setView("overview")
		g.pane = sectionNext // the cursor elsewhere: ASKED shows its first rows
		g.clamp()
		screen := ansi.Strip(g.View())
		rows := strings.Split(screen, "\n")
		ids := regexp.MustCompile(`\b(aaaaaa|bbbbbb|cccccc|dddddd|eeeeee|ffffff|gggggg)\b`)
		col := -1
		for _, line := range rows {
			loc := ids.FindStringIndex(line)
			if loc == nil || !strings.Contains(line, "feature") {
				continue
			}
			// Cell 1 is the pointer, an arrow or blank, nothing else.
			if first := []rune(line)[0]; !strings.ContainsRune(" >↑↓≡y", first) {
				t.Fatalf("at %d the gutter holds %q:\n%s", w, first, screen)
			}
			// Every id starts in the same cell.
			at := len([]rune(line[:loc[0]]))
			if col < 0 {
				col = at
			}
			if at != col {
				t.Fatalf("at %d the ids do not line up (%d, %d):\n%s", w, col, at, screen)
			}
		}
		asked := strings.Split(section(screen, "ASKED"), "\n")
		last := asked[len(asked)-1]
		// The ↓ is on the section's end rule, right under its last row
		// (v68n82), never in the row's gutter.
		end := strings.Split(screen[strings.Index(screen, strings.Join(asked, "\n"))+len(strings.Join(asked, "\n"))+1:], "\n")[0]
		if !strings.HasPrefix(end, "↓──") || strings.HasPrefix(last, "↓") || strings.Contains(last, "?") || !strings.Contains(asked[0], "shown") {
			t.Fatalf("at %d ASKED is not cut with ↓ on its end rule, or its last row holds a ? or ↓: %q, %q\n%s", w, last, end, screen)
		}
		golden = append(golden, "== overview at "+strconv.Itoa(w)+"x18, jhf", screen)
	}
	goldenFile(t, filepath.Join("testdata", "rows", "gutter.txt"), strings.Join(golden, "\n"))
}

// e3sevr check 9: on the page stop (the task itself) the footer lists the
// task-wide keys the task's state allows, each does what it says, and no
// stop on the ring has an empty footer.
func TestPageFocusListsTaskKeys(t *testing.T) {
	m := stageBoard(t)
	m.width, m.height = 140, 30
	m.resize()
	for _, id := range []string{"aaaaaa", "ffffff"} { // doing, todo
		d := openID(t, m, id)
		tk, _ := d.selected()
		for _, s := range d.focusRing(tk) {
			if s.pane != focusPage {
				continue
			}
			d.detailPane = focusPage
		}
		if d.detailPane != focusPage {
			t.Fatalf("%s: no page stop on the ring", id)
		}
		bar := ansi.Strip(d.keyBar())
		// A letter key is marked in its word (check 14): "edit" with e
		// marked; x keeps its label, drop has no x.
		want := []string{"j/k scroll", "edit", "x drop", "yank"}
		if tk.State == "doing" {
			want = append(want, "done", "wait")
		} else {
			want = append(want, "take")
		}
		for _, k := range want {
			if !strings.Contains(bar, k) {
				t.Fatalf("%s (%s) page focus: footer lacks %q: %q", id, tk.State, k, bar)
			}
		}
		for _, k := range want {
			letter := strings.Fields(k)[0]
			if letter == "j/k" {
				continue
			}
			if len(letter) > 1 { // marked in its word: the key is its first letter
				letter = letter[:1]
			}
			after := press(d, letter)
			switch letter {
			case "y":
				if after.yanked != tk.ID {
					t.Fatalf("%s: y did not yank: %q", id, after.message)
				}
			case "w":
				if after.prompt != waitPrompt {
					t.Fatalf("%s: w did not ask what it waits on: %q", id, after.message)
				}
				after = press(after, "CI run 1", "enter")
				if got, _ := after.store.Get(id); got.Task.Stage != "waiting" || got.Task.Fields["waiting_on"] != "CI run 1" {
					t.Fatalf("%s: w did not park it: stage %q on %q (%s)", id, got.Task.Stage, got.Task.Fields["waiting_on"], after.message)
				}
			default:
				if after.confirm == "" && after.prompt == "" && after.message == "" {
					t.Fatalf("%s: %q did nothing from the page stop", id, letter)
				}
			}
		}
		// No stop shows an empty footer.
		for i, s := range d.focusRing(tk) {
			at := d
			at.detailPane = s.pane
			if bar := strings.TrimSpace(ansi.Strip(at.keyBar())); bar == "" {
				t.Fatalf("%s stop %d (pane %d): empty footer", id, i, s.pane)
			}
		}
	}
}

// e3sevr check 13 (header-spec.md): the detail's bands in order:
// breadcrumb, context, a dim rule, the subject, a dim rule, the details;
// no global band. One task open and two stacked, at 80 and 140.
func TestDetailBandsInOrder(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	// The store's clock is pinned: History wraps a stamp across lines,
	// where the date mask cannot reach.
	oldTask, oldNow := task.Now, now
	task.Now = func() time.Time { return time.Date(2026, 11, 17, 17, 50, 0, 0, time.Local) }
	now = func() time.Time { return time.Date(2026, 11, 17, 17, 53, 0, 0, time.Local) }
	t.Cleanup(func() { task.Now, now = oldTask, oldNow })
	m, ids := idBoard(t)
	m.styles = newTheme()
	rule := regexp.MustCompile(`^─+$`)
	var golden []string
	for _, stacked := range []bool{false, true} {
		d := openID(t, m, ids["Main"])
		want := "HUG · list › " + ids["Main"]
		title := "Main"
		if stacked {
			d = press(d, "tab", "enter") // the dependency, from the checks
			want += " › " + ids["Dep"]
			title = "Dep"
		}
		for _, w := range []int{80, 140} {
			g := d
			g.width, g.height = w, 30
			g.resize()
			g.syncDetail()
			lines := strings.Split(ansi.Strip(g.View()), "\n")
			if got := strings.TrimRight(lines[0], " "); !strings.HasPrefix(got, want) {
				t.Fatalf("breadcrumb %q, want %q", got, want)
			}
			var rules []int
			for i, l := range lines {
				if rule.MatchString(strings.TrimRight(l, " ")) {
					rules = append(rules, i)
				}
			}
			if len(rules) < 2 || rules[1] != rules[0]+2 || strings.TrimSpace(lines[rules[0]+1]) != title {
				t.Fatalf("at %d: no rule, subject %q, rule:\n%s", w, title, strings.Join(lines, "\n"))
			}
			if !strings.Contains(strings.Join(lines[1:rules[0]], "\n"), "created ") {
				t.Fatalf("at %d: the context band (dates) is not between the breadcrumb and the first rule:\n%s", w, strings.Join(lines, "\n"))
			}
			for _, global := range []string{"view: ", "filter: none", "sort: rank"} {
				if strings.Contains(strings.Join(lines, "\n"), global) {
					t.Fatalf("at %d the global band shows %q", w, global)
				}
			}
			name := "one task"
			if stacked {
				name = "two stacked"
			}
			golden = append(golden, "== "+name+" at "+strconv.Itoa(w), strings.Join(lines, "\n"))
		}
	}
	text := strings.NewReplacer(idsReplacer(ids)...).Replace(strings.Join(golden, "\n"))
	text = regexp.MustCompile(`\d{4}-\d\d-\d\d(T\d\d:\d\d)?`).ReplaceAllStringFunc(text, func(d string) string { return strings.Repeat("#", len(d)) })
	text = regexp.MustCompile(`since \d\d:\d\d`).ReplaceAllString(text, "since ##:##")
	goldenFile(t, filepath.Join("testdata", "detail", "bands.txt"), text)
}

// Jørgen: an empty section, focused with a column marked, draws no column
// header: there is nothing under it. A section with rows still does.
func TestEmptyFocusedSectionHasNoColumnHeader(t *testing.T) {
	m := stageBoard(t)
	m.width, m.height = 100, 30
	m.resize()
	m.setView("overview")
	m.column = 3 // a marked column draws the focused section's header
	m.pane = sectionWaiting
	header := regexp.MustCompile(`(?m)^\s+ID\s+TITLE\s`) // 911j70: the id leads
	if screen := ansi.Strip(m.View()); !header.MatchString(screen) {
		t.Fatalf("WAITING has rows and a marked column, yet no header:\n%s", screen)
	}
	empty := -1
	for p := range overviewSections {
		if p != sectionLint && len(m.paneIndices(p)) == 0 {
			empty = p
		}
	}
	if empty < 0 {
		t.Fatal("the board has no empty section")
	}
	m.pane = empty
	if screen := ansi.Strip(m.View()); header.MatchString(screen) {
		t.Fatalf("empty %s focused draws a column header:\n%s", overviewSections[empty], screen)
	}
}

// goldenFile compares text with path, writing it first under
// HUG_UPDATE_GOLDEN=1; \r\n in the file reads as \n.
func goldenFile(t *testing.T, path, text string) {
	t.Helper()
	if os.Getenv("HUG_UPDATE_GOLDEN") == "1" {
		_ = os.MkdirAll(filepath.Dir(path), 0o755)
		if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("%v\nfix: HUG_UPDATE_GOLDEN=1 go test -run %s ./huginn/internal/tui/", err, t.Name())
	}
	if text != strings.ReplaceAll(string(want), "\r\n", "\n") {
		t.Fatalf("differs from %s:\n%s", path, text)
	}
}

// e3sevr check 14 (Jørgen 00:44; crab 00:10): a letter key is marked
// inside its action word, never written as a label beside a description
// that holds its letter anywhere ("n next", "o column/order", "f find"):
// a combined hint is split so each key marks its own word. Checked on the
// key bar's items in every help context at 80 and 140, and on the plain
// footer and status lines for "k kword". The mark is accent, underlined.
func TestFooterMarksTheKeyInItsWord(t *testing.T) {
	repeat := regexp.MustCompile(`(?:^|[ ·,:(])([A-Za-z]) ([A-Za-z])`)
	check := func(where, text string) {
		t.Helper()
		for _, m := range repeat.FindAllStringSubmatch(text, -1) {
			if strings.EqualFold(m[1], m[2]) {
				t.Fatalf("%s repeats the key %q before its word:\n%s", where, m[1], text)
			}
		}
	}
	ctx := helpContexts(t)
	for name, m := range ctx {
		for _, w := range []int{80, 140} {
			m.width = w
			m.resize()
			lines := strings.Split(plainMarks(ansi.Strip(m.View())), "\n")
			if len(lines) >= 3 {
				check(name+" footer", lines[len(lines)-3])
			}
			check(name+" key bar", lines[len(lines)-1])
			// A one-letter key shown as a label: its letter is nowhere in
			// its words (else it is marked there). A pair such as j/k or
			// H/L moves both ways and keeps its label (crab 00:10).
			for _, it := range itemsOf(m.barHints()) {
				if r := []rune(it.label); len(r) == 1 && unicode.IsLetter(r[0]) && strings.ContainsRune(strings.ToLower(plainMarks(it.words)), unicode.ToLower(r[0])) {
					t.Fatalf("%s at %d: key %q is a label beside %q, which holds its letter", name, w, it.label, plainMarks(it.words))
				}
			}
		}
	}
	// The marked letter is styled as a key: accent and underlined.
	m := ctx["task:checks"]
	m.styles.renderer.SetColorProfile(termenv.TrueColor)
	t.Cleanup(func() { m.styles.renderer.SetColorProfile(termenv.Ascii) })
	bar := m.keyBar()
	key := m.styles.accent.Underline(true).Render("e")
	if !strings.Contains(bar, key) || !strings.Contains(ansi.Strip(bar), "edit") {
		t.Fatalf("e is not marked inside edit: %q", bar)
	}
}

// e3sevr check 16 (Jørgen 00:45): neighbouring hints that share their last
// word are one item: the earlier end in '-', ', ' joins them, ' · '
// separates items. The file viewer reads 'next-, previous file'.
func TestFooterGroupsHintsSharingAWord(t *testing.T) {
	m := manyModel(t)
	got := plainMarks(ansi.Strip(m.renderHints([]bkey.Binding{hint("j/k", "scroll"), hint("n", "next file"), hint("p", "previous file"), hint("Enter", "edit"), hint("u/Esc", "files")}, 200)))
	if want := "j/k scroll · next-, previous file · Enter edit · u/Esc files"; got != want {
		t.Fatalf("grouped hints %q, want %q", got, want)
	}
	// Only neighbours: the same last word apart stays apart.
	got = plainMarks(ansi.Strip(m.renderHints([]bkey.Binding{hint("n", "next file"), hint("Enter", "edit"), hint("p", "previous file")}, 200)))
	if want := "next file · Enter edit · previous file"; got != want {
		t.Fatalf("apart %q, want %q", got, want)
	}
	// Narrow: whole items drop, ending in …
	if got := ansi.Strip(m.renderHints([]bkey.Binding{hint("j/k", "scroll"), hint("n", "next file"), hint("p", "previous file")}, 20)); got != "j/k scroll …" {
		t.Fatalf("narrow %q", got)
	}
}
