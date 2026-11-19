package store

import (
	"path/filepath"
	"testing"
	"time"
)

// A store created before injections.delivered_ns (x8zzzn) gets the column on
// open, with 0 for its old rows, and MarkDelivered sets it once.
func TestInjectionsDeliveredColumnOnOldDatabase(t *testing.T) {
	path := filepath.Join(t.TempDir(), "m.db")
	d, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := d.Exec(`ALTER TABLE injections DROP COLUMN delivered_ns`); err != nil {
		t.Fatal(err)
	}
	if _, err := d.Exec(`INSERT INTO injections(session,unit_id,content_hash,harness,at_ns) VALUES('jcode:s',7,'h','jcode',1)`); err != nil {
		t.Fatal(err)
	}
	d.Close()
	d, err = Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()
	var got int64
	if err := d.QueryRow(`SELECT delivered_ns FROM injections WHERE session='jcode:s'`).Scan(&got); err != nil || got != 0 {
		t.Fatalf("old row: delivered_ns=%d err=%v", got, err)
	}
	first := time.Unix(100, 0)
	if err := d.MarkDelivered("jcode:s", []int64{7}, first); err != nil {
		t.Fatal(err)
	}
	if err := d.MarkDelivered("jcode:s", []int64{7}, time.Unix(200, 0)); err != nil {
		t.Fatal(err)
	}
	d.QueryRow(`SELECT delivered_ns FROM injections WHERE session='jcode:s'`).Scan(&got)
	if got != first.UnixNano() {
		t.Fatalf("delivered_ns=%d, want the first delivery %d", got, first.UnixNano())
	}
}
