package trigger

import (
	"strings"
	"testing"
)

// jrj1vq check 1: the canonical form is what equality, storage and the
// index see; nothing stated means no trigger at all.
func TestCanonical(t *testing.T) {
	got, err := Canonical(&Trigger{When: "  reviewing   a STATBUS ticket ", Case: &Case{Paths: []string{"b.go", " a.go", "b.go", ""}, Tools: []string{" "}}})
	if err != nil {
		t.Fatal(err)
	}
	if got.When != "reviewing a STATBUS ticket" || strings.Join(got.Case.Paths, ",") != "a.go,b.go" || got.Case.Tools != nil || got.Case.Scope != nil {
		t.Fatalf("canonical %+v %+v", got, got.Case)
	}
	if want := `{"case":{"paths":["a.go","b.go"]},"when":"reviewing a STATBUS ticket"}`; Encode(got) != want {
		t.Fatalf("encode %s, want %s", Encode(got), want)
	}
	for _, empty := range []*Trigger{nil, {}, {When: "  "}, {Case: &Case{Paths: []string{" "}}}} {
		if c, err := Canonical(empty); c != nil || err != nil || Encode(empty) != "" || !Empty(empty) {
			t.Fatalf("%+v is no trigger: %+v %v %q", empty, c, err, Encode(empty))
		}
	}
	if !Equal(&Trigger{Case: &Case{Tools: []string{"b", "a"}}}, &Trigger{Case: &Case{Tools: []string{"a", "b", "a"}}}) || Equal(&Trigger{When: "x"}, &Trigger{WhenNot: "x"}) || !Equal(nil, &Trigger{}) {
		t.Fatal("equality is on the canonical form")
	}
	back, err := Decode(Encode(got))
	if err != nil || !Equal(back, got) {
		t.Fatalf("decode %+v %v", back, err)
	}
	if d, err := Decode(""); d != nil || err != nil {
		t.Fatal("empty decodes to no trigger")
	}
}

func TestLimitsEndWithFix(t *testing.T) {
	long := strings.Repeat("x", MaxText+1)
	many := make([]string, MaxItems+1)
	for i := range many {
		many[i] = string(rune('a' + i))
	}
	for _, bad := range []*Trigger{{When: long}, {WhenNot: long}, {Case: &Case{Tools: many}}, {Case: &Case{Paths: []string{strings.Repeat("p", MaxItem+1)}}}} {
		_, err := Canonical(bad)
		if err == nil || !strings.Contains(err.Error(), "\nfix: hug mem remember --when") {
			t.Fatalf("%v", err)
		}
	}
}

func TestLines(t *testing.T) {
	got := strings.Join(Lines(&Trigger{When: "w", WhenNot: "n", Case: &Case{Scope: []string{"project:hug"}, Tools: []string{"git"}}}), "|")
	if got != "case: scope project:hug; tools git|when: w|when not: n" {
		t.Fatal(got)
	}
	if Lines(nil) != nil {
		t.Fatal("no trigger, no lines")
	}
}
