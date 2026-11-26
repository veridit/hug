package run

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/veridit/hug/internal/flock"
	"github.com/veridit/hug/muninn/internal/store"
)

// 6gtv7a: an open that gave up because another hug held the schema update
// (its init lock; no SQLITE_BUSY in the text) is busy too, so hug mcp opens
// memory again instead of reporting memory unavailable for good.
func TestMCPRetriesAnOpenBehindTheInitLock(t *testing.T) {
	path := filepath.Join(t.TempDir(), "m.db")
	d, err := store.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := d.Exec(`DROP TABLE holds_ended`); err != nil {
		t.Fatal(err)
	}
	d.Close()
	release, err := flock.Acquire(path + ".init.lock")
	if err != nil {
		t.Fatal(err)
	}
	defer release()
	old := store.MigrateWait
	store.MigrateWait = 200 * time.Millisecond
	defer func() { store.MigrateWait = old }()
	_, err = store.Open(path)
	if err == nil {
		t.Fatal("open with the init lock held and a table to add: want an error")
	}
	if !busy(err) {
		t.Fatalf("busy(%q) = false: hug mcp would give up on memory instead of opening it again", err)
	}
}
