package app

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/veridit/hug/muninn/internal/harness"
)

// DeferredReader reads a deferred chunk's transcript again: for a jcode hook,
// the session file under home. It returns the session's working directory.
type DeferredReader func(harnessName, home, session string) (cwd string, messages []harness.CaptureMessage, err error)

// ReadDeferred is the hook's DeferredReader: jcode sessions only, the one
// harness hooks capture. Read-only, as in the hook itself.
func ReadDeferred(harnessName, home, session string) (string, []harness.CaptureMessage, error) {
	if harnessName != "jcode" {
		return "", nil, errors.New("deferred capture reads jcode sessions only")
	}
	t, err := harness.ReadCaptureSession(home, session)
	if err != nil {
		return "", nil, err
	}
	if t.Cwd == "" {
		return "", nil, errors.New("session cwd missing")
	}
	return t.Cwd, t.Messages, nil
}

// DrainDeferred sends one session's deferred chunks again (nmn6wf): hook
// chunks whose reply was still cut at max_tokens after the hook's retry at
// twice the cap. It runs after every hook's own capture, so the chunk of a
// session's last hook (session_end) is drained too, and at the configured
// cap (default 4096), not the hook's. One more call per chunk, once: a chunk
// still cut here leaves capture_deferred and stays in capture_backlog for
// `hug mem capture --backfill`. The oldest session goes first; one that
// fails moves to the back, so it cannot block the others.
func (a *App) DrainDeferred(ctx context.Context, read DeferredReader) (report CaptureReport, drained bool, err error) {
	var harnessName, home, session string
	err = a.DB.QueryRowContext(ctx, `SELECT harness, home, harness_session FROM capture_deferred ORDER BY at_ns LIMIT 1`).Scan(&harnessName, &home, &session)
	if errors.Is(err, sql.ErrNoRows) {
		return report, false, nil
	}
	if err != nil {
		return report, false, err
	}
	drop := func() {
		a.captureExec(context.Background(), `DELETE FROM capture_deferred WHERE harness=? AND home=? AND harness_session=?`, harnessName, home, session)
	}
	cwd, messages, err := read(harnessName, home, session)
	if err != nil {
		// The transcript is gone or unreadable: the chunk stays in the backlog.
		drop()
		return report, true, err
	}
	b := a.Bound(home, cwd)
	if b.Err != nil {
		drop()
		return report, true, b.Err
	}
	report, err = b.Capture(ctx, a, CaptureOptions{Harness: harnessName, Session: session, Home: home, Trigger: "deferred", Messages: messages, Deferred: true})
	if err != nil {
		a.captureExec(context.Background(), `UPDATE capture_deferred SET at_ns=? WHERE harness=? AND home=? AND harness_session=?`, time.Now().UnixNano(), harnessName, home, session)
		return report, true, err
	}
	// Rows whose ids are no longer in the transcript would wait forever.
	drop()
	return report, true, nil
}
