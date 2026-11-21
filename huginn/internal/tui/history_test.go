package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// xfv9pw check 1 (Jørgen 2026-11-18 11:47): after a search, finding a task
// and acting on it, Up in the empty / prompt recalls the search. Repeated
// Up goes further back, Down returns, Down past the newest is the empty
// prompt again, and Enter runs the recalled search.
func TestSearchHistoryUpDown(t *testing.T) {
	m, ids, _ := searchBoard(t)
	m.setView("list")
	// Enter shows the results; Esc leaves them, so the next / is empty
	// (5cjy88: / on the results refines the search shown).
	m = press(m, "/", "Zebra", "enter", "esc")
	m = press(m, "/", "Task 07", "enter", "esc")
	m = press(m, "/", "Task 07", "enter", "esc") // no duplicates in a row
	m = press(m, "/", "up")
	if m.prompt != "/" || m.input.Value() != "Task 07" || m.query != "Task 07" {
		t.Fatalf("Up in the empty / prompt: prompt %q value %q query %q, want the last search", m.prompt, m.input.Value(), m.query)
	}
	m = press(m, "up")
	if m.input.Value() != "Zebra" || m.query != "Zebra" {
		t.Fatalf("second Up: value %q query %q, want Zebra (the one before; a repeat is kept once)", m.input.Value(), m.query)
	}
	m = press(m, "up")
	if m.input.Value() != "Zebra" {
		t.Fatalf("Up at the oldest: value %q, want it to stay on Zebra", m.input.Value())
	}
	// ↓ never recalls in / (5cjy88, crab 09:16Z): it picks a match.
	m = press(m, "down")
	if m.input.Value() != "Zebra" {
		t.Fatalf("Down: value %q, want Zebra still (↓ picks, never recalls)", m.input.Value())
	}
	m = press(m, "enter")
	if sel, ok := m.selected(); m.prompt != "" || !ok || sel.ID != ids["Zebra"] {
		t.Fatalf("Enter on the recalled Zebra: prompt %q, selected %q, want %s", m.prompt, sel.ID, ids["Zebra"])
	}
	// An edited recalled entry becomes a new entry; the old one stays.
	m = press(m, "esc", "/", "up", "backspace", "backspace", "enter")
	if m = press(m, "esc", "/", "up"); m.input.Value() != "Zeb" {
		t.Fatalf("after editing Zebra to Zeb: Up gives %q, want Zeb", m.input.Value())
	}
	if m = press(m, "up"); m.input.Value() != "Zebra" {
		t.Fatalf("Up after Zeb: %q, want Zebra", m.input.Value())
	}
	// Esc while typing cancels the search (5cjy88): what was typed is
	// kept too.
	m = press(m, "esc", "/", "Task 09", "esc", "/", "up")
	if m.input.Value() != "Task 09" {
		t.Fatalf("after / Task 09 Esc: Up gives %q, want Task 09", m.input.Value())
	}
	// Once typed, Up moves the rows as before: history is for the empty
	// (or recalled) prompt only.
	m = press(m, "esc", "/", "Task", "down", "down")
	if m.input.Value() != "Task" {
		t.Fatalf("Down with typed text changed the prompt to %q", m.input.Value())
	}
}

// xfv9pw check 2: the : palette keeps its own history, with the same keys.
func TestCommandHistoryUpDown(t *testing.T) {
	m, _, _ := searchBoard(t)
	m = press(m, "/", "Zebra", "enter")
	m = commandLine(m, "board")
	m = commandLine(m, "list")
	if m.view != "list" {
		t.Fatalf("view %q after : list", m.view)
	}
	m = press(m, ":", "up")
	if m.prompt != ":" || m.input.Value() != "list" {
		t.Fatalf("Up in the empty : prompt: prompt %q value %q, want list", m.prompt, m.input.Value())
	}
	m = press(m, "up")
	if m.input.Value() != "board" {
		t.Fatalf("second Up: %q, want board (the search Zebra is not a command)", m.input.Value())
	}
	m = press(m, "up")
	if m.input.Value() != "board" {
		t.Fatalf("Up at the oldest: %q, want board", m.input.Value())
	}
	m = press(m, "down", "down")
	if m.input.Value() != "" {
		t.Fatalf("Down past the newest: %q, want the empty prompt", m.input.Value())
	}
	m = press(m, "up", "up", "enter")
	if m.prompt != "" || m.view != "board" {
		t.Fatalf("Enter on the recalled board: prompt %q view %q", m.prompt, m.view)
	}
	// The search history has only the search.
	m = press(m, "/", "up")
	if m.input.Value() != "Zebra" {
		t.Fatalf("/ Up after commands: %q, want Zebra", m.input.Value())
	}
	if m = press(m, "up"); m.input.Value() != "Zebra" {
		t.Fatalf("/ history holds a command: %q", m.input.Value())
	}
}

// In a task, / finds in the page; it recalls the same searches, as vim's
// / history serves both.
func TestFindRecallsSearches(t *testing.T) {
	m, id := findBoard(t)
	m = press(m, "esc")
	m = press(m, "/", "row layout", "esc") // no task matches: Esc cancels, the text kept (5cjy88)
	m = openTask(t, m, id)
	m = press(m, "/", "up")
	if m.prompt != findPrompt || m.input.Value() != "row layout" || len(m.find.hits) == 0 {
		t.Fatalf("Up in an empty find: prompt %q value %q, %d hits", m.prompt, m.input.Value(), len(m.find.hits))
	}
	// Typed, Up and Down step through the matches as before.
	m = press(m, "esc", "/", "row")
	at := m.find.at
	m = press(m, "down")
	if m.input.Value() != "row" || len(m.find.hits) < 2 || m.find.at == at {
		t.Fatalf("Down in a typed find: prompt %q, match %d of %d (was %d); want the next match", m.input.Value(), m.find.at, len(m.find.hits), at)
	}
}

// xfv9pw check 3: the history outlives the hug tui that wrote it. A new
// model on the same file (the same board) recalls it; a model on another
// board's file does not. The file is written outside the task folder.
func TestHistoryPersistsPerBoard(t *testing.T) {
	m, _, _ := searchBoard(t)
	dir := t.TempDir()
	file := filepath.Join(dir, "board-a", "tui-history.json")
	m.opts.HistoryFile = file
	m = press(m, "/", "Zebra", "enter")
	m = commandLine(m, "board")
	// Quit and open the board again: a new model, the same file.
	again := newModel(m.store, Options{User: "tester", HistoryFile: file})
	again.width, again.height = 140, 40
	again.resize()
	if got := press(again, "/", "up"); got.input.Value() != "Zebra" {
		t.Fatalf("reopened: / Up gives %q, want Zebra", got.input.Value())
	}
	if got := press(again, ":", "up"); got.input.Value() != "board" {
		t.Fatalf("reopened: : Up gives %q, want board", got.input.Value())
	}
	b, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	if want := "{\n  \"version\": 1,\n  \"search\": [\n    \"Zebra\"\n  ],\n  \"command\": [\n    \"board\"\n  ]\n}\n"; string(b) != want {
		t.Fatalf("history file:\n%s\nwant:\n%s", b, want)
	}
	// Another board has its own file, so none of this.
	other := newModel(m.store, Options{User: "tester", HistoryFile: filepath.Join(dir, "board-b", "tui-history.json")})
	if got := press(other, "/", "up"); got.input.Value() != "" {
		t.Fatalf("another board's / Up gives %q, want nothing", got.input.Value())
	}
	// Nothing landed in the task folder.
	filepath.WalkDir(m.store.Root, func(p string, d os.DirEntry, err error) error {
		if err == nil && strings.Contains(d.Name(), "history") {
			t.Errorf("history in the task folder: %s", p)
		}
		return nil
	})
	// Two hug tui on one board, open at once, keep each other's entries.
	a := newModel(m.store, Options{User: "tester", HistoryFile: file})
	b2 := newModel(m.store, Options{User: "tester", HistoryFile: file})
	a, b2 = press(a, "/", "up", "esc"), press(b2, "/", "up", "esc") // both read the file now
	press(a, "/", "Task 01", "enter")
	press(b2, "/", "Task 02", "enter")
	if got := press(newModel(m.store, Options{User: "tester", HistoryFile: file}), "/", "up", "up", "up"); got.input.Value() != "Zebra" {
		t.Fatalf("after two sessions wrote: third Up gives %q, want Zebra (Task 02, Task 01, Zebra)", got.input.Value())
	}
}
