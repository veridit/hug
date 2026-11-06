package search

import (
	"context"
	"github.com/veridit/muninn/internal/index"
	"github.com/veridit/muninn/internal/store"
	"path/filepath"
	"testing"
	"time"
)

func TestExperimentalControlsPreserveScope(t *testing.T) {
	db, err := store.Open(filepath.Join(t.TempDir(), "eval.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	sid, _, _, err := db.Source("native", "test")
	if err != nil {
		t.Fatal(err)
	}
	ix := index.Indexer{DB: db}
	st := index.Stats{}
	_, _, err = ix.SyncSource(sid, []index.Candidate{{Key: "allowed", Kind: "memory", Body: "specific_project deployment configuration", Scope: "allowed", Active: true, Updated: time.Now()}, {Key: "denied", Kind: "memory", Body: "specific_project deployment configuration", Scope: "denied", Active: true, Updated: time.Now()}}, &st)
	if err != nil {
		t.Fatal(err)
	}
	s := Searcher{DB: db}
	for _, o := range []Options{{}, {DisableConcepts: true}, {DisableCommunity: true}, {DisableFTS: true}, {DisableSemantic: true}, {SemanticSlots: 0}, {RecencyWeight: 2}, {RecencyWeight: 1, NormalisedRelevance: true}} {
		o.Scopes = []string{"allowed"}
		o.Limit = 10
		rs, err := s.Search(context.Background(), "specific_project deployment", o)
		if err != nil {
			t.Fatal(err)
		}
		for _, r := range rs {
			if r.Unit.Scope != "allowed" {
				t.Fatalf("scope leak: %+v", r)
			}
		}
	}
}
func TestNormalisedTemporalScoreRetainsEligibleResults(t *testing.T) {
	db, err := store.Open(filepath.Join(t.TempDir(), "eval.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	sid, _, _, err := db.Source("native", "test")
	if err != nil {
		t.Fatal(err)
	}
	ix := index.Indexer{DB: db}
	st := index.Stats{}
	_, _, err = ix.SyncSource(sid, []index.Candidate{{Key: "old", Kind: "memory", Body: "special_key deployment configuration", Scope: "s", Active: true, Updated: time.Now().Add(-24 * time.Hour)}, {Key: "new", Kind: "memory", Body: "special_key deployment configuration setting", Scope: "s", Active: true, Updated: time.Now()}}, &st)
	if err != nil {
		t.Fatal(err)
	}
	s := Searcher{DB: db}
	rs, err := s.Search(context.Background(), "special_key deployment", Options{Limit: 10, MinScore: 6, RecencyWeight: 1, NormalisedRelevance: true, Scopes: []string{"s"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(rs) != 2 {
		t.Fatalf("eligible normalized scores below original min-score incorrectly dropped: %d", len(rs))
	}
	for _, r := range rs {
		if r.Total < 0 || r.Total > 2 {
			t.Fatalf("not normalized: %v", r.Total)
		}
	}
}
