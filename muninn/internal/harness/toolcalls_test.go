package harness

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestParseToolCall(t *testing.T) {
	at := time.Now()
	cases := []struct {
		name, tool, input string
		edits             bool
		commit            bool
		subject           string
		hug               []HugCall
	}{
		{"edit in repo", "edit", `{"file_path":"/r/muninn/x.go"}`, true, false, "", nil},
		{"edit in tmp", "write", `{"file_path":"/r/tmp/lanes/x.md"}`, false, false, "", nil},
		{"edit on the board", "edit", `{"file_path":"/r/.hug/tasks/a.md"}`, false, false, "", nil},
		{"edit outside", "edit", `{"file_path":"/elsewhere/x.go"}`, false, false, "", nil},
		{"apply_patch", "apply_patch", `{"patch_text":"*** Begin Patch\n*** Update File: /r/a.go\n@@\n"}`, true, false, "", nil},
		{"commit", "bash", `{"command":"cd /r && git add -A && git commit -qm \"recall: board nudges (qawmnb)\n\nModel: x\""}`, true, true, "recall: board nudges (qawmnb)", nil},
		{"commit elsewhere", "bash", `{"command":"cd /other/repo && git commit -qm \"fix\""}`, false, true, "fix", nil},
		{"changelog commit", "bash", `{"command":"cd /r && git commit -qam \"CHANGELOG v0.7.5\""}`, false, true, "CHANGELOG v0.7.5", nil},
		{"changelog edit", "edit", `{"file_path":"/r/CHANGELOG.md"}`, false, false, "", nil},
		{"commit on the board", "bash", `{"command":"git -C .hug/tasks commit -m x"}`, false, false, "", nil},
		{"hug calls", "bash", `{"command":"hug task take 5g86n6 2>&1 | tail -1; hug task check 5g86n6 1; hug task note 5g86n6 \"x\""}`, false, false, "",
			[]HugCall{{"take", "5g86n6", 0}, {"check", "5g86n6", 1}, {"note", "5g86n6", 0}}},
		{"mcp take", "mcp__hug__task_take", `{"id":"qawmnb","model":"m","cwd":"/r"}`, false, false, "", []HugCall{{"take", "qawmnb", 0}}},
		{"mcp update check", "mcp__hug__task_update", `{"id":"qawmnb","action":"check","n":2}`, false, false, "", []HugCall{{"check", "qawmnb", 2}}},
		{"backlog mcp is not hug", "mcp__backlog__task_edit", `{"id":"TASK-1"}`, false, false, "", nil},
		{"batch", "batch", `{"tool_calls":[{"tool":"edit","file_path":"/r/a.go"},{"tool":"bash","command":"hug task show abc123"}]}`, true, false, "", []HugCall{{"show", "abc123", 0}}},
		// 4vxqp4: a hug command quoted inside another's argument never ran
		// (crab 2026-11-16 10:40, hug task new --goal quoting a lint next: line).
		{"quoted in a goal", "bash", `{"command":"cd ~/Projects/hug && hug task new \"Edit validation\" --goal \"lint says next: hug task take vk9td0 (unrelated)\""}`, false, false, "", []HugCall{{"new", "", 0}}},
		{"quoted in a note", "bash", `{"command":"hug task note rx74bp 'next: hug task edit qawmnb (was next: hug task take vk9td0)' && hug task check rx74bp 3"}`, false, false, "",
			[]HugCall{{"note", "rx74bp", 0}, {"check", "rx74bp", 3}}},
		{"sh -c runs", "bash", `{"command":"sh -c 'hug task take abc123'"}`, false, false, "", []HugCall{{"take", "abc123", 0}}},
		{"substitution runs", "bash", `{"command":"echo \"$(hug task show abc123 | head -1)\""}`, false, false, "", []HugCall{{"show", "abc123", 0}}},
		{"escaped quote", "bash", `{"command":"hug task note abc123 \"say \\\"hi\\\" then hug task done abc123\"; hug task done abc123"}`, false, false, "",
			[]HugCall{{"note", "abc123", 0}, {"done", "abc123", 0}}},
		{"unbalanced apostrophe", "bash", `{"command":"# don't guess\nhug task take abc123"}`, false, false, "", []HugCall{{"take", "abc123", 0}}},
	}
	for _, c := range cases {
		a := ParseToolCall(at, c.tool, json.RawMessage(c.input))
		if got := a.EditsIn("/r"); got != c.edits {
			t.Errorf("%s: edits=%v want %v (%+v)", c.name, got, c.edits, a)
		}
		if a.Commit != c.commit || a.Subject != c.subject {
			t.Errorf("%s: commit=%v %q want %v %q", c.name, a.Commit, a.Subject, c.commit, c.subject)
		}
		if len(a.Hug) != len(c.hug) {
			t.Errorf("%s: hug=%+v want %+v", c.name, a.Hug, c.hug)
			continue
		}
		for i := range a.Hug {
			if a.Hug[i] != c.hug[i] {
				t.Errorf("%s: hug[%d]=%+v want %+v", c.name, i, a.Hug[i], c.hug[i])
			}
		}
	}
}

func TestJcodeActivitySnapshotAndJournal(t *testing.T) {
	dir := t.TempDir()
	old := time.Date(2026, 11, 16, 1, 0, 0, 0, time.UTC)
	msg := func(id string, at time.Time, cmd string) string {
		// jcode writes the id first; the snapshot reader relies on it.
		c, _ := json.Marshal(cmd)
		return `{"id":"` + id + `","role":"assistant","timestamp":"` + at.Format(time.RFC3339Nano) + `","content":[{"type":"tool_use","name":"bash","input":{"command":` + string(c) + `}}]}`
	}
	snap := `{"id":"s","messages":[` + msg("message_1", old, "hug task show aaaaaa") + `,` + msg("message_2", old.Add(time.Hour), "hug task take bbbbbb") + `],"model":"claude-opus-5-5"}`
	os.WriteFile(filepath.Join(dir, "session_x_1.json"), []byte(snap), 0o600)
	j := `{"meta":{"model":"claude-opus-5-5"},"append_messages":[` + msg("message_3", old.Add(2*time.Hour), "hug task check bbbbbb 2") + `]}` + "\n"
	os.WriteFile(filepath.Join(dir, "session_x_1.journal.jsonl"), []byte(j), 0o600)
	act, err := JcodeActivity(dir, "session_x_1", old.Add(30*time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if act.Model != "claude-opus-5-5" || len(act.Actions) != 2 || act.Actions[0].Hug[0] != (HugCall{"take", "bbbbbb", 0}) || act.Actions[1].Hug[0] != (HugCall{"check", "bbbbbb", 2}) {
		t.Fatalf("%+v", act)
	}
}
