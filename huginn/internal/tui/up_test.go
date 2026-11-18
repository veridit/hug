package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

// 9xfnwr check 1: in a task, u goes back to the view it was opened from,
// with the same task selected: from the overview, the list and the board.
func TestUGoesUpFromDetail(t *testing.T) {
	for _, view := range []string{"overview", "list", "board"} {
		t.Run(view, func(t *testing.T) {
			m, ids, _ := searchBoard(t)
			m.setView(view)
			if view == "overview" {
				m = nextUp(m)
			}
			m = press(m, "j", "j", "j")
			want, _ := m.selected()
			m = press(m, "enter")
			if !m.detail {
				t.Fatal("Enter did not open the task")
			}
			m = press(m, "u")
			sel, ok := m.selected()
			if m.detail || m.view != view || !ok || sel.ID != want.ID {
				t.Fatalf("u: detail %v view %s selected %s, want %s in %s", m.detail, m.view, sel.ID, want.ID, view)
			}
			_ = ids
		})
	}
}

// 9xfnwr check 2: after following a link, u returns to the task it was
// followed from, and u again to the list.
func TestUReturnsFromFollowedLink(t *testing.T) {
	m, main, _, beta := linkModel(t)
	m = press(m, "tab", "j", "enter")
	if tk, _ := m.selected(); tk.ID != beta.ID {
		t.Fatalf("followed to %s, want %s", tk.ID, beta.ID)
	}
	m = press(m, "u")
	if tk, _ := m.selected(); !m.detail || tk.ID != main.ID {
		t.Fatalf("u after a link: detail %v open %s, want %s", m.detail, tk.ID, main.ID)
	}
	m = press(m, "u")
	if tk, _ := m.selected(); m.detail || m.view != "list" || tk.ID != main.ID {
		t.Fatalf("second u: detail %v view %s selected %s", m.detail, m.view, tk.ID)
	}
}

// 9xfnwr check 3: the detail key bar shows "u up".
func TestDetailKeyBarShowsUUp(t *testing.T) {
	for _, w := range []int{80, 140} {
		m := fixture(t)
		m.width, m.height = w, 30
		m.resize()
		m = press(m, "enter")
		bar := ansi.Strip(m.keyBar())
		if !strings.Contains(bar, "· up") { // u marked in its word (e3sevr check 14)
			t.Fatalf("at %d the detail key bar lacks 'u up': %q", w, bar)
		}
	}
	// Outside a task u is not shown: only keys that work are.
	m := fixture(t)
	if strings.Contains(ansi.Strip(m.keyBar()), " up") {
		t.Fatal("u up shown outside a task")
	}
}

// 9xfnwr check 4: inside : sort, u is still the updated field.
func TestUInSortIsUpdated(t *testing.T) {
	m := manyModel(t)
	m = press(m, "enter", ":", "sort", "enter", "u")
	if !m.detail || m.prompt != "sort" || len(m.sortDraft) != 1 || m.sortDraft[0].field != "updated" {
		t.Fatalf("u in sort: detail %v prompt %q draft %+v", m.detail, m.prompt, m.sortDraft)
	}
	m = press(manyModel(t), ":", "sort", "enter", "u", "enter")
	if len(m.sort) != 1 || m.sort[0].field != "updated" {
		t.Fatalf("sort u Enter: %+v", m.sort)
	}
}
