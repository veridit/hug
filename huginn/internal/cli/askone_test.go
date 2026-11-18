package cli

import (
	"strings"
	"testing"
)

// q2adr8 check 5: the ask/answer help and the agent instructions say one
// question per ask, and answer's help names answer ID N.
func TestHelpAndInstructionsSayOneQuestionPerAsk(t *testing.T) {
	for _, c := range []struct{ where, text string }{
		{"help ask", commandHelp("ask")},
		{"help answer", commandHelp("answer")},
		{"AGENTS.md section", agentSection},
	} {
		if !strings.Contains(c.text, "one question per ask") {
			t.Errorf("%s does not say one question per ask:\n%s", c.where, c.text)
		}
	}
	if h := commandHelp("answer"); !strings.Contains(h, "answer ID N") {
		t.Errorf("help answer does not name answer ID N:\n%s", h)
	}
	if !strings.Contains(agentSection, "hug task answer ID N") {
		t.Errorf("AGENTS.md section does not name hug task answer ID N:\n%s", agentSection)
	}
}
