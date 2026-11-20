package tui

import (
	"fmt"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/veridit/hug/huginn/internal/task"
)

// tz9z14 check 1 (Jørgen 15:19: "reloaded hug v0.7.63" stayed under a
// task's detail, "noise because it's from the main screen"): a message
// belongs to the screen that made it. After a reload the overview says
// so, moving in it keeps it, and opening a task, another view or a file
// clears it. Every message, not only an action's result (z82qsy).
func TestReloadMessageClearsOnOpeningATask(t *testing.T) {
	m, _ := glanceModel(t, 120, 30)
	m.setView("overview")
	// Run sets it before the first update, as after a handoff.
	m.message = "reloaded hug v0.7.63"
	if stay := press(m, "j"); !strings.Contains(ansi.Strip(stay.View()), "reloaded hug v0.7.63") {
		t.Fatalf("moving in the overview cleared the reload message: %q", stay.message)
	}
	for name, step := range map[string]func(model) model{
		"a task":       func(m model) model { return press(m, "enter") },
		"another view": func(m model) model { return press(m, "v", "b") },
		"a task by id": func(m model) model { return press(m, "#", "000002", "enter") },
	} {
		got := step(m)
		if name != "another view" && !got.detail {
			t.Fatalf("%s: no detail opened (%q)", name, got.message)
		}
		if strings.Contains(ansi.Strip(got.View()), "reloaded hug") {
			t.Fatalf("%s: the reload message stays: %q", name, got.message)
		}
	}
}

// tz9z14: any other message too, e.g. the hint Space leaves in a file,
// goes when the next file shows (n) or the reader goes back to FILES (u).
func TestHintClearsOnMovingOn(t *testing.T) {
	m, _ := glanceModel(t, 120, 30)
	m = openTask(t, m, "000001")
	m = press(toFiles(t, m), " ")
	if !m.glance {
		t.Fatal("Space did not show the file")
	}
	m = press(m, " ")
	hint := m.message
	if hint != "a file is read only here: Enter opens it in the editor" {
		t.Fatalf("Space in a file left %q", hint)
	}
	if stay := press(m, "j"); stay.message != hint {
		t.Fatalf("scrolling the file cleared the hint: %q", stay.message)
	}
	for name, step := range map[string]func(model) model{
		"next file": func(m model) model { return press(m, "n") },
		"back out":  func(m model) model { return press(m, "u") },
	} {
		if got := step(m); strings.Contains(ansi.Strip(got.View()), hint) {
			t.Fatalf("%s: the hint stays: %q", name, got.message)
		}
	}
}

// overviewCapture is zhvkmj's colours-overview.ans in shape: an overview
// frame, 8 blank rows, the legend and the keys, 32 lines in all.
func overviewCapture() string {
	lines := []string{
		"\x1b[1mHUG TASKS\x1b[0m · overview   7/7 tasks · ! 0 broken · ? 2 for you",
		"[overview]   list    board    !lint",
		"todo 3  ·  doing 4 (1 depends)  ·  depends 0  ·  done 0  ·  dropped 0",
		"No filters · f picker · 1 todo  2 doing  3 done  4 depends",
	}
	for i := 0; i < 17; i++ {
		lines = append(lines, fmt.Sprintf("  ◑ 1/2  row%02d   feature high     asked     A task on the captured board", i))
	}
	lines = append(lines, "")
	for i := 0; i < 8; i++ {
		lines = append(lines, "                                                                                ")
	}
	lines = append(lines, "All tasks · : filters and commands · ! lint finding · ↳ epic", "", "j/k move · Enter open · h/l Tab pane · filter · # id · / find · : cmd · ? help")
	return strings.Join(lines, "\n") + "\n"
}

// tz9z14 check 2 (Jørgen 15:19, zhvkmj's detail: the overview legend
// under the header and ~8 blank rows above DESCRIPTION). Not a leak: it
// was the capture the question shows (asked_show, 5z4zt4), drawn inline
// as it was taken, so it read as the live screen. Since qcq3tm the page
// draws no capture at all: the question carries a link line, and the
// legend never shows on the page. Space in FILES still shows it exactly
// as taken.
func TestQuestionPictureIsFramed(t *testing.T) {
	var golden []string
	for _, w := range []int{80, 140} {
		m, _ := pictureModel(t, w, 60, nil, "")
		if _, _, err := m.store.Attach("000002", "colours-overview.ans", []byte(overviewCapture()), "", false); err != nil {
			t.Fatal(err)
		}
		if _, err := m.store.Ask("000002", task.Question{What: "do the colours read as one scheme?", For: "jhf", Show: "colours-overview.ans"}, "alice"); err != nil {
			t.Fatal(err)
		}
		// A fresh model reads the board as it is now.
		m = newModel(m.store, m.opts)
		m.width, m.height = w, 60
		m.resize()
		m.setView("list")
		m = openTask(t, m, "000002")
		screen := ansi.Strip(m.View())
		if !strings.Contains(screen, "→ colours-overview.ans · Enter shows") || strings.Contains(screen, "capture, as it was taken") || strings.Contains(screen, "row05") {
			t.Fatalf("at %d the capture is drawn on the page, or its link is missing:\n%s", w, screen)
		}
		golden = append(golden, fmt.Sprintf("== 000002 asked_show colours-overview.ans at %d", w), screen)

		// Space in FILES shows it as taken: the 8 blank rows are there.
		f := toFiles(t, m)
		for i := 0; i < 5; i++ {
			if name, _, _ := f.glanceFile(mustSelected(t, f)); name == "colours-overview.ans" {
				break
			}
			f = press(f, "j")
		}
		g := press(f, " ")
		if !g.glance {
			t.Fatalf("at %d Space did not show colours-overview.ans", w)
		}
		file := strings.Split(ansi.Strip(g.frameView()), "\n")
		if !strings.Contains(file[0], "colours-overview.ans") {
			t.Fatalf("at %d Space showed %q", w, file[0])
		}
		run, longest := 0, 0
		for _, l := range file[2:] {
			if strings.HasPrefix(l, "All tasks") {
				break
			}
			if strings.TrimSpace(l) == "" {
				run++
				longest = max(longest, run)
			} else {
				run = 0
			}
		}
		if longest != 9 {
			t.Fatalf("at %d the file view shows %d blank rows above the legend, want the capture's 9:\n%s", w, longest, strings.Join(file, "\n"))
		}
	}
	goldenFile(t, "testdata/detail/picture-framed.txt", strings.Join(golden, "\n")+"\n")
}
