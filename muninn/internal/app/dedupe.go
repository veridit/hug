package app

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/veridit/hug/muninn/internal/embed"
	"github.com/veridit/hug/muninn/internal/extract"
	"github.com/veridit/hug/muninn/internal/index"
	"github.com/veridit/hug/muninn/internal/ingest"
	"github.com/veridit/hug/muninn/internal/search"
	"github.com/veridit/hug/muninn/internal/store"
	"github.com/veridit/hug/muninn/internal/trigger"
)

// DuplicateCheckError is an advisory question. Re-call with Reinforce set to
// this SID, or NewReason explaining the distinction. Neither option edits text.
type DuplicateCheckError struct {
	SID, Text string
	Score     float64
}

func (e *DuplicateCheckError) Error() string {
	return fmt.Sprintf(NotStored+" this looks like memory %s: %q (cosine %.3f). Same fact? Reinforce it (CLI --reinforce %s, MCP reinforce), or store anyway with a reason (CLI --new-reason, MCP new_reason).", e.SID, e.Text, e.Score, e.SID)
}

// NotStored begins every refusal of remember (p29rbz): a refusal must never
// read as success. A Seid agent took "not stored yet: ... Same fact?" for a
// stored memory, forgot the original, and lost both.
const NotStored = "NOT STORED:"

// Refusal reports whether err is one of remember's questions (duplicate,
// secret, place): nothing was written, and the writer must answer it.
func Refusal(err error) bool {
	var d *DuplicateCheckError
	var s *SecretCheckError
	var p *PlaceCheckError
	var tc *ingest.TriggerChangeError
	return errors.As(err, &d) || errors.As(err, &s) || errors.As(err, &p) || errors.As(err, &tc)
}

// RememberDecision carries the writer's answer to the duplicate question.
// RememberDecision carries the writer's answers to remember's checks, and
// the memory type (rule | preference | fact | record, "" = unstated).
// Holds is the condition the memory holds under (ez50rb, holds.Parse).
// Supersedes names the old memory this one corrects: it is recorded in the
// fix trail (rjk44w) and retires nothing; forget the old one separately if
// it no longer holds.
// Trigger is the situation the memory is for (jrj1vq); ClearTrigger, with
// Supersedes naming a memory with the same text, clears that memory's
// trigger on purpose (stored as None, so sync carries the clear).
type RememberDecision struct {
	Reinforce, NewReason, Type, Holds, Supersedes string
	Trigger                                       *trigger.Trigger
	ClearTrigger                                  bool
}

// MemoryTypes are the accepted memory types; see ingest.NativeMemory.Type.
var MemoryTypes = []string{"rule", "preference", "fact", "record"}

// NormalType returns t when it is a known memory type, else "".
func NormalType(t string) string {
	t = strings.ToLower(strings.TrimSpace(t))
	for _, k := range MemoryTypes {
		if t == k {
			return k
		}
	}
	return ""
}

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
	if v, ok := ctx.Value(dedupeVecKey{}).(dedupeVec); ok && v.text == content {
		o.Embedded, o.QueryVector = true, v.vec
	} else if a.DB.HasTurn(ctx) {
		// Under the write turn the embedder is never called (t0qap9): with
		// no vector embedded before the turn, the check is exact and
		// keyword only, and the next remember or capture of the same text
		// finds it by its content.
		o.Embedded = true
	}
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
	// The index takes what the save changed (t0qap9); a memory from another
	// source has no native write, and a full pass picks up its mirror.
	if !batch.saved {
		if _, err = a.Reindex(ctx); err != nil {
			return store.Unit{}, false, err
		}
	} else if err = a.indexWritten(ctx, batch.written); err != nil {
		return store.Unit{}, false, err
	}
	u, err = b.Resolve(a, u.SID)
	if err == nil {
		err = a.SaveAuditCtx(ctx, AuditRun{Time: time.Now().UTC(), Harness: "remember", Trigger: "remember", Scope: u.Scope, Report: CaptureReport{Actions: []CaptureAction{{Action: "reinforce", Target: u.SID, Score: score, Scope: u.Scope, Reason: reason}}}})
	}
	return u, true, err
}

type dedupeVecKey struct{}

// dedupeVec is a text embedded as a document before the write turn.
type dedupeVec struct {
	text string
	vec  []float32
}

// embedForDedupe embeds content the way memoryMatches compares it (a
// document under the current pipeline), before the caller takes the write
// turn, and carries the vector in ctx. A failed embed carries nil: the
// check then runs without cosine, as it does when the embedder errs.
func (a *App) embedForDedupe(ctx context.Context, content string) context.Context {
	if a.Searcher == nil || a.Searcher.Embedder == nil || content == "" || a.DB.HasTurn(ctx) {
		return ctx
	}
	e := a.Searcher.Embedder
	if a.DB.GetMeta("pipeline_hash") != index.PipelineHash(e.Model()) {
		return ctx // Search compares documents only under the current pipeline
	}
	var vec []float32
	if vecs, err := e.Embed(ctx, embed.Documents(e, []string{index.EmbeddingText("", content)})); err == nil && len(vecs) == 1 {
		vec = vecs[0]
	}
	return context.WithValue(ctx, dedupeVecKey{}, dedupeVec{content, vec})
}

// indexWritten updates the index for one write to the native store and
// nothing else (t0qap9 check 6): the units it changed, the concept graph's
// delta, their neighbour lists, and their vectors from the text the caller
// embedded before the write turn (embedForDedupe). A full pass runs only
// when the index had not recorded the file the write started from
// (index.ErrNotCurrent). ctx carries the write turn.
func (a *App) indexWritten(ctx context.Context, w ingest.Written) error {
	vecs := map[string][]float32{}
	if v, ok := ctx.Value(dedupeVecKey{}).(dedupeVec); ok && v.vec != nil {
		for _, m := range w.Changed {
			if m.Title == "" && strings.TrimSpace(m.Content) == v.text {
				vecs[index.ContentHash(index.EmbeddingText("", m.Content))] = v.vec
			}
		}
	}
	err := a.Indexer.ApplyWritten(ctx, NativePath(), w, vecs)
	if errors.Is(err, index.ErrNotCurrent) {
		_, err = a.Reindex(ctx)
	}
	if err != nil && !errors.Is(err, index.ErrBusy) {
		return fmt.Errorf("index: %w", err)
	}
	return err
}
