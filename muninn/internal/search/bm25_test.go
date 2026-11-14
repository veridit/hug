package search_test

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/veridit/hug/muninn/internal/index"
	"github.com/veridit/hug/muninn/internal/ingest"
	"github.com/veridit/hug/muninn/internal/search"
	"github.com/veridit/hug/muninn/internal/store"
)

// TestBM25RanksRareWordFirst: with flat points, a common word and a rare
// identifier count the same, so memories that repeat common words of the
// message tie with the one naming the ticket. BM25 puts the rare-word
// match first. This is the effect measured on real memories (recall4).
func TestBM25RanksRareWordFirst(t *testing.T) {
	dir := t.TempDir()
	db, err := store.Open(filepath.Join(dir, "m.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	native := filepath.Join(dir, "n.json")
	for i := 0; i < 20; i++ {
		ingest.Remember(native, fmt.Sprintf("Review note %d: the review checklist asks for a test before merge.", i), "fact", "global", nil)
	}
	ingest.Remember(native, "Brovs-4855 uses four heading rows in the cheat sheet.", "fact", "global", nil)
	if _, err := (&index.Indexer{DB: db}).Run(context.Background(), index.Options{NativePath: native}); err != nil {
		t.Fatal(err)
	}
	s := &search.Searcher{DB: db}
	q := "review the test for Brovs-4855"
	top := func(scale float64) string {
		o := search.DefaultOptions()
		o.BM25Scale = scale
		res, err := s.Search(context.Background(), q, o)
		if err != nil {
			t.Fatal(err)
		}
		if len(res) == 0 {
			t.Fatalf("scale %v: no results", scale)
		}
		return res[0].Unit.Body
	}
	if got := top(0); got == "Brovs-4855 uses four heading rows in the cheat sheet." {
		t.Fatalf("fixture does not separate the methods: flat points already rank the rare match first")
	}
	if got := top(2); got != "Brovs-4855 uses four heading rows in the cheat sheet." {
		t.Errorf("BM25: top result %q, want the Brovs-4855 memory", got)
	}
	// Coverage still counts query words, so recall's full-coverage gate works.
	o := search.DefaultOptions()
	o.BM25Scale = 2
	res, _ := s.Search(context.Background(), "Brovs-4855 cheat sheet", o)
	if len(res) == 0 || res[0].Coverage != 1 {
		t.Errorf("coverage with BM25: %+v", res)
	}
}
