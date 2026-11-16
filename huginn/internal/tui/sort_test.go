package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

func listIDs(m model) []string {
	var out []string
	for _, i := range m.indices() {
		out = append(out, m.visible[i].ID)
	}
	return out
}

// sorted opens ": sort", types keys, and returns the model before Enter.
func sorted(m model, keys ...string) model {
	m = press(m, ":", "sort", "enter")
	return press(m, keys...)
}

// mkzcqr check 1: ": sort", p, t, Enter orders by priority (critical
// first) then type, and the header says so.
func TestSortBuilderPriorityThenType(t *testing.T) {
	m := manyModel(t)
	m.width, m.height = 140, 40
	m.resize()
	m = sorted(m, "p", "t")
	if m.prompt != "sort" || !strings.Contains(ansi.Strip(m.View()), "sort: priority ↓ · type") {
		t.Fatalf("builder does not show the order being built:\n%s", ansi.Strip(m.View()))
	}
	m = press(m, "enter")
	if m.prompt != "" {
		t.Fatalf("Enter left the builder open: %q", m.prompt)
	}
	header := strings.Split(ansi.Strip(m.View()), "\n")[0]
	if !strings.Contains(header, "sort: priority ↓ · type") {
		t.Fatalf("header: %q", header)
	}
	rankOf := map[string]int{"critical": 0, "high": 1, "medium": 2, "low": 3}
	typeOf := map[string]int{"feature": 0, "bug": 1, "chore": 2, "spike": 3, "docs": 4}
	rows := m.indices()
	for n := 1; n < len(rows); n++ {
		a, b := m.visible[rows[n-1]], m.visible[rows[n]]
		pa, pb := rankOf[a.Priority], rankOf[b.Priority]
		if pa > pb || pa == pb && typeOf[a.Type] > typeOf[b.Type] {
			t.Fatalf("row %d: %s (%s %s) before %s (%s %s)", n, a.ID, a.Priority, a.Type, b.ID, b.Priority, b.Type)
		}
	}
	// ": sort rank" brings rank order back.
	m = press(m, ":", "sort rank", "enter")
	if len(m.sort) != 0 || strings.Contains(strings.Split(ansi.Strip(m.View()), "\n")[0], "sort:") {
		t.Fatalf("sort rank left %v", m.sort)
	}
}

// mkzcqr check 2: Esc in the middle of building keeps the order there was.
func TestSortBuilderEscKeepsOrder(t *testing.T) {
	m := manyModel(t)
	m = press(sorted(m, "t"), "enter")
	before := strings.Join(listIDs(m), " ")
	m = press(sorted(m, "p", "n"), "esc")
	if m.prompt != "" || sortLabel(m.sort) != "type" || strings.Join(listIDs(m), " ") != before {
		t.Fatalf("Esc changed the order: prompt %q sort %q", m.prompt, sortLabel(m.sort))
	}
	// From rank order too.
	m = manyModel(t)
	before = strings.Join(listIDs(m), " ")
	m = press(sorted(m, "p"), "esc")
	if len(m.sort) != 0 || strings.Join(listIDs(m), " ") != before {
		t.Fatalf("Esc from rank order sorted: %v", m.sort)
	}
}

// mkzcqr check 3: the same letter again flips that field; Backspace
// removes the last field.
func TestSortBuilderFlipAndBackspace(t *testing.T) {
	m := sorted(manyModel(t), "p", "t", "p")
	if got := sortLabel(m.sortDraft); got != "priority ↑ · type" {
		t.Fatalf("p t p: %q", got)
	}
	m = press(m, "t")
	if got := sortLabel(m.sortDraft); got != "priority ↑ · type ↓" {
		t.Fatalf("p t p t: %q", got)
	}
	m = press(m, "backspace")
	if got := sortLabel(m.sortDraft); got != "priority ↑" {
		t.Fatalf("after ⌫: %q", got)
	}
	m = press(m, "backspace", "backspace")
	if len(m.sortDraft) != 0 || m.prompt != "sort" {
		t.Fatalf("⌫ on nothing: %v prompt %q", m.sortDraft, m.prompt)
	}
	// Flipped priority puts low first.
	m = press(m, "p", "p", "enter")
	if first := m.visible[m.indices()[0]]; first.Priority != "low" {
		t.Fatalf("priority ↑ first row is %s", first.Priority)
	}
}

// mkzcqr check 4: with a sort other than rank, J/K, m, y and P/p are not
// in the help and say how to get rank order back when pressed.
func TestSortRefusesMoves(t *testing.T) {
	m := manyModel(t)
	m.width, m.height = 140, 60
	m.resize()
	if !strings.Contains(ansi.Strip(m.helpText()), "J / K") {
		t.Fatal("rank order: J / K missing from help")
	}
	m = press(sorted(m, "p"), "enter")
	help := press(m, "?")
	text := ansi.Strip(help.helpText())
	for _, k := range []string{"J / K", "m / y, P p"} {
		if strings.Contains(text, k) {
			t.Fatalf("sorted: help still shows %q", k)
		}
	}
	if !strings.Contains(text, ": sort rank") {
		t.Fatal("sorted: help lacks the way back to rank order")
	}
	for _, k := range []string{"J", "K", "m", "y", "P", "p"} {
		got := press(m, k)
		if got.message != sortRefusal || got.grab.active || got.yanked != "" || got.confirm != "" {
			t.Fatalf("%s while sorted: message %q grab %v yanked %q confirm %q", k, got.message, got.grab.active, got.yanked, got.confirm)
		}
		if strings.Join(listIDs(got), " ") != strings.Join(listIDs(m), " ") {
			t.Fatalf("%s while sorted changed the rows", k)
		}
	}
	if sortRefusal != "moving needs rank order: : sort rank" {
		t.Fatal(sortRefusal)
	}
}
