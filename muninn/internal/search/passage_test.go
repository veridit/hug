package search_test

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/veridit/hug/muninn/internal/index"
	"github.com/veridit/hug/muninn/internal/ingest"
	"github.com/veridit/hug/muninn/internal/search"
	"github.com/veridit/hug/muninn/internal/store"
)

// passageFixture: a long memory whose one paragraph is about the query,
// buried in unrelated paragraphs, against short memories that mention the
// query words once in passing. Whole-memory BM25 normalises the long one
// by its full length, so it ranks below the short ones; its best passage
// is as focused as any of them (c3hszg, recall4's best-passage idea).
func passageFixture(t *testing.T) *search.Searcher {
	t.Helper()
	dir := t.TempDir()
	db, err := store.Open(filepath.Join(dir, "m.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	native := filepath.Join(dir, "n.json")
	var long []string
	for i := 0; i < 12; i++ {
		long = append(long, fmt.Sprintf("Paragraph %d of the session log: the lane rebuilt its worktree, ran the parser suite, compared the generated C with the reference output and wrote the numbers into the ticket before handing over to the next lane in the queue.", i))
	}
	long[7] = "Quokka ledger migration: the quokka ledger migration moves the quokka ledger, and the migration keeps ids."
	ingest.Remember(native, strings.Join(long, "\n\n"), "fact", "global", nil)
	for i := 0; i < 6; i++ {
		ingest.Remember(native, fmt.Sprintf("Short note %d: the quokka ledger migration came up once, in passing, during a long discussion about something else entirely: the release calendar, who reviews the parser lane next week, why the bench numbers moved, which machine runs the nightly job, how the store backup is rotated, and what the coordinator wants in the weekly summary. After that the talk went to the office move, the new chairs, the coffee machine that keeps breaking, the shared calendar for holidays, the printer queue, the parking rules for visitors, the lunch order on Fridays and the plan for the summer outing.", i), "fact", "global", nil)
	}
	for i := 0; i < 200; i++ {
		ingest.Remember(native, fmt.Sprintf("Filler memory %d about parser suites and generated C output.", i), "fact", "global", nil)
	}
	if _, err := (&index.Indexer{DB: db}).Run(context.Background(), index.Options{NativePath: native}); err != nil {
		t.Fatal(err)
	}
	return &search.Searcher{DB: db}
}

func rankOfLong(t *testing.T, s *search.Searcher, mix float64) int {
	t.Helper()
	o := search.DefaultOptions()
	o.BM25Scale = 2
	o.PassageMix = mix
	res, err := s.Search(context.Background(), "quokka ledger migration", o)
	if err != nil {
		t.Fatal(err)
	}
	for i, r := range res {
		if strings.HasPrefix(r.Unit.Body, "Paragraph 0 of the session log") {
			return i + 1
		}
	}
	return 0
}

// TestBestPassageLiftsAFocusedLongMemory: with the mix off the long memory
// ranks below the passing mentions; with it on, first.
func TestBestPassageLiftsAFocusedLongMemory(t *testing.T) {
	s := passageFixture(t)
	if r := rankOfLong(t, s, 0); r == 1 {
		t.Fatalf("fixture does not separate: whole-memory BM25 already ranks the long memory first")
	}
	if r := rankOfLong(t, s, 0.65); r != 1 {
		t.Errorf("PassageMix 0.65: the long memory ranks %d, want 1", r)
	}
}

// TestPassageMixOffIsUnchanged: mix 0 gives the same results as before the
// option existed, and a memory of one passage scores the same either way.
func TestPassageMixOffIsUnchanged(t *testing.T) {
	s := passageFixture(t)
	for _, q := range []string{"parser suite generated C", "filler memory 3", "quokka"} {
		run := func(mix float64) map[int64]float64 {
			o := search.DefaultOptions()
			o.BM25Scale = 2
			o.PassageMix = mix
			o.Limit = 300
			res, err := s.Search(context.Background(), q, o)
			if err != nil {
				t.Fatal(err)
			}
			out := map[int64]float64{}
			for _, r := range res {
				if !strings.HasPrefix(r.Unit.Body, "Paragraph 0") { // the one long memory may move; nothing else may
					out[r.Unit.ID] = r.Total
				}
			}
			return out
		}
		off, on := run(0), run(0.65)
		if len(off) == 0 || len(off) != len(on) {
			t.Fatalf("%q: %d results off, %d on", q, len(off), len(on))
		}
		for id, v := range off {
			if on[id] != v {
				t.Errorf("%q: memory %d scored %v off, %v with the mix", q, id, v, on[id])
			}
		}
	}
}

// TestPassageScoreIsFTS5BM25: the passage scorer reproduces FTS5's bm25()
// for a whole row (title and tags empty), so passage and whole-memory
// scores share one scale and mix without normalising.
func TestPassageScoreIsFTS5BM25(t *testing.T) {
	s := passageFixture(t)
	for _, w := range []string{"quokka", "ledger", "parser"} {
		rows, err := s.DB.Query(`SELECT u.id, u.body, -bm25(units_fts, 2.0, 1.0, 1.0) FROM units_fts JOIN units u ON u.id = units_fts.rowid WHERE units_fts MATCH ? AND u.title = '' AND u.tags = '' LIMIT 20`, `"`+w+`"`)
		if err != nil {
			t.Fatal(err)
		}
		// Read all rows first: the store has one connection.
		type row struct {
			id   int64
			body string
			want float64
		}
		var got []row
		for rows.Next() {
			var r row
			rows.Scan(&r.id, &r.body, &r.want)
			got = append(got, r)
		}
		rows.Close()
		if len(got) == 0 {
			t.Fatalf("%q: no rows to compare", w)
		}
		for _, r := range got {
			v, err := search.PassageBM25(context.Background(), s, w, r.body)
			if err != nil {
				t.Fatal(err)
			}
			if d := v - r.want; d > 1e-9 || d < -1e-9 {
				t.Errorf("%q in memory %d: passage scorer %v, FTS5 bm25 %v", w, r.id, v, r.want)
			}
		}
	}
}
