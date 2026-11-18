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
