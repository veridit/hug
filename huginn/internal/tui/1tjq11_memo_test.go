package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
	"github.com/muesli/termenv"
	"github.com/veridit/hug/huginn/internal/task"
)

// 1tjq11 check 3: the page memo (pageMemo) never shows a page stale. A
// model without the memo (page nil) works everything out every time:
// the oracle. After each change (the task file on disk, the board, the
// width, a fold, a find, the answer box, the theme, a file attached or
// made in the checkout) the page, its ids and file links, its links,
// folds and focus ring, and the block the focus marks are the oracle's.

// unmemoed is m without its memo: every fact worked out again.
func unmemoed(m model) model {
	m.page = nil
	return m
}

// pageState is what a page shows and where its stops are, for comparing.
type pageState struct {
	Page   string
	Stops  []string
	Links  []linkDetail
	Folds  []fold
	Ring   []focusStop
	Marked [3]int
}

func stateOf(m model) pageState {
	t, ok := m.selected()
	if !ok {
		return pageState{}
	}
	var stops []string
	for _, s := range m.idStops(t) {
		id := s.target.ID
		if s.file != nil {
			id = fmt.Sprintf("file %s#%s repo=%v", s.file.name, s.file.section, s.file.repo)
		}
		stops = append(stops, fmt.Sprintf("%s/%d %q -> %s @%d", s.where, s.index, s.shown, id, s.at))
	}
	a, b, ok := m.markedBlock()
	marked := [3]int{a, b, 0}
	if ok {
		marked[2] = 1
	}
	return pageState{Page: m.detailText(t), Stops: stops, Links: m.links(t), Folds: m.folds(t), Ring: m.focusRing(t), Marked: marked}
}

// sameAsOracle fails when m's page differs from the one worked out
// without the memo. The clock is held still while both are drawn: "N min
// ago" is drawn fresh on each frame, and a minute that turns between the
// two is no stale page.
func sameAsOracle(t *testing.T, m model, when string) {
	t.Helper()
	at := now()
	old := now
	now = func() time.Time { return at }
	defer func() { now = old }()
	got, want := stateOf(m), stateOf(unmemoed(m))
	if got.Page != want.Page {
		t.Fatalf("%s: the page is stale:\n%s", when, firstDiff(ansi.Strip(got.Page), ansi.Strip(want.Page)))
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("%s: stale facts:\nmemo:   %+v\noracle: %+v", when, got, want)
	}
	if lines := m.pageLines(mustSelected(t, m)); strings.Join(lines, "\n") != ansi.Strip(want.Page) {
		t.Fatalf("%s: pageLines are stale:\n%s", when, firstDiff(strings.Join(lines, "\n"), ansi.Strip(want.Page)))
	}
}

// firstDiff shows the first line where a and b differ.
func firstDiff(a, b string) string {
	al, bl := strings.Split(a, "\n"), strings.Split(b, "\n")
	for i := 0; i < max(len(al), len(bl)); i++ {
		var x, y string
		if i < len(al) {
			x = al[i]
		}
		if i < len(bl) {
			y = bl[i]
		}
		if x != y {
			return fmt.Sprintf("line %d of %d/%d:\nmemo:   %q\noracle: %q", i, len(al), len(bl), x, y)
		}
	}
	return "same lines"
}

// memoBoard is a board of three tasks: main, which names the others by
// id, alias, task:ID and file, with notes long enough that the page,
// open, is wrapped in pieces (wrapChunk); other, and third. main asks
// jhf a question. The checkout is a folder of its own.
type memoFixture struct {
	m                   model
	s                   *task.Store
	main, other, third  string
	mainPath, otherPath string
	checkout            string
}

func memoBoard(t *testing.T) memoFixture {
	t.Helper()
	t.Setenv("NO_COLOR", "1")
	s := &task.Store{Root: t.TempDir()}
	other, err := s.New(task.NewOptions{Title: "The other task", Goal: "g", Acceptance: []string{"a"}})
	if err != nil {
		t.Fatal(err)
	}
	third, err := s.New(task.NewOptions{Title: "The third task", Goal: "g", Acceptance: []string{"a"}})
	if err != nil {
		t.Fatal(err)
	}
	o, th := other.Task.ID, third.Task.ID
	r, err := s.New(task.NewOptions{
		Title:      "Main task",
		Goal:       "Reach " + o + " and task:" + th + ".",
		Purpose:    "See plan.txt and docs/LATER.md, neither there yet.",
		Principle:  "One rule: PROJ-77 is no task yet.",
		Details:    strings.Repeat("A paragraph of details naming "+o+".\n\n", 12),
		Acceptance: []string{"first check names " + th, "second check"},
	})
	if err != nil {
		t.Fatal(err)
	}
	id := r.Task.ID
	if _, err := s.TakeAs(id, "crab", "m"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Ask(id, task.Question{What: "Shall I split it, as " + o + " did?", For: "jhf"}, "crab"); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(s.Root, filepath.FromSlash(r.Task.Path))
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var notes strings.Builder
	for i := 0; i < 60; i++ {
		fmt.Fprintf(&notes, "2026-11-%02d note %d: the probe on %s held, task:%s too; the hash %08x of this entry makes it long enough to wrap at sixty columns and more.\n\n", 1+i%28, i, o, th, i*7919)
	}
	text := strings.Replace(string(b), "\n## Notes\n", "\n## Notes\n\n"+notes.String(), 1)
	if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
	checkout := t.TempDir()
	m := newModel(s, Options{User: "jhf", Checkout: checkout})
	m.width, m.height = 100, 30
	m.resize()
	m = openTask(t, m, id)
	return memoFixture{m: m, s: s, main: id, other: o, third: th, mainPath: path, otherPath: filepath.Join(s.Root, filepath.FromSlash(other.Task.Path)), checkout: checkout}
}

// warm presses keys that work out and keep every fact of the page.
func warm(t *testing.T, m model) model {
	t.Helper()
	for _, k := range []string{"j", "tab", "tab", "tab", "shift+tab", "k"} {
		m = press(m, k)
		_ = m.View()
	}
	sameAsOracle(t, m, "warmed")
	return m
}

// openNotes focuses the Notes heading and opens it.
func openNotes(t *testing.T, m model) model {
	t.Helper()
	for n := 0; n < 80 && !strings.Contains(detailPlain(m), "▸ NOTES"); n++ {
		m = press(m, "shift+tab")
	}
	if m = press(m, "enter"); !strings.Contains(detailPlain(m), "· Enter folds") {
		t.Fatalf("the Notes did not open:\n%s", detailPlain(m))
	}
	return m
}

// editFile rewrites path on disk with each old replaced by new, and
// feeds the model the change, as the watch does.
func editFile(t *testing.T, m model, path string, pairs ...string) model {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	text := string(b)
	for i := 0; i+1 < len(pairs); i += 2 {
		if !strings.Contains(text, pairs[i]) {
			t.Fatalf("%s holds no %q", path, pairs[i])
		}
		text = strings.Replace(text, pairs[i], pairs[i+1], 1)
	}
	if err := task.AtomicWrite(path, []byte(text)); err != nil {
		t.Fatal(err)
	}
	return feed(t, m, changedMsg{})
}

// The task's own file changes on disk: its text, an id in it, a task:ID
// link, a note added; the page, the stops, the links, the Notes count
// and the focus ring follow.
func TestPageFollowsTheFileOnDisk(t *testing.T) {
	f := memoBoard(t)
	m := warm(t, openNotes(t, f.m))
	before := stateOf(m)
	m = editFile(t, m, f.mainPath,
		"Reach "+f.other+" and task:"+f.third, "Reach "+f.third+" and task:"+f.other+", rewritten on disk",
		"\n## Notes\n\n", "\n## Notes\n\n2026-11-23 a new note names "+f.other+".\n\n")
	sameAsOracle(t, m, "after the file changed on disk")
	after := stateOf(m)
	page := ansi.Strip(after.Page)
	if !strings.Contains(page, "rewritten on disk") || !strings.Contains(page, "a new note names") {
		t.Fatalf("the page does not show the new text:\n%s", page)
	}
	if after.Folds[len(after.Folds)-2].count != before.Folds[len(before.Folds)-2].count+1 {
		t.Fatalf("Notes count %v, was %v: the new note is not counted", after.Folds, before.Folds)
	}
	// The notes name task:third; the Goal now names task:other first.
	if len(before.Links) != 1 || before.Links[0].ID != f.third || len(after.Links) != 2 || after.Links[0].ID != f.other || after.Links[1].ID != f.third {
		t.Fatalf("links %+v, then %+v: want %s, then %s and %s", before.Links, after.Links, f.third, f.other, f.third)
	}
	if reflect.DeepEqual(after.Stops, before.Stops) || reflect.DeepEqual(after.Ring, before.Ring) {
		t.Fatal("the stops and the ring did not change with the file")
	}
}

// Another task gets an alias the page names (PROJ-77): a reload makes it
// a stop. Its title changes: the Links row and the stop say the new one.
func TestPageFollowsTheBoard(t *testing.T) {
	f := memoBoard(t)
	m := warm(t, f.m)
	for _, s := range stateOf(m).Stops {
		if strings.Contains(s, "PROJ-77") {
			t.Fatalf("PROJ-77 is a stop before any task has it: %s", s)
		}
	}
	m = editFile(t, m, f.otherPath, "\ntitle: ", "\naliases: [\"PROJ-77\"]\ntitle: ")
	sameAsOracle(t, m, "after another task took the alias")
	found := false
	for _, s := range stateOf(m).Stops {
		found = found || strings.Contains(s, `"PROJ-77" -> `+f.other)
	}
	if !found {
		t.Fatalf("PROJ-77 is no stop after %s took it: %v", f.other, stateOf(m).Stops)
	}
	thirdPath := filepath.Join(f.s.Root, f.third[:1], f.third+".md")
	m = editFile(t, m, thirdPath, "The third task", "The renamed third task")
	sameAsOracle(t, m, "after the linked task was renamed")
	if !strings.Contains(ansi.Strip(stateOf(m).Page), "The renamed third task") {
		t.Fatalf("the Links row keeps the old title:\n%s", ansi.Strip(stateOf(m).Page))
	}
}

// The same text on two tasks' pages: each page's stops leave out its own
// task, never the other's.
func TestStopsAreThePagesOwn(t *testing.T) {
	f := memoBoard(t)
	shared := "## Purpose\nBoth " + f.main + " and " + f.other + " say so.\n\n"
	m := editFile(t, f.m, f.mainPath, "## Purpose\n", shared+"## Old purpose\n")
	m = editFile(t, m, f.otherPath, "## Acceptance\n", shared+"## Acceptance\n")
	m = warm(t, m)
	m = press(m, "esc")
	m = openTask(t, m, f.other)
	m = warm(t, m)
	sameAsOracle(t, m, "the other task's page after main's")
	for _, s := range stateOf(m).Stops {
		if strings.Contains(s, "-> "+f.other) {
			t.Fatalf("%s's page has a stop to itself: %s", f.other, s)
		}
	}
}

// A text scanned as a linked section and as plain text: task:ID is a
// stop only in the plain one.
func TestStopsFollowLinked(t *testing.T) {
	f := memoBoard(t)
	tk := mustSelected(t, f.m)
	text := "see task:" + f.other
	linked := f.m.idsIn(tk, text, true)
	plain := f.m.idsIn(tk, text, false)
	if len(linked) != 0 || len(plain) != 1 {
		t.Fatalf("linked %v, plain %v: want none and one", linked, plain)
	}
}

// The width changes: the page is wrapped and its markdown rendered again
// at the new width, folded and open.
func TestPageFollowsTheWidth(t *testing.T) {
	f := memoBoard(t)
	m := warm(t, openNotes(t, f.m))
	for _, w := range []int{60, 140, 100} {
		next, _ := m.Update(tea.WindowSizeMsg{Width: w, Height: 30})
		m = next.(model)
		sameAsOracle(t, m, fmt.Sprintf("at width %d", w))
		for _, l := range strings.Split(ansi.Strip(stateOf(m).Page), "\n") {
			if ansi.StringWidth(l) > w {
				t.Fatalf("at width %d a line is %d wide: %q", w, ansi.StringWidth(l), l)
			}
		}
	}
}

// The theme changes (colour profile, dark or light): markdown is
// rendered again for it.
func TestMarkdownFollowsTheTheme(t *testing.T) {
	f := memoBoard(t)
	m := warm(t, f.m)
	text := "Some **bold** text and `code`."
	for _, c := range []struct {
		profile termenv.Profile
		dark    bool
	}{{termenv.Ascii, true}, {termenv.TrueColor, true}, {termenv.TrueColor, false}, {termenv.ANSI256, false}} {
		v := m
		v.styles = newTheme()
		v.styles.renderer.SetColorProfile(c.profile)
		v.styles.renderer.SetHasDarkBackground(c.dark)
		if got, want := v.markdown(text), renderMarkdown(text, v.width, c.dark, c.profile); got != want {
			t.Fatalf("profile %v dark %v: markdown is the memo's from another theme:\n%q\nwant\n%q", c.profile, c.dark, got, want)
		}
	}
}

// A fold opens and closes: its text, its ids and the ring come and go,
// and Enter's scroll lands on its heading.
func TestPageFollowsAFold(t *testing.T) {
	f := memoBoard(t)
	m := warm(t, f.m)
	folded := stateOf(m)
	m = openNotes(t, m)
	sameAsOracle(t, m, "Notes opened")
	open := stateOf(m)
	if len(open.Ring) <= len(folded.Ring) || !strings.Contains(ansi.Strip(open.Page), "note 59:") {
		t.Fatalf("opened, the Notes add no stops (%d, folded %d) or no text", len(open.Ring), len(folded.Ring))
	}
	if s, _, ok := m.markedBlock(); !ok || m.viewport.YOffset != s {
		t.Fatalf("opened, the page is at %d, the heading at %d (%v)", m.viewport.YOffset, s, ok)
	}
	for i := 0; i < 5; i++ {
		m = press(m, "tab")
		sameAsOracle(t, m, fmt.Sprintf("Tab %d in the open Notes", i+1))
	}
	for n := 0; n < 200 && !strings.Contains(detailPlain(m), "▸ NOTES"); n++ {
		m = press(m, "shift+tab")
	}
	m = press(m, "enter")
	sameAsOracle(t, m, "Notes folded again")
	if strings.Contains(ansi.Strip(stateOf(m).Page), "note 59:") {
		t.Fatal("folded again, the Notes text is still on the page")
	}
}

// A find: what is typed opens the fold holding it, its hits are the
// oracle's, and clearing it folds the Notes again.
func TestPageFollowsAFind(t *testing.T) {
	f := memoBoard(t)
	m := warm(t, f.m)
	m = typeFind(m, "note 42")
	sameAsOracle(t, m, "find typed")
	want := unmemoed(m)
	want.refind()
	if len(m.find.hits) == 0 || !reflect.DeepEqual(m.find.hits, want.find.hits) {
		t.Fatalf("hits %v, oracle %v", m.find.hits, want.find.hits)
	}
	if !m.unfolded["notes"] {
		t.Fatal("the find did not open the Notes that hold its match")
	}
	for range "note 42" {
		m = press(m, "backspace")
	}
	sameAsOracle(t, m, "find cleared")
	if m.unfolded["notes"] {
		t.Fatal("the Notes the find opened stay open after it is cleared")
	}
}

// The answer box grows line by line: the page under it moves down, the
// box and its question are the marked block, and the box stays in view.
func TestPageFollowsTheAnswerBox(t *testing.T) {
	f := memoBoard(t)
	m := warm(t, f.m)
	if m = press(m, "a"); m.prompt != answerPrompt {
		t.Fatalf("a opened %q: %q", m.prompt, m.message)
	}
	m = press(m, "y", "e", "s")
	prev := stateOf(m).Marked
	for i := 0; i < 6; i++ {
		// A new line, then text on it: an empty last line is not part
		// of the marked block (markedBlock skips blank lines).
		m, _ = key(m, "alt+enter")
		m = press(m, "n", "o")
		sameAsOracle(t, m, fmt.Sprintf("box line %d", i+1))
		now := stateOf(m).Marked
		if now[1] <= prev[1] {
			t.Fatalf("line %d: the marked block ends at %d, was %d: the box did not grow on the page", i+1, now[1], prev[1])
		}
		if now[1] < m.viewport.YOffset || now[1] >= m.viewport.YOffset+m.viewport.Height {
			t.Fatalf("line %d: the box's last line %d is out of view [%d, %d)", i+1, now[1], m.viewport.YOffset, m.viewport.YOffset+m.viewport.Height)
		}
		prev = now
	}
}

// A file is attached to the task: its name in the text becomes a link.
func TestFileLinksFollowAnAttachment(t *testing.T) {
	f := memoBoard(t)
	m := warm(t, f.m)
	if strings.Contains(fmt.Sprint(stateOf(m).Stops), "plan.txt") {
		t.Fatal("plan.txt is a link before it is attached")
	}
	if _, _, err := f.s.Attach(f.main, "plan.txt", []byte("the plan\n"), "", false); err != nil {
		t.Fatal(err)
	}
	m = feed(t, m, changedMsg{})
	sameAsOracle(t, m, "after plan.txt was attached")
	if !strings.Contains(fmt.Sprint(stateOf(m).Stops), `"plan.txt" -> file plan.txt`) {
		t.Fatalf("plan.txt is no link once attached: %v", stateOf(m).Stops)
	}
}

// A file appears in the checkout: once repoFile's answer goes stale the
// name in the text is a link, with no reload (the checkout is not
// watched).
func TestFileLinksFollowTheCheckout(t *testing.T) {
	old := repoFileFor
	repoFileFor = 20 * time.Millisecond
	t.Cleanup(func() { repoFileFor = old })
	f := memoBoard(t)
	m := warm(t, f.m)
	if strings.Contains(fmt.Sprint(stateOf(m).Stops), "docs/LATER.md") {
		t.Fatal("docs/LATER.md is a link before the checkout has it")
	}
	if err := os.MkdirAll(filepath.Join(f.checkout, "docs"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(f.checkout, "docs", "LATER.md"), []byte("# Later\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	time.Sleep(3 * repoFileFor)
	m = press(m, "j")
	sameAsOracle(t, m, "after docs/LATER.md was made")
	if !strings.Contains(fmt.Sprint(stateOf(m).Stops), `"docs/LATER.md" -> file docs/LATER.md# repo=true`) {
		t.Fatalf("docs/LATER.md is no link once the checkout has it: %v", stateOf(m).Stops)
	}
}

// Two models with other checkouts share a memo: each finds its own
// checkout's files.
func TestFileLinksAreTheCheckoutsOwn(t *testing.T) {
	f := memoBoard(t)
	other := t.TempDir()
	if err := os.MkdirAll(filepath.Join(other, "docs"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(other, "docs", "LATER.md"), []byte("# Later\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	m := warm(t, f.m)
	v := m
	v.opts.Checkout = other
	sameAsOracle(t, v, "the same page in another checkout")
	if !strings.Contains(fmt.Sprint(stateOf(v).Stops), "docs/LATER.md") {
		t.Fatalf("in a checkout with docs/LATER.md it is no link: %v", stateOf(v).Stops)
	}
}

// A text with ids before and after a file link: its stops read in order
// on every frame, not only the first (idStops sorts what it is handed;
// it must sort its own slice, not the memo's).
func TestStopsStayInReadingOrder(t *testing.T) {
	f := memoBoard(t)
	fourth, err := f.s.New(task.NewOptions{Title: "The fourth task", Goal: "g", Acceptance: []string{"a"}})
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := f.s.Attach(f.main, "plan.txt", []byte("the plan\n"), "", false); err != nil {
		t.Fatal(err)
	}
	m := editFile(t, f.m, f.mainPath, "One rule: PROJ-77 is no task yet.", "Ask "+f.other+", then "+f.third+", see plan.txt, then "+fourth.Task.ID+".")
	var want []string
	for i := 0; i < 3; i++ {
		m = press(m, "j")
		sameAsOracle(t, m, fmt.Sprintf("frame %d", i+1))
		var got []string
		for _, s := range stateOf(m).Stops {
			if strings.HasPrefix(s, "text/2 ") {
				got = append(got, s)
			}
		}
		if i == 0 {
			want = got
		}
		if len(got) != 4 || !reflect.DeepEqual(got, want) {
			t.Fatalf("frame %d: the Principle's stops %v, want 4, as on the first frame %v", i+1, got, want)
		}
	}
}

// A name written file: first (an attachment only) and then plain (a
// checkout file too): the plain one is still a link.
func TestFileWordsPrefixedThenPlain(t *testing.T) {
	f := memoBoard(t)
	if err := os.MkdirAll(filepath.Join(f.checkout, "docs"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(f.checkout, "docs", "PLAN.md"), []byte("# Plan\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	m := editFile(t, f.m, f.mainPath, "One rule: PROJ-77 is no task yet.", "Not file:docs/PLAN.md, it is no attachment; docs/PLAN.md is the checkout's.")
	m = warm(t, m)
	if !strings.Contains(fmt.Sprint(stateOf(m).Stops), `text/2 "docs/PLAN.md" -> file docs/PLAN.md# repo=true`) {
		t.Fatalf("the plain docs/PLAN.md is no link: %v", stateOf(m).Stops)
	}
}

// wrapPage, in pieces from the memo, is wrap whole: at each width, and
// for texts whose pieces begin alike and differ further on.
func TestWrapPageIsWrap(t *testing.T) {
	m := memoBoard(t).m
	line := strings.Repeat("a line of the page that is longer than a narrow screen, ", 3) + "\n"
	base := strings.Repeat(line, 3*wrapChunk/len(line))
	// other is base but for a word near the end of each piece: the
	// pieces begin alike.
	lines := strings.SplitAfter(base, "\n")
	for i := 40; i < len(lines); i += 40 {
		lines[i] = strings.Replace(lines[i], "narrow screen", "narrow SCREEN", 1)
	}
	other := strings.Join(lines, "")
	for _, s := range []string{base, other, base} {
		for _, w := range []int{100, 37, 60, 100} {
			if got, want := m.wrapPage(s, w), wrap(s, w); got != want {
				t.Fatalf("at %d: wrapPage differs from wrap:\n%s", w, firstDiff(got, want))
			}
		}
	}
}

// markSpanOf, from the memo's scan, is markSpan, for texts that begin
// alike and differ further on.
func TestMarkSpanOfIsMarkSpan(t *testing.T) {
	m := memoBoard(t).m
	head := strings.Repeat("words before the id ", 10)
	for _, s := range []string{head + "abc123 and docs/X.md", head + "zz abc123 and docs/X.md", head + "abc123"} {
		for _, w := range []string{"abc123", "docs/X.md"} {
			a, n, shown := m.markSpanOf(s, w, false)
			b, k, want := markSpan(s, w, false)
			if a != b || n != k || shown != want {
				t.Fatalf("%q in %q: memo %d,%d,%q, markSpan %d,%d,%q", w, s, a, n, shown, b, k, want)
			}
		}
	}
}
