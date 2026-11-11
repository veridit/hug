package app

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/veridit/muninn/internal/config"
	"github.com/veridit/muninn/internal/ingest"
	"github.com/veridit/muninn/internal/project"
	"github.com/veridit/muninn/internal/store"
	"github.com/veridit/muninn/internal/tidy"
)

// Correct is an owner action, not a capture/hook path. It appends a new dated
// correction and retires the previous identity without changing its text or
// dates. A private byte-exact backup precedes every write. Upstream files are
// never edited. The capture lease prevents concurrent owner/capture mutations.
func (b Binding) Correct(ctx context.Context, a *App, ref, text string) (out store.Unit, backup string, err error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return out, "", fmt.Errorf("empty correction")
	}
	b = b.Refresh(a)
	u, err := b.Resolve(a, ref)
	if err != nil {
		return out, "", err
	}
	if !u.Active {
		return out, "", fmt.Errorf("memory is retired")
	}
	if _, err = b.WriteScope(u.Scope); err != nil {
		return out, "", err
	}
	if hint := project.SecretHint(text); hint != "" {
		return out, "", &SecretCheckError{Hint: hint}
	}
	release, err := a.tidyLease(ctx)
	if err != nil {
		return out, "", err
	}
	defer release()
	// The identity may have changed while the capture lease was being acquired.
	// Never correct a replacement or a newly hidden memory using stale consent.
	fresh, err := b.Refresh(a).Resolve(a, u.SID)
	if err != nil {
		return out, "", err
	}
	if tidy.Fingerprint(fresh) != tidy.Fingerprint(u) || !fresh.Updated.Equal(u.Updated) {
		return out, "", fmt.Errorf("memory changed while acquiring the correction lease")
	}
	u = fresh
	units, err := a.DB.ResolveAll(u.SID)
	if err != nil {
		return out, "", err
	}
	files := map[string][]ingest.NativeMemory{}
	var originals []tidyFile
	locks := nativeLocks{}
	defer locks.release()
	load := func(path string) error {
		if _, ok := files[path]; ok {
			return nil
		}
		if e := locks.hold(path); e != nil {
			return e
		}
		raw, e := os.ReadFile(path)
		if e != nil && !errors.Is(e, os.ErrNotExist) {
			return e
		}
		var f struct{ Memories []ingest.NativeMemory }
		if e == nil {
			if e = json.Unmarshal(raw, &f); e != nil {
				return e
			}
		}
		seen := map[string]bool{}
		for _, m := range f.Memories {
			if seen[m.ID] {
				return fmt.Errorf("duplicate native ids: repair before correcting")
			}
			seen[m.ID] = true
		}
		files[path] = f.Memories
		originals = append(originals, tidyFile{path, raw, e == nil})
		return nil
	}
	if err = load(NativePath()); err != nil {
		return out, "", err
	}
	for _, v := range units {
		var kind, path string
		if err = a.DB.QueryRow(`SELECT kind,path FROM sources WHERE id=?`, v.SourceID).Scan(&kind, &path); err != nil {
			return out, "", err
		}
		if kind != "native" {
			continue
		}
		shown, ok := b.Present(v)
		if !ok {
			return out, "", fmt.Errorf("no visible memory")
		}
		if _, err = b.WriteScope(shown.Scope); err != nil {
			return out, "", err
		}
		if err = load(path); err != nil {
			return out, "", err
		}
		found := false
		for i, m := range files[path] {
			if m.ID == v.Key {
				if !m.Active || m.Content != v.Body || m.Category != v.Category || m.Scope != v.Scope {
					return out, "", fmt.Errorf("native memory changed: reindex before correcting")
				}
				m.Active = false
				files[path][i] = m
				found = true
			}
		}
		if !found {
			return out, "", fmt.Errorf("native memory missing: reindex before correcting")
		}
	}
	random := make([]byte, 16)
	if _, err = rand.Read(random); err != nil {
		return out, "", err
	}
	now := time.Now().UTC()
	id := "cor_" + hex.EncodeToString(random)
	files[NativePath()] = append(files[NativePath()], ingest.NativeMemory{ID: id, Content: text, Category: "correction", Scope: u.Scope, Active: true, Strength: 1, Created: now, Updated: now, CaptureSupersedes: u.SID, Trust: "high"})
	root := filepath.Join(config.DataDir(), "tidy")
	if err = os.MkdirAll(root, 0700); err != nil {
		return out, "", err
	}
	backup, err = os.MkdirTemp(root, "correction-")
	if err != nil {
		return out, "", err
	}
	if err = savePrivateJSON(filepath.Join(backup, "files.json"), originals); err != nil {
		return out, backup, err
	}
	wasSuppressed := a.DB.IsSuppressed(u.SID)
	defer func() {
		if err != nil {
			err = errors.Join(err, locks.restore(originals))
			if !wasSuppressed {
				_, e := a.DB.Exec(`DELETE FROM suppressed WHERE sid=?`, u.SID)
				err = errors.Join(err, e)
			}
			_, e := a.Reindex(context.Background())
			err = errors.Join(err, e)
		}
	}()
	for path, mems := range files {
		if err = ingest.SaveNative(path, mems); err != nil {
			return out, backup, err
		}
	}
	locks.release()
	if err = a.DB.Suppress(u.SID); err != nil {
		return out, backup, err
	}
	if _, err = a.Reindex(ctx); err != nil {
		return out, backup, err
	}
	out, err = b.Resolve(a, id)
	return out, backup, err
}
