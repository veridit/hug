package task

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func newStore(t *testing.T) (*Store, string) {
	t.Helper()
	s := &Store{Root: t.TempDir(), DepsMode: "warn"}
	r, err := s.New(NewOptions{Title: "Bench", Goal: "measure", Acceptance: []string{"numbers"}})
	if err != nil {
		t.Fatal(err)
	}
	return s, r.Task.ID
}

func TestTaskFiles(t *testing.T) {
	s, id := newStore(t)
	// attach, then read back the identical bytes
	if _, ar, err := s.Attach(id, "bench/runs.txt", []byte("12ms\n"), "baseline", false); err != nil || !ar.Changed {
		t.Fatalf("attach: %+v %v", ar, err)
	}
	if b, err := s.ReadFile(id, "bench/runs.txt"); err != nil || string(b) != "12ms\n" {
		t.Fatalf("read back %q %v", b, err)
	}
	// identical re-attach changes nothing; other content needs replace
	if _, ar, _ := s.Attach(id, "bench/runs.txt", []byte("12ms\n"), "", false); ar.Changed {
		t.Error("identical re-attach changed something")
	}
	if _, _, err := s.Attach(id, "bench/runs.txt", []byte("13ms\n"), "", false); err == nil {
		t.Error("different content without replace was accepted")
	}
	if _, ar, err := s.Attach(id, "bench/runs.txt", []byte("13ms\n"), "", true); err != nil || ar.Old == "" {
		t.Fatalf("replace: %+v %v", ar, err)
	}
	// binary and oversized are refused and nothing is written
	for name, b := range map[string][]byte{"x.bin": {'a', 0, 'b'}, "big.txt": []byte(strings.Repeat("x", MaxFileBytes+1))} {
		if _, _, err := s.Attach(id, name, b, "", false); err == nil {
			t.Errorf("%s accepted", name)
		}
		if _, err := s.ReadFile(id, name); err == nil {
			t.Errorf("%s written despite refusal", name)
		}
	}
	// unsafe names
	for _, n := range []string{"../escape.txt", "/abs.txt", ".hidden", "a/../b.txt"} {
		if _, _, err := s.Attach(id, n, []byte("x"), "", false); err == nil {
			t.Errorf("unsafe name %q accepted", n)
		}
	}
	// the files folder holds files, never tasks: a .md inside it is not a task
	if _, _, err := s.Attach(id, "notes.md", []byte("# not a task\n"), "", false); err != nil {
		t.Fatal(err)
	}
	ts, err := s.Tasks()
	if err != nil || len(ts) != 1 {
		t.Fatalf("tasks = %d %v; files folder was read as tasks", len(ts), err)
	}
	// moving the task to a group carries its folder
	if _, err := s.MoveGroup(id, "perf"); err != nil {
		t.Fatal(err)
	}
	if b, err := s.ReadFile(id, "bench/runs.txt"); err != nil || string(b) != "13ms\n" {
		t.Fatalf("after move: %q %v", b, err)
	}
	if _, err := os.Stat(filepath.Join(s.Root, "perf", id, "notes.md")); err != nil {
		t.Errorf("folder did not move with the task: %v", err)
	}
	// lint: a missing reference, and a hand-placed binary file
	s.Note(id, "see file:missing.tsv and task:"+id+"/bench/runs.txt; prose like file:line is no reference")
	os.WriteFile(filepath.Join(s.Root, "perf", id, "hand.bin"), []byte{0, 1}, 0o644)
	fs, err := s.LintFiles([]string{".jcode/scratch/"})
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, f := range fs {
		got = append(got, f.Problem)
	}
	all := strings.Join(got, "\n")
	if !strings.Contains(all, "file:missing.tsv: file not found") || !strings.Contains(all, "file-not-text") || strings.Contains(all, "file:line") || strings.Contains(all, "bench/runs.txt") {
		t.Errorf("lint findings:\n%s", all)
	}
	// detach removes the file and notes it
	if _, err := s.Detach(id, "bench/runs.txt"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.ReadFile(id, "bench/runs.txt"); err == nil {
		t.Error("detached file still readable")
	}
}
