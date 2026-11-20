package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/veridit/hug/huginn/internal/task"
)

// gmzd87 (crab 11:57 condition 2): the TUI's r and a deliver by the same
// function as hug task request and answer: Options.Deliver, called once
// per write with the task, the kind and who wrote it; the message line
// says the receipt, and REQUESTS shows each request's receipt as its one
// label.
func TestDeliverFromBoxes(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	m := stageBoard(t)
	m.opts.User = "jhf"
	m.width, m.height = 120, 30
	var calls []string
	m.opts.Deliver = func(id, kind, by string) (string, error) {
		calls = append(calls, id+" "+kind+" "+by)
		// What cli.Deliver records: the receipt line on the task.
		if _, err := m.store.Delivered(id, task.Item{Kind: kind, N: 1}, "crab", "session_crab_1_ab", ""); err != nil {
			return "", err
		}
		return "pending, delivered to crab (session_crab_1_ab)", nil
	}
	m.resize()
	m.setView("list")
	m.clamp()
	m = selectRow(t, m, "ffffff")
	m = press(m, "r")
	for _, r := range "make it per board" {
		m = press(m, string(r))
	}
	m = press(m, "enter")
	if len(calls) != 1 || calls[0] != "ffffff request jhf" {
		t.Fatalf("r delivered %q", calls)
	}
	if !strings.Contains(m.message, "requested ffffff: pending, delivered to crab (session_crab_1_ab)") {
		t.Fatalf("message %q", m.message)
	}
	page := ansi.Strip(m.detailText(mustSelected(t, m)))
	if !strings.Contains(page, "1. jhf · ") || !strings.Contains(page, "· pending, delivered to crab") {
		t.Fatalf("REQUESTS lacks the receipt:\n%s", page)
	}
	m.store.Acknowledge("ffffff", task.Item{Kind: "request", N: 1}, "crab")
	m.refresh("")
	if page = ansi.Strip(m.detailText(mustSelected(t, m))); !strings.Contains(page, "· acknowledged by crab") {
		t.Fatalf("REQUESTS after the ack:\n%s", page)
	}
	// a: the answer to eeeeee's design question goes the same way.
	if _, e := m.store.Design("eeeeee", "type stays?", "nautilus", "jhf"); e != nil {
		t.Fatal(e)
	}
	m.refresh("")
	m = press(m, "esc", "#", "eeeeee", "enter", "a")
	for _, r := range "drop it" {
		m = press(m, string(r))
	}
	m = press(m, "enter")
	if len(calls) != 2 || calls[1] != "eeeeee answer jhf" {
		t.Fatalf("a delivered %q", calls)
	}
	// A refused write delivers nothing.
	m = press(m, "a")
	if len(calls) != 2 {
		t.Fatalf("a refused answer delivered: %q", calls)
	}
}

// qsqhpe check 3 (TUI): Space on the asked check, confirmed, by the person
// asked, answers the question and delivers it as an answer, by the same
// Options.Deliver as a; the message says it answered.
func TestTickAnswerDelivers(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	m := askedBoard(t)
	// As hug tui sets it (cli/hooks.go): the store writes as the TUI user.
	m.store.Actor = m.me()
	m.width, m.height = 120, 30
	m.resize()
	var calls []string
	m.opts.Deliver = func(id, kind, by string) (string, error) {
		calls = append(calls, id+" "+kind+" "+by)
		return "pending, delivered to crab (session_crab_1_ab)", nil
	}
	m = openTask(t, m, "aaaaaa")
	m = press(m, " ")
	if m.confirm != " " {
		t.Fatalf("Space on the asked check: confirm %q", m.confirm)
	}
	m = press(m, "enter")
	if len(calls) != 1 || calls[0] != "aaaaaa answer jhf" {
		t.Fatalf("delivered %q (message %q)", calls, m.message)
	}
	// The tick moved it out of ASKED: where it went, then the receipt
	// after " · " (24403z, crab 2026-11-20 20:07Z).
	if m.message != "aaaaaa moved to IN PROGRESS · pending, delivered to crab (session_crab_1_ab)" {
		t.Fatalf("message %q", m.message)
	}
	tk := mustSelected(t, m)
	if tk.Stage != "active" {
		t.Fatalf("stage %q after the tick", tk.Stage)
	}
}
