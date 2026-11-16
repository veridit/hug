package app

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/veridit/hug/muninn/internal/ingest"
	"github.com/veridit/hug/muninn/internal/store"
)

// ForgetResult distinguishes retirement of an owned memory from local suppression.
type ForgetResult struct {
	ID         string `json:"id"`
	Suppressed bool   `json:"suppressed,omitempty"`
	Message    string `json:"message"`
}

// Forget resolves every public identifier through the same store resolver.
// Non-owned source files are never modified.
func (a *App) Forget(ctx context.Context, ref string) (ForgetResult, error) {
	out, err := a.ForgetMany(ctx, []string{ref})
	if err != nil {
		return ForgetResult{}, err
	}
	return out[0], nil
}

// ForgetMany forgets every ref in one write turn: one load and save of each
// native store, one suppression transaction and one index pass, however many
// refs. It checks every ref before it changes anything; one unknown or
// already-forgotten ref fails the whole call and nothing is forgotten.
func (a *App) ForgetMany(ctx context.Context, refs []string) ([]ForgetResult, error) {
	ctx, release, err := a.WriteTurn(ctx)
	if err != nil {
		return nil, err
	}
	defer release()
	type plan struct {
		out              ForgetResult
		retire, suppress bool
	}
	plans := make([]plan, len(refs))
	byPath := map[string][]string{}
	var problems []string
	for i, ref := range refs {
		units, err := a.DB.ResolveAll(ref)
		if err != nil {
			if len(refs) == 1 {
				return nil, err
			}
			problems = append(problems, fmt.Sprintf("%s: %v", ref, err))
			continue
		}
		p := plan{out: ForgetResult{ID: units[0].SID}}
		for _, u := range units {
			var kind, path string
			if err := a.DB.QueryRow(`SELECT kind,path FROM sources WHERE id=?`, u.SourceID).Scan(&kind, &path); err != nil {
				return nil, err
			}
			if kind != "native" {
				p.suppress = true
				continue
			}
			if u.Active {
				p.retire = true
				byPath[path] = append(byPath[path], u.Key)
			}
		}
		if !p.retire && !p.suppress {
			problems = append(problems, fmt.Sprintf("no active memory %q", ref))
		}
		plans[i] = p
	}
	if len(problems) == 1 && len(refs) == 1 {
		return nil, errors.New(problems[0])
	}
	if len(problems) > 0 {
		return nil, fmt.Errorf("nothing forgotten: %s", strings.Join(problems, "; "))
	}
	retired := false
	for path, keys := range byPath {
		done, err := ingest.ForgetMany(path, keys)
		if err != nil {
			return nil, err
		}
		retired = retired || len(done) > 0
	}
	tx, err := a.DB.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	for path, keys := range byPath {
		for _, key := range keys {
			if _, err := tx.ExecContext(ctx, `UPDATE units SET active=0 WHERE key=? AND source_id=(SELECT id FROM sources WHERE path=?)`, key, path); err != nil {
				return nil, err
			}
		}
	}
	for _, p := range plans {
		if p.suppress {
			if _, err := tx.ExecContext(ctx, `INSERT OR IGNORE INTO suppressed(sid) VALUES(?)`, p.out.ID); err != nil {
				return nil, err
			}
		}
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	if retired {
		if _, err := a.Reindex(ctx); err != nil {
			return nil, err
		}
	}
	out := make([]ForgetResult, len(plans))
	for i, p := range plans {
		r := p.out
		r.Suppressed = p.suppress
		switch {
		case p.retire && p.suppress:
			r.Message = "forgot " + r.ID + " and suppressed its mirrors in muninn (upstream files unchanged)"
		case p.suppress:
			r.Message = "suppressed " + r.ID + " in muninn (upstream files unchanged)"
		default:
			r.Message = "forgot " + r.ID
		}
		out[i] = r
	}
	return out, nil
}

// Origin describes the retained upstream mapping, not the public display id.
type Origin struct {
	SID    string `json:"sid"`
	Source string `json:"source"`
	Path   string `json:"path"`
	Key    string `json:"key"`
	Origin string `json:"origin"`
}

func (a *App) Origin(u store.Unit) (Origin, error) {
	o := Origin{SID: u.SID, Key: u.Key, Origin: u.Origin}
	err := a.DB.QueryRow(`SELECT kind,path FROM sources WHERE id=?`, u.SourceID).Scan(&o.Source, &o.Path)
	return o, err
}
