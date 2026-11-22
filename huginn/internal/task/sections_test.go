package task

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// mkzcqrOrder is how mkzcqr's file was laid out: Depends on before Purpose.
const mkzcqrOrder = `---
id: mkzcqr
title: "TUI sort builder"
state: "todo"
---

## Goal
Build a nested sort step by step.

## Depends on

## Purpose
Jørgen sorts by priority then type.

## Principle
Esc leaves the order as it was.
` + "```md\n## Not a heading inside a fence\n```\n" + `
## Acceptance
- [ ] ': sort', p, t, Enter orders rows

## Notes
2026-11-16: first note.
2026-11-16: second note.
`

func sectionBodies(t *testing.T, b []byte) map[string]string {
	t.Helper()
	d := Parse("zzz.mkzcqr.sort.md", b)
	out := map[string]string{}
	for _, s := range d.Task.Sections {
		out[s] = d.SectionText(s)
	}
	return out
}

// jnaspy check 3: hug upgrade's reorder puts mkzcqr's order right with each
// section's text unchanged, and a second run changes nothing.
func TestCanonicalOrderKeepsText(t *testing.T) {
	d := Parse("zzz.mkzcqr.sort.md", []byte(mkzcqrOrder))
	if InOrder(d.Task.Sections) {
		t.Fatal("fixture should be out of order")
	}
	out, changed := d.Canonical()
	if !changed {
		t.Fatal("Canonical changed nothing")
	}
	again := Parse("zzz.mkzcqr.sort.md", out)
	want := []string{"Goal", "Purpose", "Principle", "Depends on", "Acceptance", "Notes"}
	if strings.Join(again.Task.Sections, "|") != strings.Join(want, "|") {
		t.Fatalf("order %v, want %v\n%s", again.Task.Sections, want, out)
	}
	before, after := sectionBodies(t, []byte(mkzcqrOrder)), sectionBodies(t, out)
	for name, text := range before {
		if after[name] != text {
			t.Fatalf("section %s changed:\n%q\nwas\n%q", name, after[name], text)
		}
	}
	if !strings.Contains(string(out), "```md\n## Not a heading inside a fence\n```") {
		t.Fatalf("fenced text moved or changed:\n%s", out)
	}
	if !strings.HasPrefix(string(out), "---\nid: mkzcqr\n") {
		t.Fatalf("frontmatter not first:\n%s", out)
	}
	if strings.Contains(string(out), "\n\n\n") {
		t.Fatalf("extra blank lines:\n%s", out)
	}
	if again2, changed := again.Canonical(); changed || string(again2) != string(out) {
		t.Fatal("a second run changed the file")
	}
	for _, f := range again.Findings {
		if FindingCode(f.Problem) == "section-order" {
			t.Fatalf("still flagged: %s", f.Problem)
		}
	}
}

// Other sections keep their relative order and go before Notes; History is
// last; CRLF files stay CRLF.
func TestCanonicalOrderOthersAndCRLF(t *testing.T) {
	src := "---\nid: aaaaaa\ntitle: \"A\"\nstate: \"todo\"\n---\n\n## Goal\nG\n\n## History\nh\n\n## Notes\nn\n\n## Zeta\nz\n\n## Acceptance\n- [ ] c\n\n## Alpha\na\n"
	crlf := strings.ReplaceAll(src, "\n", "\r\n")
	for _, text := range []string{src, crlf} {
		out, changed := Parse("aaaaaa.a.md", []byte(text)).Canonical()
		if !changed {
			t.Fatal("not reordered")
		}
		got := Parse("aaaaaa.a.md", out).Task.Sections
		if strings.Join(got, "|") != "Goal|Acceptance|Zeta|Alpha|Notes|History" {
			t.Fatalf("order %v", got)
		}
		if strings.Contains(text, "\r\n") && strings.Count(string(out), "\n") != strings.Count(string(out), "\r\n") {
			t.Fatalf("CRLF lost:\n%q", out)
		}
	}
}

// jnaspy check 4: lint reports an out-of-order file with fix: hug upgrade.
func TestLintSectionOrder(t *testing.T) {
	d := Parse("zzz.mkzcqr.sort.md", []byte(mkzcqrOrder))
	var found *Finding
	for i, f := range Lint([]*Document{d}) {
		if FindingCode(f.Problem) == "section-order" {
			found = &Lint([]*Document{d})[i]
		}
	}
	if found == nil {
		t.Fatal("no section-order finding")
	}
	if found.FixLine() != "hug upgrade" {
		t.Fatalf("fix %q", found.FixLine())
	}
	if !strings.Contains(found.Problem, "Goal, Depends on, Purpose") || !strings.Contains(found.Problem, "the order is Goal, Purpose, Principle, Depends on") {
		t.Fatalf("problem %q", found.Problem)
	}
}

// Store.Reorder rewrites only files out of order, then nothing.
func TestStoreReorderIdempotent(t *testing.T) {
	root := t.TempDir()
	s := &Store{Root: root}
	r, err := s.New(NewOptions{Title: "In order", Goal: "G", Purpose: "P", Principle: "Q", Acceptance: []string{"C"}})
	if err != nil {
		t.Fatal(err)
	}
	inOrder, _ := os.ReadFile(filepath.Join(root, r.Task.Path))
	if err := os.WriteFile(filepath.Join(root, "zzz.mkzcqr.sort.md"), []byte(mkzcqrOrder), 0o644); err != nil {
		t.Fatal(err)
	}
	changed, err := s.Reorder()
	if err != nil || len(changed) != 1 || changed[0] != "zzz.mkzcqr.sort.md" {
		t.Fatalf("first run: %v %v", changed, err)
	}
	if now, _ := os.ReadFile(filepath.Join(root, r.Task.Path)); string(now) != string(inOrder) {
		t.Fatal("a file already in order was rewritten")
	}
	if changed, err = s.Reorder(); err != nil || len(changed) != 0 {
		t.Fatalf("second run: %v %v", changed, err)
	}
}

// A note on a task with History but no Notes goes above History.
func TestNoteGoesBeforeHistory(t *testing.T) {
	d := Parse("aaaaaa.a.md", []byte("---\nid: aaaaaa\ntitle: \"A\"\nstate: \"todo\"\n---\n\n## Goal\nG\n\n## Acceptance\n- [ ] c\n\n## History\nh\n"))
	d.Note("first")
	if got := strings.Join(d.Task.Sections, "|"); got != "Goal|Acceptance|Notes|History" {
		t.Fatalf("sections %s\n%s", got, d.Bytes())
	}
	if d.SectionText("Notes") != "first" || d.SectionText("History") != "h" {
		t.Fatalf("texts: %q %q", d.SectionText("Notes"), d.SectionText("History"))
	}
}

// A dependency added to a task without Depends on goes in its canonical
// place, before Acceptance and Notes, not at the end (ci/jnaspy caught the
// old append in the cache tests).
func TestNewDependsOnGoesInOrder(t *testing.T) {
	d := Parse("aaaaaa.a.md", []byte("---\nid: aaaaaa\ntitle: \"A\"\nstate: \"todo\"\n---\n\n## Goal\nG\n\n## Acceptance\n- [ ] c\n\n## Notes\nn\n"))
	d.appendDependencies([]string{"bbbbbb"})
	if got := strings.Join(d.Task.Sections, "|"); got != "Goal|Depends on|Acceptance|Notes" {
		t.Fatalf("sections %s\n%s", got, d.Bytes())
	}
	if !strings.Contains(string(d.Bytes()), "## Goal\nG\n\n## Depends on\n- bbbbbb\n\n## Acceptance\n") {
		t.Fatalf("layout:\n%s", d.Bytes())
	}
	if len(d.Task.Dependencies) != 1 || d.Task.Dependencies[0].Ref != "bbbbbb" {
		t.Fatalf("deps %v", d.Task.Dependencies)
	}
	// Only Goal: still appended after it.
	e := Parse("aaaaaa.a.md", []byte("---\nid: aaaaaa\ntitle: \"A\"\nstate: \"todo\"\n---\n\n## Goal\nG\n"))
	e.appendDependencies([]string{"bbbbbb"})
	if got := strings.Join(e.Task.Sections, "|"); got != "Goal|Depends on" {
		t.Fatalf("sections %s\n%s", got, e.Bytes())
	}
}

// 8a0ybb check 13: a file with the old "## Relevant details" reads as
// Details: its text in Texts, in canonical order where Details stands,
// and SetSection writes into it rather than adding "## Details".
func TestOldRelevantDetailsReadsAsDetails(t *testing.T) {
	data := "---\nid: abc123\ntitle: Old\nstate: todo\n---\n\n## Goal\ng\n\n## Purpose\np\n\n## Relevant details\nold facts\n\n## Depends on\n\n## Acceptance\n- [ ] a\n"
	d := Parse("a/abc123.md", []byte(data))
	if got := d.Task.Texts.Description["Details"]; got != "old facts" {
		t.Fatalf("Details %q", got)
	}
	if !InOrder(d.Task.Sections) {
		t.Fatalf("out of order: %v", d.Task.Sections)
	}
	for _, f := range d.Findings {
		if strings.Contains(f.Problem, "out of order") {
			t.Fatalf("finding %q", f.Problem)
		}
	}
	d.SetSection("Details", "new facts")
	got := string(d.Bytes())
	if !strings.Contains(got, "## Relevant details\nnew facts\n") || strings.Contains(got, "## Details") {
		t.Fatalf("SetSection on an old file:\n%s", got)
	}
}
