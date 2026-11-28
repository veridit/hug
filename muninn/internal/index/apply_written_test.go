package index_test

import (
	"context"
	"fmt"
	"math/rand"
	"path/filepath"
	"testing"
	"time"

	"github.com/veridit/hug/muninn/internal/embed"
	"github.com/veridit/hug/muninn/internal/index"
	"github.com/veridit/hug/muninn/internal/ingest"
	"github.com/veridit/hug/muninn/internal/store"
	"github.com/veridit/hug/muninn/internal/testutil"
)

// t0qap9 check 6, index side: after remembers indexed with ApplyWritten
// (the memory added, a reinforce's edit, a trigger replaced, a row a merge
// dropped) and no index pass, the index answers like one an index pass over
// the same file keeps: units, concepts, neighbour lists, the concept graph
// and, under a controlled visitation order, search results. The vector the
// remember embedded before its turn is the one stored; a write with no
// vector leaves the embedding owed, and Finish pays it.
func TestApplyWrittenMatchesIndexPass(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	native := filepath.Join(dir, "native.json")
	r := rand.New(rand.NewSource(11))
	now := time.Now().UTC()
	var mems []ingest.NativeMemory
	for i := 0; i < 40; i++ {
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
		if _, err := ix.Run(ctx, index.Options{NativePath: native}); err != nil {
			t.Fatal(err)
		}
		return db, ix
	}
	tdb, tix := open("targeted.db")
	pdb, pix := open("pass.db")
	vecOf := func(text string) map[string][]float32 {
		vs, err := fake.Embed(ctx, embed.Documents(fake, []string{index.EmbeddingText("", text)}))
		if err != nil {
			t.Fatal(err)
		}
		return map[string][]float32{index.ContentHash(index.EmbeddingText("", text)): vs[0]}
	}
	type write struct {
		name string
		do   func() (ingest.Written, map[string][]float32)
	}
	var added ingest.NativeMemory
	writes := []write{
		{"remember new", func() (ingest.Written, map[string][]float32) {
			text := "Synthetic quartz staging deploy window opens at noon."
			m, re, w, err := ingest.RememberWritten(native, ingest.Remembering{Content: text, Category: "fact", Scope: "global"})
			if err != nil || re || len(w.Changed) != 1 || w.Changed[0].ID != m.ID {
				t.Fatalf("remember: re=%v changed=%d err=%v", re, len(w.Changed), err)
			}
			added = m
			return w, vecOf(text)
		}},
		{"reinforce", func() (ingest.Written, map[string][]float32) {
			_, re, w, err := ingest.RememberWritten(native, ingest.Remembering{Content: added.Content, Category: "fact", Scope: "global"})
			if err != nil || !re || len(w.Changed) != 1 {
				t.Fatalf("reinforce: re=%v changed=%d err=%v", re, len(w.Changed), err)
			}
			return w, nil
		}},
		{"remember without a vector", func() (ingest.Written, map[string][]float32) {
			_, _, w, err := ingest.RememberWritten(native, ingest.Remembering{Content: "Synthetic banana cake release note with no vector yet.", Category: "fact", Scope: "global"})
			if err != nil {
				t.Fatal(err)
			}
			return w, nil
		}},
		{"replace trigger", func() (ingest.Written, map[string][]float32) {
			_, found, w, err := ingest.ReplaceTriggerWritten(native, added.ID, nil)
			if err != nil || !found {
				t.Fatalf("replace trigger: found=%v err=%v", found, err)
			}
			return w, nil
		}},
		{"merge drops a row", func() (ingest.Written, map[string][]float32) {
			base, err := ingest.LoadNative(native)
			if err != nil {
				t.Fatal(err)
			}
			edited := ingest.CloneNative(base)
			edited = append(edited[:3], edited[4:]...)
			w, err := ingest.MergeSaveChanged(native, base, edited)
			if err != nil || len(w.Removed) != 1 {
				t.Fatalf("merge: removed=%v err=%v", w.Removed, err)
			}
			return w, nil
		}},
	}
	for round, wr := range writes {
		w, vecs := wr.do()
		if err := tix.ApplyWritten(ctx, native, w, vecs); err != nil {
			t.Fatalf("%s: ApplyWritten: %v", wr.name, err)
		}
		if _, err := tix.Finish(ctx); err != nil {
			t.Fatalf("%s: Finish: %v", wr.name, err)
		}
		if _, err := pix.Run(ctx, index.Options{NativePath: native}); err != nil {
			t.Fatal(err)
		}
		compareIndexes(t, round, tdb, pdb, true)
	}
	if tix.Passes != 1 {
		t.Fatalf("targeted index ran %d passes, want only its first", tix.Passes)
	}
	st, err := tix.Run(ctx, index.Options{NativePath: native})
	if err != nil {
		t.Fatal(err)
	}
	if st.SourcesScanned != 0 || st.Changed+st.Added+st.Removed+st.Embedded != 0 {
		t.Fatalf("a pass after the targeted writes still found work: %s", st)
	}
	// A file some other write changed since the index saw it: refused,
	// nothing changed.
	_, _, w, err := ingest.RememberWritten(native, ingest.Remembering{Content: "Synthetic stale write.", Category: "fact", Scope: "global"})
	if err != nil {
		t.Fatal(err)
	}
	w.Before.Size++
	if err := tix.ApplyWritten(ctx, native, w, nil); err != index.ErrNotCurrent {
		t.Fatalf("stale Before: %v, want ErrNotCurrent", err)
	}
	var n int
	tdb.QueryRow(`SELECT COUNT(*) FROM units WHERE body='Synthetic stale write.'`).Scan(&n)
	if n != 0 {
		t.Fatal("a refused ApplyWritten wrote the unit")
	}
}
