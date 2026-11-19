package store

import (
	"path/filepath"
	"testing"
)

// j13xs0: a store made before rescopes joined the fix trail has memory_fixes
// without old_scope, new_scope and mode. Opening it adds them and keeps the
// rows already there.
func TestMemoryFixesGainRescopeColumns(t *testing.T) {
	path := filepath.Join(t.TempDir(), "old.db")
	d, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, q := range []string{
		`DROP TABLE memory_fixes`,
		`CREATE TABLE memory_fixes (at_ns INTEGER NOT NULL, action TEXT NOT NULL, sid TEXT NOT NULL, new_sid TEXT NOT NULL DEFAULT '', session TEXT NOT NULL DEFAULT '', harness TEXT NOT NULL DEFAULT '', instance TEXT NOT NULL DEFAULT '', surface TEXT NOT NULL DEFAULT '', reason TEXT NOT NULL DEFAULT '', shown INTEGER NOT NULL DEFAULT 0, shown_ago_ns INTEGER NOT NULL DEFAULT 0, content_hash TEXT NOT NULL DEFAULT '')`,
		`INSERT INTO memory_fixes(at_ns, action, sid) VALUES(1, 'forget', 'k7m2qx')`,
	} {
		if _, err := d.Exec(q); err != nil {
			t.Fatal(err)
		}
	}
	d.Close()
	d, err = Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()
	var action, oldScope, newScope, mode string
	if err := d.QueryRow(`SELECT action, old_scope, new_scope, mode FROM memory_fixes WHERE sid='k7m2qx'`).Scan(&action, &oldScope, &newScope, &mode); err != nil || action != "forget" || oldScope != "" || newScope != "" || mode != "" {
		t.Fatalf("old row after migration: %q %q %q %q %v", action, oldScope, newScope, mode, err)
	}
}
