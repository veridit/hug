package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// xfv9pw check 3: hug tui keeps the / and : histories per board in the
// repository's git dir, never on the task branch, and a second hug tui on
// the same board recalls what the first ran.
func TestTUIHistorySurvivesReopen(t *testing.T) {
	t.Parallel()
	a := app(t)
	execute(t, a, "init")
	execute(t, a, "new", "Alpha", "--goal", "G", "--purpose", "P", "--principle", "Q", "--accept", "C")
	session := func(keys ...string) {
		t.Helper()
		in, w := io.Pipe()
		done := make(chan error, 1)
		go func() {
			c := &App{Cwd: a.Cwd, In: in, Out: &bytes.Buffer{}, Err: &bytes.Buffer{}, Env: a.Env}
			_, e := c.Execute(context.Background(), []string{"tui"})
			done <- e
		}()
		time.Sleep(300 * time.Millisecond) // the TUI is up
		for _, k := range append(keys, "q") {
			if _, e := w.Write([]byte(k)); e != nil {
				t.Fatalf("key %q: %v", k, e)
			}
			time.Sleep(100 * time.Millisecond)
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
	}
	const up = "\x1b[A"
	session("/", "Alpha", "\r", ":", "board", "\r")
	file := filepath.Join(git(t, a.Cwd, "rev-parse", "--path-format=absolute", "--git-common-dir"), "huginn", "tui-history.json")
	read := func() (h struct{ Search, Command []string }) {
		t.Helper()
		b, e := os.ReadFile(file)
		if e != nil {
			t.Fatalf("history file: %v", e)
		}
		if e = json.Unmarshal(b, &h); e != nil {
			t.Fatalf("history file %s: %v", b, e)
		}
		return h
	}
	if h := read(); strings.Join(h.Search, ",") != "Alpha" || strings.Join(h.Command, ",") != "board" {
		t.Fatalf("after the first hug tui: search %q command %q", h.Search, h.Command)
	}
	// Reopened: Up recalls Alpha; edited, it is a new entry "Alph" (with
	// no recall it would be nothing). It still matches Alpha: Enter with no
	// match keeps the / prompt open (5cjy88), where : and q are typed.
	session("/", up, "\x7f", "\r", ":", up, "\r")
	if h := read(); strings.Join(h.Search, ",") != "Alpha,Alph" || strings.Join(h.Command, ",") != "board" {
		t.Fatalf("after the reopened hug tui: search %q command %q, want Alpha,Alph and board", h.Search, h.Command)
	}
	// Never on the task branch, never in its checkout.
	board := filepath.Join(a.Cwd, TaskDir)
	if files := git(t, board, "ls-files"); strings.Contains(files, "history") {
		t.Fatalf("history on the task branch:\n%s", files)
	}
	if st := git(t, board, "status", "--porcelain", "--ignored"); st != "" {
		t.Fatalf("the task checkout is not clean after hug tui:\n%s", st)
	}
}
