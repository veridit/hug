package cache

import (
	"context"
	"fmt"
	"path/filepath"
	"sync"
	"testing"
)

// 44w1ev on m8shdj: concurrent writes wait, they never fail. Each hug
// command opens its own connection to cache.db and syncs before it holds the
// board turn; when HEAD moved and no file changed, sync's transaction read
// board_delta's count before its first write, so a deferred transaction
// could not upgrade after another process committed: "database is locked
// (5) (SQLITE_BUSY)" (517, SQLITE_BUSY_SNAPSHOT), returned at once without
// waiting on busy_timeout (CI run 44049348743: 2 of 20 parallel notes lost).
func TestConcurrentSyncsWaitNeverFail(t *testing.T) {
	root := t.TempDir()
	db := filepath.Join(t.TempDir(), "cache.db")
	for i := 1; i <= 3; i++ {
		put(t, root, fmt.Sprintf("A.%s.task.md", testID(i)), fixture(i))
	}
	ctx := context.Background()
	first := openTest(t, root, db, &fakeGit{head: "h0"})
	if _, e := first.Sync(ctx); e != nil {
		t.Fatal(e)
	}
	first.Close()
	const writers, rounds = 8, 40
	var wg sync.WaitGroup
	errs := make(chan error, writers*rounds)
	for w := 0; w < writers; w++ {
		wg.Add(1)
		go func(w int) {
			defer wg.Done()
			for r := 0; r < rounds; r++ {
				// A new connection per round, as each command opens its own;
				// HEAD moves with no file changed, the path that failed.
				c, e := Open(root, db, &fakeGit{head: fmt.Sprintf("h%d-%d", w, r)})
				if e != nil {
					errs <- fmt.Errorf("open: %w", e)
					continue
				}
				if _, e = c.Sync(ctx); e != nil {
					errs <- fmt.Errorf("sync: %w", e)
				}
				c.Close()
			}
		}(w)
	}
	wg.Wait()
	close(errs)
	n, kinds := 0, map[string]int{}
	for e := range errs {
		n++
		kinds[e.Error()]++
	}
	for k, m := range kinds {
		t.Errorf("%d x %s", m, k)
	}
	if n > 0 {
		t.Fatalf("%d of %d concurrent syncs failed", n, writers*rounds)
	}
}
