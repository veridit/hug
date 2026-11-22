package task

import (
	"errors"
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
	for _, want := range []string{"## Goal\ng\n", "## Purpose\nwhy\n", "## Principle\nhow\n", "## Details\nfacts\n", "## Acceptance"} {
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
	if len(got) != 2 || !strings.Contains(got[0], plain.Task.ID) || got[0] != got[1] {
		t.Fatal(got)
	}
}

// Check 1 (fkpyg3): new names every missing part in one reply, with one
// fix that has all of them.
func TestNewNamesEveryMissingPart(t *testing.T) {
	s := &Store{Root: t.TempDir()}
	_, e := s.New(NewOptions{Title: "x", Pattern: true})
	var te *Error
	if !errors.As(e, &te) || te.Kind != "incomplete-task" {
		t.Fatal(e)
	}
	var codes []string
	for _, p := range te.Problems {
		codes = append(codes, p.Code)
	}
	if strings.Join(codes, ",") != "missing-goal,missing-section,missing-section,no-acceptance" {
		t.Fatal(codes)
	}
	for _, flag := range []string{"--goal", "--purpose", "--principle", "--accept"} {
		if !strings.Contains(te.Fix, flag+" ") {
			t.Fatalf("fix lacks %s: %s", flag, te.Fix)
		}
	}
	if !strings.HasPrefix(te.Fix, `hug task new "x" `) || strings.Count(te.Error(), "fix:") != 1 {
		t.Fatal(te.Error())
	}
	// A long title is named with the rest, and given a slot in the fix.
	_, e = s.New(NewOptions{Title: strings.Repeat("Long title ", 8), Goal: "G", Acceptance: []string{"C"}, Pattern: true})
	if !errors.As(e, &te) || len(te.Problems) != 3 || te.Problems[0].Code != "long-title" || !strings.Contains(te.Fix, `"short title" --goal "G" --purpose "..." --principle "..." --accept "C"`) {
		t.Fatal(e)
	}
}

// Check 2: one task, three problems, one combined fix.
func TestCombinedFixForOneTask(t *testing.T) {
	title := strings.Repeat("x", 87)
	d := Parse("q.abc123.long.md", []byte("---\nid: abc123\ntitle: "+title+"\nstate: todo\n---\n\n## Goal\ng\n\n## Acceptance\n- [ ] a\n"))
	fs := Lint([]*Document{d})
	groups := ByTask(fs)
	if len(groups) != 1 || len(groups[0].Findings) != 3 {
		t.Fatalf("%+v", groups)
	}
	want := `hug task set abc123 --title "..." --purpose "..." --principle "..."`
	if groups[0].Fix != want {
		t.Fatalf("fix %q\nwant %q", groups[0].Fix, want)
	}
}

// Check 3: what new accepts, lint never flags. Every code New can report
// is a code lint reports, and a task New wrote is lint-clean.
func TestNewAndLintShareTheCheckList(t *testing.T) {
	s := &Store{Root: t.TempDir()}
	r, e := s.New(NewOptions{Title: "Clean", Goal: "g", Purpose: "p", Principle: "q", Acceptance: []string{"a"}, Pattern: true})
	if e != nil {
		t.Fatal(e)
	}
	d, _ := s.Get(r.Task.ID)
	if fs := Lint([]*Document{d}); len(fs) != 0 {
		t.Fatal(fs)
	}
	// The write-time codes, each one a lint code too.
	write := map[string]bool{}
	for _, p := range DescriptionProblems(strings.Repeat("t", 90), "", false, false, 0) {
		write[p.Code] = true
		if !contains(CodeNames(), p.Code) {
			t.Fatalf("write-time code %s is not a lint code", p.Code)
		}
		if FindingCode(p.Text) != p.Code {
			t.Fatalf("%q reads as %s in lint, %s at write time", p.Text, FindingCode(p.Text), p.Code)
		}
	}
	if len(write) != 4 {
		t.Fatal(write)
	}
}

// Check 8: set writes Purpose, Principle and acceptance without an editor,
// in canonical order, one blank line between sections.
func TestSetTextFixesWithoutEditor(t *testing.T) {
	s := &Store{Root: t.TempDir()}
	r, e := s.New(NewOptions{Title: "Bare", Goal: "g", Acceptance: []string{"a"}})
	if e != nil {
		t.Fatal(e)
	}
	if _, e = s.SetText(r.Task.ID, map[string]string{"Purpose": "why", "Principle": "how"}, []string{"b"}); e != nil {
		t.Fatal(e)
	}
	d, _ := s.Get(r.Task.ID)
	if fs := Lint([]*Document{d}); len(fs) != 0 {
		t.Fatal(fs)
	}
	b := string(d.Bytes())
	if !strings.Contains(b, "## Goal\ng\n\n## Purpose\nwhy\n\n## Principle\nhow\n\n## Depends on\n") || !strings.Contains(b, "- [ ] a\n- [ ] b\n") {
		t.Fatal(b)
	}
	// Replacing a section keeps the rest.
	if _, e = s.SetText(r.Task.ID, map[string]string{"Purpose": "why now"}, nil); e != nil {
		t.Fatal(e)
	}
	d, _ = s.Get(r.Task.ID)
	if d.SectionText("Purpose") != "why now" || d.SectionText("Principle") != "how" {
		t.Fatal(string(d.Bytes()))
	}
}

// Checks 7 and 11: backlog leftovers are findings with hug upgrade as the
// fix; the strip removes only them, never touching Goal text, and never
// splits Purpose or Principle out of the Goal.
func TestBacklogLeftovers(t *testing.T) {
	src := "---\nid: abc123\ntitle: T\nstate: todo\ncreated_date: '2026-06-07 12:25'\nordinal: 5\nbacklog_status: \"To Do\"\nimport_type_inference: \"default\"\nmilestone: m\n---\n\n## Goal\n<!-- SECTION:DESCRIPTION:BEGIN -->\nDo it.\n\nPurpose: because.\n<!-- SECTION:DESCRIPTION:END -->\n\n## Acceptance\n<!-- AC:BEGIN -->\n- [ ] a\n<!-- AC:END -->\n"
	d := Parse("q.abc123.t.md", []byte(src))
	var got []string
	for _, f := range d.Findings {
		if FindingCode(f.Problem) == "backlog-leftover" {
			got = append(got, f.Problem)
			if f.FixLine() != "hug upgrade" {
				t.Fatal(f.FixLine())
			}
		}
	}
	want := []string{"backlog leftover: unused fields created_date, ordinal, backlog_status, import_type_inference", "backlog leftover: 4 <!-- SECTION:... --> marker lines"}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Fatalf("%q", got)
	}
	out, changed := d.StripBacklog()
	wantOut := "---\nid: abc123\ntitle: T\nstate: todo\nmilestone: m\n---\n\n## Goal\nDo it.\n\nPurpose: because.\n\n## Acceptance\n- [ ] a\n"
	if !changed || string(out) != wantOut {
		t.Fatalf("%q", out)
	}
	after := Parse("q.abc123.t.md", out)
	if contains(after.Task.Sections, "Purpose") || strings.TrimSpace(after.Task.Goal) != "Do it.\n\nPurpose: because." {
		t.Fatalf("strip moved text: %+v %q", after.Task.Sections, after.Task.Goal)
	}
	if _, again := after.StripBacklog(); again {
		t.Fatal("second strip changed bytes")
	}
	// Done and dropped tasks are history: no leftover findings.
	done := Parse("q.abc123.t.md", []byte(strings.Replace(src, "state: todo", "state: done", 1)))
	for _, f := range done.Findings {
		if FindingCode(f.Problem) == "backlog-leftover" {
			t.Fatal(f)
		}
	}
}

// Check 6: an open task created before 2026-11-16 is held to the pattern;
// a done one is not.
func TestPatternHoldsForOldOpenTasks(t *testing.T) {
	old := "---\nid: zjedy0\ntitle: Old\nstate: todo\ncreated: \"2026-06-07 12:25\"\n---\n\n## Goal\ng\n"
	d := Parse("q.zjedy0.old.md", []byte(old))
	var codes []string
	for _, g := range ByTask(Lint([]*Document{d})) {
		codes = append(codes, g.Codes...)
		if g.Fix != `hug task set zjedy0 --purpose "..." --principle "..." --accept "observable check"` {
			t.Fatal(g.Fix)
		}
	}
	if strings.Join(codes, ",") != "no-acceptance,missing-section" {
		t.Fatal(codes)
	}
	done := Parse("q.zjedy0.old.md", []byte(strings.Replace(old, "state: todo", "state: done", 1)))
	for _, f := range Lint([]*Document{done}) {
		if FindingCode(f.Problem) == "missing-section" {
			t.Fatal(f)
		}
	}
}
