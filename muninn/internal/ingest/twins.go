package ingest

import (
	"regexp"
	"time"
)

// Twins (5sc5rw). Before the fix in MergeImport, a project sync read each of
// this store's own memories back from the notes file under the origin
// repo:PROJECT/ID and, finding no memory with that origin, added a copy:
// the same text, scope and state under a new imp_ id. Every capture in a
// repository project had such a twin (2651 pairs on one machine on
// 2026-11-17), so forgetting one left the other in recall.

var repoOrigin = regexp.MustCompile(`^repo:(?:client:[^:/]+:)?[^/]+/(.+)$`)

// twinOf reports the memory a repo-origin copy duplicates: one in this store
// whose id is the copy's line id, in the same scope.
func twinOf(m NativeMemory, byID map[string]int, mems []NativeMemory) (int, bool) {
	r := repoOrigin.FindStringSubmatch(m.Origin)
	if r == nil || r[1] == m.ID {
		return 0, false
	}
	j, ok := byID[r[1]]
	if !ok || mems[j].Scope != m.Scope {
		return 0, false
	}
	return j, true
}

// FoldStats counts one fold.
type FoldStats struct {
	// Folded twins were removed; Retired of them carried a newer forget
	// that their memory now has.
	Folded, Retired int
	// Scopes counts folded twins per scope.
	Scopes map[string]int
}

// FoldTwins removes every twin a sync made of this store's own memories and
// keeps the memory itself: its id, short id, capture fields and origin stay.
// When the twin changed after the memory (a forget or an edit through it),
// the newer state and text move to the memory. With dry, nothing is written.
func FoldTwins(path string, dry bool) (FoldStats, error) {
	st := FoldStats{Scopes: map[string]int{}}
	unlock, err := LockNative(path)
	if err != nil {
		return st, err
	}
	defer unlock()
	mems, err := LoadNative(path)
	if err != nil {
		return st, err
	}
	byID := make(map[string]int, len(mems))
	for i, m := range mems {
		byID[m.ID] = i
	}
	drop := make([]bool, len(mems))
	for i, m := range mems {
		j, ok := twinOf(m, byID, mems)
		if !ok || drop[j] {
			continue
		}
		keep := &mems[j]
		if m.Updated.After(keep.Updated) {
			if keep.Active && !m.Active {
				st.Retired++
			}
			keep.Title, keep.Content, keep.Category, keep.Tags = m.Title, m.Content, m.Category, m.Tags
			keep.Active, keep.Updated = m.Active, m.Updated
		}
		if m.Strength > keep.Strength {
			keep.Strength = m.Strength
		}
		drop[i] = true
		st.Folded++
		st.Scopes[m.Scope]++
	}
	if dry || st.Folded == 0 {
		return st, nil
	}
	out := mems[:0]
	for i, m := range mems {
		if !drop[i] {
			out = append(out, m)
		}
	}
	return st, SaveNative(path, out)
}

// retireTwins retires each twin of a retired memory, and the memory of each
// retired twin, as forgetting either must (an unfolded store holds both).
func retireTwins(mems []NativeMemory, retired map[int]bool, now time.Time) {
	byID := make(map[string]int, len(mems))
	for i, m := range mems {
		byID[m.ID] = i
	}
	for i := range mems {
		j, ok := twinOf(mems[i], byID, mems)
		if !ok {
			continue
		}
		for _, k := range [][2]int{{j, i}, {i, j}} {
			if retired[k[0]] && mems[k[1]].Active {
				mems[k[1]].Active, mems[k[1]].Updated = false, now
			}
		}
	}
}
