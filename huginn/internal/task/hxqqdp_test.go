package task

import (
	"strings"
	"testing"
)

// hxqqdp (ram 2026-11-22, 6q1d1t): a hug from before 8a0ybb (v0.7.101,
// still running as hug tui or hug mcp) writing set --details to a file
// with "## Details" adds a second section, "## Relevant details". v0.7.104
// read only the Details text, so what the old hug wrote was hidden.

// bothDetails is a task file with both headings, as the old hug leaves it.
const bothDetails = "---\nid: hxq001\ntitle: \"Both\"\nstate: \"todo\"\ntype: bug\npriority: \"medium\"\ncreated: 2026-11-22\nupdated: 2026-11-22\n---\n\n" +
	"## Goal\nAim.\n\n## Purpose\nWhy.\n\n## Principle\nHow.\n\n## Details\nNew hug wrote this.\n\n" +
	"## Depends on\n\n## Acceptance\n- [ ] a\n\n## Relevant details\nOld hug wrote this.\n"

// hxqqdp check 1: Parse reads both texts as Details, the Details text
// first, and reports the second heading with fix: hug upgrade.
func TestBothDetailsHeadingsReadAndLinted(t *testing.T) {
	d := Parse("h/hxq001.md", []byte(bothDetails))
	want := "New hug wrote this.\n\nOld hug wrote this."
	if got := d.SectionText("Details"); got != want {
		t.Fatalf("Details text %q, want %q", got, want)
	}
	if got := d.Task.Texts.Description["Details"]; got != want {
		t.Fatalf("Texts.Description[Details] %q, want %q", got, want)
	}
	var found *Finding
	for i, f := range d.Findings {
		if strings.Contains(f.Problem, "Relevant details") {
			found = &d.Findings[i]
		}
	}
	if found == nil {
		t.Fatalf("no finding names the second heading: %+v", d.Findings)
	}
	if found.Line != 28 || FindingCode(found.Problem) != "old-layout" || found.FixLine() != "hug upgrade" || found.Severity() != Broken {
		t.Fatalf("finding line %d code %s fix %q severity %s: %q", found.Line, FindingCode(found.Problem), found.FixLine(), found.Severity(), found.Problem)
	}
	// One heading, either name, is not a finding.
	for _, one := range []string{
		strings.Replace(bothDetails, "\n## Relevant details\nOld hug wrote this.\n", "\n", 1),
		strings.Replace(strings.Replace(bothDetails, "## Details\nNew hug wrote this.\n\n", "", 1), "\n## Relevant details\nOld hug wrote this.\n", "\n", 1),
	} {
		for _, f := range Parse("h/hxq001.md", []byte(one)).Findings {
			if strings.Contains(f.Problem, "etails") {
				t.Fatalf("one Details heading reported: %q", f.Problem)
			}
		}
	}
}

// hxqqdp check 2: MergeDetails makes one "## Details" holding the Details
// text then the Relevant details text, every other line as it was, in
// canonical order; a file with one heading is left alone, and a second
// run changes nothing.
func TestMergeDetailsKeepsBothTexts(t *testing.T) {
	out, changed := Parse("h/hxq001.md", []byte(bothDetails)).MergeDetails()
	if !changed {
		t.Fatal("MergeDetails changed nothing")
	}
	got := string(out)
	want := strings.Replace(strings.Replace(bothDetails, "\n## Relevant details\nOld hug wrote this.\n", "\n", 1),
		"## Details\nNew hug wrote this.\n", "## Details\nNew hug wrote this.\n\nOld hug wrote this.\n", 1)
	want = strings.TrimSuffix(want, "\n")
	if got != want {
		t.Fatalf("merged:\n%s\nwant:\n%s", got, want)
	}
	d := Parse("h/hxq001.md", out)
	if len(d.Findings) != 0 {
		t.Fatalf("findings after the merge: %+v", d.Findings)
	}
	if again, changed := d.MergeDetails(); changed || string(again) != got {
		t.Fatalf("a second merge changed the file:\n%s", again)
	}
	one := strings.Replace(bothDetails, "\n## Relevant details\nOld hug wrote this.\n", "\n", 1)
	if _, changed := Parse("h/hxq001.md", []byte(one)).MergeDetails(); changed {
		t.Fatal("a file with one Details heading was changed")
	}
	// Relevant details alone (a file from before 8a0ybb) is read as it
	// is, not renamed here.
	old := strings.Replace(one, "## Details\n", "## Relevant details\n", 1)
	if _, changed := Parse("h/hxq001.md", []byte(old)).MergeDetails(); changed {
		t.Fatal("a file with only Relevant details was changed")
	}
}
