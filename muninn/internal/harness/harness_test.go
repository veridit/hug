package harness

import (
	"strings"
	"testing"
)

func TestJcodeTurnStartEdit(t *testing.T) {
	cmd := "/usr/local/bin/muninn hook jcode"
	base := "[display]\nemoji = true\n\n[hooks]\npre_tool = \"~/bin/gate\"\nturn_start = \"~/bin/probe.sh\"\n\n[ambient]\nx = 1\n"

	out, changed, err := editJcodeHook(base, "turn_start", cmd, true)
	if err != nil || !changed {
		t.Fatalf("install: %v %v", changed, err)
	}
	if !strings.Contains(out, `turn_start = ["~/bin/probe.sh","/usr/local/bin/muninn hook jcode"]`) {
		t.Fatalf("existing hook not kept:\n%s", out)
	}
	if !strings.Contains(out, "pre_tool = \"~/bin/gate\"") || !strings.Contains(out, "[ambient]\nx = 1") {
		t.Fatalf("unrelated config disturbed:\n%s", out)
	}
	if _, changed, _ := editJcodeHook(out, "turn_start", cmd, true); changed {
		t.Fatal("install not idempotent")
	}
	back, changed, err := editJcodeHook(out, "turn_start", cmd, false)
	if err != nil || !changed || !strings.Contains(back, `turn_start = "~/bin/probe.sh"`) {
		t.Fatalf("uninstall: %v %v\n%s", changed, err, back)
	}

	// No [hooks] section yet.
	out, _, _ = editJcodeHook("[display]\nemoji = true\n", "turn_start", cmd, true)
	if !strings.Contains(out, "[hooks]\n"+jcodeMarker+"\nturn_start = \"/usr/local/bin/muninn hook jcode\"") {
		t.Fatalf("fresh install:\n%s", out)
	}
	back, _, _ = editJcodeHook(out, "turn_start", cmd, false)
	if strings.Contains(back, "muninn") {
		t.Fatalf("uninstall left traces:\n%s", back)
	}
}

// The recall hook goes on turn_start, post_tool_feedback and turn_end next
// to what is there, including a multi-line array (Jørgen's turn_end), and
// comes off them again without touching the rest (z9q42h check 3).
func TestJcodeRecallHooksBesideExistingOnes(t *testing.T) {
	cmd := "/opt/homebrew/bin/muninn hook jcode"
	base := "[hooks]\n" +
		"turn_start = \"/Users/x/.local/bin/muninn hook jcode\"\n" +
		"turn_end = [\n    \"~/.jcode/bin/disk_turn_end.sh\",\n    \"/Users/x/.local/bin/muninn capture --hook\", # capture\n]\n" +
		"session_end = \"/Users/x/.local/bin/muninn capture --hook\"\n" +
		"post_tool_feedback = '~/.jcode/bin/scratch_alarm.sh'\n" +
		"post_tool_feedback_timeout_ms = 10000\n\n[ambient]\nenabled = false\n"
	out, changed, err := editJcodeHooks(base, cmd, true)
	if err != nil || !changed {
		t.Fatalf("install: %v %v\n%s", changed, err, out)
	}
	for _, want := range []string{
		`turn_start = "/opt/homebrew/bin/muninn hook jcode"`,
		`post_tool_feedback = ["~/.jcode/bin/scratch_alarm.sh","/opt/homebrew/bin/muninn hook jcode"]`,
		`turn_end = ["~/.jcode/bin/disk_turn_end.sh","/Users/x/.local/bin/muninn capture --hook","/opt/homebrew/bin/muninn hook jcode"]`,
		`session_end = "/Users/x/.local/bin/muninn capture --hook"`,
		"post_tool_feedback_timeout_ms = 10000\n\n[ambient]\nenabled = false\n",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %s in:\n%s", want, out)
		}
	}
	if strings.Count(out, "disk_turn_end") != 1 {
		t.Errorf("the multi-line array's old lines survived:\n%s", out)
	}
	if _, changed, _ := editJcodeHooks(out, cmd, true); changed {
		t.Fatal("install not idempotent")
	}
	back, changed, err := editJcodeHooks(out, cmd, false)
	if err != nil || !changed || strings.Contains(back, "hook jcode") {
		t.Fatalf("uninstall: %v %v\n%s", changed, err, back)
	}
	for _, want := range []string{`post_tool_feedback = "~/.jcode/bin/scratch_alarm.sh"`, `turn_end = ["~/.jcode/bin/disk_turn_end.sh","/Users/x/.local/bin/muninn capture --hook"]`} {
		if !strings.Contains(back, want) {
			t.Errorf("uninstall lost %s:\n%s", want, back)
		}
	}
	if _, _, err := editJcodeHooks("[hooks]\nturn_end = [\"a\",\n", cmd, true); err == nil {
		t.Error("an unclosed array must be refused, not rewritten")
	}
}

func TestPromptHookOutputShape(t *testing.T) {
	if PromptHookOutput("") != nil {
		t.Fatal("empty block must print nothing")
	}
	got := string(PromptHookOutput("[muninn recall] x"))
	want := `{"hookSpecificOutput":{"additionalContext":"[muninn recall] x","hookEventName":"UserPromptSubmit"}}` + "\n"
	if got != want {
		t.Fatalf("got %s want %s", got, want)
	}
}

func TestUserTextSkipsNoise(t *testing.T) {
	m := jcodeMessage{Role: "user"}
	m.Content = append(m.Content, struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}{"tool_result", "ignored"})
	if userText(m) != "" {
		t.Fatal("tool result treated as user text")
	}
}
