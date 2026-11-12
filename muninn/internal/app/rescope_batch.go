package app

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/veridit/hug/muninn/internal/config"
	"github.com/veridit/hug/muninn/internal/ingest"
)

// RescopeItem is one requested move in a batch.
type RescopeItem struct{ Ref, Target string }

// RescopeBatchResult reports what a batch did. Backup holds the native files
// as they were before the batch, so `cp` restores them.
type RescopeBatchResult struct {
	Moved  []RescopeItem
	Backup string
}

// RescopeBatch moves many owned memories in one pass (2026-11-09, user: the
// per-memory loop took ~2.6 s each). Every item gets the same checks as
// Rescope. All items are checked before anything is written, so one bad line
// leaves every file untouched. Each native file is written once and the index
// is rebuilt once. Capture is held off for the duration.
func (b Binding) RescopeBatch(ctx context.Context, a *App, items []RescopeItem, crossClient, dry bool) (RescopeBatchResult, error) {
	var result RescopeBatchResult
	b = b.Refresh(a)
	if b.Err != nil {
		return result, b.Err
	}
	if len(items) == 0 {
		return result, fmt.Errorf("no moves given")
	}
	type move struct {
		item RescopeItem
		key  string
		sid  string
	}
	byPath := map[string][]move{}
	var problems []error
	seen := map[string]bool{}
	for i, it := range items {
		line := fmt.Sprintf("item %d (%s)", i+1, it.Ref)
		if it.Target == "" || strings.ContainsAny(it.Target, "*!\r\n\t ") {
			problems = append(problems, fmt.Errorf("%s: target must be a literal scope", line))
			continue
		}
		u, err := a.DB.Resolve(it.Ref)
		if err != nil {
			problems = append(problems, fmt.Errorf("%s: no visible memory", line))
			continue
		}
		if seen[u.SID] {
			problems = append(problems, fmt.Errorf("%s: listed twice", line))
			continue
		}
		seen[u.SID] = true
		if clientOf(u.Scope) != clientOf(it.Target) && !crossClient {
			problems = append(problems, fmt.Errorf("%s: crossing a client boundary requires --cross-client", line))
			continue
		}
		if !crossClient {
			// b was refreshed once above, so check visibility directly
			// (Binding.Resolve refreshes per call, ~90 ms each on a live store).
			if _, ok := b.Present(u); !ok {
				problems = append(problems, fmt.Errorf("%s: no visible memory", line))
				continue
			}
			if _, err := b.WriteScope(it.Target); err != nil {
				problems = append(problems, fmt.Errorf("%s: %v", line, err))
				continue
			}
		}
		var kind, path string
		if err := a.DB.QueryRow(`SELECT kind,path FROM sources WHERE id=?`, u.SourceID).Scan(&kind, &path); err != nil {
			return result, err
		}
		if kind != "native" {
			problems = append(problems, fmt.Errorf("%s: rescope requires an owned native memory; import the upstream memory first", line))
			continue
		}
		byPath[path] = append(byPath[path], move{it, u.Key, u.SID})
	}
	if len(problems) > 0 {
		return result, fmt.Errorf("nothing moved, %d of %d items failed checks:\n%w", len(problems), len(items), errors.Join(problems...))
	}
	// Hold capture off before reading, so nothing lands between load and save.
	if !dry {
		release, err := a.tidyLease(ctx)
		if err != nil {
			return result, err
		}
		defer release()
	}
	// Plain remember/forget from other processes do not take the capture
	// lease, so also hold the store's write lock from load to save.
	locks := map[string]*ingest.NativeLock{}
	defer func() {
		for _, l := range locks {
			l.Release()
		}
	}()
	loaded := map[string][]ingest.NativeMemory{}
	for path, moves := range byPath {
		if !dry {
			l := &ingest.NativeLock{Path: path}
			if err := l.Hold(); err != nil {
				return result, err
			}
			locks[path] = l
		}
		mems, err := ingest.LoadNative(path)
		if err != nil {
			return result, err
		}
		at := map[string]int{}
		for i, m := range mems {
			at[m.ID] = i
		}
		now := time.Now().UTC()
		for _, mv := range moves {
			i, ok := at[mv.key]
			if !ok {
				return result, fmt.Errorf("nothing moved: %s is not in %s; run muninn index and retry", mv.sid, path)
			}
			mems[i].Scope = mv.item.Target
			mems[i].Updated = now
			result.Moved = append(result.Moved, RescopeItem{mv.sid, mv.item.Target})
		}
		loaded[path] = mems
	}
	if dry {
		return result, nil
	}
	base := filepath.Join(config.DataDir(), "tidy")
	if err := os.MkdirAll(base, 0700); err != nil {
		return RescopeBatchResult{}, err
	}
	var err error
	result.Backup, err = os.MkdirTemp(base, "rescope-"+time.Now().UTC().Format("20060102T150405Z")+"-")
	if err != nil {
		return RescopeBatchResult{}, err
	}
	n := 0
	for path := range loaded {
		raw, err := os.ReadFile(path)
		if err != nil {
			return RescopeBatchResult{}, err
		}
		n++
		name := fmt.Sprintf("%d-%s", n, filepath.Base(path))
		if err := os.WriteFile(filepath.Join(result.Backup, name), raw, 0600); err != nil {
			return RescopeBatchResult{}, err
		}
		if err := os.WriteFile(filepath.Join(result.Backup, name+".path"), []byte(path+"\n"), 0600); err != nil {
			return RescopeBatchResult{}, err
		}
	}
	for path, mems := range loaded {
		if err := ingest.SaveNative(path, mems); err != nil {
			return result, err
		}
	}
	for _, l := range locks {
		l.Release()
	}
	if _, err := a.Reindex(ctx); err != nil {
		return result, err
	}
	return result, nil
}

func clientOf(scope string) string {
	if !strings.HasPrefix(scope, "client:") {
		return ""
	}
	name, _, _ := strings.Cut(strings.TrimPrefix(scope, "client:"), ":")
	return name
}
