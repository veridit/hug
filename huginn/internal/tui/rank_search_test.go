package tui

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/veridit/hug/huginn/internal/cache"
	"github.com/veridit/hug/huginn/internal/task"
)

// 49n9kw check 6: with Options.Search (the task cache's index) / search
// ranks by relevance, as hug task search: the task with the words in its
// title comes before one that has them in its goal, though it is ranked
// below it on the board; a word only in the notes still finds its task.
func TestSearchRanksByIndex(t *testing.T) {
	s := &task.Store{Root: t.TempDir()}
	for _, o := range []task.NewOptions{
		{Title: "Logging", Goal: "the worker exits 137 when memory runs out", Acceptance: []string{"C"}},
		{Title: "Memory exits 137 in the worker", Goal: "Stop it", Acceptance: []string{"C"}},
		{Title: "Notes only", Goal: "G", Acceptance: []string{"C"}},
	} {
		if _, err := s.New(o); err != nil {
			t.Fatal(err)
		}
	}
	ts, _ := s.Tasks()
	if _, err := s.Note(ts[2].ID, "mentions a quokka"); err != nil {
		t.Fatal(err)
	}
	c, err := cache.Open(s.Root, filepath.Join(t.TempDir(), "cache.db"), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	s.Refresh = func() ([]task.Task, error) { return c.List(context.Background()) }
	search := func(q string) ([]string, error) { return c.Rank(context.Background(), q) }
	m := newModel(s, Options{User: "tester", NoWatch: true, NoReload: true, Search: search})
	m.setView("list")
	m = press(m, "/", "memory 137")
	if len(m.visible) != 2 || m.visible[0].Title != "Memory exits 137 in the worker" || m.visible[1].Title != "Logging" {
		t.Fatalf("ranked: %v", titles(m.visible))
	}
	m = press(newModel(s, Options{User: "tester", NoWatch: true, NoReload: true, Search: search}), "/", "quokka")
	if len(m.visible) != 1 || m.visible[0].Title != "Notes only" {
		t.Fatalf("notes word: %v", titles(m.visible))
	}
	// Without the index: substring filter in board order, as before.
	m = press(newModel(s, Options{User: "tester", NoWatch: true, NoReload: true}), "/", "137")
	if len(m.visible) != 1 || m.visible[0].Title != "Memory exits 137 in the worker" {
		t.Fatalf("substring: %v", titles(m.visible))
	}
}

func titles(ts []task.Task) []string {
	out := make([]string, len(ts))
	for i, t := range ts {
		out[i] = t.Title
	}
	return out
}
