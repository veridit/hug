package app

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/veridit/hug/muninn/internal/index"
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
	out, _, err := a.forgetMany(ctx, refs)
	return out, err
}

// forgetMany is ForgetMany that also returns each native store file it
// saved, with the file's stat right after the save, for the readback.
func (a *App) forgetMany(ctx context.Context, refs []string) ([]ForgetResult, map[string]ingest.FileStat, error) {
	saved := map[string]ingest.FileStat{}
	ctx, release, err := a.WriteTurn(ctx)
	if err != nil {
		return nil, nil, err
	}
	defer release()
	type plan struct {
		out              ForgetResult
		retire, suppress bool
		target           fixTarget
	}
	plans := make([]plan, len(refs))
	byPath := map[string][]string{}
	var problems []string
	for i, ref := range refs {
		units, err := a.DB.ResolveAll(ref)
		if err != nil {
			if len(refs) == 1 {
				return nil, nil, err
			}
			problems = append(problems, fmt.Sprintf("%s: %v", ref, err))
			continue
		}
		p := plan{out: ForgetResult{ID: units[0].SID}, target: fixTarget{sid: units[0].SID, hash: units[0].ContentHash}}
		for _, u := range units {
			p.target.ids = append(p.target.ids, u.ID)
			var kind, path string
			if err := a.DB.QueryRow(`SELECT kind,path FROM sources WHERE id=?`, u.SourceID).Scan(&kind, &path); err != nil {
				return nil, nil, err
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
		return nil, nil, errors.New(problems[0])
	}
	if len(problems) > 0 {
		return nil, nil, fmt.Errorf("nothing forgotten: %s", strings.Join(problems, "; "))
	}
	retired := false
	type applied struct {
		path string
		f    ingest.Forgotten
	}
	var writes []applied
	for path, keys := range byPath {
		f, err := ingest.ForgetManyChanged(path, keys)
		if err != nil {
			return nil, nil, err
		}
		if len(f.Done) > 0 {
			retired = true
			writes = append(writes, applied{path, f})
			saved[path] = f.After
		}
	}
	tx, err := a.DB.BeginTx(ctx, nil)
	if err != nil {
		return nil, nil, err
	}
	defer tx.Rollback()
	for path, keys := range byPath {
		for _, key := range keys {
			if _, err := tx.ExecContext(ctx, `UPDATE units SET active=0 WHERE key=? AND source_id=(SELECT id FROM sources WHERE path=?)`, key, path); err != nil {
				return nil, nil, err
			}
		}
	}
	for _, p := range plans {
		if p.suppress {
			if _, err := tx.ExecContext(ctx, `INSERT OR IGNORE INTO suppressed(sid) VALUES(?)`, p.out.ID); err != nil {
				return nil, nil, err
			}
		}
	}
	if err := tx.Commit(); err != nil {
		return nil, nil, err
	}
	if retired {
		// Update only what the write changed (4vd1mz); a full pass only
		// when the index had not recorded the file the write started from.
		full := false
		for _, w := range writes {
			if full {
				break
			}
			err := a.applyNative(ctx, w.path, w.f)
			if errors.Is(err, index.ErrNotCurrent) {
				full = true
			} else if err != nil {
				return nil, nil, err
			}
		}
		if full {
			if _, err := a.Reindex(ctx); err != nil {
				return nil, nil, err
			}
		}
	}
	out := make([]ForgetResult, len(plans))
	targets := make([]fixTarget, len(plans))
	for i, p := range plans {
		targets[i] = p.target
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
	// The trail (rjk44w): written after the forget landed, never blocking it.
	recordFixes(ctx, a, "forget", targets)
	return out, saved, nil
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

// applyNative updates the index for one forget's write to the native store
// at path (index.Indexer.ApplyNative), under the write turn and the
// index step mutex as Reindex's steps.
func (a *App) applyNative(ctx context.Context, path string, f ingest.Forgotten) error {
	ctx, release, err := a.WriteTurn(ctx)
	if err != nil {
		return fmt.Errorf("index: %w", err)
	}
	defer release()
	return a.Indexer.ApplyNative(ctx, path, f.Before, f.After, f.Changed)
}

// FreshFor makes the index current enough to resolve and forget refs, with
// a full pass only when it is not (4vd1mz): the native store changed since
// it was indexed, or a ref does not resolve yet (a harness memory written
// since). Otherwise it reads two stats and one row per ref.
func (a *App) FreshFor(ctx context.Context, refs []string) error {
	stale := false
	if path := NativePath(); path != "" {
		var mtime, size int64
		err := a.DB.QueryRow(`SELECT mtime_ns, size FROM sources WHERE kind='native' AND path=?`, path).Scan(&mtime, &size)
		st, serr := ingest.StatFile(path)
		switch {
		case serr != nil:
			// No store file yet: nothing native to be stale.
		case err != nil:
			stale = true
		default:
			stale = st.Mtime != mtime || st.Size != size
		}
	}
	for _, ref := range refs {
		if stale {
			break
		}
		if _, err := a.DB.ResolveAll(ref); err != nil {
			stale = true
		}
	}
	if !stale {
		return nil
	}
	_, err := a.Reindex(ctx)
	return err
}
