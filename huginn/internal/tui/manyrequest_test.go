package tui

import (
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/x/ansi"
	"github.com/veridit/hug/huginn/internal/task"
)

// ec6z6j check 4 (board-request-mock.md section 4): a request for several
// tasks shows, on its home's page, the text and a targets line, each
// target pending or resolved ("targets: a pending, b resolved, c
// pending"), so a partial fix reads as partial. On another target's page
// (a link, which stores no text) it shows the same text, the same targets
// line, and where it lives: "request 1 on HOME". The home's id there is a
// stop: Tab reaches it and Enter opens the home. Golden at 80x24.
func TestManyRequestPages(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	oldTask, oldNow := task.Now, now
	t.Cleanup(func() { task.Now, now = oldTask, oldNow })
	task.Now = func() time.Time { return time.Date(2026, 11, 20, 16, 12, 0, 0, time.Local) }
	now = func() time.Time { return time.Date(2026, 11, 20, 16, 16, 0, 0, time.Local) }

	m, ids, _ := moveBoard(t, "Docs page", "Spike fit", "Feature quiet", "Bug lint")
	a, b, c := ids["Docs page"], ids["Spike fit"], ids["Feature quiet"]
	if _, err := m.store.RequestFor([]string{a, b, c}, "Each of these lacks a purpose: write one from the goal.", "jhf"); err != nil {
		t.Fatal(err)
	}
	if _, err := m.store.Resolve(b, 1, "added a purpose", "crab"); err != nil {
		t.Fatal(err)
	}
	m.width, m.height = 80, 24
	m.resize()
	m.refresh("")

	var golden []string
	for _, x := range []struct{ name, id string }{{"home " + a, a}, {"link " + b + " (resolved)", b}, {"link " + c + " (pending)", c}} {
		p := openID(t, m, x.id)
		p.viewport.GotoBottom()
		screen := ansi.Strip(p.View())
		golden = append(golden, "== "+x.name+" at 80x24", screen)
		at := strings.Index(screen, "REQUESTS")
		if at < 0 {
			t.Fatalf("%s: no REQUESTS on screen:\n%s", x.name, screen)
		}
		reqs := screen[at:]
		if !strings.Contains(reqs, "Each of these lacks a purpose: write one from the goal.") {
			t.Fatalf("%s: the request's text is not shown:\n%s", x.name, screen)
		}
		if want := "targets: " + a + " pending, " + b + " resolved, " + c + " pending"; !strings.Contains(reqs, want) {
			t.Fatalf("%s: no %q:\n%s", x.name, want, screen)
		}
		if x.id != a && !strings.Contains(reqs, "request 1 on "+a) {
			t.Fatalf("%s: the link does not point to its home %s:\n%s", x.name, a, screen)
		}
		if x.id == a && strings.Contains(reqs, "request 1 on ") {
			t.Fatalf("the home points to itself:\n%s", screen)
		}
	}
	// The link's resolve stays under it, the home's text above it.
	p := openID(t, m, b)
	p.viewport.GotoBottom()
	reqs := ansi.Strip(p.View())
	if i, j := strings.Index(reqs, "lacks a purpose"), strings.Index(reqs, "└ crab: added a purpose"); i < 0 || j < i {
		t.Fatalf("the resolve is not under the text:\n%s", reqs)
	}
	// The home's id on a link is a stop: Tab reaches it, Enter opens it.
	reached := false
	for range 40 {
		p = press(p, "tab")
		if s, ok := p.markedStop(mustSelected(t, p)); ok && s.where == stopRequest && s.target.ID == a {
			reached = true
			break
		}
	}
	if !reached {
		t.Fatalf("Tab never reaches %s in the link's request", a)
	}
	if p = press(p, "enter"); mustSelected(t, p).ID != a {
		t.Fatalf("Enter on %s opened %s", a, mustSelected(t, p).ID)
	}
	goldenFile(t, "testdata/detail/manyrequest.txt", strings.NewReplacer(a, "aaaaaa", b, "bbbbbb", c, "cccccc", ids["Bug lint"], "dddddd").Replace(strings.Join(golden, "\n")+"\n"))
}
