package store

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	sqlite "modernc.org/sqlite"

	"github.com/veridit/hug/internal/flock"
)

// 6gtv7a: store.Open reads first and writes only what is missing, and never
// waits on the database while it holds the init lock. hippo measured the old
// open on a store copy (vrx1ja): with a table to add and another process
// holding the write lock, the schema exec waited busy_timeout (30 s) under
// muninn.db.init.lock, so four openers answered after 30, 60, 90 and 121 s.

// writeActions are the authorizer actions of a statement that writes.
var writeActions = map[sqlite.AuthorizerActionCode]string{
	sqlite.AuthInsert: "INSERT", sqlite.AuthUpdate: "UPDATE", sqlite.AuthDelete: "DELETE",
	sqlite.AuthCreateTable: "CREATE TABLE", sqlite.AuthCreateIndex: "CREATE INDEX",
	sqlite.AuthCreateTrigger: "CREATE TRIGGER", sqlite.AuthCreateView: "CREATE VIEW",
	sqlite.AuthCreateVTable: "CREATE VIRTUAL TABLE", sqlite.AuthDropTable: "DROP TABLE",
	sqlite.AuthDropIndex: "DROP INDEX", sqlite.AuthDropTrigger: "DROP TRIGGER",
	sqlite.AuthDropView: "DROP VIEW", sqlite.AuthDropVTable: "DROP VIRTUAL TABLE",
	sqlite.AuthAlterTable: "ALTER TABLE", sqlite.AuthReindex: "REINDEX",
	sqlite.AuthAnalyze: "ANALYZE", sqlite.AuthTransaction: "TRANSACTION",
}

var authWatch = struct {
	sync.Mutex
	once sync.Once
	logs map[string]*[]string // database path -> write statements seen
}{logs: map[string]*[]string{}}

// watchWrites records every write statement SQLite compiles on a connection
// to path opened from now on (its authorizer sees each statement as it is
// prepared), and returns what it has seen so far.
func watchWrites(t *testing.T, path string) func() []string {
	t.Helper()
	authWatch.once.Do(func() {
		sqlite.RegisterConnectionHook(func(c sqlite.ExecQuerierContext, dsn string) error {
			authWatch.Lock()
			defer authWatch.Unlock()
			for p, log := range authWatch.logs {
				if !strings.Contains(dsn, "file:"+p+"?") {
					continue
				}
				r, ok := c.(sqlite.AuthorizerRegisterer)
				if !ok {
					return fmt.Errorf("driver has no authorizer")
				}
				return r.RegisterAuthorizer(func(a sqlite.AuthorizerActionCode, arg1, arg2, _, _ string) sqlite.AuthorizerReturnCode {
					if w, ok := writeActions[a]; ok {
						authWatch.Lock()
						*log = append(*log, strings.TrimSpace(w+" "+arg1+" "+arg2))
						authWatch.Unlock()
					}
					return sqlite.AuthorizerOK
				})
			}
			return nil
		})
	})
	log := &[]string{}
	authWatch.Lock()
	authWatch.logs[path] = log
	authWatch.Unlock()
	t.Cleanup(func() {
		authWatch.Lock()
		delete(authWatch.logs, path)
		authWatch.Unlock()
	})
	return func() []string {
		authWatch.Lock()
		defer authWatch.Unlock()
		return append([]string(nil), *log...)
	}
}

// migratedStore is a fully migrated store with a row in each table a
// backfill reads, so Open's backfill checks see data, as on a real store.
func migratedStore(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "m.db")
	d, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()
	if err := d.PutEmbedding("h", "p", 1, []byte{1}, []byte{1}, []byte{1}); err != nil {
		t.Fatal(err)
	}
	for _, q := range []string{
		`INSERT INTO communities(concept_id, community_id) VALUES(1, 1)`,
		`INSERT INTO unit_concepts(unit_id, concept_id) VALUES(1, 1)`,
		`INSERT INTO unit_communities(community_id, unit_id, n) VALUES(1, 1, 1)`,
	} {
		if _, err := d.Exec(q); err != nil {
			t.Fatal(err)
		}
	}
	return path
}

func head(s []string, n int) string {
	if len(s) > n {
		return strings.Join(s[:n], "; ") + fmt.Sprintf("; ... (%d more)", len(s)-n)
	}
	return strings.Join(s, "; ")
}

// Check 1: with nothing missing, Open runs no write statement.
func TestOpenWritesNothingWhenNothingIsMissing(t *testing.T) {
	path := migratedStore(t)
	writes := watchWrites(t, path)
	d, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	d.Close()
	if w := writes(); len(w) != 0 {
		t.Fatalf("Open of a fully migrated store ran %d write statements, want none (6gtv7a): %s", len(w), head(w, 6))
	}
	// The watcher sees writes: a store missing a table gets it back, by writing.
	d, _ = Open(path)
	d.Exec(`DROP TABLE holds_ended`)
	d.Close()
	before := len(writes())
	d, err = Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()
	if len(writes()) == before {
		t.Fatal("Open of a store missing holds_ended ran no write the watcher saw: the watcher is blind")
	}
}

// With nothing missing, Open does not take the init lock: a hug that holds
// it (an older one waiting 30 s under it) cannot hold up this open.
func TestOpenSkipsTheInitLockWhenNothingIsMissing(t *testing.T) {
	path := migratedStore(t)
	release, err := flock.Acquire(path + ".init.lock")
	if err != nil {
		t.Fatal(err)
	}
	defer release()
	done := make(chan error, 1)
	start := time.Now()
	go func() {
		d, err := Open(path)
		if err == nil {
			d.Close()
		}
		done <- err
	}()
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("OPEN with the init lock held elsewhere, nothing missing: %v", time.Since(start).Round(time.Millisecond))
	case <-time.After(3 * time.Second):
		release()
		<-done
		t.Fatalf("Open of a fully migrated store waited %v for the init lock another process held; nothing was missing, so it must not take it (6gtv7a)", time.Since(start).Round(time.Millisecond))
	}
}

// lockDatabase holds the write lock of the database at path from another
// connection, as an index pass in another hug process does, until release.
func lockDatabase(t *testing.T, path string) (release func()) {
	t.Helper()
	db, err := sql.Open("sqlite", "file:"+path+"?_pragma=busy_timeout(5000)")
	if err != nil {
		t.Fatal(err)
	}
	conn, err := db.Conn(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := conn.ExecContext(context.Background(), `BEGIN IMMEDIATE`); err != nil {
		t.Fatal(err)
	}
	var once sync.Once
	release = func() {
		once.Do(func() {
			conn.ExecContext(context.Background(), `ROLLBACK`)
			conn.Close()
			db.Close()
		})
	}
	t.Cleanup(release)
	return release
}

// storeMissing is a migrated store with one table dropped.
func storeMissing(t *testing.T, table string) string {
	t.Helper()
	path := migratedStore(t)
	d, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := d.Exec(`DROP TABLE ` + table); err != nil {
		t.Fatal(err)
	}
	d.Close()
	return path
}

type opened struct {
	err  error
	took time.Duration
}

func openers(path string, n int, start time.Time) chan opened {
	out := make(chan opened, n)
	for i := 0; i < n; i++ {
		go func() {
			d, err := Open(path)
			if err == nil {
				d.Close()
			}
			out <- opened{err, time.Since(start)}
		}()
	}
	return out
}

// openBound is how long an opener may take when the database stays locked.
// Open gives up on a locked schema update after 5 s (MigrateWait: past a
// short write such as a remember or a capture, inside the recall hook's 8 s
// budget and Claude's 15 s hook timeout); 5 s more is slack for a loaded CI
// machine. The old open took 30 s for the first opener and 60 s for the
// second.
const openBound = 10 * time.Second

// Check 2: with a table to add and the database write-locked by another
// connection, two openers both return within openBound, with a busy error
// that names a fix, and neither holds the init lock while it waits.
func TestOpenersDoNotQueueBehindALockedSchemaUpdate(t *testing.T) {
	path := storeMissing(t, "holds_ended")
	unlock := lockDatabase(t, path)
	start := time.Now()
	out := openers(path, 2, start)
	time.Sleep(1500 * time.Millisecond)
	rel, free, err := flock.TryAcquire(path + ".init.lock")
	if err != nil {
		t.Fatal(err)
	}
	if free {
		rel()
	} else {
		t.Errorf("at 1.5 s an opener held muninn.db.init.lock while it waited on the locked database; it must release it before waiting (6gtv7a)")
	}
	var got []opened
	deadline := time.After(openBound - time.Since(start))
	for len(got) < 2 {
		select {
		case r := <-out:
			got = append(got, r)
		case <-deadline:
			unlock()
			for len(got) < 2 {
				got = append(got, <-out)
			}
			t.Fatalf("with the database write-locked and a table to add, %d of 2 openers had not returned after %v (6gtv7a; once unlocked they returned at %v and %v)", 2-countBefore(got, openBound), openBound, got[0].took.Round(time.Millisecond), got[1].took.Round(time.Millisecond))
		}
	}
	for i, r := range got {
		t.Logf("OPENER %d, database locked throughout: returned after %v: %v", i+1, r.took.Round(time.Millisecond), r.err)
		if r.err == nil {
			t.Errorf("opener %d opened a store missing holds_ended without adding it (the database was locked throughout)", i+1)
			continue
		}
		if msg := r.err.Error(); !strings.Contains(msg, "SQLITE_BUSY") || !strings.Contains(msg, "\nfix: ") {
			t.Errorf("opener %d: want a busy error (SQLITE_BUSY, so hug mcp opens again later) ending in a fix: line, got:\n%s", i+1, msg)
		}
	}
	unlock()
	d, err := Open(path)
	if err != nil {
		t.Fatalf("open after the lock went: %v", err)
	}
	defer d.Close()
	var n int
	d.QueryRow(`SELECT count(*) FROM sqlite_master WHERE name='holds_ended'`).Scan(&n)
	if n != 1 {
		t.Fatal("holds_ended was not added once the database was free")
	}
}

func countBefore(got []opened, bound time.Duration) int {
	n := 0
	for _, r := range got {
		if r.took < bound {
			n++
		}
	}
	return n
}

// The lock goes after 1 s: both openers add the table between them and open.
func TestOpenersUpdateTheSchemaOnceTheLockGoes(t *testing.T) {
	path := storeMissing(t, "holds_ended")
	unlock := lockDatabase(t, path)
	start := time.Now()
	out := openers(path, 2, start)
	time.Sleep(time.Second)
	unlock()
	for i := 0; i < 2; i++ {
		select {
		case r := <-out:
			t.Logf("OPENER %d, lock released at 1 s: returned after %v: %v", i+1, r.took.Round(time.Millisecond), r.err)
			if r.err != nil {
				t.Errorf("opener %d after the lock went: %v", i+1, r.err)
			}
		case <-time.After(openBound - time.Since(start)):
			t.Fatalf("opener %d had not returned %v after start, the lock went at 1 s", i+1, openBound)
		}
	}
	d, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()
	var n int
	d.QueryRow(`SELECT count(*) FROM sqlite_master WHERE name='holds_ended'`).Scan(&n)
	if n != 1 {
		t.Fatal("holds_ended missing after both openers returned")
	}
}

// An init lock held for good (an older hug waiting under it) and a table to
// add: Open gives up within openBound with a fix: line instead of waiting.
func TestOpenGivesUpOnAHeldInitLock(t *testing.T) {
	path := storeMissing(t, "holds_ended")
	release, err := flock.Acquire(path + ".init.lock")
	if err != nil {
		t.Fatal(err)
	}
	defer release()
	start := time.Now()
	out := openers(path, 1, start)
	select {
	case r := <-out:
		t.Logf("OPEN with the init lock held for good: returned after %v: %v", r.took.Round(time.Millisecond), r.err)
		if r.err == nil || !strings.Contains(r.err.Error(), "\nfix: ") {
			t.Fatalf("want an error with a fix: line, got %v", r.err)
		}
	case <-time.After(openBound):
		release()
		r := <-out
		t.Fatalf("Open waited %v on an init lock held elsewhere (returned %v after release: %v)", openBound, r.took.Round(time.Millisecond), r.err)
	}
}
