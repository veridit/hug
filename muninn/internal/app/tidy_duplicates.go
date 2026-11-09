package app

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/veridit/muninn/internal/config"
	"github.com/veridit/muninn/internal/ingest"
	"github.com/veridit/muninn/internal/store"
	"github.com/veridit/muninn/internal/tidy"
)

type TidyRetirement struct {
	SID, Origin, Keeper string
	Strength            int
}
type tidyFile struct {
	Path    string
	Data    []byte
	Existed bool
}
type duplicateReceipt struct {
	Kind        string
	Database    string
	Complete    bool
	Files       []tidyFile
	After       map[string]string
	Suppressed  []string
	Retirements []TidyRetirement
}

func rawHash(raw []byte) string { h := sha256.Sum256(raw); return hex.EncodeToString(h[:]) }
func savePrivateJSON(path string, v any) error {
	raw, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(raw, '\n'), 0600)
}
func restoreTidyFiles(files []tidyFile) error {
	var err error
	for _, f := range files {
		if f.Existed {
			err = errors.Join(err, os.WriteFile(f.Path, f.Data, 0600))
		} else {
			e := os.Remove(f.Path)
			if !errors.Is(e, os.ErrNotExist) {
				err = errors.Join(err, e)
			}
		}
	}
	return err
}

func (a *App) applyDuplicates(ctx context.Context, p tidy.Plan, dry bool) (result TidyResult, err error) {
	if p.Database != "" && filepath.Clean(p.Database) != filepath.Clean(a.DB.Path) {
		return result, fmt.Errorf("plan belongs to a different database")
	}
	if !dry {
		release, e := a.tidyLease(ctx)
		if e != nil {
			return result, e
		}
		defer release()
	}
	graph, err := tidy.ReadGraph(ctx, a.DB.Path)
	if err != nil {
		return result, err
	}
	clusters, err := tidy.SelectedDuplicates(p, graph)
	if err != nil || len(clusters) == 0 {
		return result, err
	}
	units := map[string]store.Unit{}
	for _, u := range graph.Units {
		units[u.SID] = u
	}
	files := map[string][]ingest.NativeMemory{}
	var originals []tidyFile
	load := func(path string) error {
		if _, ok := files[path]; ok {
			return nil
		}
		raw, e := os.ReadFile(path)
		if e != nil && !errors.Is(e, os.ErrNotExist) {
			return e
		}
		existed := e == nil
		var f struct{ Memories []ingest.NativeMemory }
		if existed {
			if e = json.Unmarshal(raw, &f); e != nil {
				return e
			}
		}
		ids := map[string]bool{}
		for _, m := range f.Memories {
			if ids[m.ID] {
				return fmt.Errorf("duplicate native ids: repair before tidy")
			}
			ids[m.ID] = true
		}
		files[path] = f.Memories
		originals = append(originals, tidyFile{path, raw, existed})
		return nil
	}
	if err = load(NativePath()); err != nil {
		return result, err
	}
	type location struct {
		path  string
		index int
	}
	locations := map[string][]location{}
	// Validate every native mirror against its own on-disk bytes before changing
	// any file. Imported sources are never written, only locally suppressed.
	for _, c := range clusters {
		for _, m := range append([]tidy.DuplicateMember{c.Keeper}, c.Retire...) {
			mirrors, e := a.DB.ResolveAll(m.SID)
			if e != nil {
				return result, e
			}
			for _, u := range mirrors {
				var kind, path string
				if e = a.DB.QueryRowContext(ctx, `SELECT kind,path FROM sources WHERE id=?`, u.SourceID).Scan(&kind, &path); e != nil {
					return result, e
				}
				if kind != "native" {
					continue
				}
				if err = load(path); err != nil {
					return result, err
				}
				found := false
				for i, n := range files[path] {
					if n.ID != u.Key {
						continue
					}
					if !n.Active || n.Content != u.Body || n.Title != u.Title || n.Scope != u.Scope || n.Category != u.Category || strings.Join(n.Tags, " ") != strings.Join(u.Tags, " ") {
						return result, fmt.Errorf("native file changed for %s: reindex and rescan", m.SID)
					}
					found = true
					locations[m.SID] = append(locations[m.SID], location{path, i})
				}
				if !found {
					return result, fmt.Errorf("native file changed for %s: reindex and rescan", m.SID)
				}
			}
			// A native mirror that has not been indexed cannot be silently overwritten.
			if len(locations[m.SID]) == 0 {
				for _, n := range files[NativePath()] {
					if n.SID == m.SID || n.Origin == m.Origin {
						return result, fmt.Errorf("unindexed native copy of %s: reindex and rescan", m.SID)
					}
				}
			}
		}
	}
	for _, c := range clusters {
		for _, m := range c.Retire {
			result.Retirements = append(result.Retirements, TidyRetirement{m.SID, m.Origin, c.Keeper.SID, m.Strength})
		}
	}
	if dry {
		return result, nil
	}
	base := filepath.Join(config.DataDir(), "tidy")
	if err = os.MkdirAll(base, 0700); err != nil {
		return result, err
	}
	result.Backup, err = os.MkdirTemp(base, "apply-"+time.Now().UTC().Format("20060102T150405Z")+"-")
	if err != nil {
		return result, err
	}
	receipt := duplicateReceipt{Kind: "dupes", Database: a.DB.Path, Files: originals, After: map[string]string{}, Retirements: result.Retirements}
	if err = savePrivateJSON(filepath.Join(result.Backup, "files.json"), originals); err != nil {
		return result, err
	}
	if err = savePrivateJSON(filepath.Join(result.Backup, "plan.json"), p); err != nil {
		return result, err
	}
	for _, f := range originals {
		if f.Path == NativePath() {
			if err = os.WriteFile(filepath.Join(result.Backup, "memories.json"), f.Data, 0600); err != nil {
				return result, err
			}
		}
	}
	if err = savePrivateJSON(filepath.Join(result.Backup, "receipt.json"), receipt); err != nil {
		return result, err
	}
	defer func() {
		if err == nil {
			return
		}
		err = errors.Join(err, restoreTidyFiles(originals))
		for _, sid := range receipt.Suppressed {
			_, e := a.DB.Exec(`DELETE FROM suppressed WHERE sid=?`, sid)
			err = errors.Join(err, e)
		}
		_, e := a.Reindex(context.Background())
		err = errors.Join(err, e)
		result.Retirements = nil
	}()
	copyMemory := func(u store.Unit, prefix string, active bool) ingest.NativeMemory {
		id := prefix + rawHash([]byte(u.Origin))[:24]
		origin := u.Origin
		if !active {
			origin = "tidy:dupes:" + u.Origin
		}
		return ingest.NativeMemory{ID: id, SID: u.SID, Origin: origin, Content: u.Body, Title: u.Title, Category: u.Category, Tags: slices.Clone(u.Tags), Scope: u.Scope, Active: active, Created: u.Updated, Updated: u.Updated, Strength: 1}
	}
	for _, c := range clusters {
		total := c.Keeper.Strength
		for _, m := range c.Retire {
			total += m.Strength
		}
		if len(locations[c.Keeper.SID]) == 0 {
			n := copyMemory(units[c.Keeper.SID], "tidy_keeper_", true)
			n.Strength = total
			files[NativePath()] = append(files[NativePath()], n)
		} else {
			for _, loc := range locations[c.Keeper.SID] {
				n := &files[loc.path][loc.index]
				n.Strength = total
			}
		}
		for _, m := range c.Retire {
			if len(locations[m.SID]) == 0 {
				n := copyMemory(units[m.SID], "tidy_retired_", false)
				n.Strength = m.Strength
				n.RetiredInto = c.Keeper.SID
				n.RetiredOrigin = m.Origin
				files[NativePath()] = append(files[NativePath()], n)
			} else {
				for _, loc := range locations[m.SID] {
					n := &files[loc.path][loc.index]
					n.Active = false
					n.RetiredInto = c.Keeper.SID
					n.RetiredOrigin = m.Origin
				}
			}
			// Persist suppression intent BEFORE mutation for crash recovery.
			receipt.Suppressed = append(receipt.Suppressed, m.SID)
		}
	}
	if err = savePrivateJSON(filepath.Join(result.Backup, "receipt.json"), receipt); err != nil {
		return result, err
	}
	for path, mems := range files {
		if err = ingest.SaveNative(path, mems); err != nil {
			return result, err
		}
		raw, e := os.ReadFile(path)
		if e != nil {
			return result, e
		}
		receipt.After[path] = rawHash(raw)
	}
	tx, err := a.DB.BeginTx(ctx, nil)
	if err != nil {
		return result, err
	}
	defer tx.Rollback()
	for _, sid := range receipt.Suppressed {
		if _, err = tx.ExecContext(ctx, `INSERT INTO suppressed(sid) VALUES(?)`, sid); err != nil {
			return result, err
		}
	}
	if err = tx.Commit(); err != nil {
		return result, err
	}
	if _, err = a.Reindex(ctx); err != nil {
		return result, err
	}
	// Reindex can extend a new archive SID on a collision. Hash the final bytes,
	// not the pre-index file, so a normal collision does not invalidate undo.
	for path := range files {
		raw, e := os.ReadFile(path)
		if e != nil {
			return result, e
		}
		receipt.After[path] = rawHash(raw)
	}
	receipt.Complete = true
	err = savePrivateJSON(filepath.Join(result.Backup, "receipt.json"), receipt)
	return result, err
}

// UndoTidy restores a completed duplicate batch only when every affected file
// still matches its post-apply hash. It never overwrites later remembered notes.
func (a *App) UndoTidy(ctx context.Context, backup string) (err error) {
	release, err := a.tidyLease(ctx)
	if err != nil {
		return err
	}
	defer release()
	raw, err := os.ReadFile(filepath.Join(backup, "receipt.json"))
	if err != nil {
		return err
	}
	var r duplicateReceipt
	if err = json.Unmarshal(raw, &r); err != nil {
		return err
	}
	if r.Kind != "dupes" || !r.Complete || filepath.Clean(r.Database) != filepath.Clean(a.DB.Path) || len(r.Files) == 0 {
		return fmt.Errorf("not a completed duplicate receipt for this database")
	}
	var applied []tidyFile
	for _, f := range r.Files {
		var kind string
		if err = a.DB.QueryRowContext(ctx, `SELECT kind FROM sources WHERE path=?`, f.Path).Scan(&kind); err != nil || kind != "native" {
			return fmt.Errorf("undo file is not an owned native source")
		}
		current, e := os.ReadFile(f.Path)
		if e != nil {
			return e
		}
		if r.After[f.Path] == "" || rawHash(current) != r.After[f.Path] {
			return fmt.Errorf("memories changed since apply: refusing to overwrite %s", f.Path)
		}
		applied = append(applied, tidyFile{f.Path, current, true})
	}
	tx, err := a.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for _, sid := range r.Suppressed {
		if _, err = tx.ExecContext(ctx, `DELETE FROM suppressed WHERE sid=?`, sid); err != nil {
			return err
		}
	}
	if err = restoreTidyFiles(r.Files); err != nil {
		_ = restoreTidyFiles(applied)
		return err
	}
	if err = tx.Commit(); err != nil {
		_ = restoreTidyFiles(applied)
		return err
	}
	if _, err = a.Reindex(ctx); err != nil {
		return err
	}
	r.Complete = false
	return savePrivateJSON(filepath.Join(backup, "receipt.json"), r)
}
