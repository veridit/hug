package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/veridit/hug/huginn/internal/task"
)

// sessionWriter writes jcode session files into a's JCODE_HOME, as
// TestLintStaleOwnerSessions does.
func sessionWriter(t *testing.T, a *App) func(file, status string, ago time.Duration) {
	t.Helper()
	sessions := filepath.Join(a.Env["JCODE_HOME"], "sessions")
	if err := os.MkdirAll(sessions, 0o700); err != nil {
		t.Fatal(err)
	}
	quote := func(s string) string { b, _ := json.Marshal(s); return string(b) }
	return func(file, status string, ago time.Duration) {
		t.Helper()
		name := strings.Split(file, "_")[1]
		body := `{"id":"` + strings.TrimSuffix(file, ".json") + `","title":"worker","messages":[],"working_dir":` + quote(a.Cwd) + `,"short_name":"` + name + `","status":"` + status + `","last_pid":1}`
		p := filepath.Join(sessions, file)
		if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
		at := time.Now().Add(-ago)
		if err := os.Chtimes(p, at, at); err != nil {
			t.Fatal(err)
		}
	}
}

// yf39ya check 4: hug task sync gives a doing task whose worker's session
// ended to the live coordinator, stage kept, with one History line; lint
// never writes, and until the sync its stale-owner fix is hug task sync. A
// live worker's task, a person's, a name no session knows, and every task
// when the coordinator has gone too, stay where they are, and stale-owner
// keeps reporting the ones that are stale.
func TestGoneWorkerHandsToCoordinator(t *testing.T) {
	t.Parallel()
	a := app(t)
	execute(t, a, "init")
	execute(t, a, "people", "add", "jhf", "jorgen@veridit.no", "--name", "Jørgen H. Fjeld")
	execute(t, a, "coordinator", "crab")
	session := sessionWriter(t, a)
	session("session_orca_1794905151412_33a1.json", "Closed", 3*time.Hour)
	session("session_seal_1795000000000_aaaa.json", "Active", time.Minute)
	session("session_crab_1795000000001_bbbb.json", "Active", time.Minute)
	ids := map[string]string{}
	for _, who := range []string{"orca", "seal", "jhf"} {
		ids[who] = newTicket(t, a, "Owned by "+who)
		model := "claude-opus-5-5"
		if who == "jhf" {
			model = "human"
		}
		execute(t, a, "take", ids[who], "--as", who, "--model", model)
	}
	execute(t, a, "wait", ids["orca"], "CI run 7", "--as", "orca")

	// Lint reports and does not write; its fix is the sync.
	before := taskFile(t, a, ids["orca"])
	r := execute(t, a, "lint", "--code", "stale-owner")
	fs := r["findings"].([]task.Finding)
	if len(fs) != 1 || fs[0].Fix != "hug task sync" || !strings.Contains(fs[0].Problem, "hug task sync hands it to the coordinator crab") {
		t.Fatalf("stale-owner before the sync: %+v", fs)
	}
	if taskFile(t, a, ids["orca"]) != before {
		t.Fatal("lint wrote the task")
	}

	r = execute(t, a, "sync")
	if hs, _ := r["handovers"].([]string); len(hs) != 1 || !strings.Contains(hs[0], ids["orca"]+": handed to crab: orca's session ended (closed ") {
		t.Fatalf("sync handovers %v", r["handovers"])
	}
	tk := stateOf(t, a, ids["orca"])
	if strings.Join(tk.Assignee, ",") != "crab" || tk.Stage != "waiting" || tk.Fields["waiting_on"] != "CI run 7" {
		t.Fatalf("orca's task: assignee %v stage %s fields %v", tk.Assignee, tk.Stage, tk.Fields)
	}
	if f := taskFile(t, a, ids["orca"]); strings.Count(f, "handed to crab: orca's session ended") != 1 {
		t.Fatalf("History:\n%s", f)
	}
	for _, who := range []string{"seal", "jhf"} {
		if tk := stateOf(t, a, ids[who]); strings.Join(tk.Assignee, ",") != who {
			t.Fatalf("%s's task moved to %v", who, tk.Assignee)
		}
	}
	if fs := execute(t, a, "lint", "--code", "stale-owner")["findings"].([]task.Finding); len(fs) != 0 {
		t.Fatalf("stale-owner after the handover: %+v", fs)
	}
	// Again: nothing new.
	if r := execute(t, a, "sync"); r["handovers"] != nil {
		t.Fatalf("second sync handed over %v", r["handovers"])
	}

	// The coordinator gone too: nothing is handed over, stale-owner stands
	// with its old fix.
	session("session_crab_1795000000001_bbbb.json", "Closed", 2*time.Hour)
	session("session_seal_1795000000000_aaaa.json", "Closed", 2*time.Hour)
	if r := execute(t, a, "sync"); r["handovers"] != nil {
		t.Fatalf("handed to a gone coordinator: %v", r["handovers"])
	}
	got := map[string]task.Finding{}
	for _, f := range execute(t, a, "lint", "--code", "stale-owner")["findings"].([]task.Finding) {
		id, _ := task.IDOf(f.Path)
		got[id] = f
	}
	if f, ok := got[ids["seal"]]; !ok || f.Fix != "hug task set "+ids["seal"]+" --assignee NAME" {
		t.Fatalf("seal's stale-owner with a gone coordinator: %+v", got)
	}
}

// yf39ya check 4, the background pass: a write that nudges a pass is
// enough; the pass hands the gone worker's task to the coordinator and
// pushes it, with no hug task sync run by anyone.
func TestBackgroundPassHandsOver(t *testing.T) {
	t.Parallel()
	a, b, _ := clones(t)
	jh := filepath.Join(filepath.Dir(a.Cwd), ".jcode-a")
	a.Env["JCODE_HOME"] = jh
	session := sessionWriter(t, a)
	session("session_orca_1794905151412_33a1.json", "Closed", 3*time.Hour)
	session("session_crab_1795000000001_bbbb.json", "Active", time.Minute)
	execute(t, a, "coordinator", "crab")
	x := newTicket(t, a, "Orca's")
	execute(t, a, "take", x, "--as", "orca", "--model", "m")
	settled(t, a)
	// The pass child reads JCODE_HOME from its own environment.
	a.Sync.Env = append(a.Sync.Env, "JCODE_HOME="+jh, "HOME="+filepath.Join(filepath.Dir(a.Cwd), ".no-home"))
	execute(t, a, "note", x, "nudge a pass")
	settled(t, a)
	execute(t, b, "sync")
	settled(t, b)
	if tk := stateOf(t, b, x); strings.Join(tk.Assignee, ",") != "crab" {
		t.Fatalf("after the background pass, on the other clone: assignee %v\n%s", tk.Assignee, taskFile(t, b, x))
	}
}
