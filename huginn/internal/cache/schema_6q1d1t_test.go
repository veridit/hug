package cache

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/veridit/hug/huginn/internal/task"
)

// detailsFile is 9s0vj3 as v0.7.103 writes it: ## Details (8a0ybb), then
// Depends on, Acceptance and a Questions record.
const detailsFile = "---\nid: 9s0vj3\ntitle: Release v0.7.103\nstate: todo\ntype: feature\n---\n\n" +
	"## Goal\nA screenshot lands in FILES.\n\n## Purpose\nOne release ticket per release.\n\n## Principle\nAsk one thing.\n\n" +
	"## Details\nShipped q7n2zd checks 2-4.\n\n## Depends on\n\n## Acceptance\n- [ ] Jørgen pastes a screenshot\n\n" +
	"## Questions\n- 2026-11-22T10:25 question 1 by crab to jhf (check 1):\n  Did the screenshot land?\n\n" +
	"## Notes\n\ncrab 09:26Z: shipped.\n\n## History\n- 2026-11-22T10:25 todo -> todo/design (ask)\n"

// 6q1d1t: a cache written by an older hug is not served as fresh when the
// rules changed. Observed 2026-11-22: v0.7.101 (SchemaVersion 30, Details
// unknown and ranked after Acceptance) cached 'sections out of order' for
// 9s0vj3 in cache-30.db; v0.7.103 (also 30, Details canonical, 8a0ybb)
// served it as its own.
func TestOlderRulesCacheNotServed(t *testing.T) {
	root := t.TempDir()
	put(t, root, "9/9s0vj3.md", []byte(detailsFile))
	// Untouched since: the marker matches, so only the schema can tell.
	old := time.Now().Add(-time.Hour)
	if e := os.Chtimes(filepath.Join(root, "9/9s0vj3.md"), old, old); e != nil {
		t.Fatal(e)
	}
	common := t.TempDir()

	// v0.7.101: schema "30" and SectionOrder before 8a0ybb.
	func() {
		defer func(v string, order []string) { schema, task.SectionOrder = v, order }(schema, task.SectionOrder)
		schema = "30"
		task.SectionOrder = []string{"Goal", "Purpose", "Principle", "Relevant details", "Depends on", "Acceptance"}
		c := openTest(t, root, File(common), nil)
		defer c.Close()
		fs, e := c.Lint(context.Background())
		if e != nil {
			t.Fatal(e)
		}
		if !hasCode(fs, "section-order") {
			t.Fatalf("the v0.7.101 rules should find Details out of order (the setup is wrong): %v", fs)
		}
	}()

	// This hug, on the same machine and board.
	c := openTest(t, root, File(common), nil)
	defer c.Close()
	fs, e := c.Lint(context.Background())
	if e != nil {
		t.Fatal(e)
	}
	if hasCode(fs, "section-order") {
		t.Errorf("SchemaVersion %s serves the section-order finding v0.7.101 cached under schema 30:\n%s\nfix: bump SchemaVersion in huginn/internal/cache/cache.go", SchemaVersion, findingLines(fs))
	}
}

func hasCode(fs []task.Finding, code string) bool {
	for _, f := range fs {
		if task.FindingCode(f.Problem) == code {
			return true
		}
	}
	return false
}

func findingLines(fs []task.Finding) string {
	var b strings.Builder
	for _, f := range fs {
		b.WriteString("  " + f.String() + "\n")
	}
	return b.String()
}
