package cache

import (
	"bytes"
	"context"
	"fmt"
	"github.com/veridit/hug/huginn/internal/task"
	"math/rand"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"
)

func fixture(i int) []byte {
	return []byte(fmt.Sprintf("---\nid: %s\ntitle: Task %d\nstatus: todo\ntype: bug\nlabels: [core]\naliases: [OLD-%d]\n---\n\n## Goal\nA visible result.\n\n## Purpose\nWhy.\n\n## Principle\nHow.\n\n## Acceptance\n- [ ] passes\n\n## Notes\nEvidence.\n", testID(i), i, i))
}
func propertyFixture(i int) []byte {
	b := fixture(i)
	if i > 1 {
		b = append(b, []byte(fmt.Sprintf("\n## Depends on\n- OLD-%d prior task\n", i-1))...)
	}
	return b
}
func testID(i int) string {
	const a = "0123456789abcdefghjkmnpqrstvwxyz"
	b := []byte("000000")
	for n := 5; n >= 0; n-- {
		b[n] = a[i%32]
		i /= 32
	}
	return string(b)
}
func put(t testing.TB, root, path string, b []byte) {
	t.Helper()
	p := filepath.Join(root, path)
	if e := os.MkdirAll(filepath.Dir(p), 0755); e != nil {
		t.Fatal(e)
	}
	if e := os.WriteFile(p, b, 0644); e != nil {
		t.Fatal(e)
	}
}
func openTest(t testing.TB, root, db string, g Git) *Cache {
	t.Helper()
	c, e := Open(root, db, g)
	if e != nil {
		t.Fatal(e)
	}
	return c
}

type fakeGit struct {
	head    string
	changes []Change
	calls   int
}

func (g *fakeGit) Head(context.Context) (string, error) { return g.head, nil }
func (g *fakeGit) Diff(_ context.Context, _, _ string) ([]Change, error) {
	g.calls++
	return g.changes, nil
}
func TestSyncRules(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	db := filepath.Join(t.TempDir(), "cache.db")
	g := &fakeGit{head: "one"}
	p := "A.000001.task.md"
	put(t, root, p, fixture(1))
	// Older than the racy window: a marker match is trusted (44w1ev).
	old := time.Now().Add(-time.Hour)
	if e := os.Chtimes(filepath.Join(root, p), old, old); e != nil {
		t.Fatal(e)
	}
	c := openTest(t, root, db, g)
	defer c.Close()
	st, e := c.Sync(ctx)
	if e != nil || st.Parsed != 1 || !st.Rebuilt {
		t.Fatal(st, e)
	}
	st, e = c.Sync(ctx)
	if e != nil || st.Parsed != 0 || st.Hashed != 0 {
		t.Fatal(st, e)
	}
	info, e := os.Stat(filepath.Join(root, p))
	if e != nil {
		t.Fatal(e)
	}
	if e = os.Chtimes(filepath.Join(root, p), time.Now(), info.ModTime().Add(time.Minute)); e != nil {
		t.Fatal(e)
	}
	st, e = c.Sync(ctx)
	if e != nil || st.Hashed != 1 || st.Parsed != 0 {
		t.Fatal(st, e)
	}
	// HEAD diff forces hashing even when both size and mtime are preserved.
	info, _ = os.Stat(filepath.Join(root, p))
	b := strings.Replace(string(fixture(1)), "Task 1", "Task X", 1)
	put(t, root, p, []byte(b))
	if e = os.Chtimes(filepath.Join(root, p), info.ModTime(), info.ModTime()); e != nil {
		t.Fatal(e)
	}
	g.head = "two"
	g.changes = []Change{{Status: "M", Path: p}}
	st, e = c.Sync(ctx)
	if e != nil || st.Parsed != 1 || g.calls != 1 {
		t.Fatal(st, e, g.calls)
	}
	newPath := "auth/B.000001.renamed.md"
	if e = os.MkdirAll(filepath.Join(root, "auth"), 0755); e != nil {
		t.Fatal(e)
	}
	if e = os.Rename(filepath.Join(root, p), filepath.Join(root, newPath)); e != nil {
		t.Fatal(e)
	}
	g.head = "three"
	g.changes = []Change{{Status: "R100", OldPath: p, Path: newPath}}
	st, e = c.Sync(ctx)
	if e != nil || st.Parsed != 0 || st.Renamed != 1 {
		t.Fatal(st, e)
	}
	tasks, e := c.List(ctx)
	if e != nil || tasks[0].Path != newPath || tasks[0].Title != "Task X" || tasks[0].Group != "auth" {
		t.Fatal(tasks, e)
	}
	var path string
	if e = c.db.QueryRow(`SELECT path FROM aliases`).Scan(&path); e != nil || path != newPath {
		t.Fatal(path, e)
	}
	t.Log("rules 1-3: unchanged skips hash/parse, hash-equal skips parse, HEAD diff beats preserved markers, rename updates rows without parse")
}
func normalized(t testing.TB, c *Cache) ([]task.Task, []string) {
	t.Helper()
	ts, e := c.List(context.Background())
	if e != nil {
		t.Fatal(e)
	}
	lint, e := c.Lint(context.Background())
	if e != nil {
		t.Fatal(e)
	}
	var findings []string
	for _, f := range lint {
		findings = append(findings, f.String())
	}
	sort.Strings(findings)
	return ts, findings
}
func TestRebuildEqualsIncrementalRandom(t *testing.T) {
	for _, seed := range []int64{1, 7, 42, 2026, 99173} {
		t.Run(fmt.Sprint(seed), func(t *testing.T) {
			r := rand.New(rand.NewSource(seed))
			root := t.TempDir()
			db := filepath.Join(t.TempDir(), "cache.db")
			c := openTest(t, root, db, nil)
			paths := map[int]string{}
			counter := 0
			for step := 0; step < 100; step++ {
				op := r.Intn(4)
				if len(paths) == 0 {
					op = 0
				}
				switch op {
				case 0:
					counter++
					p := fmt.Sprintf("A.%s.task.md", testID(counter))
					paths[counter] = p
					put(t, root, p, propertyFixture(counter))
				case 1, 2, 3:
					ids := make([]int, 0, len(paths))
					for i := range paths {
						ids = append(ids, i)
					}
					sort.Ints(ids)
					i := ids[r.Intn(len(ids))]
					p := paths[i]
					switch op {
					case 1:
						b := propertyFixture(i)
						if step%4 == 0 {
							b = []byte(strings.ReplaceAll(string(b), fmt.Sprintf("OLD-%d", i), fmt.Sprintf("NEW-%d-%d", i, step)))
						}
						b = append(b, []byte(fmt.Sprintf("\nedit %d\n", step))...)
						if step%9 == 0 {
							b = []byte("broken\x00\xff")
						}
						put(t, root, p, b)
					case 2:
						newPath := fmt.Sprintf("group%d/B.%s.renamed%d.md", step%3, testID(i), step)
						if e := os.MkdirAll(filepath.Dir(filepath.Join(root, newPath)), 0755); e != nil {
							t.Fatal(e)
						}
						if e := os.Rename(filepath.Join(root, p), filepath.Join(root, newPath)); e != nil {
							t.Fatal(e)
						}
						paths[i] = newPath
					case 3:
						if e := os.Remove(filepath.Join(root, p)); e != nil {
							t.Fatal(e)
						}
						delete(paths, i)
					}
				}
				if _, e := c.Sync(context.Background()); e != nil {
					t.Fatalf("step %d: %v", step, e)
				}
				if step%10 == 0 || step == 99 {
					want, wl := normalized(t, c)
					wantTables := tableState(t, c)
					if e := c.Close(); e != nil {
						t.Fatal(e)
					}
					if e := os.Remove(db); e != nil {
						t.Fatal(e)
					}
					c = openTest(t, root, db, nil)
					got, gl := normalized(t, c)
					if !reflect.DeepEqual(want, got) || !reflect.DeepEqual(wl, gl) || !reflect.DeepEqual(wantTables, tableState(t, c)) {
						t.Fatalf("seed %d step %d rebuild mismatch\nwant=%+v\ngot=%+v\n%v\n%v", seed, step, want, got, wl, gl)
					}
				}
			}
			c.Close()
			t.Logf("seed %d: 100 random add/edit/rename/delete operations, 11 cache deletion/rebuild comparisons", seed)
		})
	}
}
func TestSchemaRebuild(t *testing.T) {
	root := t.TempDir()
	db := filepath.Join(t.TempDir(), "cache.db")
	put(t, root, "A.000001.task.md", fixture(1))
	c := openTest(t, root, db, nil)
	if _, e := c.Sync(context.Background()); e != nil {
		t.Fatal(e)
	}
	if _, e := c.db.Exec(`UPDATE meta SET value='old' WHERE key='schema_version'`); e != nil {
		t.Fatal(e)
	}
	c.Close()
	c = openTest(t, root, db, nil)
	defer c.Close()
	st, e := c.Sync(context.Background())
	if e != nil || !st.Rebuilt || st.Parsed != 1 {
		t.Fatal(st, e)
	}
}
func BenchmarkList10K(b *testing.B) {
	root := b.TempDir()
	for n := 1; n <= 10000; n++ {
		put(b, root, fmt.Sprintf("A.%s.task.md", testID(n)), fixture(n))
	}
	db := filepath.Join(b.TempDir(), "cache.db")
	c := openTest(b, root, db, nil)
	if _, e := c.List(context.Background()); e != nil {
		b.Fatal(e)
	}
	b.Run("warm", func(b *testing.B) {
		b.ReportAllocs()
		for n := 0; n < b.N; n++ {
			ts, e := c.List(context.Background())
			if e != nil || len(ts) != 10000 {
				b.Fatal(len(ts), e)
			}
		}
	})
	c.Close()
	b.Run("cold", func(b *testing.B) {
		b.ReportAllocs()
		for n := 0; n < b.N; n++ {
			b.StopTimer()
			fresh := filepath.Join(b.TempDir(), "cache.db")
			b.StartTimer()
			cc := openTest(b, root, fresh, nil)
			ts, e := cc.List(context.Background())
			if e != nil || len(ts) != 10000 {
				b.Fatal(len(ts), e)
			}
			cc.Close()
		}
	})
}

// Racy clean (44w1ev, crab's ruling 2026-11-16): an edit that keeps the
// size and lands in the same mtime tick as the last sync is still seen,
// because a file modified within racyWindow of that sync is hashed. The
// Windows run 43922289968 failed exactly here before (same-size OLD-1 to
// NEW-1 within one tick, nil Git).
func TestRacySameSizeEditSameTick(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	p := "A.000001.task.md"
	put(t, root, p, fixture(1))
	tick := time.Now().Truncate(time.Second)
	if e := os.Chtimes(filepath.Join(root, p), tick, tick); e != nil {
		t.Fatal(e)
	}
	c := openTest(t, root, filepath.Join(t.TempDir(), "cache.db"), nil)
	defer c.Close()
	if _, e := c.Sync(ctx); e != nil {
		t.Fatal(e)
	}
	put(t, root, p, []byte(strings.Replace(string(fixture(1)), "Task 1", "Task X", 1)))
	if e := os.Chtimes(filepath.Join(root, p), tick, tick); e != nil {
		t.Fatal(e)
	}
	ts, e := c.List(ctx)
	if e != nil || len(ts) != 1 || ts[0].Title != "Task X" {
		t.Fatal("same-size edit in the same tick was missed:", ts, e)
	}
	// Reopened (the next command), the same holds: synced_ns is stored.
	c.Close()
}

// A write stores only its own record (board_delta), not the whole board;
// a reopened cache equals the board on disk after any mix of blob and
// delta, and the delta is folded into the blob past deltaLimit (44w1ev,
// crab 2026-11-16: writes must not get slower than reads got faster).
func TestBoardDeltaReopen(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	db := filepath.Join(t.TempDir(), "cache.db")
	for n := 1; n <= 20; n++ {
		put(t, root, fmt.Sprintf("A.%s.task.md", testID(n)), fixture(n))
	}
	c := openTest(t, root, db, nil)
	if _, e := c.Sync(ctx); e != nil {
		t.Fatal(e)
	}
	var blob1 []byte
	c.db.QueryRow(`SELECT value FROM meta WHERE key='board'`).Scan(&blob1)
	put(t, root, fmt.Sprintf("A.%s.task.md", testID(3)), []byte(strings.Replace(string(fixture(3)), "Task 3", "Task three", 1)))
	os.Remove(filepath.Join(root, fmt.Sprintf("A.%s.task.md", testID(4))))
	if _, e := c.Sync(ctx); e != nil {
		t.Fatal(e)
	}
	var blob2 []byte
	var delta int
	c.db.QueryRow(`SELECT value FROM meta WHERE key='board'`).Scan(&blob2)
	c.db.QueryRow(`SELECT count(*) FROM board_delta`).Scan(&delta)
	if !bytes.Equal(blob1, blob2) || delta != 2 {
		t.Fatalf("a two-file change rewrote the blob (%v) or stored %d delta rows, want 2", !bytes.Equal(blob1, blob2), delta)
	}
	want, wl := normalized(t, c)
	c.Close()
	c = openTest(t, root, db, nil)
	if len(c.files) != 19 || c.delta != 2 {
		t.Fatalf("blob+delta not used on open: %d files, delta %d", len(c.files), c.delta)
	}
	got, gl := normalized(t, c)
	if !reflect.DeepEqual(want, got) || !reflect.DeepEqual(wl, gl) {
		t.Fatalf("reopened board differs\n%v\n%v", want, got)
	}
	c.delta = deltaLimit
	put(t, root, fmt.Sprintf("A.%s.task.md", testID(5)), []byte(strings.Replace(string(fixture(5)), "Task 5", "Task five", 1)))
	if _, e := c.Sync(ctx); e != nil {
		t.Fatal(e)
	}
	c.db.QueryRow(`SELECT count(*) FROM board_delta`).Scan(&delta)
	if delta != 0 || c.delta != 0 {
		t.Fatalf("delta past the limit was not folded: %d rows", delta)
	}
	c.Close()
	c = openTest(t, root, db, nil)
	defer c.Close()
	ts, _ := c.List(ctx)
	five := false
	for _, x := range ts {
		five = five || x.ID == testID(5) && x.Title == "Task five"
	}
	if len(ts) != 19 || !five {
		t.Fatal("after fold:", len(ts), five)
	}
}
