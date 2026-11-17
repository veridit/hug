package task

import (
	"slices"
	"strings"
	"testing"
)

// g9dg32: the ids a subject names, from the forms used on hug's main.
func TestSubjectIDs(t *testing.T) {
	for subject, want := range map[string][]string{
		"hug task: a write walks the board once (44w1ev step 8)": {"44w1ev"},
		"commits name the model (t709he, 44w1ev)":                {"t709he", "44w1ev"},
		"rows show the branch (x1t1t1 check 4)":                  {"x1t1t1"},
		"recall nudges (qawmnb, CI run 43936088478)":             {"qawmnb"},
		"decided (j2vwk0, Jørgen's ruling)":                      {"j2vwk0"},
		"twice (abc123; abc123)":                                 {"abc123"},
		"CHANGELOG: v0.7.39":                                     nil,
		"CHANGELOG: 9xfnwr under Unreleased":                     nil, // not trailing
		"fix (Windows CI flakes, runs 43922289968, 43921466087)": nil,
		"move (and drop a stray conflict marker from 3187cf5)":   nil,
		"capital (ABC123)":                                       nil,
		"inner (abc123) then more words":                         nil,
	} {
		if got := SubjectIDs(subject); !slices.Equal(got, want) {
			t.Errorf("SubjectIDs(%q) = %v, want %v", subject, got, want)
		}
	}
}

// g9dg32 check 1 (planning half): a commit naming two tickets plans a
// landed note on each, once; a ticket that has it already gets nothing; an
// id with no ticket is skipped. Check 3: the report text.
func TestPlanLandedOncePerTicket(t *testing.T) {
	docs := map[string]*Document{
		"aaaaaa": doc(t, "aaaaaa", "wyvern", ""),
		"bbbbbb": doc(t, "bbbbbb", "", ""),
	}
	get := func(id string) *Document { return docs[id] }
	c := Commit{Hash: "abc1234def", Subject: "two tickets (aaaaaa, bbbbbb, zzzzzz)", Date: "2026-11-17"}
	got := PlanLanded([]Commit{c}, "main", get)
	if len(got) != 2 || got[0].ID != "aaaaaa" || got[1].ID != "bbbbbb" {
		t.Fatalf("%+v", got)
	}
	if got[0].Note != "landed abc1234 on main 2026-11-17: two tickets (aaaaaa, bbbbbb, zzzzzz)" || got[0].Owner != "wyvern" {
		t.Fatalf("note %q owner %q", got[0].Note, got[0].Owner)
	}
	want := `observed: commit abc1234 naming aaaaaa reached main; marked: landed abc1234; if wrong: hug task note aaaaaa "not landed: abc1234 (why)"`
	if r := got[0].Report(); r != want {
		t.Fatalf("report\n got %s\nwant %s", r, want)
	}
	// Once written, never again: a second pass, or another clone after sync.
	docs["aaaaaa"] = doc(t, "aaaaaa", "wyvern", "earlier note\n\n"+got[0].Note+"\n")
	if again := PlanLanded([]Commit{c}, "main", get); len(again) != 1 || again[0].ID != "bbbbbb" {
		t.Fatalf("again: %+v", again)
	}
}

// g9dg32 check 2 (planning half): a tag names each ticket once, even when
// several of its commits name it, and not again once written.
func TestPlanShippedOncePerTag(t *testing.T) {
	docs := map[string]*Document{"aaaaaa": doc(t, "aaaaaa", "", "")}
	get := func(id string) *Document { return docs[id] }
	cs := []Commit{{Hash: "1111111aa", Subject: "one (aaaaaa)"}, {Hash: "2222222bb", Subject: "two (aaaaaa step 2)"}}
	got := PlanShipped("v0.7.41", cs, get)
	if len(got) != 1 || got[0].Note != "shipped in v0.7.41 (1111111)" || !strings.Contains(got[0].Report(), "observed: release v0.7.41 contains commit 1111111 naming aaaaaa; marked: shipped in v0.7.41; if wrong: ") {
		t.Fatalf("%+v", got)
	}
	docs["aaaaaa"] = doc(t, "aaaaaa", "", got[0].Note+"\n")
	if again := PlanShipped("v0.7.41", cs, get); len(again) != 0 {
		t.Fatalf("again: %+v", again)
	}
	// v0.7.4 is not v0.7.41: the prefix ends at the space.
	if other := PlanShipped("v0.7.4", cs, get); len(other) != 1 {
		t.Fatalf("v0.7.4: %+v", other)
	}
}

// doc parses a minimal task file with id, assignee and Notes body.
func doc(t *testing.T, id, assignee, notes string) *Document {
	t.Helper()
	as := "[]"
	if assignee != "" {
		as = `["` + assignee + `"]`
	}
	text := "---\nid: " + id + "\ntitle: \"T\"\nstate: todo\nassignee: " + as + "\n---\n\n## Goal\nG\n\n## Acceptance\n- [ ] C\n\n## Notes\n" + notes
	d := Parse("x."+id+".t.md", []byte(text))
	if d.Task.ID != id {
		t.Fatalf("fixture %s: %+v", id, d.Findings)
	}
	return d
}

// A landed line keeps at most 80 characters of the subject, cut at a word,
// then an ellipsis; 80 characters exactly are kept whole.
func TestLandedNoteCutsSubject(t *testing.T) {
	c := Commit{Hash: "d74af5d0000", Date: "2026-11-17", Subject: "hug task tests: the commit-msg gate's cost is counted, not timed: 2 git and 1 awk per commit (t709he)"}
	want := "landed d74af5d on main 2026-11-17: hug task tests: the commit-msg gate's cost is counted, not timed: 2 git and 1…"
	if got := LandedNote(c, "main"); got != want {
		t.Fatalf("\n got %q\nwant %q", got, want)
	}
	short := Commit{Hash: "abc1234000", Date: "2026-11-17", Subject: strings.Repeat("é", 80)}
	if got := LandedNote(short, "main"); strings.HasSuffix(got, "…") || !strings.HasSuffix(got, strings.Repeat("é", 80)) {
		t.Fatalf("80 runes cut: %q", got)
	}
}
