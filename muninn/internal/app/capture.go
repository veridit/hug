package app

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/veridit/hug/muninn/internal/extract"
	"github.com/veridit/hug/muninn/internal/harness"
	"github.com/veridit/hug/muninn/internal/project"
)

type CaptureOptions struct {
	Harness, Session, Home     string
	Trigger                    string
	Messages                   []harness.CaptureMessage
	DryRun, Periodic, Backfill bool
	// Deferred sends only this session's capture_deferred chunks, at the full
	// cap: hook replies still truncated after the retry (DrainDeferred).
	Deferred bool
}
type CaptureAction struct {
	extract.Candidate
	Scope     string  `json:"scope,omitempty"`
	PlaceHint string  `json:"place_hint,omitempty"`
	Action    string  `json:"action"`
	Reason    string  `json:"reason"`
	Score     float64 `json:"cosine,omitempty"`
	Target    string  `json:"target,omitempty"`
	Origin    string  `json:"origin"`
}
type CaptureReport struct {
	CaptureTimings
	Actions         []CaptureAction `json:"actions"`
	Usage           extract.Usage   `json:"usage"`
	LatencyMS       int64           `json:"latency_ms"`
	Processed       int             `json:"processed"`
	BacklogSkipped  int             `json:"backlog_skipped"`
	Rejected        int             `json:"rejected"`
	RejectedTrivial int             `json:"rejected_trivial"`
	Skipped         string          `json:"skipped,omitempty"`
	// Split counts messages longer than one call's limit that were cut into
	// parts (8jcj3k); each part is in the run's messages with original_id.
	Split int `json:"split,omitempty"`
	// Retried counts chunks whose reply hit max_tokens and were sent once
	// more at twice the cap (nmn6wf). Truncated counts those still cut after
	// that retry: none of their reply is applied, their message ids are in
	// Deferred and in capture_deferred, and a later hook drains them
	// (DrainDeferred) or `hug mem capture --backfill` does.
	Retried   int      `json:"retried,omitempty"`
	Truncated int      `json:"truncated,omitempty"`
	Deferred  []string `json:"deferred,omitempty"`
}

type CaptureTimings struct {
	ExtractMS int64 `json:"extract_ms"`
	PlanMS    int64 `json:"plan_ms"`
	ApplyMS   int64 `json:"apply_ms"`
	ReindexMS int64 `json:"reindex_ms"`
}

func captureHash(s string) string { h := sha256.Sum256([]byte(s)); return hex.EncodeToString(h[:]) }

// Capture reuses the surface Binding for all reads and writes. The caller must
// resolve a session's persisted cwd, not the daemon's working directory.
func (b Binding) Capture(ctx context.Context, a *App, o CaptureOptions) (report CaptureReport, runErr error) {
	audit := AuditRun{Time: time.Now().UTC(), Harness: o.Harness, Session: o.Session, Home: o.Home, Trigger: o.Trigger, DryRun: o.DryRun, Backfill: o.Backfill, Model: a.Cfg.Extract.Model}
	if audit.Trigger == "" {
		audit.Trigger = "manual"
	}
	defer func() {
		audit.Report = report
		audit.CaptureTimings = report.CaptureTimings
		if runErr != nil {
			audit.Error = auditSafe(runErr.Error())
			audit.ErrorClass = extract.FailureClass(runErr)
		}
		if err := a.SaveAudit(audit); err != nil && runErr == nil {
			runErr = fmt.Errorf("capture audit: %w", err)
		}
	}()
	if a.Cfg.Extract.Provider == "off" {
		report.Skipped = "extraction off"
		return report, nil
	}
	// Automatic capture never inherits an owner's temporarily widened read view.
	if c := b.context; c != nil {
		b = a.Bound(c.home, c.cwd)
	} else {
		b = b.Refresh(a)
	}
	// Keep the session/checkpoint binding stable across the placement rollout.
	// Each new action chooses its own scope below, not this legacy top-level key.
	scope, err := b.WriteScope("")
	audit.Scope = scope
	if err != nil {
		return report, err
	}
	if o.Session == "" || o.Harness == "" {
		return report, errors.New("capture requires a harness and session identity")
	}
	budget := a.Cfg.Extract.RunTimeout()
	ctx, cancel := context.WithTimeout(ctx, budget)
	defer cancel()
	if !o.DryRun {
		tokenBytes := make([]byte, 16)
		if _, err := rand.Read(tokenBytes); err != nil {
			return report, err
		}
		token := hex.EncodeToString(tokenBytes)
		for {
			now := time.Now()
			res, err := a.captureExec(ctx, `INSERT INTO capture_lock(id,token,expires_ns) VALUES(1,?,?) ON CONFLICT(id) DO UPDATE SET token=excluded.token,expires_ns=excluded.expires_ns WHERE capture_lock.expires_ns<?`, token, now.Add(budget+10*time.Second).UnixNano(), now.UnixNano())
			if err != nil {
				return report, err
			}
			n, _ := res.RowsAffected()
			if n > 0 {
				break
			}
			if o.Periodic {
				report.Skipped = "capture already running, retry on next hook"
				return report, nil
			}
			// Final observers wait within their detached deadline rather than dropping
			// the last delta just because a periodic observer is still extracting.
			select {
			case <-ctx.Done():
				return report, ctx.Err()
			case <-time.After(100 * time.Millisecond):
			}
		}
		defer a.captureExec(context.Background(), `DELETE FROM capture_lock WHERE id=1 AND token=?`, token)
	}
	identity := captureHash(o.Harness + "\x00" + o.Home + "\x00" + o.Session)
	if !o.DryRun {
		if _, err := a.captureExec(ctx, `INSERT OR IGNORE INTO capture_bindings(session,scope) VALUES(?,?)`, identity, scope); err != nil {
			return report, err
		}
	}
	var originalScope string
	err = a.DB.QueryRowContext(ctx, `SELECT scope FROM capture_bindings WHERE session=?`, identity).Scan(&originalScope)
	if err != nil && err != sql.ErrNoRows {
		return report, err
	}
	if err == nil && originalScope != scope {
		return report, errors.New("capture session binding changed, start a new session")
	}
	session := captureHash(identity + "\x00" + scope)
	seen := map[string]bool{}
	rows, err := a.DB.QueryContext(ctx, `SELECT message FROM capture_seen WHERE session=?`, session)
	if err != nil {
		return report, err
	}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return report, err
		}
		seen[id] = true
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return report, err
	}
	if o.Backfill {
		rows, err := a.DB.QueryContext(ctx, `SELECT message FROM capture_backlog WHERE session=?`, session)
		if err != nil {
			return report, err
		}
		for rows.Next() {
			var id string
			if err := rows.Scan(&id); err != nil {
				rows.Close()
				return report, err
			}
			delete(seen, id)
		}
		err = rows.Err()
		rows.Close()
		if err != nil {
			return report, err
		}
	}
	wanted := map[string]bool{}
	if o.Deferred {
		// A deferred part's message may be seen as a whole already: unsee
		// both so the part is cut again, then send the deferred ids only.
		rows, err := a.DB.QueryContext(ctx, `SELECT message, original FROM capture_deferred WHERE session=?`, session)
		if err != nil {
			return report, err
		}
		for rows.Next() {
			var id, original string
			if err := rows.Scan(&id, &original); err != nil {
				rows.Close()
				return report, err
			}
			wanted[id] = true
			delete(seen, id)
			delete(seen, original)
		}
		err = rows.Err()
		rows.Close()
		if err != nil {
			return report, err
		}
	}
	var pending []harness.CaptureMessage
	turns := 0
	for _, m := range o.Messages {
		if !seen[m.ID] {
			// Credential-shaped tokens never leave for the extraction provider; the
			// rest of the message does, so its facts are not lost with the token.
			m.Text = project.MaskCredentials(m.Text)
			pending = append(pending, m)
			if m.Role == "user" && m.Text != "" {
				turns++
			}
		}
	}
	every := a.Cfg.Extract.EveryTurns
	if every <= 0 {
		every = 12
	}
	if len(seen) == 0 && !o.Backfill && (o.Trigger == "turn_end" || o.Trigger == "session_end") && turns > every*2 {
		remaining := every * 2
		cut := len(pending)
		for cut > 0 {
			cut--
			if pending[cut].Role == "user" && pending[cut].Text != "" {
				remaining--
				if remaining == 0 {
					break
				}
			}
		}
		older := pending[:cut]
		report.BacklogSkipped = len(older)
		if !o.DryRun {
			turnCtx, release, err := a.WriteTurn(ctx)
			if err != nil {
				return report, err
			}
			tx, err := a.DB.BeginTx(turnCtx, nil)
			if err != nil {
				release()
				return report, err
			}
			for _, m := range older {
				if _, err = tx.ExecContext(ctx, `INSERT OR IGNORE INTO capture_seen(session,message) VALUES(?,?)`, session, m.ID); err != nil {
					break
				}
				if _, err = tx.ExecContext(ctx, `INSERT OR IGNORE INTO capture_backlog(session,message) VALUES(?,?)`, session, m.ID); err != nil {
					break
				}
			}
			if err == nil {
				_, err = tx.ExecContext(ctx, `INSERT INTO capture_state(session,high_water) VALUES(?,?) ON CONFLICT(session) DO UPDATE SET high_water=(SELECT count(*) FROM capture_seen WHERE session=excluded.session)`, session, len(older))
			}
			if err != nil {
				tx.Rollback()
				release()
				return report, err
			}
			err = tx.Commit()
			release()
			if err != nil {
				return report, err
			}
		}
		for _, m := range older {
			seen[m.ID] = true
		}
		pending = pending[cut:]
		turns = every * 2
	}
	if o.Periodic && turns < every {
		report.Skipped = "waiting for extraction interval"
		return report, nil
	}
	if len(pending) == 0 {
		report.Skipped = "no new messages"
		return report, nil
	}
	// Chunk by messages (or resumable hook fragments), retaining stable ids.
	hook := o.Trigger == "turn_end" || o.Trigger == "session_end"
	chunkLimit := 48 << 10
	originals, lastFragments := map[string]string{}, map[string]string{}
	if hook || o.Deferred {
		chunkLimit = a.Cfg.Extract.HookChunkBytes
		if chunkLimit <= 0 {
			chunkLimit = 8192
		}
		chunkLimit = max(1024, min(chunkLimit, 48<<10))
	}
	// Every path cuts a message longer than one call into parts, in order
	// (8jcj3k): refusing it stopped the session's capture for good.
	pending, originals, lastFragments = captureFragments(pending, seen, chunkLimit)
	if o.Deferred {
		// A drain sends the deferred chunks only, cut as the hook cut them
		// (same limit, so the same fragment ids).
		keep := pending[:0]
		for _, m := range pending {
			if wanted[m.ID] {
				keep = append(keep, m)
			}
		}
		pending = keep
	}
	if len(pending) == 0 {
		// Only parts already captured or deferred remain.
		report.Skipped = "no new messages"
		return report, nil
	}
	report.Split = splitCount(originals)
	// Messages beyond the deadline stay pending for the next observer.
	// A hook starts a chunk, and its paid call, only when the time left covers
	// what this run has seen a chunk take (a6xm8n): the slowest search, call
	// and planning-plus-apply so far, each with a floor. A fixed 3 s checked
	// before the search, with a 2 s reserve after the call, let run 8467
	// start its last call with 4.5 s left (2.5 s for the call) and run 8468,
	// whose search then took over 2.4 s, with about 0.6 s left. The cut
	// call failed both runs after 83 and 88 applied actions. The floor for a
	// call is 5 s: of 5482 hook calls since 2026-11-12, p99 took 4.7 s
	// (max 8.0 s).
	callFloor := min(5*time.Second, a.Cfg.Extract.CallTimeout(), hookCallTimeout)
	var slowSearch, slowCall, slowAfter time.Duration
	callMargin := func() time.Duration { return max(callFloor, slowCall) }
	afterMargin := func() time.Duration { return max(2*time.Second, slowAfter) }
	covers := func(need time.Duration) bool {
		deadline, ok := ctx.Deadline()
		return !hook || !ok || time.Until(deadline) >= need
	}
	for start := 0; start < len(pending); {
		if !covers(slowSearch + callMargin() + afterMargin()) {
			report.Skipped = captureBudgetSkip
			return report, nil
		}
		auditMessages := len(audit.Messages)
		var afterStart time.Time
		var chunk strings.Builder
		end := start
		for end < len(pending) {
			m := pending[end]
			t := m.Text
			line := m.Role + ": " + t + "\n"
			if t != "" && chunk.Len()+len(line) > chunkLimit && end > start {
				break
			}
			if len(line) > chunkLimit {
				// captureFragments sizes every part to fit, so this is a bug.
				return report, fmt.Errorf("capture message part exceeds %d bytes, refusing silent loss", chunkLimit)
			}
			audit.Messages = append(audit.Messages, AuditMessage{ID: m.ID, OriginalID: originals[m.ID], Role: m.Role, Hash: captureHash(m.Text), Text: t})
			if t != "" {
				chunk.WriteString(line)
			}
			end++
		}
		text := chunk.String()
		var result extract.Result
		truncated := false
		knownByContent := map[string]string{}
		if strings.TrimSpace(text) != "" {
			opts := a.RecallOptions()
			opts.MinScore = 0
			opts.Limit = a.Cfg.Extract.TopK
			if opts.Limit <= 0 {
				opts.Limit = 12
			}
			opts.Scopes = b.Scopes
			opts.ScopeFor = b.writableScopeFor
			opts.Kinds = []string{"memory"}
			// Limit query words so transcript-sized FTS expressions remain bounded.
			query := text
			if len(query) > 4000 {
				query = query[len(query)-4000:]
			}
			planStart := time.Now()
			hits, err := a.Searcher.Search(ctx, query, opts)
			slowSearch = max(slowSearch, time.Since(planStart))
			report.PlanMS += time.Since(planStart).Milliseconds()
			if err != nil {
				return report, err
			}
			// The search took its time: check again before the paid call.
			if !covers(callMargin() + afterMargin()) {
				audit.Messages = audit.Messages[:auditMessages]
				report.Skipped = captureBudgetSkip
				return report, nil
			}
			var known []extract.Known
			knownBytes := 0
			for _, r := range hits {
				if _, err := b.WriteScope(r.Unit.Scope); err != nil {
					continue
				}
				if !project.HasCredential(r.Unit.Body) {
					if hook && knownBytes+len(r.Unit.Body) > 4096 {
						continue
					}
					knownBytes += len(r.Unit.Body)
					known = append(known, extract.Known{SID: r.Unit.SID, Content: r.Unit.Body})
					knownByContent[r.Unit.Body] = r.Unit.SID
				}
			}
			// Leave time to persist this chunk after the provider returns.
			providerCtx := ctx
			providerCancel := func() {}
			if deadline, ok := ctx.Deadline(); ok {
				reserve := 2 * time.Second
				if hook {
					reserve = afterMargin()
				}
				if left := time.Until(deadline); left < 2*reserve {
					reserve = left / 4
				}
				providerCtx, providerCancel = context.WithDeadline(ctx, deadline.Add(-reserve))
			}
			callCfg := a.Cfg.Extract
			callTimeout := time.Duration(0)
			if hook {
				if callCfg.MaxTokens <= 0 || callCfg.MaxTokens > 1024 {
					callCfg.MaxTokens = 1024
				}
				callCfg.MaxTokens = max(128, callCfg.MaxTokens)
				callTimeout = hookCallTimeout
			}
			call := func(c extract.Config, timeout time.Duration) (extract.Result, error) {
				callCtx, callCancel := providerCtx, func() {}
				if timeout > 0 {
					callCtx, callCancel = context.WithTimeout(providerCtx, timeout)
				}
				defer callCancel()
				extractStart := time.Now()
				r, err := extract.Run(callCtx, c, text, known, b.extractionPlacesFor(c.WidePlaces)...)
				slowCall = max(slowCall, time.Since(extractStart))
				report.ExtractMS += time.Since(extractStart).Milliseconds()
				audit.Calls = append(audit.Calls, r)
				return r, err
			}
			result, err = call(callCfg, callTimeout)
			// A hook reply cut at max_tokens is sent once more with twice the
			// room, never applied as it is (nmn6wf). At temperature 0 the same
			// input is cut the same way on every later hook (runs 8453, 8459,
			// 8460), so waiting for the next hook does not help. Of 1905 audited
			// hook calls 21 hit 1024 and none of the complete ones needed more
			// than 19 lines, about 1400 tokens: 2048 holds them.
			// The retry at twice the room is not started when the time left
			// cannot cover it: the chunk is deferred below as still cut.
			if errors.Is(err, extract.ErrTruncated) && hook && covers(2*callMargin()+afterMargin()) {
				addUsage(&report, result)
				report.Retried++
				callCfg.MaxTokens *= 2
				result, err = call(callCfg, 2*callTimeout)
			}
			providerCancel()
			afterStart = time.Now()
			if errors.Is(err, extract.ErrTruncated) && (hook || o.Deferred) {
				// Still cut: apply nothing from it, checkpoint the chunk so the
				// messages after it go on, and keep it for DrainDeferred and
				// --backfill. Recorded in capture_runs as truncated/deferred.
				addUsage(&report, result)
				report.Truncated++
				for _, m := range pending[start:end] {
					report.Deferred = append(report.Deferred, m.ID)
				}
				truncated = true
				result, err = extract.Result{}, nil
			}
			// A candidate naming a swarm agent is rejected (ez50rb check 3):
			// the name stops meaning anything within hours. Deterministic, no
			// model call; measured on replayed transcripts (CHANGELOG).
			if err == nil {
				result = a.rejectAgentNames(result, text)
				if !truncated {
					audit.Calls[len(audit.Calls)-1] = result
				}
			}
			if err != nil {
				addUsage(&report, result)
				return report, err
			}
		}
		addUsage(&report, result)
		report.Rejected += result.Rejected
		for _, rejection := range result.Rejections {
			if strings.HasPrefix(rejection.Reason, "trivial:") {
				report.RejectedTrivial++
			}
		}
		// Stable batch fingerprint makes retries idempotent after native-file save
		// but before the SQLite high-water checkpoint commits.
		ids := make([]string, 0, end-start)
		for _, m := range pending[start:end] {
			ids = append(ids, m.ID)
		}
		batch := captureHash(session + "\x00" + strings.Join(ids, "\x00"))
		applyStart := time.Now()
		writes, err := newCaptureBatch()
		report.ApplyMS += time.Since(applyStart).Milliseconds()
		if err != nil {
			return report, err
		}
		local := map[string]bool{}
		for n, c := range result.Candidates {
			destination, err := b.capturePlacement(c.Place, a.Cfg.Extract.WidePlaces)
			if err != nil {
				return report, err
			}
			c.Place = destination.Name
			c.PlaceReason = strings.TrimSpace(c.PlaceReason)
			if c.PlaceReason == "" {
				c.PlaceReason = "No placement reason supplied by extractor."
			}
			if local[c.Content] {
				report.Actions = append(report.Actions, CaptureAction{Candidate: c, Scope: destination.Scope, PlaceHint: b.PlacementHint(c.Content, destination.Scope), Action: "skip", Reason: "duplicate proposal in batch"})
				continue
			}
			local[c.Content] = true
			planStart := time.Now()
			action, found := writes.exact(b, a, c)
			// Reuse exact known-memory matches, never the transcript query's
			// cosine as if it were the candidate's own embedding score.
			if !found && c.SameAs == "" && c.Updates == "" && c.Supersedes == "" {
				if sid := knownByContent[c.Content]; sid != "" {
					clean := c
					clean.Supersedes = ""
					action = CaptureAction{Candidate: clean, Action: "reinforce", Reason: "exact duplicate in known-memory snapshot", Target: sid, Score: 1}
					found = true
				}
			}
			if !found || c.SameAs != "" || c.Updates != "" || c.Supersedes != "" {
				action, err = b.capturePlan(ctx, a, c)
			}
			report.PlanMS += time.Since(planStart).Milliseconds()
			if err != nil {
				return report, err
			}
			action.Scope = destination.Scope
			// Exact reinforcement and correction retain the existing target's place.
			if action.Target != "" && (action.Action == "reinforce" || action.Action == "supersede") {
				if i, ok := writes.bySID[action.Target]; ok {
					action.Scope = writes.mems[i].Scope
				} else if target, err := b.Resolve(a, action.Target); err == nil {
					action.Scope = target.Scope
				} else {
					return report, err
				}
				action.Place = b.PlaceForScope(action.Scope).Name
			}
			action.PlaceHint = b.PlacementHint(action.Content, action.Scope)
			action.Origin = fmt.Sprintf("capture:%s:%s:%s:%d", o.Harness, o.Session, batch[:12], n)
			// Include the binding/home in receipts even when harness ids collide.
			receipt := captureHash(session + "\x00" + batch + "\x00" + c.Content)
			if !o.DryRun {
				applyStart = time.Now()
				err := writes.apply(ctx, b, a, action.Scope, &action, receipt)
				report.ApplyMS += time.Since(applyStart).Milliseconds()
				if err != nil {
					return report, err
				}
			}
			report.Actions = append(report.Actions, action)
		}
		if !o.DryRun {
			// The write turn covers the save and its checkpoint, never the
			// model call before them (m8shdj).
			turnCtx, release, err := a.WriteTurn(ctx)
			if err != nil {
				return report, err
			}
			applyStart = time.Now()
			err = writes.save(turnCtx, a)
			report.ApplyMS += time.Since(applyStart).Milliseconds()
			if err != nil {
				release()
				return report, err
			}
			tx, err := a.DB.BeginTx(turnCtx, nil)
			if err != nil {
				release()
				return report, err
			}
			for _, m := range pending[start:end] {
				if _, err = tx.ExecContext(ctx, `INSERT OR IGNORE INTO capture_seen(session,message) VALUES(?,?)`, session, m.ID); err != nil {
					break
				}
				if truncated {
					// Seen, so later chunks and hooks go on. From a hook it waits
					// in capture_deferred for a later hook to drain at the full
					// cap (DrainDeferred); its message is also in the backlog for
					// `hug mem capture --backfill`.
					if original := lastFragments[m.ID]; original != "" {
						// Its last part: the message is seen as a whole, so a
						// hook stops cutting it again.
						if _, err = tx.ExecContext(ctx, `INSERT OR IGNORE INTO capture_seen(session,message) VALUES(?,?)`, session, original); err != nil {
							break
						}
					}
					for _, id := range []string{m.ID, originals[m.ID]} {
						if id == "" {
							continue
						}
						if _, err = tx.ExecContext(ctx, `INSERT OR IGNORE INTO capture_backlog(session,message) VALUES(?,?)`, session, id); err != nil {
							break
						}
					}
					if err != nil {
						break
					}
					if hook {
						if _, err = tx.ExecContext(ctx, `INSERT OR IGNORE INTO capture_deferred(session,message,original,harness,home,harness_session,at_ns) VALUES(?,?,?,?,?,?,?)`, session, m.ID, originals[m.ID], o.Harness, o.Home, o.Session, time.Now().UnixNano()); err != nil {
							break
						}
					} else if _, err = tx.ExecContext(ctx, `DELETE FROM capture_deferred WHERE session=? AND message=?`, session, m.ID); err != nil {
						break
					}
					continue
				}
				if _, err = tx.ExecContext(ctx, `DELETE FROM capture_deferred WHERE session=? AND (message=? OR original=?)`, session, m.ID, m.ID); err != nil {
					break
				}
				if original := lastFragments[m.ID]; original != "" {
					if _, err = tx.ExecContext(ctx, `INSERT OR IGNORE INTO capture_seen(session,message) VALUES(?,?)`, session, original); err != nil {
						break
					}
					// Kept in the backlog while an earlier part of it waits in
					// capture_deferred, so --backfill can still recover it.
					if _, err = tx.ExecContext(ctx, `DELETE FROM capture_backlog WHERE session=? AND message=? AND NOT EXISTS (SELECT 1 FROM capture_deferred WHERE session=? AND original=?)`, session, original, session, original); err != nil {
						break
					}
				}
				if _, err = tx.ExecContext(ctx, `DELETE FROM capture_backlog WHERE session=? AND message=?`, session, m.ID); err != nil {
					break
				}
			}
			if err == nil {
				_, err = tx.ExecContext(ctx, `INSERT INTO capture_state(session,high_water) VALUES(?,?) ON CONFLICT(session) DO UPDATE SET high_water=(SELECT count(*) FROM capture_seen WHERE session=excluded.session)`, session, len(seen)+end)
			}
			if err != nil {
				tx.Rollback()
				release()
				return report, err
			}
			err = tx.Commit()
			release()
			if err != nil {
				return report, err
			}
		}
		if !afterStart.IsZero() {
			slowAfter = max(slowAfter, time.Since(afterStart))
		}
		report.Processed += end - start
		start = end
	}
	return report, nil
}

// hookCallTimeout bounds one hook extraction call.
const hookCallTimeout = 20 * time.Second

// captureBudgetSkip ends a hook run that stopped before a chunk it had no
// time left to finish; the rest waits for the next hook.
const captureBudgetSkip = "remaining messages deferred to next hook (budget)"

func (b Binding) capturePlan(ctx context.Context, a *App, c extract.Candidate) (CaptureAction, error) {
	b = b.Refresh(a)
	out := CaptureAction{Candidate: c, Action: "add", Reason: "new memory in bound scope"}
	if c.Updates != "" {
		old, err := b.Resolve(a, c.Updates)
		if err == nil && old.Active {
			if _, err := b.WriteScope(old.Scope); err == nil {
				out.Target = old.SID
				out.Reason = "newer version linked to existing fact; old note retained"
				if c.Category == "correction" && c.Trust == "high" {
					out.Action, out.Reason = "supersede", "high-trust correction in writable scope"
				}
				return out, nil
			}
		}
		out.Updates = ""
		out.Reason = "new memory; update target unavailable or outside writable scope"
		return out, nil
	}
	if c.Supersedes != "" && c.Trust == "high" && c.Category == "correction" {
		old, err := b.Resolve(a, c.Supersedes)
		if err == nil && old.Active {
			if _, err := b.WriteScope(old.Scope); err == nil {
				out.Action = "supersede"
				out.Reason = "high-trust correction in writable scope"
				out.Target = old.SID
				return out, nil
			}
		}
	}
	// Ignore invalid/low-trust supersession requests entirely, including display.
	out.Supersedes = ""
	if c.Supersedes != "" {
		out.Reason = "new memory, supersede target unavailable, outside writable scope, or not a high-trust correction"
		return out, nil
	}
	hits, err := b.memoryMatches(ctx, a, c.Content)
	if err != nil {
		return out, err
	}
	if c.SameAs != "" {
		old, err := b.Resolve(a, c.SameAs)
		if err == nil && old.Active {
			if _, err := b.WriteScope(old.Scope); err == nil {
				out.Action, out.Target, out.Reason = "reinforce", old.SID, "extractor judged same fact in Known (same call)"
				for _, h := range hits {
					if h.Unit.SID == old.SID {
						out.Score = h.Semantic
					}
				}
				return out, nil
			}
		}
		out.SameAs = ""
		out.Reason = "new memory; same-fact target unavailable or outside writable scope"
	}
	automatic, review := a.Cfg.Extract.DedupeThresholds()
	if len(hits) > 0 {
		h := hits[0]
		if h.Semantic >= automatic {
			out.Action, out.Target, out.Score, out.Reason = "reinforce", h.Unit.SID, h.Semantic, "cosine meets dedupe threshold"
		} else if h.Semantic >= review {
			out.Target, out.Score, out.Reason = h.Unit.SID, h.Semantic, "similar fact without same_as judgment; retained as new (no second provider call)"
		}
	}
	return out, nil
}

// captureExec runs one capture statement in the write turn (m8shdj): the
// turn is taken for the statement only, so capture never holds it across a
// model call, and it never interleaves with a remember, forget or index.
func (a *App) captureExec(ctx context.Context, query string, args ...any) (sql.Result, error) {
	ctx, release, err := a.WriteTurn(ctx)
	if err != nil {
		return nil, err
	}
	defer release()
	return a.DB.ExecContext(ctx, query, args...)
}

// addUsage counts one provider call, applied or not, in the run's cost.
func addUsage(report *CaptureReport, r extract.Result) {
	report.Usage.PromptTokens += r.Usage.PromptTokens
	report.Usage.CompletionTokens += r.Usage.CompletionTokens
	report.Usage.TotalTokens += r.Usage.TotalTokens
	report.LatencyMS += r.LatencyMS
}
