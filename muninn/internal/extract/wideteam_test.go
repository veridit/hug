package extract

import (
	"strings"
	"testing"
)

// rnpdnm attempt 2 (b), PREREG2: the wide prompt carries the agents'
// working-rule sentence verbatim; the off prompt (with and without
// places) stays byte-identical, pinned in wideplaces_test.go.
func TestWidePromptPlacesAgentRulesGlobal(t *testing.T) {
	const want = "A working rule for agents (how to edit, commit, poll, report, which tools or limits apply) is global even when said while working in this project."
	on := Defaults()
	on.WidePlaces = true
	p := SystemPrompt(on, append(append([]Place{}, sessionPlaces...), WidePlaces...))
	if !strings.Contains(p, want) {
		t.Fatalf("wide prompt lacks the sentence:\n%s", p[len(prompt):])
	}
	if strings.Contains(SystemPrompt(Defaults(), sessionPlaces), "working rule for agents (how to edit") {
		t.Fatal("off prompt carries the wide sentence")
	}
}
