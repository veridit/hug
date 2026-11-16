package task

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
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

// The indexed ResolveDependency gives the scan's answers on boards with
// duplicate ids, shared aliases and aliases equal to ids, also after the
// slice is sorted in place (44w1ev).
func TestResolveDependencyIndexMatchesScan(t *testing.T) {
	var ts []Task
	for i := 0; i < 200; i++ {
		id := fmt.Sprintf("a%05d", i)
		t := Task{ID: id, Aliases: []string{fmt.Sprintf("OLD-%d", i)}}
		switch i % 10 {
		case 1:
			t.ID = "a00000" // duplicate id
		case 2:
			t.Aliases = append(t.Aliases, "SHARED-1") // alias on several tasks
		case 3:
			t.Aliases = append(t.Aliases, "a00004", "a00004") // alias equal to an id, named twice
		}
		ts = append(ts, t)
	}
	refs := []string{"a00000", "A00005", "task:a00007", "old-12", "SHARED-1", "a00004", "a00013", "nope", "OLD-3", ""}
	scan := func(ref string) (Task, error) { // the board split into small slices: no index
		ref = BareRef(ref)
		var hit []Task
		for _, t := range ts {
			if strings.EqualFold(t.ID, ref) {
				hit = append(hit, t)
			}
		}
		if len(hit) == 1 {
			return hit[0], nil
		}
		if len(hit) > 1 {
			return Task{}, &Error{Kind: "duplicate-id", ID: ref, Fix: "hug task lint"}
		}
		for _, t := range ts {
			if containsFold(t.Aliases, ref) {
				hit = append(hit, t)
			}
		}
		if len(hit) == 1 {
			return hit[0], nil
		}
		return Task{}, &Error{Kind: "dangling-dependency", Value: "no task has the id " + ref, Fix: "hug task ls"}
	}
	check := func(when string) {
		for _, r := range refs {
			got, ge := ResolveDependency(ts, r)
			want, we := scan(r)
			if got.ID != want.ID || fmt.Sprint(ge) != fmt.Sprint(we) {
				t.Fatalf("%s: %q: got %q %v, want %q %v", when, r, got.ID, ge, want.ID, we)
			}
		}
	}
	check("fresh")
	sort.Slice(ts, func(i, j int) bool { return ts[i].ID > ts[j].ID })
	check("sorted in place")
}

// LintFiles lists group folders once instead of an Lstat chain per task
// (44w1ev); a task folder that is a symlink is still never followed, and a
// symlinked task file is still not read.
func TestLintFilesSymlinkedFolderNotFollowed(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	if e := os.WriteFile(filepath.Join(outside, "bin"), []byte{0, 1, 2}, 0644); e != nil {
		t.Fatal(e)
	}
	if e := os.WriteFile(filepath.Join(root, "a0.k7m2qx.recall-gate.md"), []byte(sample), 0644); e != nil {
		t.Fatal(e)
	}
	if e := os.Symlink(outside, filepath.Join(root, "k7m2qx")); e != nil {
		t.Skip("no symlinks here:", e)
	}
	if e := os.Symlink(filepath.Join(outside, "bin"), filepath.Join(root, "a1.m3n4p5.linked.md")); e != nil {
		t.Fatal(e)
	}
	s := &Store{Root: root}
	fs, e := s.LintFiles([]string{"\x01\x02"})
	if e != nil {
		t.Fatal(e)
	}
	// Following the folder link would report outside/bin as not text, and
	// reading the file link would report it as citing the forbidden bytes.
	if len(fs) != 0 {
		t.Fatalf("followed a symlink: %+v", fs)
	}
	// The same binary file in a real task folder is reported: the check runs.
	if e := os.Remove(filepath.Join(root, "k7m2qx")); e != nil {
		t.Fatal(e)
	}
	if e := os.MkdirAll(filepath.Join(root, "k7m2qx"), 0755); e != nil {
		t.Fatal(e)
	}
	if e := os.WriteFile(filepath.Join(root, "k7m2qx", "bin"), []byte{0, 1, 2}, 0644); e != nil {
		t.Fatal(e)
	}
	if fs, _ = s.LintFiles(nil); len(fs) != 1 || !strings.Contains(fs[0].Problem, "file-not-text") {
		t.Fatalf("a binary task file was not reported: %+v", fs)
	}
}
