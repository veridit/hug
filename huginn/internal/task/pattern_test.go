package task

import (
	"strings"
	"testing"
)

func TestNewWithPatternAndLint(t *testing.T) {
	s := &Store{Root: t.TempDir()}
	r, e := s.New(NewOptions{Title: "Patterned", Goal: "g", Purpose: "why", Principle: "how", Details: "facts", Acceptance: []string{"a"}})
	if e != nil {
		t.Fatal(e)
	}
	d, _ := s.Get(r.Task.ID)
	b := string(d.Bytes())
	for _, want := range []string{"## Goal\ng\n", "## Purpose\nwhy\n", "## Principle\nhow\n", "## Relevant details\nfacts\n", "## Acceptance"} {
		if !strings.Contains(b, want) {
			t.Fatalf("missing %q in\n%s", want, b)
		}
	}
	plain, _ := s.New(NewOptions{Title: "Plain", Goal: "g", Acceptance: []string{"a"}})
	var docs []*Document
	ts, _ := s.Tasks()
	for _, x := range ts {
		d, e := s.Get(x.ID)
		if e != nil {
			t.Fatal(e)
		}
		docs = append(docs, d)
	}
	var got []string
	for _, f := range Lint(docs) {
		if FindingCode(f.Problem) == "missing-section" {
			got = append(got, f.Path)
		}
	}
	if len(got) != 1 || !strings.Contains(got[0], plain.Task.ID) {
		t.Fatal(got)
	}
}
