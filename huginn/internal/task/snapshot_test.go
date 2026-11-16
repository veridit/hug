package task

import (
	"os"
	"path/filepath"
	"testing"
)

// A Store with Snapshot reads the board once per command and again after
// each of its own writes (44w1ev): a stale board after a write would make
// the next step of the same command act on old files.
func TestSnapshotReadsOnceAndAfterWrites(t *testing.T) {
	root := t.TempDir()
	if e := os.WriteFile(filepath.Join(root, "a0.k7m2qx.recall-gate.md"), []byte(sample), 0644); e != nil {
		t.Fatal(e)
	}
	plain := &Store{Root: root}
	reads := 0
	s := &Store{Root: root, Snapshot: true, Refresh: func() ([]Task, error) { reads++; return plain.Tasks() }}
	for i := 0; i < 3; i++ {
		if ts, e := s.Tasks(); e != nil || len(ts) != 1 {
			t.Fatal(ts, e)
		}
	}
	if reads != 1 {
		t.Fatalf("3 Tasks calls read the board %d times, want 1", reads)
	}
	ts, _ := s.Tasks()
	ts[0].Title = "mutated by a caller"
	if again, _ := s.Tasks(); again[0].Title != "Recall gate" {
		t.Fatal("a caller changed the kept board:", again[0].Title)
	}
	if _, e := s.Set("k7m2qx", "title", "Renamed"); e != nil {
		t.Fatal(e)
	}
	if ts, _ := s.Tasks(); ts[0].Title != "Renamed" {
		t.Fatal("board after a write is stale:", ts[0].Title)
	}
	before := reads
	if e := os.WriteFile(filepath.Join(root, "a1.m3n4p5.other.md"), []byte("---\nid: m3n4p5\ntitle: Other\nstatus: todo\ntype: bug\n---\n\n## Goal\nx\n\n## Acceptance\n- [ ] y\n"), 0644); e != nil {
		t.Fatal(e)
	}
	s.Invalidate()
	if ts, _ := s.Tasks(); len(ts) != 2 || reads != before+1 {
		t.Fatal("Invalidate did not reread the board", len(ts), reads-before)
	}
}
