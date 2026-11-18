package search_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/veridit/hug/muninn/internal/index"
	"github.com/veridit/hug/muninn/internal/ingest"
	"github.com/veridit/hug/muninn/internal/search"
	"github.com/veridit/hug/muninn/internal/store"
)

// TestTriggersMatchWhenSeparately (jrj1vq check 3 pilot harness): a memory
// whose note shares no word with the message is found through its WHEN, and
// nil Triggers leaves search exactly as before.
func TestTriggersMatchWhenSeparately(t *testing.T) {
	dir := t.TempDir()
	db, err := store.Open(filepath.Join(dir, "m.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	native := filepath.Join(dir, "n.json")
	ingest.Remember(native, "Always export TMPDIR=/tmp before committing.", "fact", "global", nil)
	ingest.Remember(native, "The cheat sheet has four heading rows.", "fact", "global", nil)
	if _, err := (&index.Indexer{DB: db}).Run(context.Background(), index.Options{NativePath: native}); err != nil {
		t.Fatal(err)
	}
	var id int64
	if err := db.QueryRow(`SELECT id FROM units WHERE body LIKE 'Always export%'`).Scan(&id); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`CREATE VIRTUAL TABLE temp.when_fts USING fts5(w)`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO temp.when_fts(rowid, w) VALUES (?, 'running the seid gatelock suite')`, id); err != nil {
		t.Fatal(err)
	}
	// Other WHEN rows (no unit behind them) give BM25 a corpus: one row has idf ~0.
	for i := 0; i < 8; i++ {
		if _, err := db.Exec(`INSERT INTO temp.when_fts(rowid, w) VALUES (?, 'reviewing a pull request')`, 1000+i); err != nil {
			t.Fatal(err)
		}
	}
	s := &search.Searcher{DB: db}
	q := "run the gatelock suite"
	o := search.DefaultOptions()
	o.BM25Scale = 2
	plain, err := s.Search(context.Background(), q, o)
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range plain {
		if r.Unit.ID == id {
			t.Fatalf("fixture: the note already matches without a trigger: %+v", r)
		}
	}
	o.Triggers = &search.Triggers{FTS: "when_fts"}
	got, err := s.Search(context.Background(), q, o)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) == 0 || got[0].Unit.ID != id {
		t.Fatalf("with triggers: want the TMPDIR memory first, got %+v", got)
	}
}
