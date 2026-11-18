package app

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/veridit/hug/internal/cond"
	"github.com/veridit/hug/internal/flock"
	"github.com/veridit/hug/muninn/internal/config"
	"github.com/veridit/hug/muninn/internal/store"
)

// Dream on its own (h00cs2). The daily pass ran only inside hug mem serve,
// and on Jørgen's machine serve never ran: dream_last_ns was unset on
// 2026-11-18. Now the capture hook starts a detached waiter when a pass is
// due; the waiter runs it once the person has been away QuietFor and the
// machine is not busy (load below DreamLoadBelow), and gives up after
// DreamWaitMax. serve's ticker takes the same gate and the same lock.

// QuietFor is how long no session may have had a user turn before the daily
// dream pass runs. Dreaming retires memories, so it waits for a lull.
var QuietFor = 30 * time.Minute

// DreamEvery is the minimum time between automatic dream passes.
var DreamEvery = 20 * time.Hour

// DreamLoadBelow is the 5-minute load average a pass waits for, the same
// predicate as a task's "when load below 120" (crab 2026-11-18: the load was
// ~700 that night).
var DreamLoadBelow = 120.0

// DreamWaitMax is how long a waiter waits for quiet before it gives up; the
// next turn_end starts another.
var DreamWaitMax = 12 * time.Hour

// DreamClock and DreamLoad are the waiter's clock and load; tests replace
// them to count instead of waiting.
var (
	DreamClock = time.Now
	DreamLoad  = func() (float64, bool) { return cond.LoadAverage() }
)

// DreamLast is the last automatic or manual dream pass, as recorded.
type DreamLast struct {
	At      time.Time `json:"at"`
	How     string    `json:"how,omitempty"` // "while quiet", "by hand", "by hug mem serve"
	Retired int       `json:"retired"`
	Queued  int       `json:"queued"`
	Open    int       `json:"open"`
	Receipt string    `json:"receipt,omitempty"` // undo receipt, "" when nothing was retired
}

// Line is the one line a person sees where they look (stats, the TUI).
func (l DreamLast) Line() string {
	if l.At.IsZero() {
		return "dream: never ran (runs by itself when you are away)"
	}
	s := fmt.Sprintf("dream %s %s: retired %d, queued %d, open %d", l.At.Local().Format("2006-01-02 15:04"), l.How, l.Retired, l.Queued, l.Open)
	if l.Receipt != "" {
		s += ", undo: hug mem tidy undo " + l.Receipt
	}
	return s
}

const dreamLastKey = "dream_last"

// ReadDreamLast reads the last pass from db; zero At when none ran.
func ReadDreamLast(db *store.DB) DreamLast {
	var l DreamLast
	if v := db.GetMeta(dreamLastKey); v != "" {
		if err := json.Unmarshal([]byte(v), &l); err == nil && !l.At.IsZero() {
			return l
		}
	}
	// Passes before h00cs2 recorded only the time.
	if ns, err := strconv.ParseInt(db.GetMeta("dream_last_ns"), 10, 64); err == nil && ns > 0 {
		l.At, l.How = time.Unix(0, ns), "by hug mem serve"
	}
	return l
}

func writeDreamLast(db *store.DB, l DreamLast) error {
	v, err := json.Marshal(l)
	if err != nil {
		return err
	}
	if err := db.SetMeta(dreamLastKey, string(v)); err != nil {
		return err
	}
	return db.SetMeta("dream_last_ns", strconv.FormatInt(l.At.UnixNano(), 10))
}

// DreamDue reports whether a pass is due: the last is older than DreamEvery.
func DreamDue(db *store.DB, now time.Time) bool {
	l := ReadDreamLast(db)
	return l.At.IsZero() || now.Sub(l.At) >= DreamEvery
}

// DreamGate says whether a due pass may run now, and why not.
func DreamGate(db *store.DB, now time.Time) (ok bool, why string) {
	var lastTurn int64
	db.QueryRow(`SELECT COALESCE(MAX(last_ns),0) FROM session_turns`).Scan(&lastTurn)
	if lastTurn > 0 {
		if away := now.Sub(time.Unix(0, lastTurn)); away < QuietFor {
			return false, "a user turn " + away.Round(time.Minute).String() + " ago"
		}
	}
	load, known := DreamLoad()
	if known && load >= DreamLoadBelow {
		return false, "load is " + strconv.FormatFloat(load, 'f', 1, 64)
	}
	return true, ""
}

// DreamWaitLock is held by the one waiter for a pass, for its whole wait;
// a hook that finds it held starts no second one.
func DreamWaitLock() string { return filepath.Join(config.DataDir(), "dream-wait.lock") }

// TryDreamWait takes DreamWaitLock without waiting; ok false when a waiter
// holds it.
func TryDreamWait() (release func(), ok bool, err error) {
	return flock.TryAcquire(DreamWaitLock())
}

// DreamRun waits for the run lock: one pass at a time, from a waiter, hug
// mem serve's ticker or by hand.
func DreamRun() (release func(), err error) {
	return flock.Acquire(filepath.Join(config.DataDir(), "dream.lock"))
}

// RecordDream runs nothing: it stores a finished pass's result as the last
// pass, so every surface can name it.
func RecordDream(db *store.DB, at time.Time, how string, r DreamResult) (DreamLast, error) {
	l := DreamLast{At: at, How: how, Retired: r.Retired, Queued: r.Queued, Open: r.Open}
	if r.Backup != "" {
		l.Receipt = r.Backup
	}
	return l, writeDreamLast(db, l)
}

// RecordDreamFailed records a failed pass, so the next is a day away
// instead of at the next tick or turn_end; the line names the error.
func RecordDreamFailed(db *store.DB, at time.Time, err error) {
	line, _, _ := strings.Cut(err.Error(), "\n")
	_ = writeDreamLast(db, DreamLast{At: at, How: "failed (" + line + ")"})
}

// DreamWaiterArgs is the hidden command a hook spawns.
var DreamWaiterArgs = []string{"mem", "dream", "--when-quiet"}

// HugExecutable is the hug installed now: the Homebrew link when there is
// one, else this executable. A waiter that slept for hours runs this, never
// the binary that spawned it (crab 2026-11-18, the mbdfzs rule).
var HugExecutable = func() string {
	for _, p := range []string{"/opt/homebrew/bin/hug", "/usr/local/bin/hug", "/home/linuxbrew/.linuxbrew/bin/hug"} {
		if _, e := os.Stat(p); e == nil {
			return p
		}
	}
	if p, e := os.Executable(); e == nil {
		return p
	}
	return "hug"
}

// SameFile reports whether two paths are the same file now (symlinks
// followed); false when either is missing.
func SameFile(a, b string) bool {
	x, e1 := os.Stat(a)
	y, e2 := os.Stat(b)
	return e1 == nil && e2 == nil && os.SameFile(x, y)
}
