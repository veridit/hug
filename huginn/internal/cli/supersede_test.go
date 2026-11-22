package cli

import (
	"os"
	"strings"
	"testing"
)

// y8we58 check 3 (h176dp r1, calf: a ruling on bxqtkt superseded h176dp
// check 4 and neither task named the other): the instructions hug setup
// installs, and this repo's AGENTS.md, say to strike the superseded check
// where it is, naming the ruling task's id, so the link shows from it.
const supersedeLine = "A ruling that supersedes a check on another task: strike that check there,\nnaming the ruling task's id (hug task strike ID N --reason \"superseded by RULING-ID\")."

func TestInstructionsStrikeSupersededChecks(t *testing.T) {
	t.Parallel()
	if !strings.Contains(agentSection, "\n"+supersedeLine+"\n") {
		t.Fatalf("agentSection lacks the supersede line %q:\n%s", supersedeLine, agentSection)
	}
	b, e := os.ReadFile("../../../AGENTS.md")
	if e != nil {
		t.Fatal(e)
	}
	// A Windows checkout may give AGENTS.md CRLF line endings.
	if !strings.Contains(strings.ReplaceAll(string(b), "\r\n", "\n"), "\n"+supersedeLine+"\n") {
		t.Fatalf("AGENTS.md lacks the supersede line %q", supersedeLine)
	}
}
