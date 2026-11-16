package task

import "testing"

func TestNextSkipsEpicsAndEpicFilter(t *testing.T) {
	s := &Store{Root: t.TempDir()}
	epic := newTest(t, s, "Epic first")
	child := newTest(t, s, "Child")
	other := newTest(t, s, "Other")
	if _, e := s.Set(child.ID, "parent", epic.ID); e != nil {
		t.Fatal(e)
	}
	ts, _ := s.Tasks()
	r, e := Next(ts, Filter{})
	if e != nil || r.Task.ID != child.ID {
		t.Fatalf("next = %v %v, want child (epic skipped)", r.Task.ID, e)
	}
	r, e = Next(ts, Filter{Epic: epic.ID})
	if e != nil || r.Task.ID != child.ID {
		t.Fatal(r.Task.ID, e)
	}
	n := 0
	for _, x := range ts {
		if InEpic(x, ts, epic.ID) {
			n++
		}
		if x.ID == other.ID && InEpic(x, ts, epic.ID) {
			t.Fatal("other in epic")
		}
	}
	if n != 2 {
		t.Fatal("epic view should hold epic and child, got", n)
	}
}
