package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/veridit/hug/huginn/internal/task"
)

// tagBoard is a board whose hug.toml lists parser before compiler:
//
//	Lex       parser
//	Both      compiler parser (listed order puts it under parser)
//	Codegen   compiler
//	Odd       zzz (not on the list: its own heading)
//	Bare      no tags
func tagBoard(t *testing.T) (model, map[string]string) {
	t.Helper()
	s := &task.Store{Root: t.TempDir()}
	if err := os.WriteFile(filepath.Join(s.Root, task.ConfigName), []byte("tags = [\"parser\", \"compiler\"]\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	ids := map[string]string{}
	for _, c := range []struct {
		title string
		tags  []string
	}{{"Lex", []string{"parser"}}, {"Both", []string{"compiler", "parser"}}, {"Codegen", []string{"compiler"}}, {"Odd", []string{"zzz"}}, {"Bare", nil}} {
		r, err := s.New(task.NewOptions{Title: c.title, Goal: "g", Acceptance: []string{"a"}, Labels: c.tags})
		if err != nil {
			t.Fatal(err)
		}
		ids[c.title] = r.Task.ID
	}
	m := newModel(s, Options{User: "tester"})
	m.width, m.height = 100, 30
	m.resize()
	m.setView("list")
	m.move(0, true)
	return m, ids
}

func visibleTitles(m model) []string {
	var out []string
	for _, t := range m.visible {
		out = append(out, t.Title)
	}
	return out
}

// txjkrm check 4: : tag compiler keeps the tasks tagged compiler and the
// chip reads tag:compiler; an unknown tag is refused.
func TestTagFilter(t *testing.T) {
	m, _ := tagBoard(t)
	m = commandLine(m, "tag compiler")
	if got := strings.Join(visibleTitles(m), ","); got != "Both,Codegen" && got != "Codegen,Both" {
		t.Fatalf("tag compiler: %q\n%s", got, screen(m))
	}
	if !strings.Contains(m.chipText(), "tag:compiler") {
		t.Fatalf("chip %q", m.chipText())
	}
	if !strings.Contains(screen(m), "tag:compiler") && !strings.Contains(screen(m), "tag compiler") {
		t.Fatalf("the screen does not name the tag filter:\n%s", screen(m))
	}
	m = press(m, "c")
	if len(m.visible) != 5 {
		t.Fatalf("c did not clear: %v", visibleTitles(m))
	}
	m = commandLine(m, "tag nosuch")
	if !strings.Contains(m.message, "unknown tag: nosuch") || len(m.visible) != 5 {
		t.Fatalf("unknown tag: %q %v", m.message, visibleTitles(m))
	}
	// The palette offers tag values, not label.
	if vs := m.values("tag"); strings.Join(vs, ",") != "compiler,parser,zzz" {
		t.Fatalf("tag values %v", vs)
	}
}

// txjkrm check 4: g cycles flat → group → epic → tag; by tag each task
// sits under one heading, its first tag in hug.toml's order (an unlisted
// tag leads its own heading), and a task with no tag under "no tag".
func TestGroupingByTag(t *testing.T) {
	m, _ := tagBoard(t)
	m = press(m, "g", "g", "g")
	if m.grouping != "tag" {
		t.Fatalf("g g g: grouping %q", m.grouping)
	}
	s := screen(m)
	if h := strings.Split(s, "\n")[0]; !strings.Contains(h, "grouped by tag") {
		t.Fatalf("header %q", h)
	}
	for _, want := range []string{"▾ parser  [2]", "▾ compiler  [1]", "▾ zzz  [1]", "▾ no tag  [1]"} {
		if !strings.Contains(s, want) {
			t.Fatalf("by tag lacks %q:\n%s", want, s)
		}
	}
	// Both sits under parser (listed first), not compiler.
	byTitle := map[string]string{}
	for _, tk := range m.visible {
		byTitle[tk.Title] = m.groupValue(tk)
	}
	for title, want := range map[string]string{"Lex": "parser", "Both": "parser", "Codegen": "compiler", "Odd": "zzz", "Bare": ""} {
		if byTitle[title] != want {
			t.Fatalf("%s grouped under %q, want %q", title, byTitle[title], want)
		}
	}
	if m = press(m, "g"); m.grouping != "" {
		t.Fatalf("g after tag is not flat: %q", m.grouping)
	}
	m = commandLine(m, "grouping tag")
	if m.grouping != "tag" {
		t.Fatalf(": grouping tag: %q", m.grouping)
	}
	m = commandLine(m, "grouping sideways")
	if !strings.Contains(m.message, ": grouping tag") {
		t.Fatalf("the fix does not name tag: %q", m.message)
	}
}
