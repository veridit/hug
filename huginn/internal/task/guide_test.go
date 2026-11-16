package task

import (
	"strings"
	"testing"
)

func TestGuideAtStateChanges(t *testing.T) {
	epic := Task{ID: "eeeeee"}
	tk := Task{ID: "tttttt", Parent: "eeeeee", Acceptance: []Acceptance{{Text: "a"}, {Text: "b"}}}
	sib := Task{ID: "ssssss", Parent: "eeeeee", Status: "todo"}
	if g := Guide("new", tk, nil); g != nil {
		t.Fatal("new should not push into todos:", g)
	}
	if g := strings.Join(Guide("take", tk, nil), "|"); !strings.Contains(g, "one todo for tttttt") || strings.Contains(g, "1. a") {
		t.Fatal(g)
	}
	if g := Guide("check", tk, nil); g != nil {
		t.Fatal("open checks: next: says enough", g)
	}
	tk.Acceptance[0].Checked, tk.Acceptance[1].Checked = true, true
	if g := strings.Join(Guide("check", tk, nil), "|"); !strings.Contains(g, "close it") {
		t.Fatal(g)
	}
	if g := strings.Join(Guide("done", tk, []Task{epic, tk, sib}), "|"); !strings.Contains(g, "1 open task") {
		t.Fatal(g)
	}
	sib.Status = "done"
	if g := strings.Join(Guide("done", tk, []Task{epic, tk, sib}), "|"); !strings.Contains(g, "hug task done eeeeee") {
		t.Fatal(g)
	}
}
