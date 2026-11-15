package harness

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func journal(t *testing.T, lines ...string) string {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "s1.journal.jsonl"), []byte(strings.Join(lines, "\n")+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	return dir
}

const (
	userQ   = `{"meta":{},"append_messages":[{"role":"user","content":[{"type":"text","text":"why is CI red"}]}]}`
	toolUse = `{"meta":{},"append_messages":[{"role":"assistant","content":[{"type":"text","text":"Checking."},{"type":"tool_use"}]}]}`
	toolRes = `{"meta":{},"append_messages":[{"role":"user","content":[{"type":"tool_result"}]}]}`
	final   = `{"meta":{},"append_messages":[{"role":"assistant","content":[{"type":"text","text":"CI is red because of X."}]}]}`
	userQ2  = `{"meta":{},"append_messages":[{"role":"user","content":[{"type":"text","text":"and now?"}]}]}`
)

// The recall hook may only deliver while the turn it recalled for is still
// running; afterwards jcode transcript --mode send would start a fake user turn.
func TestJcodeTurnOpen(t *testing.T) {
	for _, tc := range []struct {
		name  string
		lines []string
		msg   string
		open  bool
	}{
		{"just asked", []string{userQ}, "why is CI red", true},
		{"mid tool calls", []string{userQ, toolUse, toolRes}, "why is CI red", true},
		{"answered", []string{userQ, toolUse, toolRes, final}, "why is CI red", false},
		{"newer message", []string{userQ, final, userQ2}, "why is CI red", false},
		{"no journal match", []string{userQ}, "something else", false},
		{"after an injected block", []string{userQ, toolUse, `{"meta":{},"append_messages":[{"role":"user","content":[{"type":"text","text":"[transcription] -- AI memory (muninn): not the user"}]}]}`}, "why is CI red", true},
	} {
		if got := JcodeTurnOpen(journal(t, tc.lines...), "s1", tc.msg); got != tc.open {
			t.Errorf("%s: open=%v, want %v", tc.name, got, tc.open)
		}
	}
	if JcodeTurnOpen(t.TempDir(), "s1", "x") {
		t.Error("missing journal must not count as open")
	}
}
