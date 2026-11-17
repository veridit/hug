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
	// Broken first however few (pbh9xq), then by count.
	if len(got) != 2 || got[0].Code != "missing-goal" || got[0].Severity != Broken || got[1].Code != "no-acceptance" || got[1].Count != 2 || got[1].Severity != Advice || got[1].List != "hug task lint --code no-acceptance" {
		t.Fatalf("%+v", got)
	}
	if all := Summarize(nil, true); len(all) != len(Codes) || all[0].Code != Codes[0].Code {
		t.Fatalf("all codes: %+v", all)
	}
}

// A closed task's findings, including files in its folder, are closed;
// findings on open tasks and on files no task owns are open (6neax1).
func TestOpenFindingsSplitsByStatus(t *testing.T) {
	tasks := []Task{{ID: "aaaaaa", Path: "a.aaaaaa.a.md", State: "done"}, {ID: "bbbbbb", Path: "g/b.bbbbbb.b.md", State: "todo"}, {ID: "cccccc", Path: "g/c.cccccc.c.md", State: "dropped"}}
	fs := []Finding{{Path: "a.aaaaaa.a.md"}, {Path: "g/b.bbbbbb.b.md"}, {Path: "g/cccccc/sub/log.bin"}, {Path: "broken.md"}, {Path: "aaaaaa/x.txt"}}
	open, closed := OpenFindings(fs, tasks)
	if len(open) != 2 || open[0].Path != "g/b.bbbbbb.b.md" || open[1].Path != "broken.md" || len(closed) != 3 {
		t.Fatalf("open %v closed %v", open, closed)
	}
}

// pbh9xq: broken findings are shown on open and closed tasks, broken first;
// advice is shown on open tasks and hidden on closed ones.
func TestShownFindingsBySeverity(t *testing.T) {
	tasks := []Task{{ID: "aaaaaa", Path: "a.aaaaaa.a.md", State: "done"}, {ID: "bbbbbb", Path: "b.bbbbbb.b.md", State: "todo"}}
	longTitle, cycle := "title is 90 characters (max 80): move detail into the Goal", "dependency cycle: aaaaaa -> bbbbbb -> aaaaaa"
	fs := []Finding{
		{Path: "b.bbbbbb.b.md", Problem: longTitle},                               // advice, open: shown
		{Path: "a.aaaaaa.a.md", Problem: longTitle},                               // advice, closed: hidden
		{Path: "a.aaaaaa.a.md", Problem: cycle},                                   // broken, closed: shown
		{Path: "b.bbbbbb.b.md", Problem: "done with unchecked acceptance item 2"}, // broken, open: shown
	}
	shown, hidden := ShownFindings(fs, tasks)
	if len(shown) != 3 || shown[0].Problem != cycle || shown[1].Severity() != Broken || shown[2].Path != "b.bbbbbb.b.md" || shown[2].Severity() != Advice {
		t.Fatalf("shown %+v", shown)
	}
	if len(hidden) != 1 || hidden[0].Path != "a.aaaaaa.a.md" || hidden[0].Severity() != Advice {
		t.Fatalf("hidden %+v", hidden)
	}
	for code, want := range map[string]string{"done-unchecked": Broken, "missing-goal": Broken, "dangling-dependency": Broken, "long-title": Advice, "no-acceptance": Advice, "section-order": Advice, "missing-section": Advice, "a-code-not-yet-known": Broken} {
		if got := Severity(code); got != want {
			t.Errorf("Severity(%s) = %s, want %s", code, got, want)
		}
	}
}
