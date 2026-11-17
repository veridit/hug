package tui

import (
	"context"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/veridit/hug/huginn/internal/task"
)

// np2kk2 check 1: the watch setup lists each folder at most once. On Seid
// (271 top-level folders, 1,132 in all) task.IsTaskFolder's two Globs per
// folder read the parent again for every child: 2.0 s of the first open.
func TestTaskDirsReadsEachFolderOnce(t *testing.T) {
	root := t.TempDir()
	put := func(rel string) {
		p := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	ids := []string{"aaaaaa", "bbbbbb", "cccccc", "dddddd", "eeeeee", "ffffff"}
	for i, id := range ids {
		group := []string{"todo", "doing", "done/2026"}[i%3]
		put(group + "/0" + id[:1] + "." + id + ".a-task.md")
		put(group + "/" + id + "/log.txt")         // the task's files folder
		put(group + "/" + id + "/bench/README.md") // a folder in it
	}
	put("todo/gggggg.lone-task.md") // id.slug.md names a files folder too
	put("todo/gggggg/notes.txt")
	put("todo/hhhhhh/stray.txt") // a folder named like an id, no task: a group
	put(".hidden/x.md")

	var mu sync.Mutex
	reads := map[string]int{}
	old := readDir
	readDir = func(dir string) ([]os.DirEntry, error) {
		mu.Lock()
		rel, _ := filepath.Rel(root, dir)
		reads[filepath.ToSlash(rel)]++
		mu.Unlock()
		return os.ReadDir(dir)
	}
	defer func() { readDir = old }()
	got := taskDirs(root)

	var rels []string
	for _, d := range got {
		rel, _ := filepath.Rel(root, d)
		rels = append(rels, filepath.ToSlash(rel))
	}
	sort.Strings(rels)
	want := []string{".", "doing", "done", "done/2026", "todo", "todo/hhhhhh"}
	if strings.Join(rels, " ") != strings.Join(want, " ") {
		t.Fatalf("taskDirs: %v, want %v", rels, want)
	}
	total := 0
	for dir, n := range reads {
		total += n
		if n > 1 {
			t.Errorf("%s listed %d times, want once", dir, n)
		}
	}
	// Only the folders watched are listed: never a files folder, never a
	// hidden one.
	if total != len(want) {
		t.Errorf("%d folder reads %v, want %d (one per watched folder)", total, reads, len(want))
	}
}

// np2kk2 check 2: the watch comes up after the first frame, so a change
// made between the start's refresh and the watch setup has no event. The
// TUI reads the board once more when the watch is up: the change shows.
func TestChangeBeforeWatchIsUpIsSeen(t *testing.T) {
	m := fixture(t)
	start := newModel(m.store, Options{User: "tester", Initial: &Initial{Tasks: m.all}, Refresh: func(context.Context) error { return nil }})
	// The start's refresh lands first.
	next, _ := start.Update(loadedMsg{loadSnapshot(start.store, start.loads, start.closes, start.opts.Lint), start.gen})
	start = next.(model)
	// Then, before the watch is up, a task arrives on disk.
	old := watchStarter
	watchStarter = func(root string) *watch {
		if _, err := m.store.New(task.NewOptions{Title: "Late task", Goal: "G", Acceptance: []string{"c"}}); err != nil {
			t.Error(err)
		}
		return startWatch(root)
	}
	defer func() { watchStarter = old }()
	up := start.startWatching()()
	next, cmd := start.Update(up)
	start = next.(model)
	defer start.release()
	msgs := make(chan tea.Msg, 1)
	go func() { msgs <- runBatch(cmd) }()
	select {
	case msg := <-msgs:
		next, _ = start.Update(msg)
		start = next.(model)
	case <-time.After(5 * time.Second):
		t.Fatal("no reload after the watch came up")
	}
	found := false
	for _, tk := range start.all {
		found = found || tk.Title == "Late task"
	}
	if !found {
		t.Fatalf("the change made before the watch was up is not shown: %d tasks", len(start.all))
	}
}
