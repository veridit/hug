package ingest

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"sync/atomic"
	"time"

	"github.com/veridit/hug/internal/sharefile"
	"github.com/veridit/hug/muninn/internal/shortid"
	"github.com/veridit/hug/muninn/internal/trigger"
)

// NativeMemory is a memory muninn owns (created with `hug mem remember`).
type NativeMemory struct {
	Updates           string   `json:"updates,omitempty"`
	NewReason         string   `json:"new_reason,omitempty"`
	RetiredInto       string   `json:"retired_into,omitempty"`
	RetiredOrigin     string   `json:"retired_origin,omitempty"`
	CaptureSupersedes string   `json:"capture_supersedes,omitempty"`
	Trust             string   `json:"trust,omitempty"`
	CaptureEvents     []string `json:"capture_events,omitempty"`
	ID                string   `json:"id"`
	SID               string   `json:"sid"`
	Title             string   `json:"title,omitempty"`
	Content           string   `json:"content"`
	Category          string   `json:"category,omitempty"`
	// Type says how a memory is used: "rule" (an instruction), "preference",
	// "fact" (durable knowledge) or "record" (something that happened, or a
	// state at a point in time). Empty for memories written before 2026-11.
	// Records were relevant in 17% of judged recalls against about 45% for
	// the others (muninn/docs/RECALL4.md), so recall can treat them apart.
	Type     string    `json:"type,omitempty"`
	Tags     []string  `json:"tags,omitempty"`
	Scope    string    `json:"scope,omitempty"`
	Active   bool      `json:"active"`
	Strength int       `json:"strength"`
	Created  time.Time `json:"created"`
	Updated  time.Time `json:"updated"`
	// Origin identifies an imported memory ("engram:<sync_id>",
	// "jcode@<instance>:<project>/<id>"); OriginHash is the hash of the
	// upstream text at the last import. Both are empty for native memories.
	Origin     string `json:"origin,omitempty"`
	OriginHash string `json:"origin_hash,omitempty"`
	// SafeReason records why the writer judged text that looked secret to be
	// safe to store (2026-11-06: judge on write, never hide on display).
	SafeReason string `json:"safe_reason,omitempty"`
	// Branch is the git branch the memory was remembered or captured on, set
	// when that is a linked worktree or a branch other than the main one
	// (x1t1t1). A merged or deleted branch makes the memory a dream item.
	Branch string `json:"branch,omitempty"`
	// Holds is the condition the memory holds under, stated at write time
	// (ez50rb): "until 2026-12-01", "while task:ID open", "while branch
	// NAME unmerged" (muninn/internal/holds). When it ends, recall stops
	// showing the memory and a dream item asks an agent to retire or keep
	// it; nothing is deleted automatically. Empty: holds until judged
	// otherwise.
	Holds string `json:"holds,omitempty"`
	// Trigger is the situation the memory is for, stated by its storer
	// (jrj1vq): case (scopes, paths, tools), when and when_not. The memory
	// text is the note shown (the THEN). Nil for a plain memory, whose JSON
	// then has no trigger key, as before. Kept canonical (trigger.Canonical).
	Trigger *trigger.Trigger `json:"trigger,omitempty"`
	// Repo is the main checkout of the git repository the memory was written
	// in, recorded only with Holds or Branch: where the dream pass looks up
	// the task board and the branch. Without it a task or branch condition
	// is unknown, and unknown never ends a memory.
	Repo string `json:"repo,omitempty"`
}

type nativeFile struct {
	Version  int            `json:"version"`
	Memories []NativeMemory `json:"memories"`
}

// Loads counts LoadNative calls in this process, so tests can count the
// reads a command makes (4vd1mz).
var Loads atomic.Int64

// LoadNative reads the native store; a missing file is empty. It opens the
// file so a save may rename over it meanwhile (Windows, zd3x24).
func LoadNative(path string) ([]NativeMemory, error) {
	Loads.Add(1)
	b, err := sharefile.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var f nativeFile
	if err := json.Unmarshal(b, &f); err != nil {
		return nil, err
	}
	assignNativeSIDs(f.Memories)
	seen := map[string]NativeMemory{}
	out := make([]NativeMemory, 0, len(f.Memories))
	duplicates, rekeyed := 0, 0
	reserved := map[string]bool{}
	for _, m := range f.Memories {
		reserved[m.ID] = true
	}
	for _, m := range f.Memories {
		if old, exists := seen[m.ID]; exists {
			// Retries of the old append path could differ only in write time.
			// Preserve the first row, but never collapse conflicting memories.
			compare := old
			compare.Created, compare.Updated, compare.SID = m.Created, m.Updated, m.SID
			if reflect.DeepEqual(compare, m) {
				duplicates++
				continue
			}
			// Legacy capture IDs identified an output position, not its content.
			// A regenerated response could append a different memory at that ID.
			// Keep every payload field and retain the first row's public identity.
			if !strings.HasPrefix(m.ID, "cap_") || !strings.HasPrefix(m.Origin, "capture:") || !strings.HasPrefix(old.Origin, "capture:") {
				return nil, errors.New("native store has conflicting duplicate ids, refusing data loss")
			}
			raw, err := json.Marshal(m)
			if err != nil {
				return nil, err
			}
			for salt := 0; ; salt++ {
				sum := sha256.Sum256(append(append([]byte(nil), raw...), []byte(fmt.Sprintf("\x00%d", salt))...))
				id := "cap_repair_" + hex.EncodeToString(sum[:])
				if !reserved[id] {
					m.ID, m.SID = id, ""
					reserved[id] = true
					break
				}
			}
			rekeyed++
		}
		seen[m.ID] = m
		out = append(out, m)
	}
	if duplicates > 0 || rekeyed > 0 {
		if err := SaveNative(path, out); err != nil {
			return nil, fmt.Errorf("repair native duplicates: %w", err)
		}
		// Count only, no IDs or memory text. Never print into an agent hook.
		log, err := os.OpenFile(filepath.Join(filepath.Dir(path), "native-repairs.log"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
		if err != nil {
			return nil, err
		}
		_, writeErr := fmt.Fprintf(log, "%s native duplicate repair removed=%d rekeyed=%d\n", time.Now().UTC().Format(time.RFC3339), duplicates, rekeyed)
		closeErr := log.Close()
		if writeErr != nil {
			return nil, writeErr
		}
		if closeErr != nil {
			return nil, closeErr
		}
	}
	return out, nil
}

// SaveNative writes the store atomically (temp file + rename).
func SaveNative(path string, mems []NativeMemory) error {
	ids := map[string]bool{}
	for _, m := range mems {
		if ids[m.ID] {
			return errors.New("refusing to save duplicate native memory ids")
		}
		ids[m.ID] = true
	}
	assignNativeSIDs(mems)
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	sort.Slice(mems, func(i, j int) bool { return mems[i].Created.Before(mems[j].Created) })
	b, err := json.MarshalIndent(nativeFile{Version: 1, Memories: mems}, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, append(b, '\n'), 0o600); err != nil {
		return err
	}
	// A POSIX replace: it succeeds while readers have the store open (zd3x24).
	return sharefile.Rename(tmp, path)
}

func assignNativeSIDs(mems []NativeMemory) {
	used := map[string]string{}
	for i := range mems {
		m := &mems[i]
		full := shortid.Full(shortid.Identity(m.ID, m.Origin))
		if len(m.SID) < 6 || !strings.HasPrefix(full, m.SID) {
			m.SID = ""
		}
		if m.SID != "" {
			used[m.SID] = shortid.Identity(m.ID, m.Origin)
		}
	}
	for i := range mems {
		m := &mems[i]
		if m.SID != "" {
			continue
		}
		identity := shortid.Identity(m.ID, m.Origin)
		full := shortid.Full(identity)
		for n := 6; n <= len(full); n++ {
			sid := full[:n]
			if other, ok := used[sid]; !ok || other == identity {
				m.SID = sid
				used[sid] = identity
				break
			}
		}
	}
}

func newID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return "mem_" + hex.EncodeToString(b)
}

// Remember adds a memory. Exact-duplicate content in the same category and
// scope reinforces the existing memory instead of creating a copy.
func Remember(path, content, category, scope string, tags []string) (NativeMemory, bool, error) {
	return RememberSafe(path, content, category, scope, tags, "")
}

// RememberSafe is Remember with the writer's reason for storing text that
// looked secret. The reason is kept on the memory for later review.
func RememberSafe(path, content, category, scope string, tags []string, safeReason string, newReasons ...string) (NativeMemory, bool, error) {
	return RememberTyped(path, content, category, "", scope, tags, safeReason, newReasons...)
}

// RememberTyped is RememberSafe with the memory type (rule, preference, fact,
// record; "" = unstated). Reinforcing a memory sets its type when it had none.
func RememberTyped(path, content, category, typ, scope string, tags []string, safeReason string, newReasons ...string) (NativeMemory, bool, error) {
	var reason string
	if len(newReasons) > 0 {
		reason = newReasons[0]
	}
	return RememberWith(path, Remembering{Content: content, Category: category, Type: typ, Scope: scope, Tags: tags, SafeReason: safeReason, NewReason: reason})
}

// Remembering is one memory to store with RememberWith.
type Remembering struct {
	Content, Category, Type, Scope string
	Tags                           []string
	SafeReason, NewReason          string
	Branch                         string           // see NativeMemory.Branch
	Holds                          string           // see NativeMemory.Holds (canonical form)
	Repo                           string           // see NativeMemory.Repo
	Trigger                        *trigger.Trigger // see NativeMemory.Trigger
	// OriginFor, when set, gives a new memory its origin from its id: the
	// repo origin a project sync would give it later, so its short id is
	// final when remember prints it (tn8bmn).
	OriginFor func(id string) string
}

// RememberWith stores r, or reinforces an active memory with the same text,
// category and scope. Reinforcing keeps the first branch it was seen on.
func RememberWith(path string, r Remembering) (NativeMemory, bool, error) {
	m, reinforced, _, err := RememberWritten(path, r)
	return m, reinforced, err
}

// RememberWritten is RememberWith, reporting what the write changed.
func RememberWritten(path string, r Remembering) (NativeMemory, bool, Written, error) {
	content, category, typ, scope, tags, safeReason := r.Content, r.Category, r.Type, r.Scope, r.Tags, r.SafeReason
	content = strings.TrimSpace(content)
	if content == "" {
		return NativeMemory{}, false, Written{}, errors.New("empty memory")
	}
	tr, err := trigger.Canonical(r.Trigger)
	if err != nil {
		return NativeMemory{}, false, Written{}, err
	}
	if scope == "" {
		scope = "global"
	}
	var got NativeMemory
	reinforced := false
	w, err := UpdateNativeChanged(path, func(mems []NativeMemory) ([]NativeMemory, bool, error) {
		now := time.Now().UTC()
		for i := range mems {
			m := &mems[i]
			if m.Active && m.Content == content && m.Category == category && m.Scope == scope {
				// A different trigger is a correction, not the same memory again
				// (jrj1vq, crab): refused, nothing written. None keeps the old;
				// a memory without one (plain, or cleared) takes the new one.
				if tr != nil && !trigger.Empty(m.Trigger) && !trigger.Equal(tr, m.Trigger) {
					got = *m
					return mems, false, &TriggerChangeError{ID: m.ID, SID: m.SID, Old: m.Trigger, New: tr}
				}
				if tr != nil && trigger.Empty(m.Trigger) {
					m.Trigger = tr
				}
				if safeReason != "" {
					m.SafeReason = safeReason
				}
				if m.Type == "" {
					m.Type = typ
				}
				if m.Branch == "" {
					m.Branch = r.Branch
					if m.Repo == "" && m.Branch != "" {
						m.Repo = r.Repo
					}
				}
				// Reinforcing never changes a memory's condition: one that held
				// without one keeps holding, one with a condition keeps it.
				m.Strength++
				m.Updated = now
				got, reinforced = *m, true
				return mems, true, nil
			}
		}
		m := NativeMemory{ID: newID(), Content: content, Category: category, Type: typ, Tags: tags, Scope: scope,
			Active: true, Strength: 1, Created: now, Updated: now, SafeReason: strings.TrimSpace(safeReason),
			NewReason: strings.TrimSpace(r.NewReason), Branch: strings.TrimSpace(r.Branch), Holds: strings.TrimSpace(r.Holds), Trigger: tr}
		if m.Branch != "" || m.Holds != "" {
			m.Repo = strings.TrimSpace(r.Repo)
		}
		if r.OriginFor != nil {
			m.Origin = r.OriginFor(m.ID)
		}
		mems = append(mems, m)
		assignNativeSIDs(mems)
		got = mems[len(mems)-1]
		return mems, true, nil
	})
	if err == nil {
		// The saved row: SaveNative's ids are final.
		for _, m := range w.Changed {
			if m.ID == got.ID {
				got = m
			}
		}
	}
	return got, reinforced, w, err
}

// TriggerChangeError refuses a reinforce whose trigger differs from the
// memory's (jrj1vq, crab): a changed trigger is a correction, stated with
// supersedes. Nothing was written.
type TriggerChangeError struct {
	ID, SID  string
	Old, New *trigger.Trigger
}

func (e *TriggerChangeError) Error() string {
	ref := e.SID
	if ref == "" {
		ref = e.ID
	}
	return fmt.Sprintf("NOT STORED: memory %s has this text with another trigger (%s), not %s. A changed trigger is a correction: say which memory it corrects.\nfix: hug mem remember --supersedes %s --when ... \"TEXT\" (MCP supersedes), or --supersedes %s --no-trigger to clear it",
		ref, trigger.Encode(e.Old), trigger.Encode(e.New), ref, ref)
}

// ReplaceTrigger sets the trigger of the active memory with this id (or
// sid) in place, under the write lock: a supersede whose text is the same
// (jrj1vq). tr nil or empty stores None, a deliberate clear that sync
// carries to other clones. It reports whether the memory was found.
func ReplaceTrigger(path, ref string, tr *trigger.Trigger) (NativeMemory, bool, error) {
	m, found, _, err := ReplaceTriggerWritten(path, ref, tr)
	return m, found, err
}

// ReplaceTriggerWritten is ReplaceTrigger, reporting what the write changed.
func ReplaceTriggerWritten(path, ref string, tr *trigger.Trigger) (NativeMemory, bool, Written, error) {
	stored, err := trigger.Stored(tr)
	if err != nil {
		return NativeMemory{}, false, Written{}, err
	}
	if stored == nil {
		stored = trigger.None()
	}
	var got NativeMemory
	found := false
	w, err := UpdateNativeChanged(path, func(mems []NativeMemory) ([]NativeMemory, bool, error) {
		for i := range mems {
			m := &mems[i]
			if !m.Active || (m.ID != ref && m.SID != ref) {
				continue
			}
			found = true
			if trigger.Encode(m.Trigger) == trigger.Encode(stored) && trigger.Cleared(m.Trigger) == trigger.Cleared(stored) {
				got = *m
				return mems, false, nil
			}
			m.Trigger, m.Updated = stored, time.Now().UTC()
			got = *m
			return mems, true, nil
		}
		return mems, false, nil
	})
	return got, found, w, err
}

// Forget deactivates a memory by id (kept on disk, excluded from recall).
func Forget(path, id string) (bool, error) {
	done, err := ForgetMany(path, []string{id})
	return done[id], err
}

// ForgetMany deactivates every listed memory (by id or sid) with one load and
// one save under the write lock. The result marks the ids that were active.
func ForgetMany(path string, ids []string) (map[string]bool, error) {
	f, err := ForgetManyChanged(path, ids)
	return f.Done, err
}

// FileStat is a file's (mtime ns, size), as the indexer records per source.
type FileStat struct{ Mtime, Size int64 }

// StatFile returns path's FileStat.
func StatFile(path string) (FileStat, error) {
	fi, err := os.Stat(path)
	if err != nil {
		return FileStat{}, err
	}
	return FileStat{fi.ModTime().UnixNano(), fi.Size()}, nil
}

// Forgotten is what one ForgetManyChanged call did to the store file.
type Forgotten struct {
	Done map[string]bool // listed ids that were active
	// Changed holds every memory the call deactivated, twins included, as
	// saved: what an index needs to update, and nothing else changed.
	Changed []NativeMemory
	// Before and After are the file's stat just before the load and just
	// after the save, both under the lock: if an index recorded Before, the
	// file at After differs from what it indexed by Changed only.
	Before, After FileStat
}

// ForgetManyChanged is ForgetMany that also reports what it changed (4vd1mz).
func ForgetManyChanged(path string, ids []string) (Forgotten, error) {
	f := Forgotten{Done: map[string]bool{}}
	done := f.Done
	unlock, err := LockNative(path)
	if err != nil {
		return f, err
	}
	defer unlock()
	if f.Before, err = StatFile(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return f, err
	}
	mems, err := LoadNative(path)
	if err != nil {
		return f, err
	}
	if after, err := StatFile(path); err == nil && after != f.Before {
		// LoadNative repaired duplicates and saved: the file is no longer
		// the one an index may have recorded. Make Before say so.
		f.Before = FileStat{-1, -1}
	}
	want := map[string]bool{}
	for _, id := range ids {
		want[id] = true
	}
	now := time.Now().UTC()
	retired := map[int]bool{}
	for i := range mems {
		m := &mems[i]
		if !m.Active {
			continue
		}
		for _, id := range []string{m.ID, m.SID} {
			if id != "" && want[id] {
				m.Active = false
				m.Updated = now
				done[id] = true
				retired[i] = true
			}
		}
	}
	if len(done) == 0 {
		f.After = f.Before
		return f, nil
	}
	wasActive := make(map[string]bool, len(mems))
	for i, m := range mems {
		if m.Active || retired[i] {
			wasActive[m.ID] = true
		}
	}
	// A twin a sync made (5sc5rw) is the same memory: it goes too, or
	// recall shows it in the forgotten one's place.
	retireTwins(mems, retired, now)
	if err := SaveNative(path, mems); err != nil {
		return f, err
	}
	for _, m := range mems {
		if wasActive[m.ID] && !m.Active {
			f.Changed = append(f.Changed, m)
		}
	}
	f.After, err = StatFile(path)
	return f, err
}
