package tui

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/veridit/hug/huginn/internal/task"
)

// yf39ya check 3 (TUI): a shipped task the release asked of jhf about its
// only open check (an after-release confirmation) is in his ASKED with the
// question (golden at 80 and 140), and Space on that check, his tick, goes
// through the same helper as hug task check: the task closes with the
// release in History and the message says so.
func TestShippedAskedClosesOnTick(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	m := stageBoard(t)
	m.opts.User = "jhf"
	if err := os.WriteFile(filepath.Join(m.store.Root, "hug.toml"), []byte("[people.jhf]\nname = \"Jørgen H. Fjeld\"\nemail = [\"jorgen@veridit.no\"]\n\n[board]\ncoordinator = \"crab\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	// What the release pass wrote: the shipped note, then the question.
	if _, err := m.store.Note("aaaaaa", "shipped in v0.1.1 (abcdef0)"); err != nil {
		t.Fatal(err)
	}
	if _, err := m.store.Ask("aaaaaa", task.Question{What: "Jørgen confirms on his board after release", For: "jhf", Check: 2}, "crab"); err != nil {
		t.Fatal(err)
	}
	m.refresh("")
	var golden []string
	for _, w := range []int{80, 140} {
		view := focusAt(t, m, w, "aaaaaa")
		asked := section(view, "ASKED")
		if !strings.Contains(asked, "aaaaaa") || !strings.Contains(view, "asked: check 2 · Jørgen confirms on his board") {
			t.Fatalf("at %d not in ASKED with its question:\n%s", w, view)
		}
		golden = append(golden, "== overview at "+strconv.Itoa(w), view)
	}
	goldenFile(t, "testdata/overview/shipped-asked.txt", strings.Join(golden, "\n")+"\n")

	m.store.Actor = m.me()
	m.width, m.height = 120, 30
	m.resize()
	m = openTask(t, m, "aaaaaa")
	m = press(m, " ")
	m = press(m, "enter")
	// Closed by the release: where it went, then what the release did
	// (24403z, crab 2026-11-20 20:07Z).
	if m.message != "aaaaaa moved to RECENTLY DONE · after the release: done" {
		t.Fatalf("message %q", m.message)
	}
	d, err := m.store.Get("aaaaaa")
	if err != nil {
		t.Fatal(err)
	}
	if d.Task.State != "done" || d.Task.Fields["done_release"] != "v0.1.1" {
		t.Fatalf("after the tick: %s done_release %q", d.Task.State, d.Task.Fields["done_release"])
	}
	if !strings.Contains(ansi.Strip(string(d.Bytes())), "(done after jhf ticked check 2 (release v0.1.1))") {
		t.Fatalf("History:\n%s", d.Bytes())
	}
}
