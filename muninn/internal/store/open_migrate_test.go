package store

import (
	"database/sql"
	"path/filepath"
	"testing"
)

// 6gtv7a: Open now checks what is missing before it writes. Every kind of
// gap an older store can have must still be filled: a table, an index, a
// trigger, a column, the old-hug view, a backfill, and the short ids of a
// store from before them.
func TestOpenFillsEveryKindOfGap(t *testing.T) {
	count := func(t *testing.T, d *DB, q string, args ...any) int {
		t.Helper()
		var n int
		if err := d.QueryRow(q, args...).Scan(&n); err != nil {
			t.Fatalf("%s: %v", q, err)
		}
		return n
	}
	object := `SELECT count(*) FROM sqlite_master WHERE name=?`
	for _, c := range []struct {
		name   string
		break_ []string
		check  func(t *testing.T, d *DB)
	}{
		{"a table", []string{`DROP TABLE holds_ended`}, func(t *testing.T, d *DB) {
			if count(t, d, object, "holds_ended") != 1 {
				t.Fatal("holds_ended not added")
			}
		}},
		{"an index", []string{`DROP INDEX unit_anchors_anchor`}, func(t *testing.T, d *DB) {
			if count(t, d, object, "unit_anchors_anchor") != 1 {
				t.Fatal("unit_anchors_anchor not added")
			}
		}},
		{"a trigger", []string{`DROP TRIGGER units_au`}, func(t *testing.T, d *DB) {
			if count(t, d, object, "units_au") != 1 {
				t.Fatal("units_au not added")
			}
		}},
		{"a column", []string{`ALTER TABLE units DROP COLUMN mtrigger`}, func(t *testing.T, d *DB) {
			if count(t, d, `SELECT count(*) FROM pragma_table_info('units') WHERE name='mtrigger'`) != 1 {
				t.Fatal("units.mtrigger not added")
			}
		}},
		{"the old-hug view", []string{`DROP VIEW graph_state`, `CREATE VIEW graph_state(id, state) AS SELECT 1, NULL WHERE 0`}, func(t *testing.T, d *DB) {
			var def string
			d.QueryRow(`SELECT sql FROM sqlite_master WHERE name='graph_state'`).Scan(&def)
			if def != OldHugView {
				t.Fatalf("graph_state is %q, want OldHugView", def)
			}
		}},
		{"a backfill", []string{`DELETE FROM sketches`, `DELETE FROM qvecs`}, func(t *testing.T, d *DB) {
			if count(t, d, `SELECT count(*) FROM sketches`) != 1 || count(t, d, `SELECT count(*) FROM qvecs`) != 1 {
				t.Fatal("sketches and qvecs not backfilled from embeddings")
			}
		}},
		{"the unit_communities backfill", []string{`DELETE FROM unit_communities`}, func(t *testing.T, d *DB) {
			if count(t, d, `SELECT count(*) FROM unit_communities`) != 1 {
				t.Fatal("unit_communities not backfilled")
			}
		}},
		{"short ids", []string{`DELETE FROM meta WHERE key='shortids_schema'`}, func(t *testing.T, d *DB) {
			if d.GetMeta("shortids_schema") != "2" {
				t.Fatal("shortids_schema not set again")
			}
		}},
	} {
		t.Run(c.name, func(t *testing.T) {
			path := migratedStore(t)
			d, err := Open(path)
			if err != nil {
				t.Fatal(err)
			}
			for _, q := range c.break_ {
				if _, err := d.Exec(q); err != nil {
					t.Fatalf("%s: %v", q, err)
				}
			}
			d.Close()
			// What Open saw as missing, before it filled it.
			raw, err := sql.Open("sqlite", "file:"+path)
			if err != nil {
				t.Fatal(err)
			}
			missing, err := (&DB{DB: raw, Path: path}).missingSchema()
			raw.Close()
			if err != nil || len(missing) == 0 {
				t.Fatalf("missingSchema after breaking %s: %v %v, want it named", c.name, missing, err)
			}
			writes := watchWrites(t, path)
			d, err = Open(path)
			if err != nil {
				t.Fatal(err)
			}
			c.check(t, d)
			d.Close()
			if len(writes()) == 0 {
				t.Fatal("no write seen")
			}
			// Filled once: the next open writes nothing.
			n := len(writes())
			d, err = Open(path)
			if err != nil {
				t.Fatal(err)
			}
			d.Close()
			if w := writes()[n:]; len(w) != 0 {
				t.Fatalf("after %s was filled, the next open still wrote: %s", c.name, head(w, 6))
			}
		})
	}
}

// A new store gets the whole schema, and its second open writes nothing.
func TestOpenNewStoreThenNoWrites(t *testing.T) {
	path := filepath.Join(t.TempDir(), "new.db")
	writes := watchWrites(t, path)
	d, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	var n int
	d.QueryRow(`SELECT count(*) FROM sqlite_master WHERE name IN ('units','units_fts','units_ai','memory_ids','graph_state')`).Scan(&n)
	d.Close()
	if n != 5 {
		t.Fatalf("new store has %d of 5 checked objects", n)
	}
	before := len(writes())
	d, err = Open(path)
	if err != nil {
		t.Fatal(err)
	}
	d.Close()
	if w := writes()[before:]; len(w) != 0 {
		t.Fatalf("second open of a new store wrote: %s", head(w, 6))
	}
}

// Six first runs at once on a new store (2026-11-10: two failed "schema:
// database is locked"): each opens, one of them creates the schema, the
// others find it there.
func TestParallelFirstOpens(t *testing.T) {
	path := filepath.Join(t.TempDir(), "first.db")
	errs := make(chan error, 6)
	for i := 0; i < 6; i++ {
		go func() {
			d, err := Open(path)
			if err == nil {
				err = d.Close()
			}
			errs <- err
		}()
	}
	for i := 0; i < 6; i++ {
		if err := <-errs; err != nil {
			t.Errorf("parallel first open: %v", err)
		}
	}
}
