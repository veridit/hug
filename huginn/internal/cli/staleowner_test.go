package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/veridit/hug/huginn/internal/task"
)

// 9ahvgt: hug task lint --code stale-owner on a board whose owners are
// jcode sessions shaped as on 2026-11-18: a reaped worker (Closed, hours
// ago, in this repository), a live one (Active, just now), a name taken
// with --as by a session told "You are NAME" (kestrel by calf), and a
// session of another repository with the same name that is live there.
func TestLintStaleOwnerSessions(t *testing.T) {
	t.Parallel()
	a := app(t)
	execute(t, a, "init")
	jh := a.Env["JCODE_HOME"]
	sessions := filepath.Join(jh, "sessions")
	if err := os.MkdirAll(sessions, 0o700); err != nil {
		t.Fatal(err)
	}
	// jcode writes the folder as JSON: a Windows path's \ escaped.
	quote := func(s string) string { b, _ := json.Marshal(s); return string(b) }
	session := func(file, wd, status, title string, ago time.Duration) {
		t.Helper()
		name := strings.Split(file, "_")[1]
		body := `{"id":"` + strings.TrimSuffix(file, ".json") + `","title":"` + title + `","messages":[],"working_dir":` + quote(wd) + `,"short_name":"` + name + `","status":"` + status + `","last_pid":1}`
		p := filepath.Join(sessions, file)
		if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
		at := time.Now().Add(-ago)
		if err := os.Chtimes(p, at, at); err != nil {
			t.Fatal(err)
		}
	}
	session("session_nautilus_1794905151412_33a1.json", a.Cwd, "Closed", "You are vole, a worker", 6*time.Hour)
	session("session_macaque_1795000000000_aaaa.json", filepath.Join(a.Cwd, "sub"), "Active", "Role: engineer", time.Minute)
	session("session_calf_1794909717965_ba2b.json", a.Cwd, "Closed", "You are kestrel, a worker on hug", 5*time.Hour)
	session("session_wyvern_1795008787421_c706.json", "/elsewhere/statbus", "Active", "Statbus", time.Minute)
	session("session_wyvern_1794905939974_6ffa.json", a.Cwd, "Closed", "You are humpback", 4*time.Hour)
	ids := map[string]string{}
	for _, who := range []string{"nautilus", "macaque", "kestrel", "wyvern", "jhf"} {
		id := execute(t, a, "new", "Owned by "+who, "--goal", "G", "--purpose", "P", "--principle", "Q", "--accept", "C")["task"].(Reply)["id"].(string)
		model := "claude-opus-5-5"
		if who == "jhf" {
			model = "human"
		}
		execute(t, a, "take", id, "--as", who, "--model", model)
		ids[who] = id
	}
	r := execute(t, a, "lint", "--code", "stale-owner")
	got := map[string]string{}
	for _, f := range r["findings"].([]task.Finding) {
		id, _ := task.IDOf(f.Path)
		got[id] = f.Problem
	}
	for _, who := range []string{"nautilus", "kestrel", "wyvern"} {
		if got[ids[who]] == "" {
			t.Errorf("%s's task %s not reported; findings %q", who, ids[who], got)
		}
	}
	for _, who := range []string{"macaque", "jhf"} {
		if p := got[ids[who]]; p != "" {
			t.Errorf("%s's task reported: %s", who, p)
		}
	}
	if p := got[ids["kestrel"]]; !strings.Contains(p, "(kestrel was session calf)") {
		t.Errorf("kestrel: %s", p)
	}
	if r["warnings"] != nil {
		t.Errorf("warnings with readable sessions: %v", r["warnings"])
	}
	// A session folder hug cannot read is said once, with what to run.
	if runtime.GOOS == "windows" || os.Geteuid() == 0 {
		return
	}
	if err := os.Chmod(sessions, 0); err != nil {
		t.Fatal(err)
	}
	defer os.Chmod(sessions, 0o700)
	r = execute(t, a, "lint", "--code", "stale-owner")
	ws, _ := r["warnings"].([]string)
	if len(ws) != 1 || !strings.Contains(ws[0], "stale-owner cannot read "+sessions) || !strings.Contains(ws[0], "\nfix: ls -ld "+sessions) {
		t.Fatalf("unreadable sessions: warnings %q", ws)
	}
}

// 9ahvgt check 3 (mizaru on txjkrm, 2026-11-18): take on a task whose
// owner is gone (the stale-owner test) is refused, as before, but the fix
// takes it over in one runnable line; a live owner's task keeps "hug task
// next". Running the fix makes the task the taker's.
func TestTakeFromGoneOwner(t *testing.T) {
	t.Parallel()
	a := app(t)
	execute(t, a, "init")
	sessions := filepath.Join(a.Env["JCODE_HOME"], "sessions")
	if err := os.MkdirAll(sessions, 0o700); err != nil {
		t.Fatal(err)
	}
	quote := func(s string) string { b, _ := json.Marshal(s); return string(b) }
	session := func(file, status string, ago time.Duration) {
		t.Helper()
		name := strings.Split(file, "_")[1]
		p := filepath.Join(sessions, file)
		body := `{"title":"x","messages":[],"working_dir":` + quote(a.Cwd) + `,"short_name":"` + name + `","status":"` + status + `"}`
		if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
		at := time.Now().Add(-ago)
		if err := os.Chtimes(p, at, at); err != nil {
			t.Fatal(err)
		}
	}
	session("session_nautilus_1794905151412_33a1.json", "Closed", 6*time.Hour)
	session("session_macaque_1795000000000_aaaa.json", "Active", time.Minute)
	ids := map[string]string{}
	for _, who := range []string{"nautilus", "macaque"} {
		id := execute(t, a, "new", "Owned by "+who, "--goal", "G", "--purpose", "P", "--principle", "Q", "--accept", "C")["task"].(Reply)["id"].(string)
		execute(t, a, "take", id, "--as", who, "--model", "claude-opus-5-5")
		ids[who] = id
	}
	gone, live := ids["nautilus"], ids["macaque"]
	code, out := runText(a, "take", gone, "--as", "mizaru", "--model", "claude-opus-5-5/high")
	want := "fix: hug task set " + gone + " --assignee mizaru && hug task take " + gone + " --as mizaru --model claude-opus-5-5/high\n"
	if code != 1 || !strings.Contains(out, "taken by nautilus ("+gone+")") || !strings.Contains(out, "no live session") || !strings.HasSuffix(out, want) {
		t.Fatalf("exit %d, want 1 ending %q:\n%s", code, want, out)
	}
	if code, out := runText(a, "take", live, "--as", "mizaru", "--model", "claude-opus-5-5"); code != 1 || !strings.HasSuffix(out, "fix: hug task next\n") {
		t.Fatalf("a live owner's task: exit %d\n%s", code, out)
	}
	// The fix runs: both commands, as written.
	execute(t, a, "set", gone, "--assignee", "mizaru")
	execute(t, a, "take", gone, "--as", "mizaru", "--model", "claude-opus-5-5/high")
	if f := taskFile(t, a, gone); !strings.Contains(f, `assignee: ["mizaru"]`) {
		t.Fatalf("after the fix:\n%s", f)
	}
}
