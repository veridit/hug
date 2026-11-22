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
// AGENTS.md section hug setup writes and the MCP set field.
func TestDetailsWordingForAgents(t *testing.T) {
	flat := strings.Join(strings.Fields(agentSection), " ")
	if !strings.Contains(flat, "Principle (what we do and do not do), Details, then Acceptance.") || strings.Contains(agentSection, "Relevant details") {
		t.Fatalf("agentSection:\n%s", agentSection)
	}
	if strings.Contains(commandHelp("upgrade"), "Relevant details") {
		t.Fatalf("help upgrade names Relevant details:\n%s", commandHelp("upgrade"))
	}
}
