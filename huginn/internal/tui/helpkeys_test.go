package tui

import (
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

// state is what a key can change: where you are, what is marked, what the
// screen asks or says, the filters and loads from disk. A key that changes
// none of it and starts no command did nothing.
func state(m model) string {
	loads := int64(0)
	if m.loads != nil {
		loads = m.loads.n.Load()
	}
	return fmt.Sprintf("%s|%d|%d|%v|%s|%d|%d|%d|%d|%d|%d|%d|%q|%q|%q|%v|%v|%d|%d|%s|%v|%v|%v|%d|%s|%q|%v|%d|%d|%s",
		m.view, m.pane, m.cursor, m.detail, m.detailPath, m.detailPane, m.acceptance, m.dependency, m.link, m.file, m.idAt, m.column,
		m.message, m.confirm, m.prompt, m.help, m.glance, m.viewport.YOffset, len(m.detailStack), m.query, m.grab.active, m.pendingV, m.grouping+"/"+m.head,
		len(m.sort), m.yanked, m.input.Value(), m.filter, loads, m.completion, m.pickerField) + fmt.Sprintf("|%v|%v|%v|%v", m.columnHinted, m.glanceMD.raw, m.glanceMD.loading, m.pickerNot)
}

// pressed reports whether k did something in m: changed its state, or
// returned a command (an editor, a write, a quit). A key that moves is
// tried from the middle as well as from where m is, since at the top k
// has nowhere to go; a key that clears filters, with a filter set.
func pressed(m model, k string) (model, bool) {
	if m.loads == nil {
		m.loads = &loadCounter{}
	}
	for _, from := range []model{m, middle(m), short(m), filtered(m), elsewhere(m)} {
		before := state(from)
		after, cmd := key(from, k)
		if cmd != nil || state(after) != before {
			return after, true
		}
	}
	return m, false
}

// middle is m moved off the first row, check and page, so k, <, p, [ and
// K have somewhere to go.
func middle(m model) model {
	if m.prompt != "" || m.confirm != "" {
		return m
	}
	if m.detail {
		// A short screen, so the page is longer than it and J/K scroll.
		m.height = 14
		m.resize()
		if m.glance {
			m = press(m, "n")
		} else {
			// Down one task, its last check, the page scrolled.
			m = press(m, "n")
			if t, ok := m.selected(); ok && len(t.Acceptance) > 1 {
				m.acceptance = len(t.Acceptance) - 1
			}
		}
		m.syncDetail()
		m.viewport.SetYOffset(2)
		return m
	}
	return press(m, "j", "j")
}

// short is m on a short screen, so the open task's page is longer than
// it and J/K have a page to scroll.
func short(m model) model {
	if !m.detail {
		return m
	}
	m.height = 22 // a page of a few lines under the header, the text longer
	m.resize()
	m.syncDetail()
	m.viewport.SetYOffset(m.viewport.Height) // and scrolled, so K goes back
	return m
}

// elsewhere is m in another view, so a view key (i, b, !) has somewhere
// to go from it.
func elsewhere(m model) model {
	if m.prompt != "" || m.confirm != "" || m.detail {
		return m
	}
	other := "overview"
	if m.view == "overview" {
		other = "list"
	}
	m.setView(other)
	return m
}

// filtered is m with a filter chip set, as Backspace, F and c need.
func filtered(m model) model {
	if m.prompt != "" || m.confirm != "" {
		return m
	}
	m.setFilter("type", "feature")
	return m
}

// helpKeyOf is the key that opens help in m: F1 in text (t64v7d), else ?.
func helpKeyOf(m model) string {
	if m.textInput() {
		return "f1"
	}
	return "?"
}

// helpContexts opens ? from every place hug tui has keys: the four views,
// a task in each detail focus, a task's file, and each prompt.
func helpContexts(t *testing.T) map[string]model {
	t.Helper()
	out := map[string]model{}
	m := manyModel(t)
	m.width, m.height = 140, 40
	m.resize()
	for _, v := range []string{"overview", "list", "board", "lint"} {
		mm := m
		mm.setView(v)
		mm.clamp()
		out[v] = mm
	}
	task := openTask(t, m, "000001") // checks and files
	out["task:checks"] = task
	tk, _ := task.selected()
	for _, pane := range panesOf(task, tk) {
		mm := task
		mm.detailPane = pane
		mm.syncDetail()
		out[fmt.Sprintf("task:%d", pane)] = mm
	}
	ids, idMap := idBoard(t)
	out["task:ids"] = press(openID(t, ids, idMap["Main"]), "shift+tab", "shift+tab") // the stage's id
	g, _ := glanceModel(t, 140, 40)
	out["file"] = press(toFiles(t, openTask(t, g, "000001")), " ")
	out["palette"] = press(m, ":")
	out["filter"] = press(m, "f")
	out["find"] = press(m, "/")
	out["results"] = press(m, "/", "task", "enter") // Enter's results (5cjy88)
	out["jump"] = press(m, "#")
	out["confirm"] = press(m, "t")
	return out
}

// jmx10s checks 1, 2 and 4: ? leads with the keys of where it was opened,
// under a heading naming it, then EVERYWHERE; and every key a row lists
// does something there, pressed in that very context.
func TestHelpKeysAreBound(t *testing.T) {
	for name, m := range helpContexts(t) {
		help := press(m, helpKeyOf(m))
		if !help.help {
			t.Fatalf("%s: %s did not open help", name, helpKeyOf(m))
		}
		text := ansi.Strip(help.helpText())
		sections := help.helpSections()
		if !strings.HasPrefix(text, sections[0].title) || !strings.Contains(text, "EVERYWHERE") || strings.Index(text, "EVERYWHERE") < len(sections[0].title) {
			t.Fatalf("%s: help does not lead with its context, then EVERYWHERE:\n%s", name, text)
		}
		for _, s := range sections {
			for _, r := range s.rows {
				for _, k := range r.probe {
					// Pressed where ? was opened, not inside help.
					if _, ok := pressed(m, k); !ok {
						t.Errorf("%s: %q (%s, under %s) does nothing there", name, k, r.key, s.title)
					}
				}
			}
		}
	}
	// The context's heading names it.
	for name, want := range map[string]string{"list": "IN THE LIST", "board": "ON THE BOARD", "overview": "IN THE OVERVIEW", "lint": "IN LINT",
		"task:checks": "IN THIS TASK · 000001", "file": "IN THIS FILE", "palette": "IN THE : PALETTE", "filter": "IN THE FILTER PICKER",
		"find": "IN / FIND", "jump": "IN # JUMP", "confirm": "IN THIS CONFIRM"} {
		m := helpContexts(t)[name]
		if got := ansi.Strip(press(m, helpKeyOf(m)).helpText()); !strings.HasPrefix(got, want+" ") && !strings.HasPrefix(got, want+"\n") {
			t.Fatalf("%s: help starts %q, want %q", name, strings.SplitN(got, "\n", 2)[0], want)
		}
	}
}

// jmx10s check 2: the list's help shows no key that is not bound there.
// Inside a task, the task's keys lead; Tab names task ids only where the
// task shows some (4bwcgp), and the row keys (o, m, J/K moves) are not
// listed for a task.
func TestHelpShowsOnlyKeysThatWorkHere(t *testing.T) {
	ctx := helpContexts(t)
	list := ansi.Strip(press(ctx["list"], "?").helpText())
	if strings.Contains(list, "IN THIS TASK") || strings.Contains(list, "tick the check") {
		t.Fatalf("list help shows task keys:\n%s", list)
	}
	listOnly := ansi.Strip(ctx["list"].sectionText(ctx["list"].contextHelp(), 140))
	if strings.Contains(listOnly, "h / l") {
		t.Fatalf("the list has one pane, yet help lists h / l:\n%s", listOnly)
	}
	task := ansi.Strip(press(ctx["task:checks"], "?").helpText())
	taskOnly := ansi.Strip(ctx["task:checks"].sectionText(ctx["task:checks"].contextHelp(), 140))
	_ = task
	for _, absent := range []string{"order by the marked column", "grab:", "move below / above"} {
		if strings.Contains(taskOnly, absent) {
			t.Fatalf("task help lists a row key %q:\n%s", absent, taskOnly)
		}
	}
	if ids := ansi.Strip(press(ctx["task:ids"], "?").helpText()); !strings.Contains(ids, "h / l  Tab") || !strings.Contains(ids, "open the task id marked ▸") {
		t.Fatalf("a task with ids: help lacks Tab and Enter:\n%s", ids)
	}
	// A prompt's help lists its keys; the global letters are typed text there.
	pal := ansi.Strip(press(ctx["palette"], "?").helpText())
	if strings.Contains(pal, "v o/i/b/!") || !strings.Contains(pal, "complete the choice") {
		t.Fatalf("palette help:\n%s", pal)
	}
}

// jmx10s check 3: typing in help filters rows by key or word; Esc clears
// the filter, then closes help.
func TestHelpFindsByKeyOrWord(t *testing.T) {
	m := press(helpContexts(t)["task:checks"], "?", "/")
	for _, r := range "strike" {
		m = press(m, string(r))
	}
	got := ansi.Strip(m.View())
	if !strings.Contains(got, "strike it out as not relevant") || strings.Contains(got, "tick the check") || !strings.Contains(got, "1 of ") {
		t.Fatalf("help find 'strike':\n%s", got)
	}
	m = press(m, "esc")
	if !m.help || m.helpQuery != "" || !strings.Contains(ansi.Strip(m.View()), "tick the check") {
		t.Fatal("first Esc clears the find and keeps help open")
	}
	if m = press(m, "esc"); m.help {
		t.Fatal("second Esc closes help")
	}
}

// jmx10s check 5: at 80x24 every context's help has no line wider than the
// terminal, and the screen keeps its height (golden at 80 and 140).
func TestHelpFitsAndGolden(t *testing.T) {
	ctx := helpContexts(t)
	var shots []string
	for _, w := range []int{80, 140} {
		for _, name := range []string{"list", "task:checks", "task:ids", "palette"} {
			m := ctx[name]
			m.width, m.height = w, 24
			if w == 140 {
				m.height = 40
			}
			m.resize()
			m.syncDetail()
			m = press(m, helpKeyOf(m))
			screen := ansi.Strip(m.View())
			lines := strings.Split(screen, "\n")
			if len(lines) != m.height {
				t.Fatalf("%s at %d: %d lines, want %d", name, w, len(lines), m.height)
			}
			for _, l := range lines {
				if n := ansi.StringWidth(l); n > w {
					t.Fatalf("%s at %d: a line is %d wide: %q", name, w, n, l)
				}
			}
			body := ansi.Strip(m.helpText())
			for _, l := range strings.Split(body, "\n") {
				if n := ansi.StringWidth(l); n > w {
					t.Fatalf("%s at %d: a help line is %d wide: %q", name, w, n, l)
				}
			}
			shots = append(shots, fmt.Sprintf("== %s at %d", name, w), body)
		}
	}
	text := strings.Join(shots, "\n") + "\n"
	text = regexp.MustCompile(`IN THIS TASK · [0-9a-z]{6}`).ReplaceAllString(text, "IN THIS TASK · ######")
	golden(t, "testdata/help/contexts.txt", "TestHelpFitsAndGolden", text)
}
