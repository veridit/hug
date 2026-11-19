package tui

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/veridit/hug/huginn/internal/task"
)

// gateGit makes git wait until the returned release is called: a machine
// too loaded to start a process (adqdfe). The cleanup releases it.
func gateGit(t *testing.T) (release func()) {
	t.Helper()
	gate := make(chan struct{})
	real := git
	git = func(root string, args ...string) (string, error) {
		<-gate
		return real(root, args...)
	}
	done := false
	release = func() {
		if !done {
			done = true
			close(gate)
		}
	}
	t.Cleanup(func() { release(); git = real })
	return release
}

// messagesOf runs cmd and each command of a batch (nested too) in the
// background and sends their messages on the returned channel.
func messagesOf(cmd tea.Cmd) <-chan tea.Msg {
	out := make(chan tea.Msg, 16)
	var run func(c tea.Cmd)
	run = func(c tea.Cmd) {
		if c == nil {
			return
		}
		go func() {
			msg := c()
			if b, ok := msg.(tea.BatchMsg); ok {
				for _, c := range b {
					run(c)
				}
				return
			}
			out <- msg
		}()
	}
	run(cmd)
	return out
}

// adqdfe check 2: a reload does not wait on git. The board lands while
// git cannot start; the close times follow once it can, and RECENTLY DONE
// takes their order then.
func TestReloadDoesNotWaitOnGit(t *testing.T) {
	m, ids, commit := closeBoard(t)
	m.setView("overview")
	// A task closed elsewhere: the commit is new to the TUI.
	s := &task.Store{Root: m.store.Root}
	if _, err := s.Check(ids["November"], 1); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Take(ids["November"], "tester"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Done(ids["November"]); err != nil {
		t.Fatal(err)
	}
	commit(time.Date(2026, 11, 2, 0, 0, 0, 0, time.UTC), "hug task done "+ids["November"])
	release := gateGit(t)
	cmd := m.reload()
	msgs := make(chan tea.Msg, 1)
	go func() { msgs <- cmd() }()
	var msg tea.Msg
	select {
	case msg = <-msgs:
	case <-time.After(10 * time.Second):
		t.Fatal("the reload waited on git: no board while git could not start")
	}
	next, after := m.Update(msg)
	m = next.(model)
	if !strings.Contains(strings.Join(sectionIDs(m, sectionRecent), " "), ids["November"]) {
		t.Fatalf("the reload does not show November done: RECENTLY DONE %v", sectionIDs(m, sectionRecent))
	}
	afterMsgs := messagesOf(after)
	release()
	deadline := time.After(10 * time.Second)
	for got := false; !got; {
		select {
		case msg = <-afterMsgs:
			_, got = msg.(closesMsg)
		case <-deadline:
			t.Fatal("no close times after the reload, once git could start")
		}
	}
	next, _ = m.Update(msg)
	m = next.(model)
	got := strings.Join(sectionIDs(m, sectionRecent)[:3], " ")
	want := strings.Join([]string{ids["November"], ids["Mike"], ids["Lima"]}, " ")
	if got != want {
		t.Fatalf("after the close times: %s, want November Mike Lima %s", got, want)
	}
}
