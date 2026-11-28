package index_test

import (
	"context"
	"fmt"
	"math/rand"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/veridit/hug/muninn/internal/index"
	"github.com/veridit/hug/muninn/internal/ingest"
	"github.com/veridit/hug/muninn/internal/store"
	"github.com/veridit/hug/muninn/internal/testutil"
)

// t0qap9: a pass reads and plans a source file with no lock, so another
// writer can change a unit between the plan and the pass's write. The pass
// then leaves that unit to the other writer, does not record the file's
// stat, and the next pass reads the file again; afterwards the index equals
// one built from the file in one go.
func TestPassSkipsUnitWrittenAfterItsPlan(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	native := filepath.Join(dir, "native.json")
	r := rand.New(rand.NewSource(5))
	now := time.Now().UTC()
	var mems []ingest.NativeMemory
	for i := 0; i < 30; i++ {
		mems = append(mems, ingest.NativeMemory{ID: fmt.Sprintf("mem_%03d", i), Content: synthetic(r, i), Active: true, Scope: "global", Created: now, Updated: now})
	}
	if err := ingest.SaveNative(native, mems); err != nil {
		t.Fatal(err)
	}
	fake := &testutil.FakeEmbedder{Dim: 64}
	open := func(name string) (*store.DB, *index.Indexer) {
		db, err := store.Open(filepath.Join(dir, name))
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { db.Close() })
		ix := &index.Indexer{DB: db, Embedder: fake}
		index.ControlledGraphOrder(ix)
		return db, ix
	}
	tdb, tix := open("t.db")
	if _, err := tix.Run(ctx, index.Options{NativePath: native}); err != nil {
		t.Fatal(err)
	}
	// The file changes; while the pass plans it, a remember edits mem_004.
	mems[3].Content = synthetic(r, 103)
	mems[4].Content = synthetic(r, 104)
	if err := ingest.SaveNative(native, mems); err != nil {
		t.Fatal(err)
	}
	raced := false
	index.AfterPlan(func(string) {
		if raced {
			return
		}
		raced = true
		w, err := ingest.UpdateNativeChanged(native, func(cur []ingest.NativeMemory) ([]ingest.NativeMemory, bool, error) {
			for i := range cur {
				if cur[i].ID == "mem_004" {
					cur[i].Content = synthetic(r, 204)
				}
			}
			return cur, true, nil
		})
		if err != nil {
			t.Error(err)
			return
		}
		if err := tix.ApplyWritten(ctx, native, w, nil); err != nil {
			t.Errorf("ApplyWritten during the pass: %v", err)
		}
	})
	defer index.AfterPlan(nil)
	st, err := tix.Run(ctx, index.Options{NativePath: native})
	if err != nil {
		t.Fatal(err)
	}
	if !raced || st.Changed != 1 {
		t.Fatalf("raced pass: hook ran %v, %s (want only mem_003 changed by the pass)", raced, st)
	}
	// The remember's newer text stands: the pass did not overwrite it
	// with the text it had planned from the older file.
	var body string
	tdb.QueryRow(`SELECT body FROM units WHERE key='mem_004'`).Scan(&body)
	if cur, _ := ingest.LoadNative(native); body != cur[4].Content {
		t.Fatalf("after the raced pass mem_004 is %q, want the remember's %q", body, cur[4].Content)
	}
	var mtime, size int64
	tdb.QueryRow(`SELECT mtime_ns, size FROM sources WHERE kind='native'`).Scan(&mtime, &size)
	if fs, _ := ingest.StatFile(native); fs.Mtime == mtime && fs.Size == size {
		t.Fatal("a raced pass recorded the file's stat")
	}
	// The next pass reads the file again and finds it indexed.
	st, err = tix.Run(ctx, index.Options{NativePath: native})
	if err != nil {
		t.Fatal(err)
	}
	if st.SourcesScanned != 1 || st.Changed+st.Added+st.Removed != 0 {
		t.Fatalf("pass after the race: %s (want the file scanned, nothing changed)", st)
	}
	pdb, pix := open("p.db")
	if _, err := pix.Run(ctx, index.Options{NativePath: native}); err != nil {
		t.Fatal(err)
	}
	for _, q := range []string{
		`SELECT u.key || ' ' || u.active || ' ' || u.ident FROM units u ORDER BY u.key`,
		`SELECT u.key || ' ' || c.term FROM unit_concepts uc JOIN units u ON u.id=uc.unit_id JOIN concepts c ON c.id=uc.concept_id ORDER BY 1`,
		`SELECT u.key || ' ' || v.key || ' ' || printf('%.5f', n.sim) FROM neighbors n JOIN units u ON u.id=n.unit_id JOIN units v ON v.id=n.nbr_id ORDER BY 1`,
	} {
		if ga, gb := dumpRows(t, tdb, q), dumpRows(t, pdb, q); ga != gb {
			t.Fatalf("%s differ\nraced:\n%s\none pass:\n%s", q, diffLines(ga, gb), diffLines(gb, ga))
		}
	}
	// Concept ids follow creation order, which the race changed: compare
	// the concept graphs by term.
	graph := func(db *store.DB) string {
		adj, k, m, err := index.GraphEdges(db)
		if err != nil {
			t.Fatal(err)
		}
		term := map[int]string{}
		rows, err := db.Query(`SELECT id, term FROM concepts`)
		if err != nil {
			t.Fatal(err)
		}
		for rows.Next() {
			var id int
			var s string
			rows.Scan(&id, &s)
			term[id] = s
		}
		rows.Close()
		var lines []string
		for a, nb := range adj {
			for b, w := range nb {
				lines = append(lines, fmt.Sprintf("%s %s %v", term[a], term[b], w))
			}
			lines = append(lines, fmt.Sprintf("k %s %v", term[a], k[a]))
		}
		sort.Strings(lines)
		return fmt.Sprintf("M %v\n%s", m, strings.Join(lines, "\n"))
	}
	if ga, gb := graph(tdb), graph(pdb); ga != gb {
		t.Fatalf("concept graphs differ\nraced:\n%s\none pass:\n%s", diffLines(ga, gb), diffLines(gb, ga))
	}
}
