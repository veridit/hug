package store

import (
	"database/sql"
	"path/filepath"
	"testing"
)

// The read-only URI must open a real database whose path has a space, as
// "~/Library/Application Support/muninn/muninn.db" does.
func TestFileURIOpensReadOnly(t *testing.T) {
	path := filepath.Join(t.TempDir(), "Application Support", "muninn.db")
	s, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	s.Close()
	db, err := sql.Open("sqlite", FileURI(path)+"?mode=ro&_pragma=query_only(1)")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	var n int
	if err := db.QueryRow(`SELECT count(*) FROM sqlite_master`).Scan(&n); err != nil || n == 0 {
		t.Fatalf("read-only open: n=%d err=%v", n, err)
	}
}
