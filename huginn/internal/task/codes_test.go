package task

import (
	"slices"
	"testing"
)

// Every code FindingCode can return has a plain meaning in Codes, and
// Codes lists nothing FindingCode cannot return (6neax1).
func TestCodesCoverFindingCode(t *testing.T) {
	want := []string{"missing-task-ref", "missing-file", "file-content", "other"}
	for _, c := range findingCodes {
		if !slices.Contains(want, c.code) {
			want = append(want, c.code)
		}
	}
	names := CodeNames()
	for _, w := range want {
		if !slices.Contains(names, w) {
			t.Errorf("code %s has no meaning in Codes", w)
		}
	}
	for _, n := range names {
		if !slices.Contains(want, n) {
			t.Errorf("Codes lists %s, which FindingCode never returns", n)
		}
	}
}

func TestSummarizeCountsByCode(t *testing.T) {
	fs := []Finding{{Problem: "missing Goal"}, {Problem: "Acceptance requires at least one item"}, {Problem: "Acceptance requires at least one item"}}
	got := Summarize(fs, false)
	if len(got) != 2 || got[0].Code != "no-acceptance" || got[0].Count != 2 || got[0].List != "hug task lint --code no-acceptance" || got[1].Code != "missing-goal" {
		t.Fatalf("%+v", got)
	}
	if all := Summarize(nil, true); len(all) != len(Codes) || all[0].Code != Codes[0].Code {
		t.Fatalf("all codes: %+v", all)
	}
}
