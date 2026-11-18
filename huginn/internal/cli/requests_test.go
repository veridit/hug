package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// hbgybn checks 1 and 2 (CLI): hug task request ID - stores a multi-line
// request verbatim with who and when; hug task requests lists every open
// one across the board with its resolve command; hug task resolve ID N
// closes it with the account and who; an unknown N is refused with a fix.
func TestRequestCommands(t *testing.T) {
	t.Parallel()
	a := app(t)
	execute(t, a, "init")
	id := execute(t, a, "new", "Tags", "--goal", "g", "--purpose", "p", "--principle", "q", "--accept", "a")["task"].(Reply)["id"].(string)
	other := execute(t, a, "new", "Rows", "--goal", "g", "--purpose", "p", "--principle", "q", "--accept", "a")["task"].(Reply)["id"].(string)
	text := "make it per board: \"tags\"\nand split the TUI part"
	a.In = strings.NewReader(text + "\n")
	execute(t, a, "request", id, "-", "--as", "jhf")
	execute(t, a, "request", other, "shorter title", "--as", "jhf")
	f := taskFile(t, a, id)
	if !strings.Contains(f, "## Requests\n- 20") || !strings.Contains(f, " request 1 by jhf:\n  make it per board: \"tags\"\n  and split the TUI part\n") {
		t.Fatalf("request not stored verbatim:\n%s", f)
	}
	code, out := runText(a, "requests")
	for _, want := range []string{id + " request 1 by jhf, ", "  make it per board: \"tags\"\n  and split the TUI part\n", "  fix: hug task resolve " + id + " 1 \"what changed\"\n", other + " request 1 by jhf"} {
		if code != 0 || !strings.Contains(out, want) {
			t.Fatalf("requests lacks %q (exit %d):\n%s", want, code, out)
		}
	}
	if code, out = runText(a, "resolve", id, "4", "x", "--as", "nautilus"); code != 2 || !strings.Contains(out, id+" has no request 4; its requests are 1 (open: 1)") || !strings.Contains(out, "fix: hug task requests "+id+"\n") {
		t.Fatalf("resolve 4: exit %d\n%s", code, out)
	}
	a.In = strings.NewReader("goal now per board\nnew task for the TUI\n")
	execute(t, a, "resolve", id, "1", "-", "--as", "nautilus")
	if f = taskFile(t, a, id); !strings.Contains(f, "  and split the TUI part\n  resolved 20") || !strings.Contains(f, " by nautilus:\n    goal now per board\n    new task for the TUI\n") {
		t.Fatalf("resolve not under the request:\n%s", f)
	}
	if _, out = runText(a, "requests"); strings.Contains(out, id+" request 1") || !strings.Contains(out, other+" request 1") {
		t.Fatalf("a resolved request still listed, or the open one lost:\n%s", out)
	}
	if _, out = runText(a, "requests", id, "--all"); !strings.Contains(out, "resolved by nautilus") || !strings.Contains(out, "    new task for the TUI\n") {
		t.Fatalf("requests --all:\n%s", out)
	}
	// show prints the request and its account.
	if _, out = runText(a, "show", id); !strings.Contains(out, "request 1 by jhf, ") || !strings.Contains(out, "  resolved by nautilus, ") {
		t.Fatalf("show:\n%s", out)
	}
	if log := git(t, filepath.Join(a.Cwd, TaskDir), "log", "--format=%s", "-3"); !strings.Contains(log, "hug task resolve "+id+" 1") || !strings.Contains(log, "hug task request "+other) {
		t.Fatalf("commits:\n%s", log)
	}
}

// hbgybn (crab 00:58): an open request is the first thing hug task next
// prints for whoever it routes to: the assignee; on an unowned task, the
// coordinator (hug task coordinator NAME, in hug.toml [board]); with none
// set, everyone. hug task coordinator writes and reads the name.
func TestRequestRoutesThroughNext(t *testing.T) {
	t.Parallel()
	a := app(t)
	execute(t, a, "init")
	first := execute(t, a, "new", "First", "--goal", "g", "--purpose", "p", "--principle", "q", "--accept", "a")["task"].(Reply)["id"].(string)
	id := execute(t, a, "new", "Later", "--goal", "g", "--purpose", "p", "--principle", "q", "--accept", "a")["task"].(Reply)["id"].(string)
	next := func(who string) Reply {
		t.Helper()
		return execute(t, a, "next", "--as", who)
	}
	if got := next("wyvern")["task"].(Reply)["id"]; got != first {
		t.Fatalf("before any request next is %v, want %s", got, first)
	}
	execute(t, a, "request", id, "the goal is vague", "--as", "jhf")
	// No coordinator: everyone hears an unowned request.
	for _, who := range []string{"wyvern", "crab"} {
		r := next(who)
		if r["task"].(Reply)["id"] != id || !strings.Contains(r["next"].(string), "hug task resolve "+id+" 1 \"what changed\"") {
			t.Fatalf("next for %s: %v", who, r)
		}
	}
	// Setup names the missing coordinator, with the command (crab 00:58).
	if st, _ := setupStep(t, a, "coordinator"); st.Done || st.Do != "hug task coordinator NAME" || !strings.Contains(st.Detail, "no coordinator set") {
		t.Fatalf("setup without a coordinator: %+v", st)
	}
	// A coordinator: only they hear it.
	if c := execute(t, a, "coordinator")["coordinator"]; c != "" {
		t.Fatalf("coordinator before any: %v", c)
	}
	execute(t, a, "coordinator", "crab")
	if c := execute(t, a, "coordinator")["coordinator"]; c != "crab" {
		t.Fatalf("coordinator: %v", c)
	}
	if st, _ := setupStep(t, a, "coordinator"); !st.Done || !strings.Contains(st.Detail, "coordinator is crab") {
		t.Fatalf("setup with a coordinator: %+v", st)
	}
	toml, _ := os.ReadFile(filepath.Join(a.Cwd, TaskDir, "hug.toml"))
	if !strings.Contains(string(toml), "[board]\ncoordinator = \"crab\"\n") || !strings.HasPrefix(string(toml), layoutLine) {
		t.Fatalf("hug.toml:\n%s", toml)
	}
	if got := next("wyvern")["task"].(Reply)["id"]; got != first {
		t.Fatalf("wyvern, not the coordinator, got the request: %v", got)
	}
	if got := next("crab")["task"].(Reply)["id"]; got != id {
		t.Fatalf("the coordinator did not get the request: %v", got)
	}
	// Taken: the assignee hears it, not the coordinator.
	execute(t, a, "take", id, "--as", "nautilus", "--model", "m")
	if got := next("crab")["task"].(Reply)["id"]; got == id {
		t.Fatalf("the coordinator still gets a request on a held task")
	}
	if r := execute(t, a, "next", "--as", "nautilus"); r["task"].(Reply)["id"] != id {
		t.Fatalf("the assignee does not get its request: %v", r)
	}
	// Resolved: next goes back to normal.
	execute(t, a, "resolve", id, "1", "goal rewritten", "--as", "nautilus")
	if got := next("crab")["task"].(Reply)["id"]; got == id {
		t.Fatalf("a resolved request still comes first")
	}
	execute(t, a, "coordinator", "--clear")
	if toml, _ = os.ReadFile(filepath.Join(a.Cwd, TaskDir, "hug.toml")); strings.Contains(string(toml), "coordinator") {
		t.Fatalf("--clear kept it:\n%s", toml)
	}
}

// 41hhgn check 11 (task half): after a person answers, the asker's next
// write on that task is recorded under the answer as its outcome, in the
// read line, and show --json gives it as "outcome".
func TestAnswerOutcomeRecorded(t *testing.T) {
	t.Parallel()
	a := app(t)
	execute(t, a, "init")
	id := execute(t, a, "new", "Look", "--goal", "g", "--purpose", "p", "--principle", "q", "--accept", "a", "--accept", "b")["task"].(Reply)["id"].(string)
	execute(t, a, "take", id, "--as", "nautilus", "--model", "m")
	execute(t, a, "ask", id, "--to", "jhf", "--check", "2", "which rows?", "--as", "nautilus")
	execute(t, a, "answer", id, "the open ones", "--as", "jhf")
	if o := execute(t, a, "show", id)["task"].(Reply)["outcome"]; o != nil {
		t.Fatalf("outcome before the asker acted: %v", o)
	}
	a.Env["HUG_USER"] = "nautilus"
	execute(t, a, "note", id, "rows now list the open ones first")
	f := taskFile(t, a, id)
	if !strings.Contains(f, ` nautilus read the answer: hug task note `+id+` "rows now list the open ones first"`+"\n") {
		t.Fatalf("the outcome is not under the answer:\n%s", f)
	}
	o, _ := execute(t, a, "show", id)["task"].(Reply)["outcome"].(map[string]any)
	if o["for"] != "nautilus" || o["by"] != "jhf" || o["stage"] != "asked" || o["text"] != `hug task note `+id+` "rows now list the open ones first"` {
		t.Fatalf("show --json outcome: %#v", o)
	}
	// Only the first write after the answer: a second note changes nothing.
	execute(t, a, "note", id, "and again")
	if f = taskFile(t, a, id); strings.Count(f, "read the answer") != 1 {
		t.Fatalf("a second write marked it again:\n%s", f)
	}
}
