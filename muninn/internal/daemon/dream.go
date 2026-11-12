package daemon

import (
	"context"
	"strconv"
	"sync"
	"time"

	"github.com/veridit/hug/muninn/internal/app"
)

// QuietFor is how long no session may have had a user turn before the daily
// dream pass runs. Dreaming retires memories, so it waits for a lull.
var QuietFor = 30 * time.Minute

// DreamEvery is the minimum time between automatic dream passes.
var DreamEvery = 20 * time.Hour

// dreamWhenQuiet runs one dream pass when the last one is older than
// DreamEvery and no harness session has been active for QuietFor.
func dreamWhenQuiet(ctx context.Context, a *app.App, mu *sync.Mutex, logf func(string, ...any)) {
	now := time.Now()
	var lastTurn int64
	a.DB.QueryRow(`SELECT COALESCE(MAX(last_ns),0) FROM session_turns`).Scan(&lastTurn)
	if lastTurn > 0 && now.Sub(time.Unix(0, lastTurn)) < QuietFor {
		return
	}
	var last string
	a.DB.QueryRow(`SELECT value FROM meta WHERE key='dream_last_ns'`).Scan(&last)
	if ns, err := strconv.ParseInt(last, 10, 64); err == nil && now.Sub(time.Unix(0, ns)) < DreamEvery {
		return
	}
	mu.Lock()
	defer mu.Unlock()
	r, err := a.Dream(ctx, false)
	a.DB.Exec(`INSERT INTO meta(key,value) VALUES('dream_last_ns',?) ON CONFLICT(key) DO UPDATE SET value=excluded.value`, strconv.FormatInt(now.UnixNano(), 10))
	if err != nil {
		logf("dream: %v", err)
		return
	}
	logf("dream: retired %d, queued %d, open %d, undo %s", r.Retired, r.Queued, r.Open, r.Backup)
}
