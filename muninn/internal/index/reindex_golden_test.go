package index_test

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
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

// t0qap9 check 7: the neighbour lists and topic groups (communities) a
// reindex leaves are the ones hug left before the pass was split into
// lock-free compute and short write steps. The golden file was written by
// the code before the split (v0.7.118, b5e4450c) and must not be
// regenerated to make a change pass: a diff here is a changed result.
//
// The scenario runs every kind of pass the index has: a cold index of three
// source kinds, edits, additions, deletions, a removed source file, targeted
// forgets (ApplyNative), a capped embed (MaxEmbed) followed by the rest, and
// a model switch that rebuilds every neighbour list. After each step it
// dumps every index table by names (memory keys and concept terms), never
// by row ids. Communities come from the controlled visit schedule: in
// production the order is Go map order, so two runs of the same code differ.
// Similarities are rounded to 6 decimals, so a last-bit difference between
// CPUs (fused multiply-add on arm64) does not count as a changed result.
func TestReindexMatchesGolden(t *testing.T) {
	got := reindexGoldenScenario(t)
	path := filepath.Join("testdata", "reindex_golden.txt")
	if os.Getenv("HUG_UPDATE_GOLDEN") == "1" {
		if err := os.MkdirAll("testdata", 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("%v (HUG_UPDATE_GOLDEN=1 writes it)", err)
	}
	if got != string(want) {
		gl, wl := strings.Split(got, "\n"), strings.Split(string(want), "\n")
		for i := 0; i < len(gl) || i < len(wl); i++ {
			var g, w string
			if i < len(gl) {
				g = gl[i]
			}
			if i < len(wl) {
				w = wl[i]
			}
			if g != w {
				t.Fatalf("reindex results differ from the golden at line %d\n got %q\nwant %q", i+1, g, w)
			}
		}
	}
}

func reindexGoldenScenario(t *testing.T) string {
	t.Helper()
	ctx := context.Background()
	dir := t.TempDir()
	db, err := store.Open(filepath.Join(dir, "m.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	native := filepath.Join(dir, "native.json")
	jdir := filepath.Join(dir, "jcode")
	notes := filepath.Join(dir, "notes")
	for _, d := range []string{filepath.Join(jdir, "projects"), notes} {
		if err := os.MkdirAll(d, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	r := rand.New(rand.NewSource(11))
	at := time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)
	tick := 0
	// touch gives every write its own mtime, so no pass skips a changed file.
	touch := func(path string) {
		tick++
		mt := at.Add(time.Duration(tick) * time.Second)
		if err := os.Chtimes(path, mt, mt); err != nil {
			t.Fatal(err)
		}
	}
	var mems []ingest.NativeMemory
	next := 0
	addMem := func() {
		mems = append(mems, ingest.NativeMemory{ID: fmt.Sprintf("mem_%03d", next), Content: synthetic(r, next), Active: true, Scope: "global", Created: at, Updated: at})
		next++
	}
	for i := 0; i < 50; i++ {
		addMem()
	}
	saveNative := func() {
		if err := ingest.SaveNative(native, mems); err != nil {
			t.Fatal(err)
		}
		touch(native)
	}
	type jmem struct {
		ID        string   `json:"id"`
		Content   string   `json:"content"`
		Category  string   `json:"category"`
		Tags      []string `json:"tags"`
		UpdatedAt string   `json:"updated_at"`
	}
	writeJcode := func(name string, from, n int) {
		f := struct {
			Memories map[string]jmem `json:"memories"`
		}{map[string]jmem{}}
		for i := from; i < from+n; i++ {
			id := fmt.Sprintf("j_%s_%02d", strings.TrimSuffix(filepath.Base(name), ".json"), i)
			f.Memories[id] = jmem{ID: id, Content: synthetic(r, 1000+i), Category: "fact", Tags: []string{vocab[i%len(vocab)]}, UpdatedAt: at.Format(time.RFC3339Nano)}
		}
		b, _ := json.Marshal(f)
		if err := os.WriteFile(name, b, 0o600); err != nil {
			t.Fatal(err)
		}
		touch(name)
	}
	writeNote := func(name string, secs int) {
		var sb strings.Builder
		for i := 0; i < secs; i++ {
			fmt.Fprintf(&sb, "## Section %d\n\n%s\n\n", i, synthetic(r, 2000+i))
		}
		p := filepath.Join(notes, name)
		if err := os.WriteFile(p, []byte(sb.String()), 0o600); err != nil {
			t.Fatal(err)
		}
		touch(p)
	}
	saveNative()
	// One new or changed jcode file per pass: IndexJcode walks its files in
	// map order (indexer.go IndexJcode), so two in one pass get their concept
	// ids in either order, and the controlled schedule visits by id.
	writeJcode(filepath.Join(jdir, "global.json"), 0, 12)
	writeNote("one.md", 6)
	writeNote("two.md", 4)
	o := index.Options{NativePath: native, Jcode: []index.JcodeSource{{Dir: jdir, Prefix: "jcode@test"}}, NotesRoots: []string{notes}}
	fake := &testutil.FakeEmbedder{Dim: 64}
	ix := &index.Indexer{DB: db, Embedder: fake}
	index.ControlledGraphOrder(ix)
	var out strings.Builder
	step := func(name string) {
		t.Helper()
		if _, err := ix.Run(ctx, o); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		fmt.Fprintf(&out, "=== %s\n%s", name, dumpIndex(t, db, dir))
	}
	step("cold")
	writeJcode(filepath.Join(jdir, "projects", "alpha.json"), 0, 8)
	step("project added")

	// Edits, additions and deletions across all three kinds.
	for _, i := range []int{3, 9, 27} {
		mems[i].Content = synthetic(r, 500+i)
	}
	mems[14].Tags = []string{"kernel", "socket"}
	for i := 0; i < 6; i++ {
		addMem()
	}
	mems = append(mems[:20], mems[22:]...)
	saveNative()
	writeJcode(filepath.Join(jdir, "global.json"), 4, 12)
	writeNote("one.md", 4)
	step("edits")

	// A source file goes away; another arrives.
	if err := os.Remove(filepath.Join(jdir, "projects", "alpha.json")); err != nil {
		t.Fatal(err)
	}
	writeJcode(filepath.Join(jdir, "projects", "beta.json"), 0, 5)
	step("source removed")

	// Targeted forgets, as hug mem forget does them. Forget stamps Updated
	// with the clock, so the file's size is not compared.
	for _, id := range []string{"mem_005", "mem_031"} {
		f, err := ingest.ForgetManyChanged(native, []string{id})
		if err != nil || !f.Done[id] {
			t.Fatalf("forget %s: %+v %v", id, f, err)
		}
		if err := ix.ApplyNative(ctx, native, f.Before, f.After, f.Changed); err != nil {
			t.Fatalf("ApplyNative %s: %v", id, err)
		}
	}
	fmt.Fprintf(&out, "=== targeted forgets\n%s", dumpIndex(t, db, dir))
	if loaded, err := ingest.LoadNative(native); err != nil {
		t.Fatal(err)
	} else {
		mems = loaded
	}

	// A hook's capped pass embeds one text; the next pass embeds the rest.
	for i := 0; i < 5; i++ {
		addMem()
	}
	mems[40].Content = synthetic(r, 940)
	saveNative()
	ix.MaxEmbed = 1
	step("capped embed")
	ix.MaxEmbed = 0
	step("rest embedded")

	// A new model: every vector and neighbour list is rebuilt.
	ix.Embedder = &testutil.FakeEmbedder{Dim: 48, Name: "fake-embed-2"}
	step("model switch")

	// Nothing changed: a pass changes nothing.
	step("idle")
	return out.String()
}

// dumpIndex is every index table by names: units, concepts, neighbour lists,
// communities, the concept graph, per-unit community counts, anchors and
// the pipeline meta.
func dumpIndex(t *testing.T, db *store.DB, dir string) string {
	t.Helper()
	unit := `(s.kind || ':' || replace(s.path, '` + dir + `', '') || '#' || u.key)`
	var sb strings.Builder
	for _, q := range []struct{ name, sql string }{
		{"sources", `SELECT kind || ' ' || replace(path, '` + dir + `', '') FROM sources ORDER BY 1`},
		{"units", `SELECT ` + unit + ` || ' ' || u.active || ' ' || u.ident || ' ' || u.content_hash FROM units u JOIN sources s ON s.id=u.source_id ORDER BY 1`},
		{"concepts", `SELECT ` + unit + ` || ' ' || group_concat(c.term, ',') FROM (SELECT uc.unit_id, c.term FROM unit_concepts uc JOIN concepts c ON c.id=uc.concept_id ORDER BY c.term) c
		  JOIN units u ON u.id=c.unit_id JOIN sources s ON s.id=u.source_id GROUP BY u.id ORDER BY 1`},
		{"neighbors", `SELECT ` + unit + ` || ' ' || group_concat(nb, ' ') FROM (
		  SELECT n.unit_id, (vs.kind || ':' || replace(vs.path, '` + dir + `', '') || '#' || v.key) || '=' || printf('%.6f', n.sim) AS nb
		  FROM neighbors n JOIN units v ON v.id=n.nbr_id JOIN sources vs ON vs.id=v.source_id ORDER BY n.unit_id, n.sim DESC, n.nbr_id) x
		  JOIN units u ON u.id=x.unit_id JOIN sources s ON s.id=u.source_id GROUP BY u.id ORDER BY 1`},
		{"communities", `SELECT cc.term || ' ' || g.term FROM communities m JOIN concepts cc ON cc.id=m.concept_id
		  LEFT JOIN concepts g ON g.id=m.community_id ORDER BY 1`},
		{"community totals", `SELECT ifnull(c.term, '#' || g.community_id) || ' ' || printf('%.6f', g.sum_tot) FROM graph_community g
		  LEFT JOIN concepts c ON c.id=g.community_id WHERE abs(g.sum_tot) > 1e-9 ORDER BY 1`},
		{"unit communities", `SELECT ` + unit + ` || ' ' || group_concat(x, ',') FROM (
		  SELECT uc.unit_id, ifnull(c.term, '#' || uc.community_id) || '=' || uc.n AS x FROM unit_communities uc
		  LEFT JOIN concepts c ON c.id=uc.community_id ORDER BY uc.unit_id, c.term) y
		  JOIN units u ON u.id=y.unit_id JOIN sources s ON s.id=u.source_id GROUP BY u.id ORDER BY 1`},
		{"anchors", `SELECT ` + unit + ` || ' ' || group_concat(a.anchor, ',') FROM (SELECT unit_id, anchor FROM unit_anchors ORDER BY anchor) a
		  JOIN units u ON u.id=a.unit_id JOIN sources s ON s.id=u.source_id GROUP BY u.id ORDER BY 1`},
		{"meta", `SELECT key || ' ' || value FROM meta WHERE key IN ('pipeline_hash','pipeline_query_prefix','graph_m','source_recipe','anchor_recipe') ORDER BY 1`},
		{"vectors", `SELECT count(*) || ' ' || pipeline_hash FROM qvecs GROUP BY pipeline_hash ORDER BY 1`},
	} {
		fmt.Fprintf(&sb, "-- %s\n", q.name)
		rows := dumpRows(t, db, q.sql)
		if rows != "" {
			sb.WriteString(rows)
			sb.WriteByte('\n')
		}
	}
	// The concept graph by terms: each concept's degree and weighted edges.
	term := map[int]string{}
	for _, l := range strings.Split(dumpRows(t, db, `SELECT id || ' ' || term FROM concepts`), "\n") {
		var id int
		var s string
		if _, err := fmt.Sscan(l, &id, &s); err == nil {
			term[id] = s
		}
	}
	adj, k, m, err := index.GraphEdges(db)
	if err != nil {
		t.Fatal(err)
	}
	var lines []string
	for v, nb := range adj {
		var es []string
		for j, w := range nb {
			es = append(es, fmt.Sprintf("%s=%g", term[j], w))
		}
		sort.Strings(es)
		lines = append(lines, fmt.Sprintf("%s k=%g %s", term[v], k[v], strings.Join(es, ",")))
	}
	sort.Strings(lines)
	fmt.Fprintf(&sb, "-- graph m=%g\n%s\n", m, strings.Join(lines, "\n"))
	return sb.String()
}
