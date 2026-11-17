package tui

import (
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/x/ansi"
	"github.com/veridit/hug/huginn/internal/task"
)

// e3sevr check 7: column 1 holds only the pointer, the grab or yank mark
// and the scroll arrows; the lint mark has its own cell right before the
// id, so ids line up and an arrow never shares a cell with a mark. The
// golden has a waiting-on-you task as the last row shown, under ↓.
func TestGutterHoldsOnlyPointerAndArrows(t *testing.T) {
	m := stageBoard(t)
	m.opts.User = "jhf" // eeeeee's design question is for jhf
	m.refresh("")
	var golden []string
	for _, w := range []int{80, 140} {
		g := m
		g.width, g.height = w, 14 // short: WAITING is cut, ↓ on its last row
		g.resize()
		g.setView("overview")
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
		waiting := strings.Split(section(screen, "WAITING"), "\n")
		last := waiting[len(waiting)-1]
		if !strings.HasPrefix(last, "↓") || strings.Contains(last, "?") {
			t.Fatalf("at %d WAITING's last shown row has no ↓, or a ?: %q\n%s", w, last, screen)
		}
		if !strings.Contains(waiting[1], "eeeeee") {
			t.Fatalf("at %d the question for jhf is not first in WAITING:\n%s", w, screen)
		}
		golden = append(golden, "== overview at "+strconv.Itoa(w)+"x14, jhf", screen)
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
		want := []string{"j/k scroll", "e edit", "x drop", "y yank"}
		if tk.State == "doing" {
			want = append(want, "d done", "w wait")
		} else {
			want = append(want, "t take")
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
		want := "HUG TASKS · list › " + ids["Main"]
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
			for _, global := range []string{"[list]", "f picker", "No filters"} {
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
	header := regexp.MustCompile(`(?m)^\s+✓\s+ID\s`)
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
