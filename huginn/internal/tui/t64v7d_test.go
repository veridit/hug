package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

// textInputs opens each text input of the TUI: the answer, wait, strike,
// drop and new-title prompts, the filter's text after its field, / find,
// the : palette (its argument) and # jump.
func textInputs(t *testing.T) map[string]model {
	t.Helper()
	out := map[string]model{}
	m := stageBoard(t)
	m.opts.User = "jhf"
	m.width, m.height = 100, 30
	if _, err := m.store.Design("eeeeee", "drop type from the rows?", "crab", "jhf"); err != nil {
		t.Fatal(err)
	}
	m.refresh("")
	m.setView("list")
	open := func(id string) model {
		mm := press(m, "#", id, "enter")
		if !mm.detail {
			t.Fatalf("%s not open", id)
		}
		return mm
	}
	out["answer"] = press(open("eeeeee"), "a")
	out["wait"] = press(open("aaaaaa"), "w")
	out["strike"] = press(open("aaaaaa"), "j", "-") // check 2: check 1 is ticked
	out["drop"] = press(open("ffffff"), "x")
	out["new title"] = press(m, "N")
	out["filter text"] = press(m, "f", "g")
	out["find"] = press(m, "/")
	out["palette"] = press(m, ":", "g", "r", "o", "u", "p", " ")
	out["jump"] = press(m, "#")
	for name, mm := range out {
		if mm.prompt == "" || !mm.textInput() {
			t.Fatalf("%s: no text input open (prompt %q, message %q)", name, mm.prompt, mm.message)
		}
	}
	return out
}

// field is what the open text input holds: the answer box or the line.
func field(m model) string {
	if m.prompt == answerPrompt {
		return m.answerBox.Value()
	}
	return m.input.Value()
}

// t64v7d check 1: in each text input, typing "done by morning?" leaves
// exactly that text in the field, and help stays closed (Jørgen
// 2026-11-18 01:52: ? at the end of his question opened help).
func TestQuestionMarkIsTextInEveryInput(t *testing.T) {
	const typed = "done by morning?"
	for name, m := range textInputs(t) {
		before := field(m)
		for _, r := range typed {
			m = press(m, string(r))
		}
		if m.help {
			t.Errorf("%s: ? opened help", name)
		}
		if got := field(m); got != before+typed {
			t.Errorf("%s: the field holds %q, want %q", name, got, before+typed)
		}
	}
}

// t64v7d check 2: F1 in a text input opens help with the typed text
// intact underneath; Esc or F1 returns to it. The prompt's footer and its
// help name F1.
func TestF1IsHelpInATextInput(t *testing.T) {
	for name, m := range textInputs(t) {
		m = press(m, "d", "u", "e", "?")
		typed := field(m)
		if bar := ansi.Strip(m.keyBar()); !strings.Contains(bar, "F1 help") || strings.Contains(bar, "? help") {
			t.Errorf("%s: the footer does not name F1 for help: %q", name, bar)
		}
		for _, back := range []string{"esc", "f1"} {
			h := press(m, "f1")
			if !h.help {
				t.Fatalf("%s: F1 did not open help", name)
			}
			if text := ansi.Strip(h.helpText()); !strings.Contains(text, "F1") {
				t.Errorf("%s: the help does not name F1:\n%s", name, text)
			}
			r := press(h, back)
			if r.help || r.prompt != m.prompt || field(r) != typed {
				t.Errorf("%s: %s after F1: help %v, prompt %q, field %q; want prompt %q, field %q", name, back, r.help, r.prompt, field(r), m.prompt, typed)
			}
		}
	}
}

// t64v7d check 3: outside text input ? still opens help: a list, a task,
// the sort builder, the filter's field chooser and a confirm.
func TestQuestionMarkIsHelpOutsideText(t *testing.T) {
	m := stageBoard(t)
	m.width, m.height = 100, 30
	m.setView("list")
	for name, mm := range map[string]model{
		"list": m, "task": press(m, "enter"), "sort": press(m, ":", "s", "o", "r", "t", "enter"),
		"filter fields": press(m, "f"), "confirm": press(m, "t"),
	} {
		if mm.textInput() {
			t.Fatalf("%s: a text input (prompt %q)", name, mm.prompt)
		}
		if h := press(mm, "?"); !h.help {
			t.Errorf("%s: ? did not open help (prompt %q, confirm %q)", name, mm.prompt, mm.confirm)
		}
	}
}
