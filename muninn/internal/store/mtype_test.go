package store

import (
	"path/filepath"
	"testing"
)

// TestMemoryTypeColumnOnOldDatabase: a database whose shortids schema is
// already at version 2 (every database in use before 2026-11-15) still gets
// units.mtype on open. The first version only migrated inside the shortids
// step, which returns early at version 2, and failed live with
// "no such column: mtype".
func TestMemoryTypeColumnOnOldDatabase(t *testing.T) {
	path := filepath.Join(t.TempDir(), "m.db")
	d, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := d.Exec(`ALTER TABLE units DROP COLUMN mtype`); err != nil {
		t.Fatal(err)
	}
	var v string
	if err := d.QueryRow(`SELECT value FROM meta WHERE key='shortids_schema'`).Scan(&v); err != nil || v != "2" {
		t.Fatalf("fixture: shortids_schema=%q err=%v", v, err)
	}
	d.Close()
	d, err = Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()
	src, _, _, err := d.Source("native", filepath.Join(t.TempDir(), "n.json"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := d.UpsertUnit(Unit{SourceID: src, Key: "k", ContentHash: "h", Kind: "memory", Body: "b", Type: "rule", Active: true}); err != nil {
		t.Fatalf("upsert after reopen: %v", err)
	}
	if _, err := d.UntypedUnits(0); err != nil {
		t.Fatalf("untyped after reopen: %v", err)
	}
}
