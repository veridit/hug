package task

import (
	"errors"
	"strings"
	"testing"
)

func TestEpicParentRules(t *testing.T) {
	s := &Store{Root: t.TempDir()}
	epic := newTest(t, s, "Epic")
	child := newTest(t, s, "Child")
	grand := newTest(t, s, "Grandchild")
	if _, e := s.Set(child.ID, "parent", epic.ID); e != nil {
		t.Fatal(e)
	}
	ts, _ := s.Tasks()
	if !IsEpic(epic, ts) || len(Children(epic, ts)) != 1 {
		t.Fatal("epic not recognised")
	}
	// Three levels only: a child cannot be a parent, an epic cannot get one.
	var te *Error
	if _, e := s.Set(grand.ID, "parent", child.ID); !errors.As(e, &te) || te.Kind != "invalid-parent" {
		t.Fatal("grandchild accepted:", e)
	}
	if _, e := s.Set(epic.ID, "parent", grand.ID); !errors.As(e, &te) || te.Kind != "invalid-parent" {
		t.Fatal("epic got a parent:", e)
	}
	if _, e := s.Set(child.ID, "parent", child.ID); !errors.As(e, &te) {
		t.Fatal("self parent accepted")
	}
	// Done refuses an epic with an open child (exit 3), then succeeds.
	if _, e := s.Check(epic.ID, 1); e != nil {
		t.Fatal(e)
	}
	if _, e := s.Take(epic.ID, "x"); e != nil { // take before done (6ephqk)
		t.Fatal(e)
	}
	if _, e := s.Done(epic.ID); !errors.As(e, &te) || te.ExitCode() != 3 || te.Field != "children" {
		t.Fatal("epic done with open child:", e)
	}
	if _, e := s.Drop(child.ID, "test reason"); e != nil {
		t.Fatal(e)
	}
	if _, e := s.Done(epic.ID); e != nil {
		t.Fatal(e)
	}
	// none clears.
	if _, e := s.Set(child.ID, "parent", "none"); e != nil {
		t.Fatal(e)
	}
	ts, _ = s.Tasks()
	for _, x := range ts {
		if x.ID == child.ID && x.Parent != "" {
			t.Fatal("parent not cleared")
		}
	}
}

func TestLintParentsAndTitles(t *testing.T) {
	mk := func(path, id, extra string) *Document {
		return Parse(path, []byte(strings.Replace(strings.Replace(sample, "id: k7m2qx", "id: "+id, 1), "state:", extra+"state:", 1)))
	}
	long := strings.Repeat("x", 81)
	docs := []*Document{
		mk("a.aaaaaa.a.md", "aaaaaa", ""),
		mk("b.bbbbbb.b.md", "bbbbbb", "parent: aaaaaa\n"),
		mk("c.cccccc.c.md", "cccccc", "parent: bbbbbb\n"),
		mk("d.dddddd.d.md", "dddddd", "parent: zzzzzz\n"),
		mk("e.eeeeee.e.md", "eeeeee", "title: \""+long+"\"\n"),
	}
	var probs []string
	for _, f := range Lint(docs) {
		probs = append(probs, f.Path+" "+f.Problem)
	}
	all := strings.Join(probs, "\n")
	for _, want := range []string{"c.cccccc.c.md parent bbbbbb has parent aaaaaa", "d.dddddd.d.md missing parent zzzzzz", "e.eeeeee.e.md title is 81 characters"} {
		if !strings.Contains(all, want) {
			t.Fatalf("missing %q in:\n%s", want, all)
		}
	}
	if strings.Contains(all, "b.bbbbbb.b.md parent") {
		t.Fatal("valid parent flagged:\n" + all)
	}
}
