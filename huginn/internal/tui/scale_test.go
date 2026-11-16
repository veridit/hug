package tui

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/veridit/hug/huginn/internal/cache"
	"github.com/veridit/hug/huginn/internal/task"
	"github.com/veridit/hug/huginn/internal/tui/testfixture"
)

// 44w1ev check 4: with Options.Lint (findings from the task cache) a reload
// parses no task file, and a task's file is read when its detail opens. What
// the reader sees is the same as when every file is parsed: overview, list,
// board, every task's detail with its sections and files, and the LINT pane.
func TestCacheLintShowsTheSameBoard(t *testing.T) {
	root := t.TempDir()
	if err := testfixture.Seed(root); err != nil {
		t.Fatal(err)
	}
	c, err := cache.Open(root, filepath.Join(t.TempDir(), "cache.db"), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	cached := &task.Store{Root: root}
	cached.Refresh = func() ([]task.Task, error) { return c.List(context.Background()) }
	plain := newModel(&task.Store{Root: root}, Options{User: "tester", NoWatch: true, NoReload: true})
	fast := newModel(cached, Options{User: "tester", NoWatch: true, NoReload: true, Lint: func() ([]task.Finding, error) { return c.LintListed(), nil }})
	// The first screen may read the selected task (its preview), no more.
	if len(fast.docs) > 1 {
		t.Fatalf("a reload with Options.Lint parsed %d task files", len(fast.docs))
	}
	if len(plain.findings) == 0 || len(plain.findings) != len(fast.findings) {
		t.Fatalf("findings: parsed %d, cache %d", len(plain.findings), len(fast.findings))
	}
	for _, m := range []*model{&plain, &fast} {
		m.width, m.height = 140, 40
		m.resize()
	}
	for _, view := range []string{"overview", "list", "board"} {
		plain.setView(view)
		fast.setView(view)
		if a, b := ansi.Strip(plain.View()), ansi.Strip(fast.View()); a != b {
			t.Fatalf("%s differs\nparsed:\n%s\ncache:\n%s", view, a, b)
		}
	}
	if len(plain.all) != len(fast.all) {
		t.Fatalf("tasks: %d vs %d", len(plain.all), len(fast.all))
	}
	for i, tk := range plain.all {
		if a, b := ansi.Strip(plain.detailText(tk)), ansi.Strip(fast.detailText(fast.all[i])); a != b {
			t.Fatalf("detail of %s differs\nparsed:\n%s\ncache:\n%s", tk.ID, a, b)
		}
	}
	if len(fast.docs) != len(fast.all) {
		t.Fatalf("opening every detail parsed %d of %d files", len(fast.docs), len(fast.all))
	}
	// Task folders come from one listing per group: each task's files
	// equal a walk of its own folder, and the seeded board has some.
	withFiles := 0
	for _, tk := range fast.all {
		want, got := taskFiles(root, tk), fast.filesOf(tk)
		if fmt.Sprint(want) != fmt.Sprint(got) {
			t.Fatalf("files of %s: %q, its folder has %q", tk.ID, got, want)
		}
		if len(got) > 0 {
			withFiles++
		}
	}
	if withFiles == 0 {
		t.Fatal("the seeded board has no task with files: the check above proves nothing")
	}
}
