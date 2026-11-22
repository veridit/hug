package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// zbdbtm check 4 (Jørgen 2026-11-20, hpqt99 request 1, point 3): a
// document written for one task is attached to that task; repo docs hold
// only decisions. Both the AGENTS.md section hug setup writes and this
// repository's AGENTS.md say so, in these words.
const attachRule = "A document for one task is attached to that task (hug task attach ID FILE); only decisions go into repo docs."

func TestAgentsSayTaskDocumentsAreAttached(t *testing.T) {
	repo, err := os.ReadFile(filepath.Join("..", "..", "..", "AGENTS.md"))
	if err != nil {
		t.Fatal(err)
	}
	flat := func(s string) string { return strings.Join(strings.Fields(s), " ") }
	for _, c := range []struct{ name, text string }{
		{"AGENTS.md section (hug setup)", agentSection},
		{"repository AGENTS.md", string(repo)},
	} {
		if !strings.Contains(flat(c.text), attachRule) {
			t.Errorf("%s does not say %q", c.name, attachRule)
		}
	}
}
