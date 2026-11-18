package tui

import (
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/muesli/termenv"
	"github.com/veridit/hug/huginn/internal/task"
)

// rxr4z8: in lists Space marks tasks and the footer acts on the marked.

// markBoard is moveBoard in the list, six tasks A..F in rank order, its
// ids in that order.
func markBoard(t *testing.T) (model, []string, *int) {
	t.Helper()
	return grabBoard(t)
}

// gutter is the first cell of the row that shows id, plain, and the rest
// of the row as drawn (styles included).
func gutter(t *testing.T, m model, id string) (string, string) {
	t.Helper()
	for _, line := range strings.Split(m.View(), "\n") {
		plain := ansi.Strip(line)
		at := strings.Index(plain, id+" ")
		if at > 0 && at < 20 && strings.HasPrefix(plain[len(string([]rune(plain)[0])):], " ") {
			r := []rune(plain)
			rest := ansi.Cut(line, 1, ansi.StringWidth(line))
			// Cut keeps the gutter's own style codes, now empty: drop them,
			// so the rest is the row after its first cell.
			rest = emptySGR.ReplaceAllString(rest, "")
			return string(r[0]), rest
		}
	}
	t.Fatalf("%s not on screen:\n%s", id, screen(m))
	return "", ""
}

// emptySGR is a style that styles nothing: a code straight followed by a
// reset.
var emptySGR = regexp.MustCompile(`\x1b\[[0-9;]*m\x1b\[0m`)

// keyBarText is the key bar plain, marks dropped.
func keyBarText(m model) string { return ansi.Strip(m.keyBar()) }

// footerText is the footer line above the input and key bar, plain.
func footerText(m model) string {
	lines := strings.Split(m.View(), "\n")
	return strings.TrimRight(ansi.Strip(lines[len(lines)-3]), " ")
}

// Check 1: Space marks the focused task and steps to the next row; again
// unmarks it. A marked row shows ◆ in its gutter and nothing else changes.
func TestSpaceMarksAndSteps(t *testing.T) {
	for _, view := range []string{"list", "overview", "board"} {
		t.Run(view, func(t *testing.T) {
			m, o, commits := markBoard(t)
			if view == "overview" {
				m = nextUp(m)
			} else {
				m.setView(view)
			}
			m.clamp()
			rows := shown(m)
			m = selectRow(t, m, rows[0])
			m = press(m, " ")
			if !m.isMarked(rows[0]) {
				t.Fatalf("Space did not mark %s", rows[0])
			}
			if sel, _ := m.selected(); sel.ID != rows[1] {
				t.Fatalf("Space did not step to the next row: on %s, want %s", sel.ID, rows[1])
			}
			if g, _ := gutter(t, m, rows[0]); g != markSign {
				t.Fatalf("marked row's gutter %q, want %q:\n%s", g, markSign, screen(m))
			}
			// The last row: Space marks it and stays.
			m = selectRow(t, m, rows[len(rows)-1])
			m = press(m, " ")
			if sel, _ := m.selected(); !m.isMarked(rows[len(rows)-1]) || sel.ID != rows[len(rows)-1] {
				t.Fatalf("Space on the last row: marked %v, on %s", m.isMarked(rows[len(rows)-1]), sel.ID)
			}
			// Space again unmarks.
			m = selectRow(t, m, rows[0])
			m = press(m, " ")
			if m.isMarked(rows[0]) {
				t.Fatal("Space on a marked row did not unmark it")
			}
			if *commits != 0 {
				t.Fatalf("marking wrote: %d commits", *commits)
			}
			_ = o
		})
	}
}

// Check 1 with crab's ruling (20:31): the mark is ◆ in the gutter only. No
// background on the row, and the cells after the gutter are byte-identical
// marked and unmarked, in colour.
func TestMarkIsTheGutterOnly(t *testing.T) {
	m, o, _ := markBoard(t)
	m.styles = newTheme()
	m.styles.renderer.SetColorProfile(termenv.TrueColor)
	m.styles.renderer.SetHasDarkBackground(true)
	m = selectRow(t, m, o[0])
	_, before := gutter(t, m, o[2])
	m = press(m, "j", "j", " ") // o[2] marked, cursor on o[3]
	g, after := gutter(t, m, o[2])
	if g != markSign {
		t.Fatalf("gutter %q, want %s", g, markSign)
	}
	if after != before {
		t.Fatalf("a marked row changed beyond its gutter:\nbefore %q\nafter  %q", before, after)
	}
	for _, line := range strings.Split(m.View(), "\n") {
		if strings.Contains(ansi.Strip(line), o[2]) && strings.Contains(line, "48;2") {
			t.Fatalf("a marked row has a background: %q", line)
		}
	}
}

// Check 7: the cursor on a marked row shows one sign, cursorOnMark, not >
// or ◆ alone.
func TestCursorOnMarkedRow(t *testing.T) {
	m, o, _ := markBoard(t)
	m = selectRow(t, m, o[0])
	m = press(m, " ", "k")
	if g, _ := gutter(t, m, o[0]); g != cursorOnMark {
		t.Fatalf("cursor on a marked row: %q, want %q", g, cursorOnMark)
	}
	if g, _ := gutter(t, m, o[1]); g != " " {
		t.Fatalf("an unmarked row without the cursor: %q", g)
	}
	m = press(m, "j")
	if g, _ := gutter(t, m, o[1]); g != ">" {
		t.Fatalf("cursor on an unmarked row: %q", g)
	}
}

// Check 2: with N marked the footer reads "N marked ·", the actions valid
// for all of them, "Esc unmark"; the section says it too; Esc clears all.
func TestMarkedFooterAndEsc(t *testing.T) {
	m, o, _ := markBoard(t)
	m = selectRow(t, m, o[0])
	m = press(m, " ", "j", " ") // o[0] and o[2]: both todo
	f := plainMarks(footerText(m))
	if !strings.HasPrefix(f, "2 marked · ") || !strings.HasSuffix(f, "Esc unmark") {
		t.Fatalf("footer with 2 marked: %q", f)
	}
	for _, want := range []string{"move", "drop", "edit"} {
		if !strings.Contains(f, want) {
			t.Fatalf("footer %q lacks %q", f, want)
		}
	}
	// d on todo tasks is refused (take before done) and w needs doing:
	// neither is offered.
	for _, not := range []string{"done", "wait"} {
		if strings.Contains(f, not) {
			t.Fatalf("footer %q offers %q, which every marked task refuses", f, not)
		}
	}
	if !strings.Contains(keyBarText(m), "Space mark") {
		t.Fatalf("key bar lacks Space mark: %q", keyBarText(m))
	}
	m = press(m, "esc")
	if n := len(m.marks); n != 0 {
		t.Fatalf("Esc left %d marks", n)
	}
	if strings.Contains(footerText(m), " marked ·") {
		t.Fatalf("footer after Esc: %q", footerText(m))
	}
	if !strings.Contains(m.message, "unmarked 2") {
		t.Fatalf("Esc says %q", m.message)
	}
}

// Check 2: the overview's section heading counts the marks, so one
// scrolled out of sight is not forgotten.
func TestMarkedCountInTheHeading(t *testing.T) {
	m, _, _ := markBoard(t)
	m = nextUp(m)
	m.clamp()
	m = press(m, " ", " ")
	if !strings.Contains(screen(m), "· 2 marked") {
		t.Fatalf("NEXT UP heading lacks the count:\n%s", screen(m))
	}
}

// t with 2+ marked (byj479, crab 21:11): one confirm naming them, then the
// first is taken and the rest are taken with it (doing_with: the first),
// one commit each. With one marked, t takes it after one confirm.
func TestTakeMarked(t *testing.T) {
	m, o, commits := markBoard(t)
	m = selectRow(t, m, o[0])
	m = press(m, " ", " ")
	if f := plainMarks(footerText(m)); !strings.Contains(f, "take") {
		t.Fatalf("footer with 2 todo marked does not offer take: %q", f)
	}
	m = press(m, "t")
	if q := ansi.Strip(m.confirmLine()); !strings.Contains(q, "Take 2 tasks: "+o[0]+", "+o[1]+", "+o[1]+" with "+o[0]) {
		t.Fatalf("t with 2 marked: confirm %q, message %q", q, m.message)
	}
	m = press(m, "enter")
	first, _ := m.taskAt(pathOf(t, m, o[0]))
	second, _ := m.taskAt(pathOf(t, m, o[1]))
	if first.State != "doing" || second.State != "doing" || task.TakenWith(second) != o[0] {
		t.Fatalf("after Enter: %s %s, %s %s with %q; message %q", o[0], first.State, o[1], second.State, task.TakenWith(second), m.message)
	}
	if *commits != 2 || len(m.marks) != 0 || m.message != "took 2 of 2" {
		t.Fatalf("commits %d marks %v message %q", *commits, m.marks, m.message)
	}
	// A doing task among the marked: take is not offered for them all.
	m = selectRow(t, m, o[0])
	m = press(m, " ")
	m = selectRow(t, m, o[2])
	m = press(m, " ")
	if f := plainMarks(footerText(m)); strings.Contains(f, "take") {
		t.Fatalf("take offered with a doing task marked: %q", f)
	}
	m = press(m, "t")
	if m.confirm != "" || !strings.Contains(m.message, o[0]+" is doing") {
		t.Fatalf("t with a doing task marked: confirm %q message %q", m.confirm, m.message)
	}
	m = press(m, "esc")
	m = selectRow(t, m, o[3])
	m = press(m, " ", "t")
	if !strings.Contains(ansi.Strip(m.View()), "Take 1 task: "+o[3]) {
		t.Fatalf("t with 1 marked does not confirm naming it:\n%s", screen(m))
	}
}

// The first marked task refused (busy: the taker is on another one): the
// rest are not taken, all stay marked, and the fix: line is the refusal's.
func TestTakeMarkedFirstRefused(t *testing.T) {
	m, o, commits := markBoard(t)
	if _, err := m.store.Take(o[5], m.me()); err != nil { // tester is on F
		t.Fatal(err)
	}
	m.refresh("")
	m = selectRow(t, m, o[0])
	m = press(m, " ", " ", "t", "enter")
	for _, id := range o[:2] {
		if got, _ := m.taskAt(pathOf(t, m, id)); got.State != "todo" {
			t.Fatalf("%s is %s, want todo", id, got.State)
		}
	}
	if *commits != 0 || len(m.marks) != 2 {
		t.Fatalf("commits %d marks %v", *commits, m.marks)
	}
	if !strings.HasPrefix(m.message, "took 0 of 2 · "+o[0]+" refused: busy") || !strings.Contains(m.message, o[1]+" not taken: "+o[0]+" was not") || !strings.Contains(m.message, "\nfix: hug task done "+o[5]) {
		t.Fatalf("message %q", m.message)
	}
}

func pathOf(t *testing.T, m model, id string) string {
	t.Helper()
	for _, v := range m.all {
		if v.ID == id {
			return v.Path
		}
	}
	t.Fatalf("no task %s", id)
	return ""
}

// doingBoard is three doing tasks, the second with an open check; its
// ids in rank order.
func doingBoard(t *testing.T) (model, []string, *int) {
	t.Helper()
	m, o, commits := markBoard(t)
	for i, id := range o[:3] {
		if i != 1 {
			if _, err := m.store.Check(id, 1); err != nil {
				t.Fatal(err)
			}
		}
		if _, err := m.store.Take(id, "tester"+string(rune('a'+i))); err != nil { // one doing per name (n2hq8p)
			t.Fatal(err)
		}
	}
	m.refresh("")
	m.setView("list")
	return m, o, commits
}

// Checks 3 and 9: d on 3 marked asks once, naming the count and the ids;
// Enter marks 2 done, names the refused one with its fix: line, and only
// the refused one stays marked.
func TestDoneMarkedOneRefused(t *testing.T) {
	m, o, commits := doingBoard(t)
	m = selectRow(t, m, o[0])
	m = press(m, " ", " ", " ")
	if len(m.marks) != 3 {
		t.Fatalf("marked %d", len(m.marks))
	}
	f := plainMarks(footerText(m))
	if !strings.Contains(f, "done") || !strings.Contains(f, "wait") || strings.Contains(f, "take") {
		t.Fatalf("footer with 3 doing marked: %q", f)
	}
	m = press(m, "d")
	confirm := ansi.Strip(m.confirmLine())
	if !strings.Contains(confirm, "Done 3 tasks: "+o[0]+", "+o[1]+", "+o[2]) {
		t.Fatalf("confirm %q", confirm)
	}
	if *commits != 0 {
		t.Fatal("wrote before the confirm")
	}
	m = press(m, "enter")
	for i, id := range o[:3] {
		got, _ := m.taskAt(pathOf(t, m, id))
		want := "done"
		if i == 1 {
			want = "doing"
		}
		if got.State != want {
			t.Fatalf("%s is %s, want %s", id, got.State, want)
		}
	}
	if *commits != 2 {
		t.Fatalf("commits %d, want 2 (one per task that went through)", *commits)
	}
	lines := strings.Split(m.message, "\n")
	if !strings.HasPrefix(lines[0], "marked done 2 of 3 · "+o[1]+" refused: ") {
		t.Fatalf("message %q", m.message)
	}
	if !strings.Contains(m.message, "\nfix: hug task check "+o[1]+" 1") {
		t.Fatalf("no fix line for the refused task: %q", m.message)
	}
	if len(m.marks) != 1 || !m.isMarked(o[1]) {
		t.Fatalf("marks after: %v, want only %s", m.marks, o[1])
	}
}

// Check 3: x asks the reason once and drops every marked task with it.
func TestDropMarked(t *testing.T) {
	m, o, commits := markBoard(t)
	m = selectRow(t, m, o[1])
	m = press(m, " ", " ")
	m = press(m, "x")
	if m.prompt != dropPrompt {
		t.Fatalf("x with marks: prompt %q", m.prompt)
	}
	m = press(m, "not needed", "enter")
	if !strings.Contains(ansi.Strip(m.confirmLine()), "Drop 2 tasks: "+o[1]+", "+o[2]+": not needed") {
		t.Fatalf("confirm %q", ansi.Strip(m.confirmLine()))
	}
	m = press(m, "enter")
	for _, id := range o[1:3] {
		if got, _ := m.taskAt(pathOf(t, m, id)); got.State != "dropped" {
			t.Fatalf("%s is %s", id, got.State)
		}
	}
	if *commits != 2 || len(m.marks) != 0 || m.message != "dropped 2 of 2" {
		t.Fatalf("commits %d marks %v message %q", *commits, m.marks, m.message)
	}
}

// Check 3: w asks what they wait on once and parks every marked task.
func TestWaitMarked(t *testing.T) {
	m, o, _ := doingBoard(t)
	m = selectRow(t, m, o[0])
	m = press(m, " ", " ")
	m = press(m, "w")
	if m.prompt != waitPrompt {
		t.Fatalf("w with marks: prompt %q", m.prompt)
	}
	m = press(m, "the release", "enter", "enter")
	for _, id := range o[:2] {
		got, _ := m.taskAt(pathOf(t, m, id))
		if task.Stage(got, m.all) != "waiting" {
			t.Fatalf("%s stage %s", id, task.Stage(got, m.all))
		}
	}
	if len(m.marks) != 0 {
		t.Fatalf("marks after: %v", m.marks)
	}
}

// Check 4: m grabs the marked tasks as one block, gathered at the first
// marked row in their order; j moves the block; Enter writes it in order.
func TestGrabMarkedBlock(t *testing.T) {
	m, o, commits := markBoard(t)
	m = selectRow(t, m, o[0])
	m = press(m, " ", "j", " ") // o[0], o[2]
	m = press(m, "m")
	if !m.grab.active {
		t.Fatalf("m with marks did not grab: %q", m.message)
	}
	if got := strings.Join(shown(m), " "); got != pick(o, 0, 2, 1, 3, 4, 5) {
		t.Fatalf("block not gathered: %s", got)
	}
	m = press(m, "j", "j")
	if got := strings.Join(shown(m), " "); got != pick(o, 1, 3, 0, 2, 4, 5) {
		t.Fatalf("block after j j: %s", got)
	}
	if *commits != 0 {
		t.Fatal("written before Enter")
	}
	m = press(m, "enter")
	if got := rankOrder(t, m); got != pick(o, 1, 3, 0, 2, 4, 5) {
		t.Fatalf("order on disk: %s", got)
	}
	if m.grab.active || len(m.marks) != 0 {
		t.Fatalf("after the drop: grabbed %v marks %v", m.grab.active, m.marks)
	}
	// Esc puts the block back as it was.
	m, o, _ = markBoard(t)
	m = selectRow(t, m, o[1])
	m = press(m, " ", "j", "j", " ", "m", "k", "esc")
	if got := strings.Join(shown(m), " "); got != pick(o, 0, 1, 2, 3, 4, 5) || m.grab.active {
		t.Fatalf("Esc did not put the block back: %s", got)
	}
}

// Check 8: under 70 columns the footer is "N marked · ? actions · Esc
// unmark", never bare letters; ? lists the actions with their words.
func TestNarrowMarkedFooter(t *testing.T) {
	m, o, _ := markBoard(t)
	m.width = 60
	m.resize()
	m = selectRow(t, m, o[0])
	m = press(m, " ", " ", " ")
	if f := plainMarks(footerText(m)); f != "3 marked · ? actions · Esc unmark" {
		t.Fatalf("narrow footer %q", f)
	}
	help := ansi.Strip(press(m, "?").helpText())
	if !strings.HasPrefix(help, "3 MARKED") {
		t.Fatalf("? with marks does not lead with the marked actions:\n%s", help)
	}
	for _, want := range []string{"move the marked as one block", "drop every marked task", "unmark all"} {
		if !strings.Contains(help, want) {
			t.Fatalf("help lacks %q:\n%s", want, help)
		}
	}
}

// Check 5 and the heading rule: Space on a heading still folds; inside a
// task Space still ticks (detail tests keep that).
func TestSpaceOnHeadingStillFolds(t *testing.T) {
	m, _, _ := markBoard(t)
	m.setGrouping("next")
	m.move(0, true)
	if m.head == "" {
		t.Skip("no heading at the top")
	}
	m = press(m, " ")
	if len(m.marks) != 0 || !m.folded[m.head] && len(m.folded) == 0 {
		t.Fatalf("Space on a heading: marks %v folded %v", m.marks, m.folded)
	}
}

// Golden: the list at 80 and 140 and narrow at 60, three marked, the
// cursor on one of them, plain.
func TestMarkedGolden(t *testing.T) {
	var b strings.Builder
	for _, w := range []int{80, 140, 60} {
		m := manyModel(t)
		m.width, m.height = w, 24
		m.resize()
		m = press(m, " ", "j", " ", " ", "k")
		b.WriteString("== list at " + strconv.Itoa(w) + "\n" + screen(m) + "\n")
		o := m
		o.setView("overview")
		b.WriteString("== overview at " + strconv.Itoa(w) + "\n" + screen(o) + "\n")
	}
	golden(t, "testdata/marks/marked.txt", "TestMarkedGolden", b.String())
}
