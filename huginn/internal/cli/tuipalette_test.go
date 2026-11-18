package cli

import (
	"bytes"
	"context"
	"io"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// 05r3pg check 5: a keyless action in hug tui's : palette runs the hug task
// command itself, through Execute (the code MCP task_update runs): a note
// typed in the palette is in the file and committed on the task branch,
// and the TUI quits clean.
func TestTUIPaletteNoteCommits(t *testing.T) {
	t.Parallel()
	a := app(t)
	execute(t, a, "init")
	id := execute(t, a, "new", "Alpha", "--goal", "G", "--purpose", "P", "--principle", "Q", "--accept", "C")["task"].(Reply)["id"].(string)
	in, w := io.Pipe()
	done := make(chan error, 1)
	go func() {
		c := &App{Cwd: a.Cwd, In: in, Out: &bytes.Buffer{}, Err: &bytes.Buffer{}, Env: a.Env}
		_, e := c.Execute(context.Background(), []string{"tui"})
		done <- e
	}()
	time.Sleep(300 * time.Millisecond) // the TUI is up
	// List view, Alpha selected; the palette's note acts on it; Enter
	// asks "Run hug task note ...", Enter again runs it.
	for _, k := range []string{"vl", ":", "note written in the palette", "\r", "\r", "q"} {
		if _, e := w.Write([]byte(k)); e != nil {
			t.Fatalf("key %q: %v", k, e)
		}
		time.Sleep(150 * time.Millisecond)
	}
	select {
	case e := <-done:
		if e != nil {
			t.Fatalf("tui: %v", e)
		}
	case <-time.After(60 * time.Second):
		t.Fatal("the TUI did not quit")
	}
	w.Close()
	shown := execute(t, a, "show", id)["task"].(Reply)
	if !strings.Contains(shown["notes"].(string), "written in the palette") {
		t.Fatalf("the palette's note is not on %s: notes %q", id, shown["notes"])
	}
	board := filepath.Join(a.Cwd, TaskDir)
	if log := git(t, board, "log", "-1", "-p", "--format=%s"); !strings.Contains(log, "written in the palette") {
		t.Fatalf("the note is not in the last commit on the task branch:\n%s", log)
	}
	if st := git(t, board, "status", "--porcelain"); st != "" {
		t.Fatalf("the task checkout is not clean after the palette's note:\n%s", st)
	}
}
