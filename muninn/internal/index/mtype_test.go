package index_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/veridit/hug/muninn/internal/index"
	"github.com/veridit/hug/muninn/internal/ingest"
	"github.com/veridit/hug/muninn/internal/store"
)

// TestMemoryTypeIndexed: a typed memory carries its type into the unit; an
// untyped memory keeps an empty type and its ident does not change when the
// type field is introduced, so upgrading re-embeds nothing.
func TestMemoryTypeIndexed(t *testing.T) {
	dir := t.TempDir()
	db, err := store.Open(filepath.Join(dir, "m.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	native := filepath.Join(dir, "n.json")
	typed, _, err := ingest.RememberTyped(native, "Never push to main without a green CI run.", "preference", "rule", "global", nil, "")
	if err != nil {
		t.Fatal(err)
	}
	plain, _, err := ingest.Remember(native, "The staging database runs Postgres 16 on the ops host.", "fact", "global", nil)
	if err != nil {
		t.Fatal(err)
	}
	ix := &index.Indexer{DB: db}
	if _, err := ix.Run(context.Background(), index.Options{NativePath: native}); err != nil {
		t.Fatal(err)
	}
	get := func(key string) store.Unit {
		var id int64
		if err := db.QueryRow(`SELECT id FROM units WHERE key=?`, key).Scan(&id); err != nil {
			t.Fatal(err)
		}
		u, err := db.GetUnit(id)
		if err != nil {
			t.Fatal(err)
		}
		return u
	}
	if u := get(typed.ID); u.Type != "rule" {
		t.Errorf("typed memory: unit type %q, want rule", u.Type)
	}
	if u := get(plain.ID); u.Type != "" {
		t.Errorf("untyped memory: unit type %q, want empty", u.Type)
	}
	// A second pass with no changes touches nothing.
	st, err := ix.Run(context.Background(), index.Options{NativePath: native})
	if err != nil {
		t.Fatal(err)
	}
	if st.Changed != 0 || st.Added != 0 {
		t.Errorf("re-run changed units: %+v", st)
	}
	// Reinforcing an untyped memory with a type sets it once.
	if _, reinforced, err := ingest.RememberTyped(native, plain.Content, "fact", "fact", "global", nil, ""); err != nil || !reinforced {
		t.Fatalf("reinforce: reinforced=%v err=%v", reinforced, err)
	}
	if _, err := ix.Run(context.Background(), index.Options{NativePath: native}); err != nil {
		t.Fatal(err)
	}
	if u := get(plain.ID); u.Type != "fact" {
		t.Errorf("after typed reinforce: unit type %q, want fact", u.Type)
	}
}
