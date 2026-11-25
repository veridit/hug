package tui

import (
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/x/ansi"
	"github.com/veridit/hug/huginn/internal/task"
)

// textBoard is qtj177's case (0ga6m0, Jørgen 11:45Z): Mainta is asked of
// jhf about check 2, "Jørgen opens Checkd and says it reads clearly", and
// the question names Askedq and Checkd again. Its Goal, Purpose,
// Principle and Details each name a task; a request names
// Reques and its account Resolv. Answrd carries carol's unread answer to
// jhf naming Answer. Every title is six letters, as wide as an id.
func textBoard(t *testing.T) (model, map[string]string) {
	t.Helper()
	old, oldTask := now, task.Now
	now = func() time.Time { return time.Date(2026, 11, 18, 11, 53, 0, 0, time.Local) }
	task.Now = func() time.Time { return time.Date(2026, 11, 18, 11, 50, 0, 0, time.Local) }
	t.Cleanup(func() { now, task.Now = old, oldTask })
	s := &task.Store{Root: t.TempDir()}
	ids := map[string]string{}
	add := func(o task.NewOptions) {
		if o.Goal == "" {
			o.Goal = "g"
		}
		if len(o.Acceptance) == 0 {
			o.Acceptance = []string{"a"}
		}
		r, err := s.New(o)
		if err != nil {
			t.Fatal(err)
		}
		ids[o.Title] = r.Task.ID
	}
	for _, title := range []string{"Goalid", "Purpos", "Princi", "Detail", "Checkd", "Askedq", "Reques", "Resolv", "Answer"} {
		add(task.NewOptions{Title: title})
	}
	add(task.NewOptions{Title: "Mainta", Top: true,
		Goal:       "Builds on " + ids["Goalid"] + ".",
		Purpose:    "So " + ids["Purpos"] + " reads clearly.",
		Principle:  "As " + ids["Princi"] + " does.",
		Details:    "Seen in " + ids["Detail"] + ".",
		Acceptance: []string{"a", "Jørgen opens " + ids["Checkd"] + " and says it reads clearly"},
	})
	add(task.NewOptions{Title: "Answrd"})
	must := func(_ task.Result, err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	must(s.TakeAs(ids["Mainta"], "wyvern", "m"))
	must(s.Ask(ids["Mainta"], task.Question{What: "after the release: open " + ids["Askedq"] + " (or " + ids["Checkd"] + "): does it read clearly?", For: "jhf", Check: 2}, "wyvern"))
	must(s.Request(ids["Mainta"], "Please look at "+ids["Reques"]+" too.", "jhf"))
	must(s.Resolve(ids["Mainta"], 1, "Done, as in "+ids["Resolv"]+".", "crab"))
	must(s.TakeAs(ids["Answrd"], "wyvern", "m"))
	must(s.Ask(ids["Answrd"], task.Question{What: "which one?", For: "carol"}, "jhf"))
	must(s.Answer(ids["Answrd"], "Use "+ids["Answer"]+".", "carol"))
	m := newModel(s, Options{User: "jhf"})
	m.width, m.height = 120, 60
	m.resize()
	return m, ids
}

// stopsOf is t's id stops as where:TITLE, in order.
func stopsOf(m model, t task.Task, ids map[string]string) string {
	byID := map[string]string{}
	for title, id := range ids {
		byID[id] = title
	}
	var out []string
	for _, s := range m.idStops(t) {
		out = append(out, s.where+":"+byID[s.target.ID])
	}
	return strings.Join(out, " ")
}

// 0ga6m0 check 1 (Jørgen 11:45Z on qtj177): the asked check names an id
// and so does its question; Tab from the check reaches the id in the
// check, then in the question; Enter opens it, u comes back. Golden at
// 80 and 140 with the check's id marked.
func TestTabReachesIDInCheckAndQuestion(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	var golden []string
	for _, w := range []int{80, 140} {
		m, ids := textBoard(t)
		m.width, m.height = w, 60
		m.resize()
		m = openTask(t, m, ids["Mainta"])
		if m.detailPane != focusAcceptance || m.acceptance != 1 {
			t.Fatalf("at %d: opened on pane %d check %d, want the asked check 2", w, m.detailPane, m.acceptance+1)
		}
		m = press(m, "tab")
		screen := ansi.Strip(m.View())
		if !strings.Contains(screen, "2. Jørgen opens ▸"+ids["Checkd"]+" and says") {
			t.Fatalf("at %d Tab from the asked check does not mark %s in it:\n%s", w, ids["Checkd"], screen)
		}
		g := screen
		for _, want := range []string{"open ▸" + ids["Askedq"] + " (or", "(or ▸" + ids["Checkd"] + ")"} {
			m = press(m, "tab")
			if screen := ansi.Strip(m.View()); !strings.Contains(screen, want) {
				t.Fatalf("at %d Tab does not mark %q in the question:\n%s", w, want, screen)
			}
		}
		m = press(m, "enter")
		if tk, _ := m.selected(); tk.ID != ids["Checkd"] {
			t.Fatalf("at %d Enter opened %s, want %s", w, tk.ID, ids["Checkd"])
		}
		m = press(m, "u")
		if tk, _ := m.selected(); tk.ID != ids["Mainta"] || !m.detail {
			t.Fatalf("at %d u: open %s detail %v, want %s", w, tk.ID, m.detail, ids["Mainta"])
		}
		text := strings.NewReplacer(idsReplacer(ids)...).Replace(g)
		text = regexp.MustCompile(`\d{4}-\d\d-\d\d(T\d\d:\d\d)?`).ReplaceAllStringFunc(text, func(d string) string { return strings.Repeat("#", len(d)) })
		golden = append(golden, "== check id marked at "+strconv.Itoa(w), text)
	}
	goldenFile(t, "testdata/detail/check-ids.txt", strings.Join(golden, "\n")+"\n")
}

// 0ga6m0 check 2: ids in Goal, Purpose, Principle, the checks, the
// question under a check, a request and its account are stops, in
// reading order; the ring has them where they stand (the description's
// after the page, the checks' after the checks), and Tab marks each where
// it stands. An unread answer's id is a stop in the header, before the
// page. Details, one line here, is open after the Principle (hya1st): its
// id is a stop there; only HISTORY is folded.
func TestIDsInTextsAreStopsInReadingOrder(t *testing.T) {
	m, ids := textBoard(t)
	m = openTask(t, m, ids["Mainta"])
	tk, _ := m.selected()
	want := "text:Goalid text:Purpos text:Princi details:Detail check:Checkd asked:Askedq asked:Checkd request:Reques resolved:Resolv"
	if got := stopsOf(m, tk, ids); got != want {
		t.Fatalf("stops\n %s\nwant\n %s", got, want)
	}
	var ring []string
	for _, s := range m.focusRing(tk) {
		ring = append(ring, strconv.Itoa(s.pane))
	}
	p, a, i, f := strconv.Itoa(focusPage), strconv.Itoa(focusAcceptance), strconv.Itoa(focusIDs), strconv.Itoa(focusFold)
	if got, want := strings.Join(ring, " "), strings.Join([]string{p, i, i, i, i, a, i, i, i, i, i, f}, " "); got != want {
		t.Fatalf("ring %s, want %s (page, 3 description ids, the Details id, checks, 3 check ids, 2 request ids, HISTORY folded)", got, want)
	}
	// From the page, Tab marks each where it stands.
	m.detailPane = focusPage
	marks := []string{"Builds on ▸" + ids["Goalid"], "So ▸" + ids["Purpos"], "As ▸" + ids["Princi"], "Seen in ▸" + ids["Detail"],
		"", "opens ▸" + ids["Checkd"], "open ▸" + ids["Askedq"], "(or ▸" + ids["Checkd"], "at ▸" + ids["Reques"], "in ▸" + ids["Resolv"]}
	for k, mark := range marks {
		m = press(m, "tab")
		text := detailPlain(m)
		if mark == "" {
			if m.detailPane != focusAcceptance {
				t.Fatalf("Tab %d: pane %d, want the checks", k+1, m.detailPane)
			}
			continue
		}
		if !strings.Contains(text, mark) || strings.Count(text, "▸") != 1 {
			t.Fatalf("Tab %d: no single mark %q:\n%s", k+1, mark, text)
		}
	}
	// Each opens with Enter.
	m = press(m, "enter")
	if tk, _ := m.selected(); tk.ID != ids["Resolv"] {
		t.Fatalf("Enter on the account's id opened %s", tk.ID)
	}
	// The answer jhf has not read names Answer: a stop in the header, and
	// again in History, where the answer block stands too, once History
	// is open (8a0ybb: folded, it shows none of its text).
	m = openTask(t, press(m, "esc", "esc"), ids["Answrd"])
	tk, _ = m.selected()
	if got := stopsOf(m, tk, ids); got != "answer:Answer" {
		t.Fatalf("Answrd stops %q, want answer:Answer", got)
	}
	if got := stopsOf(m.withUnfolded("history"), tk, ids); got != "answer:Answer history:Answer" {
		t.Fatalf("Answrd stops with History open %q, want answer:Answer history:Answer", got)
	}
	m = press(m, "shift+tab")
	m = press(m, "shift+tab")
	head := ansi.Strip(strings.Join(m.detailHeader(tk), "\n"))
	if !strings.Contains(head, "carol answered asked: Use ▸"+ids["Answer"]) {
		t.Fatalf("Shift-Tab does not mark the answer's id in the header:\n%s", head)
	}
	if m = press(m, "enter"); mustSelected(t, m).ID != ids["Answer"] {
		t.Fatalf("Enter on the answer's id opened %s", mustSelected(t, m).ID)
	}
}

// 0ga6m0 check 3: an id named in two places is a stop in each; named
// twice in one place it is one stop there, the first.
func TestIDStopsOncePerPlace(t *testing.T) {
	m, ids := textBoard(t)
	tk := mustTask(t, m, ids["Mainta"])
	// Checkd stands in check 2 and in its question: two stops. The
	// question names Askedq then Checkd, each once.
	if got := stopsOf(m, tk, ids); strings.Count(got, ":Checkd") != 2 {
		t.Fatalf("Checkd is not a stop in both the check and the question: %s", got)
	}
	if _, err := m.store.Note(ids["Mainta"], "both "+ids["Goalid"]+" and "+ids["Goalid"]+" again"); err != nil {
		t.Fatal(err)
	}
	m.refresh("")
	tk = mustTask(t, m, ids["Mainta"])
	if got := stopsOf(m.withUnfolded("notes"), tk, ids); !strings.HasSuffix(got, "resolved:Resolv notes:Goalid") {
		t.Fatalf("an id twice in the Notes: stops %s, want one notes:Goalid last", got)
	}
}

func mustTask(t *testing.T, m model, id string) task.Task {
	t.Helper()
	for _, tk := range m.all {
		if tk.ID == id {
			return tk
		}
	}
	t.Fatalf("no task %s", id)
	return task.Task{}
}
