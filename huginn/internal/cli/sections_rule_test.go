package cli

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/veridit/hug/huginn/internal/task"
)

// 32ym01 (Jørgen 2026-11-25 08:30Z: "I would like those reasons included
// in instructions for AI agents", from his 11-24 18:24Z order "What, Why,
// How, Notes (to explain those over)"): one sentence names each section's
// job, said the same in every text that tells an agent how to write a task.
const sectionsRule = "Describe a task with Goal (what: where we aim), Purpose (why), Principle (how: what we do and do not do), Details (notes that explain those), then Acceptance."

// oldSectionsRule is the sentence before 32ym01, as AGENTS.md files hug
// setup wrote from v0.7.103 on still have it.
const oldSectionsRule = "Describe a task with Goal (where we aim), Purpose (why), Principle (what we do and do not do), Details, then Acceptance."

func flatText(s string) string { return strings.Join(strings.Fields(s), " ") }

// 32ym01 check 1: AGENTS.md, hug setup's template, help new and the
// missing-section lint texts say it in the same words (MCP task_new:
// huginn/internal/mcp, TestTaskNewSaysSectionsRule).
func TestSectionsRuleWording(t *testing.T) {
	repo, err := os.ReadFile(filepath.Join("..", "..", "..", "AGENTS.md"))
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct{ where, text string }{
		{"repository AGENTS.md", string(repo)},
		{"AGENTS.md section (hug setup)", agentSection},
		{"help new", commandHelp("new")},
	} {
		if flat := flatText(c.text); !strings.Contains(flat, sectionsRule) || strings.Contains(flat, oldSectionsRule) {
			t.Errorf("%s does not say %q:\n%s", c.where, sectionsRule, c.text)
		}
	}
	// Lint names a missing section with its job, as the sentence does.
	var texts []string
	for _, p := range task.DescriptionProblems("t", "g", false, false, 1) {
		texts = append(texts, p.Text)
	}
	if got, want := strings.Join(texts, "; "), "missing section Purpose (why); missing section Principle (how: what we do and do not do)"; got != want {
		t.Errorf("lint texts\n %s\nwant\n %s", got, want)
	}
	for _, job := range []string{"Purpose (why)", "Principle (how: what we do and do not do)"} {
		if !strings.Contains(sectionsRule, job) {
			t.Errorf("the sentence does not hold the lint text's %q", job)
		}
	}
}

// 32ym01 check 2: hug setup on a repository whose AGENTS.md hug section
// has the old sentence lists the update as an open step, naming the old
// sentence and the new one, and the section to put in.
func TestSetupFlagsOldSectionsSentence(t *testing.T) {
	dir := t.TempDir()
	for _, args := range [][]string{{"init", "-q", "-b", "main"}, {"-c", "user.email=t@t", "-c", "user.name=t", "commit", "-q", "--allow-empty", "-m", "root"}} {
		c := exec.Command("git", args...)
		c.Dir = dir
		if out, err := c.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v %s", args, err, out)
		}
	}
	t.Setenv("HOME", t.TempDir())
	step := func() map[string]any {
		t.Helper()
		r, err := (&App{Cwd: dir}).Execute(context.Background(), []string{"setup", "--json"})
		if err != nil {
			t.Fatal(err)
		}
		b, _ := json.Marshal(r["setup"])
		var steps []map[string]any
		json.Unmarshal(b, &steps)
		for _, s := range steps {
			if s["step"] == "agent instructions" {
				return s
			}
		}
		t.Fatal("no agent instructions step")
		return nil
	}
	// This hug's section, with the sentence as hug wrote it before:
	// wrapped, the Goal rule and the title rule after it.
	old := regexp.MustCompile(`(?s)Describe a task with.*?Titles stay under 80 characters\.`).ReplaceAllString(agentSection,
		"Describe a task with Goal (where we aim), Purpose (why), Principle (what we do\nand do not do), Details, then Acceptance. The Goal is one sentence of\nat most 140 characters; detail goes in Details. Titles stay under 80 characters.")
	if err := os.WriteFile(filepath.Join(dir, "AGENTS.md"), []byte("# Repo\n\n"+old+"\n\n## Other\nkept\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	s := step()
	do, _ := s["do"].(string)
	if s["done"] != false || !strings.Contains(do, `in AGENTS.md, replace the sentence "`+oldSectionsRule+`" with "`+sectionsRule+`"`) || !strings.Contains(flatText(do), sectionsRule) {
		t.Fatalf("old sentence: step %v", s)
	}
	cur := "# Repo\n\n" + agentSection + "\n\n## Other\nkept\n"
	if err := os.WriteFile(filepath.Join(dir, "AGENTS.md"), []byte(cur), 0o644); err != nil {
		t.Fatal(err)
	}
	if s := step(); s["done"] != true {
		t.Fatalf("current section not done: %v", s)
	}
}
