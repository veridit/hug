package app

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/veridit/hug/muninn/internal/config"
	"github.com/veridit/hug/muninn/internal/tidy"
)

// Dreaming (2026-11-12, user): memory housekeeping runs by itself, with undo.
// Mechanical changes are applied automatically. Anything that needs judgement
// is queued and handed to the driving agent, never to a cheap model: the
// agent sees the memories and the user's context, and acts with
// memory_forget / memory_remember.

// DreamResult reports one dream pass.
type DreamResult struct {
	Retired int    `json:"retired"`          // duplicates retired automatically
	Backup  string `json:"backup,omitempty"` // undo receipt for the retirements
	Queued  int    `json:"queued"`           // new judgement items
	Open    int    `json:"open"`             // judgement items waiting in total
	DryRun  bool   `json:"dry_run"`
}

// DreamItem is one judgement item for the agent.
type DreamItem struct {
	ID          int64    `json:"id"`
	Kind        string   `json:"kind"`
	Instruction string   `json:"instruction"`
	SIDs        []string `json:"sids"`
	Memories    []string `json:"memories"` // "sid: text" lines, as the agent sees them
}

// Dream runs one pass. Exact and near-identical duplicates (same scope, the
// keeper keeps all the text) are retired with an undo receipt; duplicates in
// the review band, and ones where a retired copy holds more detail, are
// queued for judgement.
func (a *App) Dream(ctx context.Context, dry bool) (DreamResult, error) {
	res := DreamResult{DryRun: dry}
	g, err := tidy.ReadGraph(ctx, config.DBPath())
	if err != nil {
		return res, err
	}
	auto, review := a.Cfg.Extract.DedupeCosine, a.Cfg.Extract.ReviewCosine
	if auto <= 0 {
		auto = 0.95
	}
	if review <= 0 || review >= auto {
		review = 0.88
	}
	plan, err := tidy.ScanDuplicates(g, auto, review)
	if err != nil {
		return res, err
	}
	var safe []tidy.DuplicateCluster
	var judge []tidy.DuplicateCluster
	for _, c := range plan.Duplicates {
		if c.Band == "automatic" && !losesDetail(c) {
			safe = append(safe, c)
		} else {
			judge = append(judge, c)
		}
	}
	if !dry && len(safe) > 0 {
		plan.Duplicates = safe
		plan.Database = config.DBPath()
		r, err := a.ApplyTidy(ctx, plan, "", false)
		if err != nil {
			return res, err
		}
		res.Retired, res.Backup = len(r.Retirements), r.Backup
		a.logRetired(res.Retired)
	} else {
		for _, c := range safe {
			res.Retired += len(c.Retire)
		}
	}
	for _, c := range judge {
		q, err := a.queueDupes(c, dry)
		if err != nil {
			return res, err
		}
		if q {
			res.Queued++
		}
	}
	n, err := a.queueUnrecalled(dry, time.Now())
	if err != nil {
		return res, err
	}
	res.Queued += n
	err = a.DB.QueryRow(`SELECT count(*) FROM dream_items WHERE resolved_ns=0`).Scan(&res.Open)
	return res, err
}

// DreamSummary counts the last day of dreaming: merged by the pass, items
// handed out to agents, resolved, and still open.
type DreamSummary struct {
	Retired  int `json:"retired"`
	Handed   int `json:"handed"`
	Resolved int `json:"resolved"`
	Open     int `json:"open"`
}

// DreamSummarySince reports dream activity since t. Retired counts memories
// suppressed by dream undo receipts written since t.
func (a *App) DreamSummarySince(t time.Time) DreamSummary {
	var s DreamSummary
	ns := t.UnixNano()
	a.DB.QueryRow(`SELECT count(*) FROM dream_items WHERE handed_ns>=?`, ns).Scan(&s.Handed)
	a.DB.QueryRow(`SELECT count(*) FROM dream_items WHERE resolved_ns>=?`, ns).Scan(&s.Resolved)
	a.DB.QueryRow(`SELECT count(*) FROM dream_items WHERE resolved_ns=0`).Scan(&s.Open)
	var v string
	if a.DB.QueryRow(`SELECT value FROM meta WHERE key='dream_retired_log'`).Scan(&v) == nil {
		for _, line := range strings.Split(v, "\n") {
			var at int64
			var n int
			if _, err := fmt.Sscanf(line, "%d %d", &at, &n); err == nil && at >= ns {
				s.Retired += n
			}
		}
	}
	return s
}

func (a *App) logRetired(n int) {
	if n <= 0 {
		return
	}
	var v string
	a.DB.QueryRow(`SELECT value FROM meta WHERE key='dream_retired_log'`).Scan(&v)
	lines := strings.Split(strings.TrimSpace(v), "\n")
	if len(lines) > 60 {
		lines = lines[len(lines)-60:]
	}
	lines = append(lines, fmt.Sprintf("%d %d", time.Now().UnixNano(), n))
	a.DB.Exec(`INSERT INTO meta(key,value) VALUES('dream_retired_log',?) ON CONFLICT(key) DO UPDATE SET value=excluded.value`, strings.TrimSpace(strings.Join(lines, "\n")))
}

// losesDetail reports a cluster where a copy to retire is clearly longer than
// the keeper: merging could drop facts, so a person or agent should decide.
func losesDetail(c tidy.DuplicateCluster) bool {
	for _, r := range c.Retire {
		if len(r.Text) > len(c.Keeper.Text)*13/10 {
			return true
		}
	}
	return false
}

func (a *App) queueDupes(c tidy.DuplicateCluster, dry bool) (bool, error) {
	sids := []string{c.Keeper.SID}
	for _, r := range c.Retire {
		sids = append(sids, r.SID)
	}
	sort.Strings(sids)
	sum := sha256.Sum256([]byte("dupes\x00" + strings.Join(sids, " ")))
	key := "dupes:" + hex.EncodeToString(sum[:8])
	instruction := "These memories say the same thing. Keep the one that is complete and correct; " +
		"if one copy holds a detail the others lack, memory_remember a merged version first. " +
		"Then memory_forget the others."
	if dry {
		var one int
		return a.DB.QueryRow(`SELECT 1 FROM dream_items WHERE key=?`, key).Scan(&one) != nil, nil
	}
	r, err := a.DB.Exec(`INSERT OR IGNORE INTO dream_items(kind,key,sids,scope,instruction,created_ns) VALUES('dupes',?,?,?,?,?)`,
		key, strings.Join(sids, " "), c.Keeper.EffectiveScope, instruction, time.Now().UnixNano())
	if err != nil {
		return false, err
	}
	n, _ := r.RowsAffected()
	return n > 0, nil
}

// DreamItems hands up to max open items visible in this binding to a session
// and marks them handed. An item stays visible only if every memory in it is
// still active and visible: resolved-by-hand items close themselves.
func (b Binding) DreamItems(a *App, session string, max int) ([]DreamItem, error) {
	b = b.Refresh(a)
	if b.Err != nil {
		return nil, b.Err
	}
	if max <= 0 || max > 5 {
		max = 5
	}
	rows, err := a.DB.Query(`SELECT id,kind,sids,instruction FROM dream_items WHERE resolved_ns=0 ORDER BY handed_ns, id`)
	if err != nil {
		return nil, err
	}
	type raw struct {
		id                      int64
		kind, sids, instruction string
	}
	var all []raw
	for rows.Next() {
		var r raw
		if err := rows.Scan(&r.id, &r.kind, &r.sids, &r.instruction); err == nil {
			all = append(all, r)
		}
	}
	rows.Close()
	var out []DreamItem
	now := time.Now().UnixNano()
	for _, r := range all {
		if len(out) >= max {
			break
		}
		item := DreamItem{ID: r.id, Kind: r.kind, Instruction: r.instruction}
		active, visible := 0, true
		for _, sid := range strings.Fields(r.sids) {
			u, err := a.DB.Resolve(sid)
			if err != nil || !u.Active || a.DB.IsSuppressed(u.SID) {
				continue
			}
			shown, ok := b.Present(u)
			if !ok {
				visible = false
				break
			}
			active++
			item.SIDs = append(item.SIDs, shown.SID)
			item.Memories = append(item.Memories, shown.SID+": "+shown.Body)
		}
		if !visible {
			continue
		}
		need := 2 // dupes: done once one copy is left
		if r.kind == "vet" {
			need = 1 // vet: done once the memory is gone
		}
		if active < need {
			// Someone already resolved it (forgot all but one).
			a.DB.Exec(`UPDATE dream_items SET resolved_ns=?, resolution='resolved outside dream' WHERE id=?`, now, r.id)
			continue
		}
		a.DB.Exec(`UPDATE dream_items SET handed_ns=?, handed_to=? WHERE id=?`, now, session, r.id)
		out = append(out, item)
	}
	return out, nil
}

// ResolveDream closes an item the agent has handled.
func (b Binding) ResolveDream(a *App, id int64, note string) error {
	r, err := a.DB.Exec(`UPDATE dream_items SET resolved_ns=?, resolution=? WHERE id=? AND resolved_ns=0`, time.Now().UnixNano(), strings.TrimSpace(note), id)
	if err != nil {
		return err
	}
	if n, _ := r.RowsAffected(); n == 0 {
		return fmt.Errorf("no open dream item %d", id)
	}
	return nil
}

// OpenDreamCount counts open items this binding could be handed.
func (b Binding) OpenDreamCount(a *App) int {
	b = b.Refresh(a)
	if b.Err != nil {
		return 0
	}
	rows, err := a.DB.Query(`SELECT sids FROM dream_items WHERE resolved_ns=0`)
	if err != nil {
		return 0
	}
	// The store has one connection: read all rows before resolving units.
	var firsts []string
	for rows.Next() {
		var sids string
		if rows.Scan(&sids) != nil {
			continue
		}
		if f := strings.Fields(sids); len(f) > 0 {
			firsts = append(firsts, f[0])
		}
	}
	rows.Close()
	n := 0
	for _, sid := range firsts {
		if u, err := a.DB.Resolve(sid); err == nil {
			if _, ok := b.Present(u); ok {
				n++
			}
		}
	}
	return n
}

// RecallStats is how often recall has put a memory into a prompt.
type RecallStats struct {
	Count    int       `json:"recalled"` // injections across all sessions
	Sessions int       `json:"sessions"` // distinct sessions
	Last     time.Time `json:"last_recalled,omitempty"`
}

// Recalled reports recall statistics for a unit (and its mirrors by SID).
func (a *App) Recalled(sid string) RecallStats {
	var s RecallStats
	var last int64
	a.DB.QueryRow(`SELECT count(*), count(DISTINCT i.session), COALESCE(MAX(i.at_ns),0)
		FROM injections i JOIN units u ON u.id=i.unit_id WHERE u.sid=?`, sid).Scan(&s.Count, &s.Sessions, &last)
	if last > 0 {
		s.Last = time.Unix(0, last)
	}
	return s
}

// RecallLogSince is when recall logging began (first injection), zero if never.
func (a *App) RecallLogSince() time.Time {
	var first int64
	a.DB.QueryRow(`SELECT COALESCE(MIN(at_ns),0) FROM injections`).Scan(&first)
	if first == 0 {
		return time.Time{}
	}
	return time.Unix(0, first)
}

// UnusedAfter is how long recall must have been logging, and how long a
// memory must have gone unchanged and unrecalled, before it is vetted.
var UnusedAfter = 60 * 24 * time.Hour

// VetPerPass caps vetting items queued by one dream pass, so the queue stays
// a few minutes of agent work, not a backlog.
var VetPerPass = 10

// queueUnrecalled queues "still true?" items for memories recall has never
// used. Never archived automatically (user, 2026-11-12, memory 156x77): rare
// knowledge is often the most critical, needed almost never but crucial when
// it is. The agent keeps anything still true and retires only what is wrong.
func (a *App) queueUnrecalled(dry bool, now time.Time) (int, error) {
	since := a.RecallLogSince()
	if since.IsZero() || now.Sub(since) < UnusedAfter {
		return 0, nil
	}
	cutoff := now.Add(-UnusedAfter).UnixNano()
	rows, err := a.DB.Query(`SELECT u.sid, u.scope FROM units u JOIN sources s ON s.id=u.source_id
		WHERE u.active=1 AND u.kind='memory' AND u.updated_ns < ? AND u.sid IS NOT NULL
		AND NOT EXISTS (SELECT 1 FROM suppressed x WHERE x.sid=u.sid)
		AND NOT EXISTS (SELECT 1 FROM injections i WHERE i.unit_id=u.id)
		AND NOT EXISTS (SELECT 1 FROM dream_items d WHERE d.kind='vet' AND d.sids=u.sid)
		ORDER BY u.updated_ns LIMIT ?`, cutoff, VetPerPass)
	if err != nil {
		return 0, err
	}
	type cand struct{ sid, scope string }
	var cs []cand
	for rows.Next() {
		var c cand
		if rows.Scan(&c.sid, &c.scope) == nil {
			cs = append(cs, c)
		}
	}
	rows.Close()
	if dry {
		return len(cs), nil
	}
	instruction := "Recall has never used this memory in 60+ days. Vet it, do not judge by usefulness: " +
		"rare knowledge is often the most critical. If it is still true, leave it (resolve with note kept). " +
		"If it is wrong, memory_remember the correction then memory_forget it. If it is obsolete, memory_forget it."
	n := 0
	for _, c := range cs {
		r, err := a.DB.Exec(`INSERT OR IGNORE INTO dream_items(kind,key,sids,scope,instruction,created_ns) VALUES('vet',?,?,?,?,?)`,
			"vet:"+c.sid, c.sid, c.scope, instruction, now.UnixNano())
		if err != nil {
			return n, err
		}
		if k, _ := r.RowsAffected(); k > 0 {
			n++
		}
	}
	return n, nil
}
