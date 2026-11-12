// Package testfixture seeds deterministic, entirely synthetic terminal acceptance data.
package testfixture

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/veridit/hug/huginn/internal/task"
)

// Seed writes 60 tasks covering all statuses, types, priorities, owners and lint.
// Dates, ids and ranks are fixed so real-terminal captures are reproducible.
func Seed(root string) error {
	for i := 0; i < 60; i++ {
		id := fmt.Sprintf("%06d", i+1)
		status := []string{"todo", "doing", "done", "dropped"}[i%4]
		kind := task.Types[i%len(task.Types)]
		priority := []string{"critical", "high", "medium", "low"}[(i/5)%4]
		group := []string{"compiler", "runtime", "docs"}[(i/3)%3]
		title := fmt.Sprintf("%s task %02d: make the terminal workflow observable", strings.ToUpper(kind[:1])+kind[1:], i+1)
		if i%7 == 0 {
			title += " with a deliberately long title that must truncate cleanly at the right edge 界 🤖"
		}
		goal := "Deliver **compiler-lang** support with `huginn` and *readable evidence*.\n\n- Preserve the public interface\n- See [the guide](https://example.invalid/guide) for the synthetic contract.\n\n### Verification\n\n```go\nfmt.Println(\"synthetic acceptance\")\n```\n"
		assignee := "[]"
		if status == "doing" {
			assignee = `["` + []string{"alice", "bob", "carol"}[(i/5)%3] + `"]`
		}
		deps := ""
		if i%8 == 4 {
			deps = "- 000001 waiting for the ranked prerequisite\n"
		}
		if i%13 == 12 {
			deps = "- zzzzzz synthetic missing dependency\n"
		}
		box := " "
		if status == "done" {
			box = "x"
		}
		acceptance := "- [" + box + "] First acceptance check\n- [" + box + "] Second acceptance check\n"
		notes := "Evidence: integration tests are synthetic."
		history := "2026-11-10: created by the terminal acceptance fixture."
		if i == 2 {
			title = "Render compiler-lang task detail without storage markers"
			kind = "feature"
			priority = "medium"
			acceptance = ""
			notes = ""
			history = ""
			goal = "<!-- SECTION:DESCRIPTION:BEGIN -->\nProvide **compiler-lang** support with `huginn` and *clear sections*.\n\n- Render Markdown, not source\n- Keep the header fixed while scrolling\n\nRead [the guide](https://example.invalid/guide).\n\n```go\nfmt.Println(\"rendered code\")\n```\n<!-- SECTION:DESCRIPTION:END -->\n"
		}
		quoted, _ := json.Marshal(title)
		body := fmt.Sprintf("---\nid: %s\ntitle: %s\nstatus: %s\ntype: %s\npriority: %s\nassignee: %s\nlabels: [\"terminal\", \"%s\"]\naliases: [\"TASK-%d\"]\ncreated: 2026-11-01\nupdated: 2026-11-10\n---\n\n## Goal\n%s\n## Depends on\n%s\n## Acceptance\n%s\n## Notes\n%s\n\n## History\n%s\n", id, quoted, status, kind, priority, assignee, group, 1783+i, goal, deps, acceptance, notes, history)
		// Keep synthetic filename slugs clear of credential-shaped token patterns.
		slug := strings.ReplaceAll(task.Slug(title), "-task-", "-item-")
		path := filepath.Join(root, group, task.BuildName(fmt.Sprintf("%04d", i+1), id, slug))
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			return err
		}
		if err := os.WriteFile(path, []byte(body), 0644); err != nil {
			return err
		}
	}
	return nil
}
