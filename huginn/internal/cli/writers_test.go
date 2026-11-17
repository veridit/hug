package cli

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

// 95ahtk check 1: Seid (2026-11-17, item 3b) saw two agents writing at once
// race on git's index and on temp files. Eight writers on one board, twenty
// rounds each of note, check and set, all at once: every write succeeds and
// every write is its own commit on the board branch.
func TestEightWritersMixedCommandsAllLand(t *testing.T) {
	t.Parallel()
	const writers, rounds = 8, 20
	a := app(t)
	execute(t, a, "init")
	ids := make([]string, writers)
	for i := range ids {
		args := []string{"new", "Writer " + strconv.Itoa(i), "--goal", "G", "--purpose", "P", "--principle", "Q"}
		for n := 0; n < rounds; n++ {
			args = append(args, "--accept", fmt.Sprintf("item %d", n+1))
		}
		ids[i] = execute(t, a, args...)["task"].(Reply)["id"].(string)
	}
	board := a.Cwd + "/" + TaskDir
	before, _ := strconv.Atoi(git(t, board, "rev-list", "--count", "HEAD"))
	var wg sync.WaitGroup
	errs := make(chan error, writers*rounds)
	start := make(chan struct{})
	for i := 0; i < writers; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start
			for r := 0; r < rounds; r++ {
				var args []string
				switch r % 3 {
				case 0:
					args = []string{"note", ids[i], fmt.Sprintf("w%d r%d", i, r)}
				case 1:
					args = []string{"check", ids[i], strconv.Itoa(r + 1)}
				default:
					args = []string{"set", ids[i], "--priority", fmt.Sprintf("p%d-%d", i, r)}
				}
				c := &App{Cwd: a.Cwd, In: strings.NewReader(""), Out: &bytes.Buffer{}, Err: &bytes.Buffer{}, Env: a.Env}
				if _, e := c.Execute(context.Background(), args); e != nil {
					errs <- fmt.Errorf("%v: %w", args, e)
				}
			}
		}(i)
	}
	close(start)
	wg.Wait()
	close(errs)
	for e := range errs {
		t.Errorf("parallel write: %v", e)
	}
	if out := git(t, board, "status", "--porcelain"); out != "" {
		t.Fatalf("board left uncommitted: %s", out)
	}
	after, _ := strconv.Atoi(git(t, board, "rev-list", "--count", "HEAD"))
	if got := after - before; got != writers*rounds {
		t.Errorf("%d commits for %d writes", got, writers*rounds)
	}
	// Per writer: notes at rounds 0, 3, ..., 18; checks at 1, 4, ..., 19
	// (7 items); sets at 2, 5, ..., 17, the last one winning.
	for i, id := range ids {
		f := taskFile(t, a, id)
		for r := 0; r < rounds; r += 3 {
			if !strings.Contains(f, fmt.Sprintf("w%d r%d\n", i, r)) && !strings.HasSuffix(f, fmt.Sprintf("w%d r%d", i, r)) {
				t.Errorf("%s: note w%d r%d missing", id, i, r)
			}
		}
		if n := strings.Count(f, "[x]"); n != 7 {
			t.Errorf("%s: %d of 7 items checked", id, n)
		}
		if !strings.Contains(f, fmt.Sprintf("p%d-17", i)) {
			t.Errorf("%s: priority is not p%d-17", id, i)
		}
	}
}

// 95ahtk check 2: readers never wait for a writer. With the board turn held
// (a writer mid-command), ls and show still answer at once.
func TestReadersDoNotWaitOnHeldTurn(t *testing.T) {
	t.Parallel()
	a := app(t)
	execute(t, a, "init")
	id := execute(t, a, "new", "Read me", "--goal", "G", "--purpose", "P", "--principle", "Q", "--accept", "C")["task"].(Reply)["id"].(string)
	s, e := a.openBoard(context.Background(), false)
	if e != nil {
		t.Fatal(e)
	}
	release := s.turn()
	s.c.Close()
	defer release()
	for _, args := range [][]string{{"ls"}, {"show", id}, {"next"}, {"search", "Read"}} {
		done := make(chan error, 1)
		go func(args []string) {
			c := &App{Cwd: a.Cwd, In: strings.NewReader(""), Out: &bytes.Buffer{}, Err: &bytes.Buffer{}, Env: a.Env}
			_, e := c.Execute(context.Background(), args)
			done <- e
		}(args)
		select {
		case e := <-done:
			if e != nil {
				t.Errorf("%v: %v", args, e)
			}
		case <-time.After(20 * time.Second):
			t.Fatalf("%v waited on the held writer turn", args)
		}
	}
}

// 95ahtk, the race Seid hit (2026-11-17 16:49:23: an agent's hug task
// attach failed on index.lock in the same second as a "hug task tui"
// commit): the TUI ticks and unticks a check while an agent adds notes
// with hug task note, at the same moment, on one board. Every TUI write is
// its own commit, every note lands, and nothing fails.
func TestTUIWriteAndCLINoteAtOnce(t *testing.T) {
	t.Parallel()
	const toggles, notes = 30, 30
	a := app(t)
	execute(t, a, "init")
	id := execute(t, a, "new", "Shared", "--goal", "G", "--purpose", "P", "--principle", "Q", "--accept", "C")["task"].(Reply)["id"].(string)
	board := a.Cwd + "/" + TaskDir
	in, keys := io.Pipe()
	tuiDone := make(chan error, 1)
	go func() {
		c := &App{Cwd: a.Cwd, In: in, Out: &bytes.Buffer{}, Err: &bytes.Buffer{}, Env: a.Env}
		_, e := c.Execute(context.Background(), []string{"tui"})
		tuiDone <- e
	}()
	press := func(k string) {
		if _, e := keys.Write([]byte(k)); e != nil {
			t.Fatalf("key %q: %v", k, e)
		}
	}
	// The TUI is up when its first key is read; open the task.
	time.Sleep(300 * time.Millisecond)
	press("\r")
	time.Sleep(200 * time.Millisecond)
	var wg sync.WaitGroup
	errs := make(chan error, notes)
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < notes; i++ {
			c := &App{Cwd: a.Cwd, In: strings.NewReader(""), Out: &bytes.Buffer{}, Err: &bytes.Buffer{}, Env: a.Env}
			if _, e := c.Execute(context.Background(), []string{"note", id, fmt.Sprintf("agent note %d", i)}); e != nil {
				errs <- e
			}
		}
	}()
	for i := 0; i < toggles; i++ {
		press(" ") // asks: toggle acceptance?
		press("\r")
		time.Sleep(15 * time.Millisecond)
	}
	wg.Wait()
	close(errs)
	time.Sleep(200 * time.Millisecond)
	press("q")
	select {
	case e := <-tuiDone:
		if e != nil {
			t.Fatalf("tui: %v", e)
		}
	case <-time.After(60 * time.Second):
		t.Fatal("the TUI did not quit")
	}
	keys.Close()
	for e := range errs {
		t.Errorf("hug task note beside the TUI: %v", e)
	}
	subjects := git(t, board, "log", "--format=%s")
	if n := strings.Count(subjects, "hug task note "+id); n != notes {
		t.Errorf("%d of %d note commits", n, notes)
	}
	tuiCommits := strings.Count(subjects, "hug task check "+id) + strings.Count(subjects, "hug task uncheck "+id)
	if tuiCommits != toggles {
		t.Errorf("%d of %d TUI writes have their own commit", tuiCommits, toggles)
	}
	if out := git(t, board, "status", "--porcelain"); out != "" {
		t.Errorf("board left uncommitted: %s", out)
	}
	f := taskFile(t, a, id)
	for i := 0; i < notes; i++ {
		if !strings.Contains(f, fmt.Sprintf("agent note %d\n", i)) && !strings.HasSuffix(strings.TrimSpace(f), fmt.Sprintf("agent note %d", i)) {
			t.Errorf("note %d missing", i)
		}
	}
}
