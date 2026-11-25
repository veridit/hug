package store

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/veridit/hug/internal/timing"
)

// 2026-11-16 (me51r2): waiters behind a long holder got SQLITE_BUSY once
// busy_timeout's 30 s ran out for the queue as a whole. Through the write
// turn, a queue of processes whose combined time is several times a deadline
// all succeed, one at a time: the clock only runs per holder.
func TestWriteTurnQueueOutlastsPerHolderDeadline(t *testing.T) {
	if p := os.Getenv("TURN_CHILD_DB"); p != "" {
		db, err := Open(p)
		if err != nil {
			os.Stdout.WriteString("open: " + err.Error())
			os.Exit(1)
		}
		defer db.Close()
		TurnStuckAfter, TurnPoll = time.Second, 20*time.Millisecond
		_, release, err := db.WriteTurn(context.Background())
		if err != nil {
			os.Stdout.WriteString("turn: " + err.Error())
			os.Exit(1)
		}
		// Prove exclusivity: nobody else is inside while we are.
		marker := p + ".inside"
		if _, err := os.Stat(marker); err == nil {
			os.Stdout.WriteString("two holders at once")
			os.Exit(1)
		}
		os.WriteFile(marker, nil, 0o600)
		time.Sleep(400 * time.Millisecond)
		if _, err := db.Exec(`INSERT INTO meta(key,value) VALUES(?, 'x')`, os.Getenv("TURN_CHILD_ID")); err != nil {
			os.Stdout.WriteString("write: " + err.Error())
			os.Exit(1)
		}
		os.Remove(marker)
		release()
		os.Exit(0)
	}
	path := filepath.Join(t.TempDir(), "m.db")
	db, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	db.Close()
	// 8 x 0.4 s = 3.2 s of queue, against a 1 s per-holder deadline.
	const children = 8
	cmds := make([]*exec.Cmd, children)
	outs := make([]*strings.Builder, children)
	for i := range cmds {
		cmds[i] = exec.Command(os.Args[0], "-test.run=^TestWriteTurnQueueOutlastsPerHolderDeadline$")
		cmds[i].Env = append(os.Environ(), "TURN_CHILD_DB="+path, "TURN_CHILD_ID=c"+strconv.Itoa(i))
		outs[i] = &strings.Builder{}
		cmds[i].Stdout, cmds[i].Stderr = outs[i], outs[i]
		if err := cmds[i].Start(); err != nil {
			t.Fatal(err)
		}
	}
	for i, c := range cmds {
		if err := c.Wait(); err != nil {
			t.Fatalf("child %d: %v\n%s", i, err, outs[i])
		}
	}
	db, err = Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	var n int
	db.QueryRow(`SELECT count(*) FROM meta WHERE value='x'`).Scan(&n)
	if n != children {
		t.Fatalf("writes %d, want %d", n, children)
	}
}

// A holder that keeps the turn far past any normal write: the waiter gives
// up, names the holder and gives a fix line, and does not leave the lock taken.
func TestWriteTurnStuckHolderIsNamed(t *testing.T) {
	path := filepath.Join(t.TempDir(), "m.db")
	holder, _ := Open(path)
	defer holder.Close()
	waiter, _ := Open(path)
	defer waiter.Close()
	oldStuck, oldPoll := TurnStuckAfter, TurnPoll
	TurnStuckAfter, TurnPoll = 300*time.Millisecond, 20*time.Millisecond
	defer func() { TurnStuckAfter, TurnPoll = oldStuck, oldPoll }()
	var noticed atomic.Value
	var notices atomic.Int32
	oldNotice, oldAfter, oldEvery := TurnNotice, TurnNoticeAfter, TurnNoticeEvery
	TurnNotice = func(h string, waited time.Duration) { noticed.Store(h); notices.Add(1) }
	TurnNoticeAfter, TurnNoticeEvery = 50*time.Millisecond, 100*time.Millisecond
	defer func() { TurnNotice, TurnNoticeAfter, TurnNoticeEvery = oldNotice, oldAfter, oldEvery }()

	_, release, err := holder.WriteTurn(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	_, _, err = waiter.WriteTurn(context.Background())
	var stuck *TurnStuckError
	if !errors.As(err, &stuck) {
		t.Fatalf("want TurnStuckError, got %v", err)
	}
	pid := strconv.Itoa(os.Getpid())
	if msg := err.Error(); !strings.Contains(msg, "pid "+pid) || !strings.Contains(msg, "\nfix: ") {
		t.Fatalf("error must name the holder and end with a fix line:\n%s", msg)
	}
	if h, _ := noticed.Load().(string); !strings.Contains(h, "pid "+pid) {
		t.Fatalf("waiting notice must name the holder, got %q", h)
	}
	// 300 ms of waiting, notices from 50 ms every 100 ms: the waiter keeps
	// saying it is waiting, not once and then silence.
	if n := notices.Load(); n < 2 {
		t.Fatalf("waiting notice repeated %d times, want at least 2", n)
	}
	release()
	// The abandoned wait must not keep the lock. Its goroutine wakes on the
	// release, takes the lock and drops it again, so the turn can be held
	// for a moment (Windows CI, run 43921466087): free within 2 s, not at once.
	var again func()
	var ok bool
	for deadline := time.Now().Add(2 * time.Second); ; time.Sleep(10 * time.Millisecond) {
		_, again, ok, err = waiter.TryWriteTurn(context.Background())
		if err != nil || ok || time.Now().After(deadline) {
			break
		}
	}
	if err != nil || !ok {
		t.Fatalf("turn not free after the holder released: ok=%v err=%v", ok, err)
	}
	again()
	if h, ok := TurnHolder(path); ok {
		t.Fatalf("holder file not cleared: %+v", h)
	}
}

// The turn is reentrant through its context, so nested write flows (forget
// calls Reindex) do not wait on themselves; a hook's TryWriteTurn never waits.
func TestWriteTurnNestsAndTryDoesNotWait(t *testing.T) {
	timing.Budget(t)
	path := filepath.Join(t.TempDir(), "m.db")
	db, _ := Open(path)
	defer db.Close()
	other, _ := Open(path)
	defer other.Close()
	ctx, release, err := db.WriteTurn(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() {
		_, inner, err := db.WriteTurn(ctx)
		if err == nil {
			inner()
		}
		done <- err
	}()
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("nested WriteTurn waited on its own turn")
	}
	if !db.HasTurn(context.WithoutCancel(ctx)) {
		t.Fatal("WithoutCancel must keep the turn")
	}
	t0 := time.Now()
	if _, _, ok, err := other.TryWriteTurn(context.Background()); ok || err != nil || time.Since(t0) > time.Second {
		t.Fatalf("TryWriteTurn while held: ok=%v err=%v after %s", ok, err, time.Since(t0))
	}
	release()
}

func TestExplainBusyNamesHolder(t *testing.T) {
	path := filepath.Join(t.TempDir(), "m.db")
	db, _ := Open(path)
	defer db.Close()
	busy := errors.New("database is locked (5) (SQLITE_BUSY)")
	if err := ExplainBusy(path, busy); !strings.Contains(err.Error(), "\nfix: ") {
		t.Fatalf("no fix line: %v", err)
	}
	_, release, _ := db.WriteTurn(context.Background())
	defer release()
	err := ExplainBusy(path, busy)
	if !strings.Contains(err.Error(), "held by: pid "+strconv.Itoa(os.Getpid())) || !strings.Contains(err.Error(), "\nfix: wait for pid") {
		t.Fatalf("holder not named: %v", err)
	}
	if other := errors.New("other"); ExplainBusy(path, other) != other {
		t.Fatal("non-busy errors pass through")
	}
}
