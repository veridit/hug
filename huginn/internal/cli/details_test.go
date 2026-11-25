package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// 8a0ybb check 13 (Jørgen 2026-11-20 on ddpxs8): 'Relevant details' is
// shortened to 'Details'. hug task new writes "## Details" and show heads
// it "details:"; a file written before, with "## Relevant details", still
// reads the same in show, show --json and lint, and set --details writes
// into that section, never a second one.
func TestDetailsSectionNameAndOldName(t *testing.T) {
	t.Parallel()
	a := app(t)
	r := execute(t, a, "new", "Fresh", "--goal", "Where we aim", "--purpose", "Why we aim", "--principle", "How we go", "--details", "Facts that matter", "--accept", "Observable check")
	id := r["task"].(Reply)["id"].(string)
	m, _ := globID(filepath.Join(a.Cwd, TaskDir), id)
	b, _ := os.ReadFile(m[0])
	if !strings.Contains(string(b), "\n## Details\nFacts that matter\n") || strings.Contains(string(b), "Relevant details") {
		t.Fatalf("new task file:\n%s", b)
	}
	_, out := runText(a, "show", id)
	if !strings.Contains(out, "\ndetails: Facts that matter\n") || strings.Contains(out, "relevant details") {
		t.Fatalf("show:\n%s", out)
	}
	// The same task as an old file has it.
	old := strings.Replace(string(b), "\n## Details\n", "\n## Relevant details\n", 1)
	if err := os.WriteFile(m[0], []byte(old), 0o644); err != nil {
		t.Fatal(err)
	}
	git(t, filepath.Join(a.Cwd, TaskDir), "commit", "-qam", "old heading")
	_, out = runText(a, "show", id)
	if !strings.Contains(out, "\ndetails: Facts that matter\n") {
		t.Fatalf("show of an old file:\n%s", out)
	}
	if j := execute(t, a, "show", id, "--json")["task"].(Reply); j["details"] != "Facts that matter" {
		t.Fatalf("json details %q", j["details"])
	}
	if code, out := runText(a, "lint", id); code != 0 || strings.Contains(out, "section-order") {
		t.Fatalf("lint of an old file: exit %d\n%s", code, out)
	}
	execute(t, a, "set", id, "--details", "New facts")
	b, _ = os.ReadFile(m[0])
	if got := string(b); strings.Count(got, "etails\n") != 1 || !strings.Contains(got, "## Relevant details\nNew facts\n") {
		t.Fatalf("set --details on an old file:\n%s", got)
	}
}

// 8a0ybb check 13: what agents read names the section Details: the
// AGENTS.md section hug setup writes and the MCP set field (the sentence
// itself is 32ym01's, TestSectionsRuleWording).
func TestDetailsWordingForAgents(t *testing.T) {
	flat := strings.Join(strings.Fields(agentSection), " ")
	if !strings.Contains(flat, "Principle (how: what we do and do not do), Details (notes that explain those), then Acceptance.") || strings.Contains(agentSection, "Relevant details") {
		t.Fatalf("agentSection:\n%s", agentSection)
	}
	if strings.Contains(commandHelp("upgrade"), "Relevant details") {
		t.Fatalf("help upgrade names Relevant details:\n%s", commandHelp("upgrade"))
	}
}

// hxqqdp (ram 2026-11-22): an older hug (v0.7.101, running as hug tui or
// hug mcp) writing set --details to a "## Details" file adds a second
// section, "## Relevant details". Check 1: show and show --json print both
// texts, the Details text first; lint reports the second heading, with
// fix: hug upgrade. Check 2: hug upgrade merges them into one
// "## Details", the old heading's text after, no text lost, and commits;
// a second upgrade changes nothing.
func TestBothDetailsHeadingsShownLintedMerged(t *testing.T) {
	t.Parallel()
	a := app(t)
	r := execute(t, a, "new", "Both", "--goal", "Where we aim", "--purpose", "Why we aim", "--principle", "How we go", "--details", "New hug wrote this", "--accept", "Observable check")
	id := r["task"].(Reply)["id"].(string)
	m, _ := globID(filepath.Join(a.Cwd, TaskDir), id)
	b, _ := os.ReadFile(m[0])
	// What v0.7.101's set --details does to it: SetSection finds no
	// "## Relevant details" and inserts one before the first section
	// ranked after it, "## Details", a heading that hug does not know.
	both := strings.Replace(string(b), "\n## Details\n", "\n## Relevant details\nOld hug wrote this\n\n## Details\n", 1)
	if err := os.WriteFile(m[0], []byte(both), 0o644); err != nil {
		t.Fatal(err)
	}
	git(t, filepath.Join(a.Cwd, TaskDir), "commit", "-qam", "an old hug's set --details")
	_, out := runText(a, "show", id)
	if !strings.Contains(out, "\ndetails: New hug wrote this\n\nOld hug wrote this\n") {
		t.Fatalf("show hides a Details text:\n%s", out)
	}
	if j := execute(t, a, "show", id, "--json")["task"].(Reply); j["details"] != "New hug wrote this\n\nOld hug wrote this" {
		t.Fatalf("json details %q", j["details"])
	}
	if _, out := runText(a, "lint", id); !strings.Contains(out, "broken [old-layout] old layout: two Details sections") || !strings.Contains(out, "next: hug upgrade") {
		t.Fatalf("lint:\n%s", out)
	}
	did := func() string { return strings.Join(execute(t, a, "upgrade", "--apply")["upgrade"].([]string), "; ") }
	if got := did(); !strings.Contains(got, "1 tasks with two Details sections merged") {
		t.Fatalf("upgrade: %s", got)
	}
	after, _ := os.ReadFile(m[0])
	if got := string(after); strings.Contains(got, "Relevant details") || strings.Count(got, "## Details\n") != 1 ||
		!strings.Contains(got, "\n## Details\nNew hug wrote this\n\nOld hug wrote this\n\n## Depends on\n") {
		t.Fatalf("merged file:\n%s", got)
	}
	if st := git(t, filepath.Join(a.Cwd, TaskDir), "status", "--porcelain"); st != "" {
		t.Fatalf("not committed: %s", st)
	}
	if log := git(t, filepath.Join(a.Cwd, TaskDir), "log", "--format=%s"); !strings.Contains("\n"+log+"\n", "\nhug upgrade: two Details sections merged in 1 tasks\n") {
		t.Fatalf("no merge commit:\n%s", log)
	}
	if code, out := runText(a, "lint", id); code != 0 || strings.Contains(out, "Details") {
		t.Fatalf("lint after upgrade: exit %d\n%s", code, out)
	}
	if got := did(); strings.Contains(got, "Details") {
		t.Fatalf("second upgrade: %s", got)
	}
	if again, _ := os.ReadFile(m[0]); string(again) != string(after) {
		t.Fatal("second upgrade changed the file")
	}
}
