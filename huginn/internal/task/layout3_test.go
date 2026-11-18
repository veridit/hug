package task

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// w40cmk checks 2 and 3: in layout 3 a move is a line edit. new writes
// C/ID.md; move --top, --before, --after and a rebalance change only rank:
// lines; a group move changes only group:; nothing is renamed, the files
// folder stays at C/ID/, and updated: is never touched (Jørgen 01:21: a
// reorder is not an update).
func TestLayout3MovesEditLinesOnly(t *testing.T) {
	s := &Store{Root: t.TempDir()}
	a := newTest(t, s, "First task")
	b := newTest(t, s, "Second task")
	c := newTest(t, s, "Third task")
	for _, x := range []Task{a, b, c} {
		if x.Path != PathFor(x.ID) {
			t.Fatalf("new wrote %s, want %s", x.Path, PathFor(x.ID))
		}
	}
	if _, _, err := s.Attach(b.ID, "bench/runs.txt", []byte("13ms\n"), "", false); err != nil {
		t.Fatal(err)
	}
	// An updated: from another day, so a write that bumps it shows.
	for _, x := range []Task{a, b, c} {
		p := filepath.Join(s.Root, filepath.FromSlash(x.Path))
		raw, _ := os.ReadFile(p)
		old := strings.Replace(string(raw), "\nupdated: ", "\nupdated: 2020-01-02\nx-was: ", 1)
		if err := os.WriteFile(p, []byte(old), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	files := func() map[string]string {
		out := map[string]string{}
		filepath.WalkDir(s.Root, func(p string, d os.DirEntry, e error) error {
			if e == nil && !d.IsDir() {
				rel, _ := filepath.Rel(s.Root, p)
				b, _ := os.ReadFile(p)
				out[filepath.ToSlash(rel)] = string(b)
			}
			return nil
		})
		return out
	}
	order := func() string {
		ts, err := s.Tasks()
		if err != nil {
			t.Fatal(err)
		}
		var ids []string
		for _, x := range ts {
			ids = append(ids, x.ID)
		}
		return strings.Join(ids, " ")
	}
	moves := []struct {
		name  string
		do    func() (Result, error)
		order string
		field string
	}{
		{"top", func() (Result, error) { return s.Move(c.ID, MoveOptions{Top: true}) }, c.ID + " " + a.ID + " " + b.ID, "rank: "},
		{"before", func() (Result, error) { return s.Move(b.ID, MoveOptions{Before: a.ID}) }, c.ID + " " + b.ID + " " + a.ID, "rank: "},
		{"after", func() (Result, error) { return s.Move(c.ID, MoveOptions{After: a.ID}) }, b.ID + " " + a.ID + " " + c.ID, "rank: "},
		{"rebalance", func() (Result, error) { return s.Rebalance(nil) }, b.ID + " " + a.ID + " " + c.ID, "rank: "},
		{"group", func() (Result, error) { return s.MoveGroup(b.ID, "perf") }, b.ID + " " + a.ID + " " + c.ID, "group: "},
	}
	for _, m := range moves {
		before := files()
		r, err := m.do()
		if err != nil {
			t.Fatalf("%s: %v", m.name, err)
		}
		if len(r.Renames) != 0 {
			t.Errorf("%s renamed files: %+v", m.name, r.Renames)
		}
		after := files()
		if len(after) != len(before) {
			t.Fatalf("%s: %d files before, %d after", m.name, len(before), len(after))
		}
		for path, was := range before {
			now, ok := after[path]
			if !ok {
				t.Fatalf("%s: %s is gone (renamed?)", m.name, path)
			}
			if now == was {
				continue
			}
			if !strings.HasSuffix(path, ".md") || strings.Contains(path, "/"+b.ID+"/") {
				t.Fatalf("%s changed a file that is not a task: %s", m.name, path)
			}
			wl, nl := strings.SplitAfter(was, "\n"), strings.SplitAfter(now, "\n")
			if len(wl) != len(nl) {
				t.Fatalf("%s: %s went from %d to %d lines:\n%s", m.name, path, len(wl), len(nl), now)
			}
			for i := range wl {
				if wl[i] != nl[i] && !strings.HasPrefix(nl[i], m.field) {
					t.Fatalf("%s: %s line %d changed %q -> %q (only %q may)", m.name, path, i+1, wl[i], nl[i], m.field)
				}
			}
			if !strings.Contains(now, "\nupdated: 2020-01-02\n") {
				t.Fatalf("%s bumped updated: in %s:\n%s", m.name, path, now)
			}
		}
		if got := order(); got != m.order {
			t.Fatalf("%s: order %s, want %s", m.name, got, m.order)
		}
	}
	all, err := s.Tasks()
	if err != nil {
		t.Fatal(err)
	}
	got, err := Resolve(all, b.ID)
	if err != nil || got.Group != "perf" || got.Path != PathFor(b.ID) {
		t.Fatalf("after the group move: group %q path %s %v", got.Group, got.Path, err)
	}
	if rb, err := s.ReadFile(b.ID, "bench/runs.txt"); err != nil || string(rb) != "13ms\n" {
		t.Fatalf("files after the moves: %q %v", rb, err)
	}
	if _, err := os.Stat(filepath.Join(s.Root, "perf")); !os.IsNotExist(err) {
		t.Fatalf("a group move made a folder: %v", err)
	}
}

// w40cmk check 4: an old file name, with or without its group and .md, and
// the slug, still find the task after it moved to C/ID.md.
func TestLayout3OldNamesResolve(t *testing.T) {
	s := &Store{Root: t.TempDir()}
	x := newTest(t, s, "Task state and stage hard")
	ts, err := s.Tasks()
	if err != nil {
		t.Fatal(err)
	}
	for _, ref := range []string{
		"00i." + x.ID + ".task-state-stage-hard",
		"00i." + x.ID + ".task-state-stage-hard.md",
		"later/00i." + x.ID + ".task-state-stage-hard.md",
		x.ID + ".task-state-stage-hard.md",
		"task:00i." + x.ID + ".old-slug",
		PathFor(x.ID),
		x.Slug,
		x.ID,
	} {
		got, err := Resolve(ts, ref)
		if err != nil || got.ID != x.ID {
			t.Errorf("%q: %v %v", ref, got.ID, err)
		}
	}
	// A dotted word that names no task file is not taken for one.
	if got, err := Resolve(ts, "v0.7.55"); err == nil {
		t.Errorf("v0.7.55 resolved to %s", got.ID)
	}
}

func TestLayout3KeepsLegacyBareSlugAsAlias(t *testing.T) {
	s := &Store{Root: t.TempDir()}
	old := "a.8q6v7b.seal-the-ir-and-typed-ast.md"
	raw := []byte("---\nid: 8q6v7b\ntitle: Seal IR and typed AST\nstate: todo\naliases: [TASK-2034]\nupdated: 2026-11-17\n---\n\n## Goal\nG\n")
	if e := os.WriteFile(filepath.Join(s.Root, old), raw, 0o644); e != nil {
		t.Fatal(e)
	}
	plan, e := s.PlanLayout3()
	if e != nil {
		t.Fatal(e)
	}
	if _, e := s.ApplyLayout3(plan); e != nil {
		t.Fatal(e)
	}
	ts, e := s.Tasks()
	if e != nil {
		t.Fatal(e)
	}
	for _, ref := range []string{"seal-the-ir-and-typed-ast", "seal-ir-typed-ast", "TASK-2034", old} {
		if got, e := Resolve(ts, ref); e != nil || got.ID != "8q6v7b" {
			t.Errorf("legacy ref %q: id=%s error=%v", ref, got.ID, e)
		}
	}
	got, e := os.ReadFile(filepath.Join(s.Root, filepath.FromSlash(PathFor("8q6v7b"))))
	if e != nil || !strings.Contains(string(got), "updated: 2026-11-17\n") {
		t.Fatalf("migration changed updated: e=%v\n%s", e, got)
	}
}
