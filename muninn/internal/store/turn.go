package store

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/veridit/hug/muninn/internal/flock"
)

// The write turn: one hug process at a time runs a write flow (an index pass,
// a forget, a remember) against this database. Waiting for SQLite's lock
// alone failed under load (2026-11-16, seid: 233 of 239 forgets): its busy
// handler polls, so a late arrival can jump the queue, and busy_timeout is
// one budget for the whole queue, not per holder: 20 forgets of ~1.5 s each,
// or one cold index pass of 32 s, used it up and the waiters got
// "database is locked (5) (SQLITE_BUSY)". The turn is a lock file waited on
// in the kernel, with no deadline while the queue moves: a waiter gives up
// only when one holder has held it for TurnStuckAfter, and then names it.

// TurnStuckAfter is how long one holder may hold the turn before a waiter
// gives up. Far beyond any normal write: a cold index of 12,700 memories took
// 33 s (2026-11-16); a full re-embed can take minutes.
var TurnStuckAfter = 20 * time.Minute

// TurnNoticeAfter, TurnNoticeEvery and TurnNotice tell a waiter why nothing
// happens: first after TurnNoticeAfter, then every TurnNoticeEvery or when the
// holder changes, so an agent never watches minutes of silence.
var (
	TurnNoticeAfter = 2 * time.Second
	TurnNoticeEvery = 15 * time.Second
	TurnNotice      func(holder string, waited time.Duration)
)

// TurnPoll is how often a waiter looks at who holds the turn.
var TurnPoll = 250 * time.Millisecond

// BusyTimeout is the SQLite busy_timeout in milliseconds for every connection.
// Inside the turn a writer only waits for short statements outside it.
const BusyTimeout = 30000

// Holder is who holds the write turn, as it wrote itself down.
type Holder struct {
	PID   int       `json:"pid"`
	Cmd   string    `json:"cmd"`
	Since time.Time `json:"since"`
}

func (h Holder) String() string {
	if h.PID == 0 {
		return "another hug process"
	}
	return fmt.Sprintf("pid %d (%s) since %s", h.PID, h.Cmd, h.Since.Local().Format("15:04:05"))
}

// TurnStuckError is returned when one holder kept the turn past TurnStuckAfter.
type TurnStuckError struct {
	Holder Holder
	Held   time.Duration
}

func (e *TurnStuckError) Error() string {
	fix := "fix: wait for the other hug process to finish, then retry"
	if e.Holder.PID != 0 {
		fix = fmt.Sprintf("fix: wait for pid %d to finish, or stop it (kill %d), then retry", e.Holder.PID, e.Holder.PID)
	}
	return fmt.Sprintf("memory writes are held by %s for %s, far longer than any normal write\n%s", e.Holder, e.Held.Round(time.Second), fix)
}

func (d *DB) turnPath() string   { return d.Path + ".write.lock" }
func (d *DB) holderPath() string { return d.Path + ".writer" }

// TurnHolder reads who holds the write turn of the database at path, if anyone wrote it down.
func TurnHolder(dbPath string) (Holder, bool) {
	b, err := os.ReadFile(dbPath + ".writer")
	if err != nil || len(b) == 0 {
		return Holder{}, false
	}
	var h Holder
	if json.Unmarshal(b, &h) != nil {
		return Holder{}, false
	}
	return h, h.PID != 0
}

func (d *DB) took(release func()) func() {
	cmd := filepath.Base(os.Args[0])
	if len(os.Args) > 1 {
		cmd += " " + strings.Join(os.Args[1:], " ")
	}
	if len(cmd) > 120 {
		cmd = cmd[:117] + "..."
	}
	b, _ := json.Marshal(Holder{PID: os.Getpid(), Cmd: cmd, Since: time.Now().UTC()})
	os.WriteFile(d.holderPath(), b, 0o600)
	var once sync.Once
	return func() {
		once.Do(func() {
			os.WriteFile(d.holderPath(), nil, 0o600)
			release()
		})
	}
}

type turnKey struct{}

// HasTurn reports whether ctx carries this process's write turn on d. Pass
// that context (or context.WithoutCancel of it) to nested write flows: a
// second wait on the turn from inside it would block forever.
func (d *DB) HasTurn(ctx context.Context) bool { return ctx.Value(turnKey{}) == d.Path }

// TryWriteTurn takes the turn only if it is free (hooks never queue).
func (d *DB) TryWriteTurn(ctx context.Context) (context.Context, func(), bool, error) {
	if d.HasTurn(ctx) {
		return ctx, func() {}, true, nil
	}
	release, ok, err := flock.TryAcquire(d.turnPath())
	if err != nil || !ok {
		return ctx, func() {}, false, err
	}
	return context.WithValue(ctx, turnKey{}, d.Path), d.took(release), true, nil
}

// WriteTurn waits for the write turn and returns a context that carries it.
// The release func is safe to call more than once.
func (d *DB) WriteTurn(ctx context.Context) (context.Context, func(), error) {
	if d.HasTurn(ctx) {
		return ctx, func() {}, nil
	}
	type result struct {
		release func()
		err     error
	}
	got := make(chan result, 1)
	var mu sync.Mutex
	abandoned := false
	go func() {
		release, err := flock.Acquire(d.turnPath())
		mu.Lock()
		defer mu.Unlock()
		if abandoned {
			if err == nil {
				release()
			}
			return
		}
		got <- result{release, err}
	}()
	// A waiter that gives up must not leave the lock taken behind it.
	abandon := func() {
		mu.Lock()
		defer mu.Unlock()
		abandoned = true
		select {
		case r := <-got:
			if r.err == nil {
				r.release()
			}
		default:
		}
	}
	start := time.Now()
	var seen Holder
	seenAt := start
	var noticedAt time.Time
	var noticedHolder Holder
	tick := time.NewTicker(TurnPoll)
	defer tick.Stop()
	for {
		select {
		case r := <-got:
			if r.err != nil {
				return ctx, func() {}, r.err
			}
			return context.WithValue(ctx, turnKey{}, d.Path), d.took(r.release), nil
		case <-ctx.Done():
			abandon()
			return ctx, func() {}, ctx.Err()
		case <-tick.C:
			// The stuck clock is per holder and counts only what this waiter
			// watched: a queue that moves never times out, and a holder file
			// left by a killed process (the kernel already freed its lock)
			// cannot make a fresh waiter give up.
			h, _ := TurnHolder(d.Path)
			if h != seen {
				seen, seenAt = h, time.Now()
			}
			if TurnNotice != nil && time.Since(start) >= TurnNoticeAfter &&
				(noticedAt.IsZero() || time.Since(noticedAt) >= TurnNoticeEvery || (h != noticedHolder && time.Since(noticedAt) >= TurnNoticeAfter)) {
				noticedAt, noticedHolder = time.Now(), h
				TurnNotice(h.String(), time.Since(start))
			}
			if held := time.Since(seenAt); held > TurnStuckAfter {
				abandon()
				return ctx, func() {}, &TurnStuckError{Holder: h, Held: held}
			}
		}
	}
}

// ExplainBusy adds who holds the write turn and a fix line to a SQLITE_BUSY
// error; other errors pass through.
func ExplainBusy(dbPath string, err error) error {
	if err == nil || !strings.Contains(err.Error(), "SQLITE_BUSY") {
		return err
	}
	h, ok := TurnHolder(dbPath)
	if !ok {
		return fmt.Errorf("%w\nheld by: a short write outside the queue\nfix: retry the command", err)
	}
	return fmt.Errorf("%w\nheld by: %s\nfix: wait for pid %d to finish, then retry", err, h, h.PID)
}
