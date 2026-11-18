package index_test

import (
	"context"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/veridit/hug/muninn/internal/index"
	"github.com/veridit/hug/muninn/internal/ingest"
	"github.com/veridit/hug/muninn/internal/store"
)

// TestAnchorsIndexed (n7jbmx): indexing a memory stores the anchors its text
// names; a change replaces them, a forget removes them from what recall can
// match, and a store indexed before anchors existed is backfilled once.
func TestAnchorsIndexed(t *testing.T) {
	dir := t.TempDir()
	db, err := store.Open(filepath.Join(dir, "m.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	native := filepath.Join(dir, "n.json")
	m, _, err := ingest.RememberTyped(native, "Run `go test` niced before editing muninn/run/main.go.", "preference", "rule", "global", nil, "")
	if err != nil {
		t.Fatal(err)
	}
	ix := &index.Indexer{DB: db}
	run := func() {
		t.Helper()
		if _, err := ix.Run(context.Background(), index.Options{NativePath: native}); err != nil {
			t.Fatal(err)
		}
	}
	anchors := func() []string {
		t.Helper()
		rows, err := db.Query(`SELECT a.anchor FROM unit_anchors a JOIN units u ON u.id=a.unit_id WHERE u.key=? AND u.active=1 ORDER BY 1`, m.ID)
		if err != nil {
			t.Fatal(err)
		}
		defer rows.Close()
		var out []string
		for rows.Next() {
			var s string
			rows.Scan(&s)
			out = append(out, s)
		}
		return out
	}
	run()
	want := []string{"cmd:go test", "file:main.go", "path:muninn/run/main.go"}
	if got := anchors(); !reflect.DeepEqual(got, want) {
		t.Fatalf("indexed anchors %v, want %v", got, want)
	}
	// A store indexed before anchors existed: the table is empty and the
	// recipe unset. The next pass backfills without any source change.
	if _, err := db.Exec(`DELETE FROM unit_anchors`); err != nil {
		t.Fatal(err)
	}
	if err := db.SetMeta("anchor_recipe", ""); err != nil {
		t.Fatal(err)
	}
	run()
	if got := anchors(); !reflect.DeepEqual(got, want) {
		t.Fatalf("backfilled anchors %v, want %v", got, want)
	}
	if err := ingest.UpdateNative(native, func(ms []ingest.NativeMemory) ([]ingest.NativeMemory, bool, error) {
		for i := range ms {
			if ms[i].ID == m.ID {
				ms[i].Content = "Run `git status` first; see huginn/internal/task/store.go."
			}
		}
		return ms, true, nil
	}); err != nil {
		t.Fatal(err)
	}
	run()
	want = []string{"cmd:git status", "file:store.go", "path:huginn/internal/task/store.go"}
	if got := anchors(); !reflect.DeepEqual(got, want) {
		t.Fatalf("changed anchors %v, want %v", got, want)
	}
	if _, err := ingest.Forget(native, m.ID); err != nil {
		t.Fatal(err)
	}
	run()
	var n int
	db.QueryRow(`SELECT COUNT(*) FROM unit_anchors a JOIN units u ON u.id=a.unit_id WHERE u.active=1`).Scan(&n)
	if n != 0 {
		t.Fatalf("%d anchors left on active units after forget", n)
	}
}

// 4vd1mz: the shared candidate path keeps main's anchor maintenance even
// when forget no longer runs a full pass (and its one-time anchor backfill).
func TestTargetedForgetClearsAnchors(t *testing.T) {
	dir := t.TempDir()
	db, err := store.Open(filepath.Join(dir, "m.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	native := filepath.Join(dir, "native.json")
	m, _, err := ingest.RememberTyped(native, "Run `go test` before editing muninn/run/main.go.", "preference", "rule", "global", nil, "")
	if err != nil {
		t.Fatal(err)
	}
	ix := &index.Indexer{DB: db}
	if _, err := ix.Run(context.Background(), index.Options{NativePath: native}); err != nil {
		t.Fatal(err)
	}
	count := func() int {
		t.Helper()
		var n int
		if err := db.QueryRow(`SELECT COUNT(*) FROM unit_anchors a JOIN units u ON u.id=a.unit_id WHERE u.key=?`, m.ID).Scan(&n); err != nil {
			t.Fatal(err)
		}
		return n
	}
	if n := count(); n == 0 {
		t.Fatal("fixture has no anchors before forget")
	}
	f, err := ingest.ForgetManyChanged(native, []string{m.ID})
	if err != nil || !f.Done[m.ID] {
		t.Fatalf("forget: %+v, %v", f, err)
	}
	if err := ix.ApplyNative(context.Background(), native, f.Before, f.After, f.Changed); err != nil {
		t.Fatal(err)
	}
	if n := count(); n != 0 {
		t.Fatalf("targeted forget left %d anchor rows", n)
	}
	if ix.Passes != 1 {
		t.Fatalf("targeted forget ran a full pass: %d passes", ix.Passes)
	}
}
