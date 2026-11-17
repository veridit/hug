package project

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/veridit/hug/muninn/internal/ingest"
)

// 5sc5rw check 3: FoldTwins merges the pairs older syncs made: the local
// memory stays (with its own fields), its twin goes, and the newer state wins;
// a preview changes nothing. Memories from other clones (an imp_ row whose
// line id is no memory here) stay.
func TestFoldTwinsKeepsTheLocalMemory(t *testing.T) {
	p := Project{ID: "rc-demo", Label: "demo", Root: t.TempDir()}
	store := filepath.Join(t.TempDir(), "memories.json")
	captured(t, store, p.Scope(), "cap_aaaaaaaaaaaaaaaa", "Alpha fact")
	captured(t, store, p.Scope(), "cap_bbbbbbbbbbbbbbbb", "Beta fact")
	old := time.Date(2026, 11, 16, 10, 0, 0, 0, time.UTC)
	items := []ingest.Imported{
		{Origin: originFor(p, "cap_aaaaaaaaaaaaaaaa"), Content: "Alpha fact", Category: "fact", Scope: p.Scope(), Active: true, Created: old, Updated: old},
		// The twin was forgotten after the capture's last change: forgotten wins.
		{Origin: originFor(p, "cap_bbbbbbbbbbbbbbbb"), Content: "Beta fact", Category: "fact", Scope: p.Scope(), Active: false, Created: old, Updated: time.Now().Add(time.Hour)},
		{Origin: originFor(p, "mem_other_clone"), Content: "From the other clone", Category: "fact", Scope: p.Scope(), Active: true, Created: old, Updated: old},
	}
	if _, err := ingest.MergeImport(store, items, false); err != nil {
		t.Fatal(err)
	}
	before, _ := os.ReadFile(store)
	n, err := ingest.FoldTwins(store, true)
	if err != nil || n.Folded != 2 || n.Retired != 1 {
		t.Fatalf("preview: %+v %v", n, err)
	}
	if after, _ := os.ReadFile(store); string(after) != string(before) {
		t.Fatal("preview wrote the store")
	}
	if n, err = ingest.FoldTwins(store, false); err != nil || n.Folded != 2 {
		t.Fatalf("apply: %+v %v", n, err)
	}
	mems, _ := ingest.LoadNative(store)
	ids := map[string]ingest.NativeMemory{}
	for _, m := range mems {
		ids[m.ID] = m
	}
	if len(mems) != 3 {
		t.Fatalf("%d memories after the fold, want 3: %+v", len(mems), mems)
	}
	a, b := ids["cap_aaaaaaaaaaaaaaaa"], ids["cap_bbbbbbbbbbbbbbbb"]
	if !a.Active || a.Trust != "high" || a.Origin != "capture:jcode:session_x_1_y:abc:0" {
		t.Fatalf("alpha: %+v", a)
	}
	if b.Active {
		t.Fatalf("beta's newer forget was lost: %+v", b)
	}
	other := 0
	for _, m := range mems {
		if m.Origin == originFor(p, "mem_other_clone") && m.Active {
			other++
		}
	}
	if other != 1 {
		t.Fatalf("the other clone's memory was folded away: %+v", mems)
	}
	if n, _ = ingest.FoldTwins(store, false); n.Folded != 0 {
		t.Fatalf("a second fold found %d", n.Folded)
	}
	// The next sync makes no new twin.
	if _, err := Sync(p, store, false); err != nil {
		t.Fatal(err)
	}
	if mems, _ = ingest.LoadNative(store); len(mems) != 3 {
		t.Fatalf("sync after the fold: %d memories", len(mems))
	}
}
