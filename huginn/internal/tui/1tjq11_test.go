package tui

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/veridit/hug/huginn/internal/task"
)

// 1tjq11 (Jørgen 2026-11-22 21:37Z, 21:49Z): on a big task's page every
// key took most of a second (p39w4a in Seid, 668 KB and 759 notes: one j
// p50 769 ms). A key on the page answers within 50 ms, on a generated
// task as big: about 700 KB with 750 notes, ids, task:ID links and file
// names in its text, a question asked of the reader and open requests.

// bigTaskKeyBudget is how long one key and the frame after it may take.
const bigTaskKeyBudget = 50 * time.Millisecond

// bigBoard is a board with a few small tasks and one big one (its id
// returned), the big task asked of jhf, opened at 180x50.
func bigBoard(tb testing.TB) (model, string) {
	tb.Helper()
	tb.Setenv("NO_COLOR", "1")
	s := &task.Store{Root: tb.TempDir()}
	if err := os.WriteFile(filepath.Join(s.Root, "DESIGN.md"), []byte("# Design\n\n## Imported data\nx\n"), 0o644); err != nil {
		tb.Fatal(err)
	}
	var small []string
	for i := 0; i < 6; i++ {
		r, err := s.New(task.NewOptions{Title: fmt.Sprintf("Small task %d", i), Goal: "a small goal", Acceptance: []string{"it holds"}})
		if err != nil {
			tb.Fatal(err)
		}
		small = append(small, r.Task.ID)
	}
	r, err := s.New(task.NewOptions{
		Title:      "Typed-module bytes depend on load history",
		Goal:       "Identical source gives identical bytes, see task:" + small[0] + " and " + small[1] + ".",
		Purpose:    "One reproducible compiled form; DESIGN.md#Imported data names it.",
		Principle:  "Only content and dependencies determine identity, as " + small[2] + " says.",
		Details:    bigText("Details paragraph", 40, small),
		Acceptance: []string{"The same module compiles to the same bytes (" + small[3] + ")", "The cause is named with file:line", "The variant chain disappears"},
	})
	if err != nil {
		tb.Fatal(err)
	}
	id := r.Task.ID
	for i := 0; i < 4; i++ {
		if _, err := s.Request(id, fmt.Sprintf("request %d: make this clearer, see %s", i, small[i%len(small)]), "jhf"); err != nil {
			tb.Fatal(err)
		}
	}
	if _, err := s.TakeAs(id, "orangutan", "claude-opus-5-5"); err != nil {
		tb.Fatal(err)
	}
	if _, err := s.Ask(id, task.Question{What: "Do I split the lane into pieces, as " + small[4] + " did?", For: "jhf"}, "orangutan"); err != nil {
		tb.Fatal(err)
	}
	// The notes and history written straight into the file: 750 notes
	// through Store.Note would rewrite a growing file 750 times.
	path := filepath.Join(s.Root, filepath.FromSlash(r.Task.Path))
	b, err := os.ReadFile(path)
	if err != nil {
		tb.Fatal(err)
	}
	text := string(b)
	notes := bigNotes(750, small)
	text = strings.Replace(text, "\n## Notes\n", "\n## Notes\n\n"+notes, 1)
	var hist strings.Builder
	for i := 0; i < 120; i++ {
		fmt.Fprintf(&hist, "- 2026-11-%02dT%02d:%02d doing/active -> doing/waiting (wait); was waiting_on: %s\n", 1+i%28, i%24, i%60, small[i%len(small)])
	}
	text = strings.Replace(text, "\n## History\n", "\n## History\n"+hist.String(), 1)
	if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
		tb.Fatal(err)
	}
	if n := len(text); n < 650_000 || n > 760_000 {
		tb.Fatalf("the big task is %d bytes, want about 700 KB", n)
	}
	m := newModel(s, Options{User: "jhf"})
	m.width, m.height = 180, 50
	m.resize()
	m = press(m, "#", id, "enter")
	if sel, ok := m.selected(); !ok || sel.ID != id || !m.detail {
		tb.Fatalf("could not open %s: %q", id, m.message)
	}
	return m, id
}

// bigText is n paragraphs of prose naming ids, links and files.
func bigText(what string, n int, ids []string) string {
	var b strings.Builder
	for i := 0; i < n; i++ {
		fmt.Fprintf(&b, "%s %d: the checked reference carries a fingerprint (typecheck.ml:%d), never a locator; %s and task:%s agree, DESIGN.md says why. `canon_hash` must match.\n\n", what, i, 7000+i, ids[i%len(ids)], ids[(i+1)%len(ids)])
	}
	return b.String()
}

// bigNotes are n notes of about 870 bytes, each one line, as agents write.
func bigNotes(n int, ids []string) string {
	var b strings.Builder
	for i := 0; i < n; i++ {
		fmt.Fprintf(&b, "2026-11-%02d %02d:%02d (orangutan) ", 1+i%28, i%24, i%60)
		for k := 0; k < 4; k++ {
			fmt.Fprintf(&b, "Probe %d.%d: cold rc0 hash %08x and warm rc1 hash %08x differ; compiler.ml:%d qualified by cid only on collision, see %s and task:%s, evidence in .jcode/scratch/TASK-1960/marker-census.md. ", i, k, i*7919+k, i*104729+k, 1500+i+k, ids[(i+k)%len(ids)], ids[(i+k+1)%len(ids)])
		}
		b.WriteString("\n\n")
	}
	return b.String()
}

// keyTimes presses k on m n times (from the same m each time, so the
// page stays where it was) and returns how long each key and the frame
// after it took, sorted.
func keyTimes(m model, n int, k string) []time.Duration {
	var out []time.Duration
	for i := 0; i < n; i++ {
		start := time.Now()
		next := press(m, k)
		_ = next.View()
		out = append(out, time.Since(start))
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

// TestBigTaskKeys: j, k, Tab and a character typed in the request and
// the answer box each answer within bigTaskKeyBudget at the median.
func TestBigTaskKeys(t *testing.T) {
	if raceEnabled {
		t.Skip("timing under -race measures the race detector")
	}
	page, _ := bigBoard(t)
	request := press(page, "r")
	if request.prompt != requestPrompt {
		t.Fatalf("r opened %q", request.prompt)
	}
	answer := press(page, "a")
	if answer.prompt != answerPrompt {
		t.Fatalf("a opened %q: %q", answer.prompt, answer.message)
	}
	for _, c := range []struct {
		name string
		m    model
		key  string
	}{
		{"j", page, "j"},
		{"k", press(page, "j", "j"), "k"},
		{"tab", page, "tab"},
		{"request box", press(request, "w"), "x"},
		{"answer box", press(answer, "w"), "x"},
	} {
		times := keyTimes(c.m, 9, c.key)
		med := times[len(times)/2]
		t.Logf("%s: p50 %v max %v (load %s)", c.name, med, times[len(times)-1], loadAvg())
		if med > bigTaskKeyBudget {
			t.Errorf("%s on a 700 KB task: p50 %v, want under %v", c.name, med, bigTaskKeyBudget)
		}
	}
}

// BenchmarkBigTaskJ is one j and the frame after it on the big task.
func BenchmarkBigTaskJ(b *testing.B) {
	m, _ := bigBoard(b)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = press(m, "j").View()
	}
}

// BenchmarkBigTaskRequestChar is one character typed in the request box
// and the frame after it on the big task.
func BenchmarkBigTaskRequestChar(b *testing.B) {
	m, _ := bigBoard(b)
	m = press(m, "r", "w")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = press(m, "x").View()
	}
}

// loadAvg is the machine's load, for the log next to a time.
func loadAvg() string {
	if out, err := os.ReadFile("/proc/loadavg"); err == nil {
		return strings.Join(strings.Fields(string(out))[:3], " ")
	}
	if out, err := exec.Command("sysctl", "-n", "vm.loadavg").Output(); err == nil {
		return strings.Trim(strings.TrimSpace(string(out)), "{} ")
	}
	return "unknown"
}
