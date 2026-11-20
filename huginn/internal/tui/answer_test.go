package tui

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/x/ansi"
	"github.com/veridit/hug/huginn/internal/task"
)

// 3jtfw9 check 1: in the detail of a task whose stage waits on you, the
// question is at the top with who asked; a opens an input right under it;
// Enter stores the answer as one History line with your name, the stage
// re-derives, and the asker's unread answer shows in its place (golden at
// 100: the question block, the open input, the answered state).
func TestAnswerInDetail(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	m := stageBoard(t) // the TUI's clock: 09:53
	m.opts.User = "jhf"
	m.width, m.height = 100, 30
	// The question is asked at 09:45, by the task's clock.
	oldNow := task.Now
	task.Now = func() time.Time { return time.Date(2026, 11, 17, 9, 45, 0, 0, time.Local) }
	t.Cleanup(func() { task.Now = oldNow })
	// eeeeee asks jhf a design question; give it an asker.
	if _, e := m.store.Design("eeeeee", "drop type from the rows, keep it, or show it only when not feature?", "crab", "jhf"); e != nil {
		t.Fatal(e)
	}
	m.refresh("")
	m.setView("list")
	m = press(m, "#", "eeeeee", "enter")
	if !m.detail {
		t.Fatal("detail not open")
	}
	// The header and the page's first lines: the question is at the top
	// of the details, once (0gztsy), the header holds its stage badge.
	head := func(m model) string {
		tk, _ := m.selected()
		page := strings.Split(m.viewport.View(), "\n")
		return ansi.Strip(strings.Join(append(m.detailHeader(tk), page[:min(5, len(page))]...), "\n"))
	}
	var got []string
	asked := head(m)
	for _, want := range []string{"[design]", "> ? design: drop type from the rows, keep it, or show it only when not feature?", "└ crab · 8 min ago · answer"} {
		if !strings.Contains(asked, want) {
			t.Fatalf("detail lacks %q:\n%s", want, asked)
		}
	}
	if m.detailPane != focusQuestion || strings.Contains(asked, "└─ question for you") {
		t.Fatalf("not opened on the question (pane %d), or it shows twice:\n%s", m.detailPane, asked)
	}
	got = append(got, "== asked", asked)
	// a on a task that asks someone else is refused.
	m.opts.User = "carol"
	if m2 := press(m, "a"); m2.prompt != "" || !strings.Contains(m2.message, "asks you nothing") {
		t.Fatalf("carol: prompt %q message %q", m2.prompt, m2.message)
	}
	m.opts.User = "jhf"
	m = press(m, "a")
	if m.prompt != answerPrompt {
		t.Fatalf("prompt %q", m.prompt)
	}
	// An empty answer writes nothing.
	if m2 := press(m, "enter"); !strings.Contains(m2.message, "not answered") {
		t.Fatalf("empty: %q", m2.message)
	}
	for _, r := range "drop it" {
		m = press(m, string(r))
	}
	typing := head(m)
	if !strings.Contains(typing, "answer: drop it") || strings.Contains(typing, "· answer") {
		t.Fatalf("input not under the question:\n%s", typing)
	}
	got = append(got, "== typing", typing)
	m = press(m, "enter")
	if m.message != "eeeeee moved to NEXT UP" { // answered: out of ASKED (24403z)
		t.Fatalf("message %q", m.message)
	}
	var tk task.Task
	for _, x := range m.all {
		if x.ID == "eeeeee" {
			tk = x
		}
	}
	b, _ := os.ReadFile(filepath.Join(m.store.Root, tk.Path))
	if tk.Stage != "ready" || !strings.Contains(string(b), "answered design, by jhf, for crab (todo/design -> todo/ready):\n  > drop type from the rows, keep it, or show it only when not feature?\n  drop it\n") {
		t.Fatalf("stage %q:\n%s", tk.Stage, b)
	}
	answered := head(m)
	if strings.Contains(answered, "? design") || !regexp.MustCompile(`✓ jhf answered design: drop it\s*\n  unread by crab`).MatchString(answered) {
		t.Fatalf("after:\n%s", answered)
	}
	got = append(got, "== answered", answered)
	text := strings.Join(got, "\n") + "\n"
	path := filepath.Join("testdata", "detail", "answer-100.txt")
	if os.Getenv("HUG_UPDATE_GOLDEN") == "1" {
		if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("%v\nfix: HUG_UPDATE_GOLDEN=1 go test -run TestAnswerInDetail ./huginn/internal/tui/", err)
	}
	if text != strings.ReplaceAll(string(want), "\r\n", "\n") {
		t.Fatalf("answer detail differs from %s:\n%s", path, text)
	}
}
