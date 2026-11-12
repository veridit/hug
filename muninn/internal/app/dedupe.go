package app

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/veridit/hug/muninn/internal/extract"
	"github.com/veridit/hug/muninn/internal/search"
	"github.com/veridit/hug/muninn/internal/store"
)

// DuplicateCheckError is an advisory question. Re-call with Reinforce set to
// this SID, or NewReason explaining the distinction. Neither option edits text.
type DuplicateCheckError struct {
	SID, Text string
	Score     float64
}

func (e *DuplicateCheckError) Error() string {
	return fmt.Sprintf("not stored yet: this looks like memory %s: %q (cosine %.3f). Same fact? Reinforce it (CLI --reinforce %s, MCP reinforce), or store anyway with a reason (CLI --new-reason, MCP new_reason).", e.SID, e.Text, e.Score, e.SID)
}

// RememberDecision carries the writer's answer to the duplicate question.
type RememberDecision struct{ Reinforce, NewReason string }

func (b Binding) writableScopeFor(u store.Unit) (string, bool) {
	scope, ok := b.ScopeFor(u)
	if !ok {
		return "", false
	}
	_, err := b.WriteScope(scope)
	return scope, err == nil
}

func (b Binding) rememberMatches(ctx context.Context, a *App, content string) ([]search.Result, error) {
	batch, err := newCaptureBatch()
	if err != nil {
		return nil, err
	}
	if action, ok := batch.exact(b, a, extract.Candidate{Content: content}); ok {
		m := batch.mems[batch.bySID[action.Target]]
		return []search.Result{{Unit: store.Unit{SID: m.SID, Key: m.ID, Scope: m.Scope, Body: m.Content, Active: true, Kind: "memory"}, Semantic: 1, SemanticOn: true}}, nil
	}
	return b.memoryMatches(ctx, a, content)
}

// memoryMatches never uses keyword score as cosine, and filters both visibility
// and write permission before a candidate can be reinforced or shown in a question.
func (b Binding) memoryMatches(ctx context.Context, a *App, content string) ([]search.Result, error) {
	where, args := b.SQL("u.scope", "u.source_id")
	args = append(args, content)
	rows, err := a.DB.QueryContext(ctx, `SELECT u.id FROM units u WHERE u.active=1 AND u.kind='memory' AND (`+where+`) AND u.body=? AND NOT EXISTS(SELECT 1 FROM suppressed s WHERE s.sid=u.sid) AND u.id=(SELECT c.id FROM units c JOIN sources s ON s.id=c.source_id WHERE c.sid=u.sid ORDER BY (s.kind='native') DESC,c.updated_ns DESC,c.id LIMIT 1) ORDER BY u.id`, args...)
	if err != nil {
		return nil, err
	}
	var ids []int64
	for rows.Next() {
		var id int64
		if err = rows.Scan(&id); err != nil {
			rows.Close()
			return nil, err
		}
		ids = append(ids, id)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return nil, err
	}
	for _, id := range ids {
		u, err := a.DB.GetUnit(id)
		if err != nil {
			return nil, err
		}
		shown, ok := b.Present(u)
		if !ok {
			continue
		}
		if _, err := b.WriteScope(shown.Scope); err == nil {
			return []search.Result{{Unit: shown, Semantic: 1, SemanticOn: true}}, nil
		}
	}
	if a.Searcher.Embedder == nil {
		return nil, nil
	}
	o := a.RecallOptions()
	o.Limit, o.MinScore, o.SemanticSlots, o.SemanticFloor = 32, 0, 32, 0
	o.Scopes, o.ScopeFor, o.Kinds = b.Scopes, b.writableScopeFor, []string{"memory"}
	o.DocumentQuery = true
	hits, err := a.Searcher.Search(ctx, content, o)
	if err != nil {
		return nil, err
	}
	out := hits[:0]
	for _, h := range hits {
		if h.SemanticOn {
			if _, err := b.WriteScope(h.Unit.Scope); err == nil {
				out = append(out, h)
			}
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Semantic == out[j].Semantic {
			return out[i].Unit.SID < out[j].Unit.SID
		}
		return out[i].Semantic > out[j].Semantic
	})
	return out, nil
}

func (b Binding) reinforceMemory(ctx context.Context, a *App, u store.Unit, score float64, reason string) (store.Unit, bool, error) {
	// Reuse capture's atomic native reinforcement and upstream-local event path.
	batch, err := newCaptureBatch()
	if err != nil {
		return store.Unit{}, false, err
	}
	action := CaptureAction{Action: "reinforce", Target: u.SID}
	if err = batch.apply(ctx, b, a, u.Scope, &action, fmt.Sprintf("remember:%d", time.Now().UnixNano())); err != nil {
		return store.Unit{}, false, err
	}
	if err = batch.save(ctx, a); err != nil {
		return store.Unit{}, false, err
	}
	if _, err = a.Reindex(ctx); err != nil {
		return store.Unit{}, false, err
	}
	u, err = b.Resolve(a, u.SID)
	if err == nil {
		err = a.SaveAudit(AuditRun{Time: time.Now().UTC(), Harness: "remember", Trigger: "remember", Scope: u.Scope, Report: CaptureReport{Actions: []CaptureAction{{Action: "reinforce", Target: u.SID, Score: score, Scope: u.Scope, Reason: reason}}}})
	}
	return u, true, err
}
