package project

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/veridit/hug/muninn/internal/ingest"
)

// captured writes a memory as capture does: its own id and a capture origin,
// not a repo origin (5sc5rw).
func captured(t *testing.T, path, scope, id, text string) {
	t.Helper()
	mems, err := ingest.LoadNative(path)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	mems = append(mems, ingest.NativeMemory{ID: id, Content: text, Category: "fact", Scope: scope, Active: true, Strength: 1,
		Created: now, Updated: now, Origin: "capture:jcode:session_x_1_y:abc:0", Trust: "high"})
	if err := ingest.SaveNative(path, mems); err != nil {
		t.Fatal(err)
	}
}

func active(t *testing.T, path, scope string) []ingest.NativeMemory {
	t.Helper()
	mems, err := ingest.LoadNative(path)
	if err != nil {
		t.Fatal(err)
	}
	var out []ingest.NativeMemory
	for _, m := range mems {
		if m.Scope == scope && m.Active {
			out = append(out, m)
		}
	}
	return out
}

// 5sc5rw check 1: a captured memory synced to the notes file and read back
// stays one memory. On 2026-11-17 every capture in a repo project also
// existed as an imp_ row (2651 pairs on one machine): the file's line id is
// the capture's own id, but the import looked it up by repo origin only.
func TestSyncReadsOwnCaptureBackAsTheSameMemory(t *testing.T) {
	repo := t.TempDir()
	p := Project{ID: "rc-demo", Label: "demo", Root: repo}
	store := filepath.Join(t.TempDir(), "memories.json")
	captured(t, store, p.Scope(), "cap_0123456789abcdef", "Run make check before pushing")
	for i := 0; i < 3; i++ {
		if _, err := Sync(p, store, false); err != nil {
			t.Fatal(err)
		}
	}
	got := active(t, store, p.Scope())
	if len(got) != 1 || got[0].ID != "cap_0123456789abcdef" || got[0].Trust != "high" {
		t.Fatalf("after 3 syncs, %d active memories, want the capture alone: %+v", len(got), got)
	}
	// Forgotten here, then synced: nothing of it stays active, and the file
	// says so for other clones.
	if _, err := ingest.Forget(store, "cap_0123456789abcdef"); err != nil {
		t.Fatal(err)
	}
	if _, err := Sync(p, store, false); err != nil {
		t.Fatal(err)
	}
	if got := active(t, store, p.Scope()); len(got) != 0 {
		t.Fatalf("a forgotten capture is active again after sync: %+v", got)
	}
	lines, err := ReadFile(p.File())
	if err != nil || len(lines) != 1 || lines[0].Active || lines[0].ID != "cap_0123456789abcdef" {
		t.Fatalf("file lines %+v (%v)", lines, err)
	}
}
