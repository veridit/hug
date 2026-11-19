package run

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/veridit/hug/muninn/internal/app"
	"github.com/veridit/hug/muninn/internal/config"
	"github.com/veridit/hug/muninn/internal/ingest"
	"github.com/veridit/hug/muninn/internal/scopetest"
)

// Mid-turn recall (x8zzzn step 2). jcode starts a turn with the person's
// message and names it in every journal line's meta.model_usage_turn_id; a
// message typed while the turn runs (a soft interrupt) is appended inside
// that turn and keeps the turn's id. jcode runs no turn_start for it, so with
// recall.jcode_midturn on, the next tool result carries its recall.

// turnUser starts a jcode turn: the message names the turn.
func (j *jcodeTurns) turnUser(id, text string) {
	j.turn = id
	j.add(j.meta() + `"append_messages":[{"id":"` + id + `","role":"user","content":[{"type":"text","text":"` + text + `"}]}]}`)
}

// pasted is a message typed into the running turn: jcode keeps the turn id.
func (j *jcodeTurns) pasted(id, text string) {
	j.add(j.meta() + `"append_messages":[{"id":"` + id + `","role":"user","content":[{"type":"text","text":"` + text + `"}]}]}`)
}

// system is a soft interrupt jcode wrote, not the person (swarm DM, task).
func (j *jcodeTurns) system(id, text string) {
	j.add(j.meta() + `"append_messages":[{"id":"` + id + `","role":"user","display_role":"system","content":[{"type":"text","text":"` + text + `"}]}]}`)
}

func (j *jcodeTurns) turnTool() {
	j.add(j.meta() + `"append_messages":[{"role":"assistant","content":[{"type":"text","text":"Looking."},{"type":"tool_use"}]}]}`)
	j.add(j.meta() + `"append_messages":[{"role":"user","content":[{"type":"tool_result","text":"ok"}]}]}`)
}

func (j *jcodeTurns) turnAnswer() {
	j.add(j.meta() + `"append_messages":[{"role":"assistant","content":[{"type":"text","text":"Done."}]}]}`)
}

func (j *jcodeTurns) meta() string {
	return `{"meta":{"model_usage_turn_id":"` + j.sid + `:` + j.turn + `"},`
}

func setMidturn(t *testing.T, on bool) {
	t.Helper()
	cfg, err := config.Load()
	if err != nil {
		t.Fatal(err)
	}
	cfg.Recall.JcodeMidturn = &on
	if err := config.Save(cfg); err != nil {
		t.Fatal(err)
	}
}

// kilnQuery finds one memory the fixture's Query does not: a second message
// with recall of its own.
const kilnQuery = "tarragon kiln calibration"

func addKilnMemory(t *testing.T, f *scopetest.Fixture) {
	t.Helper()
	if _, _, err := ingest.RememberTyped(app.NativePath(), "Tarragon kiln calibration uses the blue gauge, never the red one.", "fact", "rule", "global", nil, ""); err != nil {
		t.Fatal(err)
	}
	if _, err := f.App.Reindex(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func midturnFixture(t *testing.T) (*scopetest.Fixture, *jcodeTurns) {
	t.Helper()
	f := scopetest.New(t)
	f.Bind(t, "global")
	t.Chdir(f.Cwd)
	listed := staleMCPListed
	staleMCPListed = false
	t.Cleanup(func() { staleMCPListed = listed })
	return f, newJcodeTurns(t, f)
}

// The switch is off by default: config.json without it reads as off.
func TestJcodeMidturnIsOffByDefault(t *testing.T) {
	memHome(t)
	cfg, err := config.Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.JcodeMidturn() {
		t.Fatal("recall.jcode_midturn is on by default")
	}
}

// A message pasted into a running turn gets its recall in the next tool
// result, once; with the switch off it gets none (the step 1 finding).
// Nothing is ever printed outside a tool result: turn_start and turn_end
// stay silent, so an idle session is never woken for a memory.
func TestJcodeMidturnRecallsAPastedMessage(t *testing.T) {
	for _, on := range []bool{false, true} {
		t.Run(map[bool]string{false: "off", true: "on"}[on], func(t *testing.T) {
			_, j := midturnFixture(t)
			setMidturn(t, on)
			j.turnUser("message_1", "something unrelated to any memory here")
			if out := j.event("turn_start"); out != "" {
				t.Fatalf("turn_start printed %q", out)
			}
			j.turnTool()
			if out := j.event("post_tool_feedback"); strings.Contains(out, recallMarker) {
				t.Fatalf("first tool result has a block for an unrelated message:\n%s", out)
			}
			j.pasted("message_2", scopetest.Query)
			j.turnTool()
			got := j.event("post_tool_feedback")
			if !on {
				if strings.Contains(got, recallMarker) {
					t.Fatalf("switch off, yet the pasted message was recalled:\n%s", got)
				}
				return
			}
			if !strings.Contains(got, recallMarker) || !strings.Contains(got, "Matrix(") {
				t.Fatalf("the tool result after a pasted message lacks its recall:\n%s", got)
			}
			j.turnTool()
			if again := j.event("post_tool_feedback"); strings.Contains(again, recallMarker) {
				t.Fatalf("the pasted message was recalled twice:\n%s", again)
			}
			// A system soft interrupt (swarm DM, background task) is not the
			// person typing: no recall for it.
			j.system("message_3", scopetest.Query+" again")
			j.turnTool()
			if out := j.event("post_tool_feedback"); strings.Contains(out, recallMarker) {
				t.Fatalf("a system soft interrupt was recalled:\n%s", out)
			}
			// Pasted after the last tool call: no tool result follows, so it
			// is counted at turn_end, never pushed into the session.
			j.pasted("message_4", kilnQuery)
			j.turnAnswer()
			if out := j.event("turn_end"); out != "" {
				t.Fatalf("turn_end printed %q", out)
			}
			log := hookLog(t)
			if !strings.Contains(log, "midturn turn=message_1 msg=message_2") {
				t.Fatalf("hook.log lacks the mid-turn delivery line:\n%s", log)
			}
			if !strings.Contains(log, "midturn turn=message_1 missed=1") {
				t.Fatalf("hook.log lacks the missed count at turn_end:\n%s", log)
			}
			j.noTranscript()
		})
	}
}

// A message pasted before the turn's first tool result no longer makes the
// parked block stale: the tool result carries the parked block first, then
// the pasted message's own. Off, the old path drops the parked block.
func TestJcodeMidturnPasteKeepsTheParkedBlock(t *testing.T) {
	for _, on := range []bool{false, true} {
		t.Run(map[bool]string{false: "off", true: "on"}[on], func(t *testing.T) {
			f, j := midturnFixture(t)
			addKilnMemory(t, f)
			setMidturn(t, on)
			j.turnUser("message_1", scopetest.Query)
			j.event("turn_start")
			j.pasted("message_2", kilnQuery)
			j.turnTool()
			got := j.event("post_tool_feedback")
			log := hookLog(t)
			if !on {
				if strings.Contains(got, recallMarker) || !strings.Contains(log, "dropped=stale parked=message_1") {
					t.Fatalf("switch off: expected the old stale drop\nout:\n%s\nlog:\n%s", got, log)
				}
				return
			}
			parked, pasted := strings.Index(got, "Matrix("), strings.Index(got, "Tarragon kiln")
			if parked < 0 || pasted < 0 || parked > pasted {
				t.Fatalf("want the parked block, then the pasted message's block:\n%s", got)
			}
			if strings.Contains(log, "dropped=stale") {
				t.Fatalf("a paste still drops the parked block:\n%s", log)
			}
			if !strings.Contains(log, "delivered turn=message_1") {
				t.Fatalf("hook.log lacks the parked delivery:\n%s", log)
			}
			j.noTranscript()
		})
	}
}

// The model answered with no tool call and the person typed into the turn
// (jcode point B: the same turn goes on). The parked block and the new
// message's recall both arrive at the next tool result.
func TestJcodeMidturnAfterAnAnswerInTheSameTurn(t *testing.T) {
	f, j := midturnFixture(t)
	addKilnMemory(t, f)
	setMidturn(t, true)
	j.turnUser("message_1", scopetest.Query)
	j.event("turn_start")
	j.turnAnswer()
	j.pasted("message_2", kilnQuery)
	j.turnTool()
	got := j.event("post_tool_feedback")
	if !strings.Contains(got, "Matrix(") || !strings.Contains(got, "Tarragon kiln") {
		t.Fatalf("point B: want both blocks:\n%s", got)
	}
}

// A new turn is still a new turn: a block parked for an answered turn never
// lands in the next one, switch on or not (z9q42h).
func TestJcodeMidturnNeverLandsInALaterTurn(t *testing.T) {
	_, j := midturnFixture(t)
	setMidturn(t, true)
	j.turnUser("message_1", scopetest.Query)
	j.event("turn_start")
	j.turnAnswer()
	// turn_end lost; the person starts a new turn with the same words.
	j.turnUser("message_2", scopetest.Query)
	j.turnTool()
	if out := j.event("post_tool_feedback"); strings.Contains(out, recallMarker) {
		t.Fatalf("message_1's block showed in message_2's turn:\n%s", out)
	}
	if log := hookLog(t); !strings.Contains(log, "dropped=stale parked=message_1") {
		t.Fatalf("hook.log lacks the stale drop:\n%s", log)
	}
}

// The recall-path rule: mid-turn delivery changes when a block arrives, not
// what is recalled. The same message, recalled as a turn's first message
// (old path, switch off) in one session and pasted mid-turn (new path) in
// another session of the same store, prints byte-identical output.
func TestJcodeMidturnRecallIsByteIdentical(t *testing.T) {
	f, j := midturnFixture(t)
	addKilnMemory(t, f)
	const msg = scopetest.Query + " and " + kilnQuery

	setMidturn(t, false)
	j.sid = "session_oldpath_1"
	t.Setenv("JCODE_HOOK_SESSION_ID", j.sid)
	j.lines = nil
	j.turnUser("message_1", msg)
	j.event("turn_start")
	j.turnTool()
	old := j.event("post_tool_feedback")

	setMidturn(t, true)
	j.sid = "session_newpath_1"
	t.Setenv("JCODE_HOOK_SESSION_ID", j.sid)
	j.lines = nil
	j.turnUser("message_1", "something unrelated to any memory here")
	j.event("turn_start")
	j.turnTool()
	if out := j.event("post_tool_feedback"); out != "" {
		t.Fatalf("new path: unrelated first message printed %q", out)
	}
	j.pasted("message_2", msg)
	j.turnTool()
	mid := j.event("post_tool_feedback")

	if !strings.Contains(old, "Matrix(") || !strings.Contains(old, "Tarragon kiln") {
		t.Fatalf("old path recalled too little to compare:\n%s", old)
	}
	if old != mid {
		t.Fatalf("recall differs between the paths\nold (%d bytes):\n%q\nmid-turn (%d bytes):\n%q", len(old), old, len(mid), mid)
	}
	t.Logf("byte-identical: %d bytes through both paths", len(old))
}

// Every memory hug hands to the agent gets a delivered time in injections;
// a block dropped with no tool call leaves it 0, so the store can count
// delivered against recalled.
func TestJcodeInjectionsRecordDelivery(t *testing.T) {
	f, j := midturnFixture(t)
	setMidturn(t, false)
	j.turnUser("message_1", scopetest.Query)
	j.event("turn_start")
	j.turnTool()
	j.event("post_tool_feedback")

	count := func(session string) (all, delivered int) {
		t.Helper()
		if err := f.App.DB.QueryRow(`SELECT COUNT(*), COALESCE(SUM(delivered_ns>0),0) FROM injections WHERE session=?`, session).Scan(&all, &delivered); err != nil {
			t.Fatal(err)
		}
		return
	}
	if all, d := count("jcode:" + j.sid); all == 0 || d != all {
		t.Fatalf("delivered block: %d injections, %d with a delivered time", all, d)
	}

	j.sid = "session_dropped_1"
	t.Setenv("JCODE_HOOK_SESSION_ID", j.sid)
	j.lines = nil
	j.turnUser("message_1", scopetest.Query)
	j.event("turn_start")
	j.turnAnswer()
	j.event("turn_end")
	if all, d := count("jcode:" + j.sid); all == 0 || d != 0 {
		t.Fatalf("dropped block: %d injections, %d with a delivered time (want 0)", all, d)
	}
}

// A park overwritten before any tool call or turn_end took it says so in
// hook.log; before, it vanished and showed only as "never resolved".
func TestJcodeParkOverwriteIsLogged(t *testing.T) {
	f, j := midturnFixture(t)
	addKilnMemory(t, f)
	setMidturn(t, false)
	j.turnUser("message_1", scopetest.Query)
	j.event("turn_start")
	j.turnAnswer()
	j.turnUser("message_2", kilnQuery)
	j.event("turn_start")
	if log := hookLog(t); !strings.Contains(log, "replaced parked=message_1 by=message_2") {
		t.Fatalf("hook.log lacks the overwrite line:\n%s", log)
	}
}

// A hook error (a timeout among them) names the session and the event, so
// it can be tied to a turn.
func TestHookErrorLineNamesTheSession(t *testing.T) {
	home := memHome(t)
	jh := filepath.Join(home, "jc")
	os.MkdirAll(filepath.Join(jh, "sessions"), 0o700)
	t.Setenv("JCODE_HOME", jh)
	t.Setenv("JCODE_HOOK_SESSION_ID", "session_errtest_17950")
	t.Setenv("JCODE_HOOK_EVENT", "turn_start")
	if _, errOut, code := runCLI(t, "", "hook", "jcode"); code != 0 {
		t.Fatalf("hook exit %d: %s", code, errOut)
	}
	if log := hookLog(t); !strings.Contains(log, "jcode session_errtest_17950 turn_start: ") {
		t.Fatalf("the error line lacks the session and event:\n%s", log)
	}
}
