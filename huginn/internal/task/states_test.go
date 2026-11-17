package task

import (
	"strings"
	"testing"
)

// 6ephqk check 1: the tree has the four states and the stages Jørgen
// ruled, every node field is named after its node, and StatesText prints
// every node, field and setter from the one table.
func TestStatesTree(t *testing.T) {
	if got := strings.Join(StateNames(), " "); got != "todo doing done dropped" {
		t.Fatalf("states %q", got)
	}
	if got := strings.Join(StageNames("todo"), " "); got != "deferred depends design draft review ready" {
		t.Fatalf("todo stages (precedence order) %q", got)
	}
	if got := strings.Join(StageNames("doing"), " "); got != "active waiting depends review" {
		t.Fatalf("doing stages %q", got)
	}
	for _, s := range States {
		if s.Means == "" {
			t.Errorf("%s has no meaning", s.Name)
		}
		for _, f := range s.Fields {
			if !strings.HasPrefix(f.Name, s.Name+"_") {
				t.Errorf("%s field %s is not named after it", s.Name, f.Name)
			}
		}
		for _, g := range s.Stages {
			if g.Means == "" || g.SetBy == "" {
				t.Errorf("%s/%s lacks meaning or setter", s.Name, g.Name)
			}
			for _, f := range g.Fields {
				if !strings.HasPrefix(f.Name, g.Name+"_") {
					t.Errorf("%s/%s field %s is not named after it", s.Name, g.Name, f.Name)
				}
			}
		}
	}
	owners := NodeFields()
	if got := strings.Join(owners["review_by"], " "); got != "todo/review doing/review" {
		t.Fatalf("review_by owners %q", got)
	}
	text := StatesText()
	for _, s := range States {
		if !strings.Contains(text, s.Name+" ") {
			t.Errorf("states text lacks %s", s.Name)
		}
		for _, f := range s.Fields {
			if !strings.Contains(text, f.Name) {
				t.Errorf("states text lacks %s", f.Name)
			}
		}
		for _, g := range s.Stages {
			if !strings.Contains(text, "  "+g.Name+" ") || !strings.Contains(text, g.Means+" (set by "+string(g.SetBy)+")") {
				t.Errorf("states text lacks %s/%s with its setter", s.Name, g.Name)
			}
			for _, f := range g.Fields {
				if !strings.Contains(text, f.Name) || !strings.Contains(text, "(by "+string(f.SetBy)) {
					t.Errorf("states text lacks %s", f.Name)
				}
			}
		}
	}
	for _, want := range []string{"waiting_on       what it waits on (by agent, required)", "todo/review    -> doing/review   hug task take ID", "the first wins: deferred, depends, design, draft, review, ready"} {
		if !strings.Contains(text, want) {
			t.Errorf("states text lacks %q:\n%s", want, text)
		}
	}
}
