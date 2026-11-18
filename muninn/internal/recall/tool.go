package recall

import (
	"context"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/veridit/hug/muninn/internal/anchor"
	"github.com/veridit/hug/muninn/internal/search"
	"github.com/veridit/hug/muninn/internal/store"
)

// ToolRequest is one tool call, as jcode's post_tool_feedback hook sees it
// (n7jbmx): the tool, its command (bash) and the paths it touched.
type ToolRequest struct {
	Session, Harness string
	Tool, Command    string
	Touched          []string
	// Root and Home make absolute paths relative: the project, then home.
	Root, Home string
	Scopes     []string
	ScopeFor   func(store.Unit) (string, bool) `json:"-"`
	// Limit caps the memories shown per call (0 = DefaultToolLimit).
	Limit int
	// MaxShared drops an anchor more memories than this share: it says
	// nothing about any one of them (0 = DefaultMaxShared).
	MaxShared int
	// MinCos > 0 also asks the call's text to be this close to the memory
	// by embedding. Similar computes it (default: Engine.Searcher's
	// Similarities); with neither, or when it fails, nothing is shown: the
	// gate is part of what was measured, so recall never runs without it.
	MinCos  float64
	Similar func(ctx context.Context, text string, ids []int64) (map[int64]float64, error) `json:"-"`
	// Near keeps only path anchors that end at a touched or named path or
	// its folder (anchor.CallNear).
	Near bool
	// NeedPath drops a memory whose only shared anchors are commands.
	NeedPath bool
}

const (
	DefaultToolLimit = 2
	DefaultMaxShared = 40
	// ShipMinCos is the cosine gate of the arm chosen on the dev replay
	// (n7jbmx, 300 calls of 2026-11-16..18): with Near and NeedPath, 9.0
	// noise and 10.7 useful memories per 100 calls, 54% precise.
	ShipMinCos = 0.65
)

// Shipped is the tool-call recall the hook runs: the arm fixed on the dev
// replay before the fresh held-out set was drawn.
func (r ToolRequest) Shipped() ToolRequest {
	r.Near, r.NeedPath, r.MinCos = true, true, ShipMinCos
	return r
}

// ToolTypes are the memory types a tool call may surface: something that
// changes what the agent does next. A record (what happened) never does on
// its own, and an untyped memory is not trusted to be one of these.
var ToolTypes = []string{"rule", "fact", "preference"}

// ToolRecall finds memories that name an anchor of the call exactly (a
// path, a file name, a command head), keeps those of a ToolTypes type in
// scope and not yet shown this session, ranks them by how rare their shared
// anchors are, and renders at most Limit of them.
func (e *Engine) ToolRecall(ctx context.Context, r ToolRequest) (Response, error) {
	t0 := time.Now()
	anchors := anchor.CallNear(r.Tool, r.Command, r.Touched, r.Root, r.Home, r.Near)
	if len(anchors) == 0 {
		return Response{Skipped: "no anchors"}, nil
	}
	limit, maxShared := r.Limit, r.MaxShared
	if limit <= 0 {
		limit = DefaultToolLimit
	}
	if maxShared <= 0 {
		maxShared = DefaultMaxShared
	}
	args := make([]any, len(anchors))
	for i, a := range anchors {
		args[i] = a
	}
	in := "?" + strings.Repeat(",?", len(anchors)-1)
	rows, err := e.DB.QueryContext(ctx, `SELECT anchor, COUNT(*) FROM unit_anchors WHERE anchor IN (`+in+`) GROUP BY anchor`, args...)
	if err != nil {
		return Response{}, err
	}
	df := map[string]int{}
	for rows.Next() {
		var a string
		var n int
		if err := rows.Scan(&a, &n); err != nil {
			rows.Close()
			return Response{}, err
		}
		if n <= maxShared {
			df[a] = n
		}
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return Response{}, err
	}
	if len(df) == 0 {
		return Response{Elapsed: time.Since(t0).Round(time.Millisecond).String()}, nil
	}
	var total float64
	e.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM units WHERE active=1`).Scan(&total)
	args, in = args[:0], ""
	for a := range df {
		args = append(args, a)
	}
	in = "?" + strings.Repeat(",?", len(args)-1)
	rows, err = e.DB.QueryContext(ctx, `SELECT unit_id, anchor FROM unit_anchors WHERE anchor IN (`+in+`)`, args...)
	if err != nil {
		return Response{}, err
	}
	score := map[int64]float64{}
	why := map[int64][]string{}
	for rows.Next() {
		var id int64
		var a string
		if err := rows.Scan(&id, &a); err != nil {
			rows.Close()
			return Response{}, err
		}
		score[id] += math.Log(1 + total/float64(df[a]))
		why[id] = append(why[id], a)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return Response{}, err
	}
	type cand struct {
		u     store.Unit
		score float64
	}
	var cands []cand
	for id, s := range score {
		if r.NeedPath && !namesPath(why[id]) {
			continue
		}
		u, err := e.DB.GetUnit(id)
		if err != nil || !u.Active || !isToolType(u.Type) || !e.DB.IsCanonical(u) {
			continue
		}
		if r.ScopeFor != nil {
			var ok bool
			if u.Scope, ok = r.ScopeFor(u); !ok {
				continue
			}
		}
		if !search.Allowed(u.Scope, r.Scopes) || e.holdsEnded(u, time.Now()) {
			continue
		}
		if r.Session != "" && e.injected(r.Session, u.ContentHash) {
			continue
		}
		cands = append(cands, cand{u, s})
	}
	if r.MinCos > 0 && len(cands) > 0 {
		similar := r.Similar
		if similar == nil && e.Searcher != nil && e.Searcher.Embedder != nil {
			similar = e.Searcher.Similarities
		}
		if similar == nil {
			return Response{Skipped: "no embedder", Elapsed: time.Since(t0).Round(time.Millisecond).String()}, nil
		}
		ids := make([]int64, len(cands))
		for i, c := range cands {
			ids[i] = c.u.ID
		}
		sims, err := similar(ctx, toolText(r), ids)
		if err != nil {
			return Response{Skipped: "embed failed: " + err.Error(), Elapsed: time.Since(t0).Round(time.Millisecond).String()}, nil
		}
		kept := cands[:0]
		for _, c := range cands {
			if sims[c.u.ID] >= r.MinCos {
				kept = append(kept, c)
			}
		}
		cands = kept
	}
	sort.Slice(cands, func(i, j int) bool {
		if cands[i].score != cands[j].score {
			return cands[i].score > cands[j].score
		}
		if !cands[i].u.Updated.Equal(cands[j].u.Updated) {
			return cands[i].u.Updated.After(cands[j].u.Updated)
		}
		return cands[i].u.ID < cands[j].u.ID
	})
	var hits []Hit
	for _, c := range cands {
		if len(hits) >= limit {
			break
		}
		sort.Strings(why[c.u.ID])
		hits = append(hits, Hit{SID: c.u.SID, UnitID: c.u.ID, Key: c.u.Key, Source: e.sourceKind(c.u.SourceID), Text: c.u.Body, Kind: c.u.Kind,
			Scope: c.u.Scope, Score: c.score, Why: "anchors " + strings.Join(why[c.u.ID], ", "), Updated: c.u.Updated})
		if r.Session != "" {
			e.record(r.Session, c.u.ID, c.u.ContentHash, r.Harness)
		}
	}
	return Response{Hits: hits, Block: Render(hits, true), Elapsed: time.Since(t0).Round(time.Millisecond).String()}, nil
}

// toolText is a call as one line of text, for the embedding arm.
func toolText(r ToolRequest) string {
	return strings.TrimSpace(r.Tool + " " + strings.Join(r.Touched, " ") + " " + Truncate(r.Command, 400))
}

func namesPath(anchors []string) bool {
	for _, a := range anchors {
		if !strings.HasPrefix(a, "cmd:") {
			return true
		}
	}
	return false
}

func isToolType(t string) bool {
	for _, x := range ToolTypes {
		if t == x {
			return true
		}
	}
	return false
}
