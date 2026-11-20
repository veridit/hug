package task

import (
	"errors"
	"strings"
	"testing"
)

func TestPromote(t *testing.T) {
	s := &Store{Root: t.TempDir()}
	epic := newTest(t, s, "Epic")
	r, e := s.New(NewOptions{Title: "Source", Goal: "g", Acceptance: []string{"first", "second thing to do", "third"}, Type: "bug"})
	if e != nil {
		t.Fatal(e)
	}
	src := r.Task
	if _, e = s.Set(src.ID, "parent", epic.ID); e != nil {
		t.Fatal(e)
	}
	p, e := s.Promote(src.ID, 2)
	if e != nil {
		t.Fatal(e)
	}
	nt := p.Task
	if nt.Title != "second thing to do" || nt.Parent != epic.ID || nt.Type != "bug" {
		t.Fatalf("new task %+v", nt)
	}
	// Ranked right after the source.
	ts, _ := s.Tasks()
	Sort(ts)
	for i, x := range ts {
		if x.ID == src.ID && (i+1 >= len(ts) || ts[i+1].ID != nt.ID) {
			t.Fatal("not ranked after source")
		}
	}
	d, _ := s.Get(src.ID)
	if got := d.Task.Acceptance[1].Text; !strings.Contains(got, "done when task:"+nt.ID+" is done") {
		t.Fatal(got)
	}
	// Promoting again is refused; ticked items too.
	var te *Error
	if _, e = s.Promote(src.ID, 2); !errors.As(e, &te) || te.Kind != "already-promoted" {
		t.Fatal(e)
	}
	// Done on the new task ticks the waiting item.
	if _, e = s.Check(nt.ID, 1); e != nil {
		t.Fatal(e)
	}
	if _, e = s.Take(nt.ID, "x"); e != nil { // take before done (6ephqk)
		t.Fatal(e)
	}
	res, e := s.Done(nt.ID)
	if e != nil {
		t.Fatal(e)
	}
	d, _ = s.Get(src.ID)
	if !d.Task.Acceptance[1].Checked || d.Task.Acceptance[0].Checked {
		t.Fatal("item 2 not ticked (or wrong item)", d.Task.Acceptance, res.Warnings)
	}
	// Promoting from an epic makes a child of that epic.
	q, e := s.Promote(epic.ID, 1)
	if e != nil {
		t.Fatal(e)
	}
	if q.Task.Parent != epic.ID {
		t.Fatal("epic item did not become a child", q.Task)
	}
}

// heype5 check 1: the child of a task with Purpose and Principle starts with
// both (and Relevant details), so hug task lint reports it clean.
func TestPromoteChildLintsClean(t *testing.T) {
	s := &Store{Root: t.TempDir()}
	r, e := s.New(NewOptions{Title: "Source", Goal: "g", Purpose: "Why we do it.", Principle: "What we do and do not do.\n\nSecond paragraph.", Details: "Where to look.", Acceptance: []string{"first", "second"}})
	if e != nil {
		t.Fatal(e)
	}
	p, e := s.Promote(r.Task.ID, 2)
	if e != nil {
		t.Fatal(e)
	}
	d, e := s.Get(p.Task.ID)
	if e != nil {
		t.Fatal(e)
	}
	for sec, want := range map[string]string{"Purpose": "Why we do it.", "Principle": "What we do and do not do.\n\nSecond paragraph.", "Relevant details": "Where to look.\n\nPromoted from item 2 of task:" + r.Task.ID + ".", "Goal": "second"} {
		if got := d.sectionText(sec); got != want {
			t.Errorf("%s: %q, want %q", sec, got, want)
		}
	}
	all, e := s.Tasks()
	if e != nil {
		t.Fatal(e)
	}
	var docs []*Document
	for _, x := range all {
		doc, e := s.Get(x.ID)
		if e != nil {
			t.Fatal(e)
		}
		docs = append(docs, doc)
	}
	for _, f := range Lint(docs) {
		t.Errorf("finding: %s", f)
	}
	// A source without them gives a child without them: nothing is invented.
	bare, e := s.New(NewOptions{Title: "Bare", Goal: "g", Acceptance: []string{"one"}})
	if e != nil {
		t.Fatal(e)
	}
	q, e := s.Promote(bare.Task.ID, 1)
	if e != nil {
		t.Fatal(e)
	}
	if d, _ := s.Get(q.Task.ID); contains(d.Task.Sections, "Purpose") || contains(d.Task.Sections, "Principle") {
		t.Fatalf("invented sections: %v", d.Task.Sections)
	}
}
