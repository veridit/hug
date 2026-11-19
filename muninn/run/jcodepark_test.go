package run

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/veridit/hug/muninn/internal/config"
	"github.com/veridit/hug/muninn/internal/harness"
	"github.com/veridit/hug/muninn/internal/scopetest"
)

// jcodeTurns is a jcode session the recall hook reads: the person typed
// scopetest.Query, so recall finds the fixture's memories.
type jcodeTurns struct {
	t                *testing.T
	home, sid, stamp string
	lines            []string
	// turn is the message that started the running turn (jcodemidturn_test).
	turn string
}

func newJcodeTurns(t *testing.T, f *scopetest.Fixture) *jcodeTurns {
	home := filepath.Join(f.Root, "global", ".jcode")
	for _, d := range []string{"sessions", "bin"} {
		if err := os.MkdirAll(filepath.Join(home, d), 0o700); err != nil {
			t.Fatal(err)
		}
	}
	j := &jcodeTurns{t: t, home: home, sid: "session_parktest_1", stamp: filepath.Join(home, "transcript-called")}
	// A jcode binary that records any transcript call: recall must never
	// reach the person's input box again (z9q42h).
	// (Windows cannot run the /bin/sh stub; there nothing would run it.)
	if runtime.GOOS != "windows" {
		stub := "#!/bin/sh\n{ printf '%s\\n' \"$@\"; cat; } >> \"" + j.stamp + "\"\n"
		if err := os.WriteFile(filepath.Join(home, "bin", "jcode"), []byte(stub), 0o700); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("PATH", filepath.Join(home, "bin")+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("JCODE_HOME", home)
	t.Setenv("JCODE_HOOK_SESSION_ID", j.sid)
	t.Setenv("JCODE_HOOK_CWD", f.Cwd)
	return j
}

func (j *jcodeTurns) add(line string) {
	j.lines = append(j.lines, line)
	if err := os.WriteFile(filepath.Join(j.home, "sessions", j.sid+".journal.jsonl"), []byte(strings.Join(j.lines, "\n")+"\n"), 0o600); err != nil {
		j.t.Fatal(err)
	}
}

func (j *jcodeTurns) user(id, text string) {
	j.add(`{"meta":{},"append_messages":[{"id":"` + id + `","role":"user","content":[{"type":"text","text":"` + text + `"}]}]}`)
}
func (j *jcodeTurns) toolCall() {
	j.add(`{"meta":{},"append_messages":[{"role":"assistant","content":[{"type":"text","text":"Looking."},{"type":"tool_use"}]}]}`)
}
func (j *jcodeTurns) answer() {
	j.add(`{"meta":{},"append_messages":[{"role":"assistant","content":[{"type":"text","text":"Done."}]}]}`)
}

// event runs the hook for one jcode event and returns what it printed.
func (j *jcodeTurns) event(name string) string {
	j.t.Helper()
	j.t.Setenv("JCODE_HOOK_EVENT", name)
	j.t.Setenv("JCODE_HOOK_TOOL_NAME", "bash")
	out, err := captureSurface(j.t, func() error { return runHook("jcode", 5*time.Second) })
	if err != nil {
		j.t.Fatalf("%s: %v", name, err)
	}
	return out
}

func (j *jcodeTurns) noTranscript() {
	j.t.Helper()
	if b, err := os.ReadFile(j.stamp); err == nil {
		j.t.Fatalf("recall went through jcode transcript (the person's input box):\n%s", b)
	}
}

const recallMarker = "-- AI memory (muninn):"

// A turn's recall reaches the agent once, on the first tool result of that
// turn, and never through jcode transcript (z9q42h checks 1, 2).
func TestJcodeRecallRidesTheFirstToolResult(t *testing.T) {
	f := scopetest.New(t)
	f.Bind(t, "global")
	t.Chdir(f.Cwd)
	j := newJcodeTurns(t, f)

	j.user("message_1", scopetest.Query)
	if out := j.event("turn_start"); out != "" {
		t.Fatalf("turn_start printed %q; it is an observer, the block must wait for a tool result", out)
	}
	j.noTranscript()
	j.toolCall()
	first := j.event("post_tool_feedback")
	if !strings.Contains(first, recallMarker) || !strings.Contains(first, "Matrix(") {
		t.Fatalf("first tool result of the turn lacks the recall block:\n%s", first)
	}
	if again := j.event("post_tool_feedback"); again != "" {
		t.Fatalf("second tool result repeated the block:\n%s", again)
	}
	j.answer()
	j.event("turn_end")
	j.noTranscript()

	// A turn with no tool call: the block is dropped at turn_end and never
	// shows in the next turn's tool results.
	j.user("message_2", scopetest.Query)
	j.event("turn_start")
	j.answer()
	j.event("turn_end")
	j.user("message_3", "something unrelated to any memory here")
	j.event("turn_start")
	j.toolCall()
	if out := j.event("post_tool_feedback"); strings.Contains(out, recallMarker) {
		t.Fatalf("turn 2's block showed in turn 3:\n%s", out)
	}
	j.noTranscript()
}

// A block parked for a turn that has since answered or been followed by a
// new message is dropped, even when turn_end never removed it (the observer
// hooks are detached, so jcode does not order them).
func TestJcodeRecallNeverLandsInALaterTurn(t *testing.T) {
	f := scopetest.New(t)
	f.Bind(t, "global")
	t.Chdir(f.Cwd)
	j := newJcodeTurns(t, f)
	j.user("message_1", scopetest.Query)
	j.event("turn_start")
	j.answer()
	// turn_end lost or late; the person types again, the same words: the
	// turn is the message id, not its text.
	j.user("message_2", scopetest.Query)
	j.toolCall()
	if out := j.event("post_tool_feedback"); strings.Contains(out, recallMarker) {
		t.Fatalf("message_1's block showed in message_2's turn:\n%s", out)
	}
	if harness.HasPark(harness.ParkDir(config.DataDir()), j.sid) {
		t.Fatal("the stale park was not removed")
	}
	j.noTranscript()
}

// A tool call with nothing parked costs one stat: no store open, no recall,
// no journal read (z9q42h check 4). Every recall and every claim writes a
// hook.log line or calls the probe, so both stay silent.
func TestJcodeFeedbackWithNothingParkedIsOneStat(t *testing.T) {
	f := scopetest.New(t)
	t.Chdir(f.Cwd)
	j := newJcodeTurns(t, f)
	j.user("message_1", scopetest.Query)
	logPath := filepath.Join(config.DataDir(), "hook.log")
	before, _ := os.ReadFile(logPath)
	opened := 0
	feedbackProbe = func(what string) { opened++ }
	defer func() { feedbackProbe = nil }()
	for i := 0; i < 3; i++ {
		if out := j.event("post_tool_feedback"); out != "" {
			t.Fatalf("nothing parked, yet printed %q", out)
		}
	}
	after, _ := os.ReadFile(logPath)
	if opened != 0 || len(after) != len(before) {
		t.Fatalf("feedback with nothing parked did work: %d claims, hook.log grew by:\n%s", opened, after[len(before):])
	}
	j.noTranscript()
}

// hug mem upgrade moves an install still on the turn_start-only hook (which
// typed recall into the input box) onto the tool-result hooks: preview with
// exit 5 and nothing written, --apply with a backup, then nothing to do
// (z9q42h check 3).
func TestMemUpgradeMovesJcodeRecallOffTheInputBox(t *testing.T) {
	home := memHome(t)
	jh := filepath.Join(home, "inst", ".jcode")
	if err := os.MkdirAll(jh, 0o700); err != nil {
		t.Fatal(err)
	}
	cfg := config.Defaults()
	cfg.JcodeHomes = []string{jh}
	if err := config.Save(cfg); err != nil {
		t.Fatal(err)
	}
	old := "[hooks]\n# muninn: automatic local memory recall\nturn_start = \"/old/bin/muninn hook jcode\"\nturn_end = [\n    \"~/.jcode/bin/disk_turn_end.sh\",\n    \"/old/bin/muninn capture --hook\",\n]\npost_tool_feedback = \"~/.jcode/bin/scratch_alarm.sh\"\n"
	path := filepath.Join(jh, "config.toml")
	if err := os.WriteFile(path, []byte(old), 0o600); err != nil {
		t.Fatal(err)
	}
	out, errOut, code := runCLI(t, "", "upgrade")
	if code != ExitUpgradePending || !strings.Contains(out, "would deliver jcode[") || !strings.Contains(out, "/old/bin/muninn hook jcode on turn_start, post_tool_feedback and turn_end") || !strings.Contains(out, "fix: hug mem upgrade --apply") {
		t.Fatalf("preview: exit %d\n%s%s", code, out, errOut)
	}
	if b, _ := os.ReadFile(path); string(b) != old {
		t.Fatalf("the preview wrote:\n%s", b)
	}
	out, errOut, code = runCLI(t, "", "upgrade", "--apply")
	if code != 0 || !strings.Contains(out, "recall now arrives in a tool result") {
		t.Fatalf("apply: exit %d\n%s%s", code, out, errOut)
	}
	b, _ := os.ReadFile(path)
	for _, want := range []string{
		`turn_start = "/old/bin/muninn hook jcode"`,
		`post_tool_feedback = ["~/.jcode/bin/scratch_alarm.sh","/old/bin/muninn hook jcode"]`,
		`turn_end = ["~/.jcode/bin/disk_turn_end.sh","/old/bin/muninn capture --hook","/old/bin/muninn hook jcode"]`,
	} {
		if !strings.Contains(string(b), want) {
			t.Errorf("after --apply, missing %s:\n%s", want, b)
		}
	}
	if out, errOut, code = runCLI(t, "", "upgrade"); code != 0 || !strings.Contains(out, "memory: nothing to do") {
		t.Fatalf("second run: exit %d\n%s%s", code, out, errOut)
	}
}
