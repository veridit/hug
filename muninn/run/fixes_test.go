package run

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/veridit/hug/muninn/internal/app"
	"github.com/veridit/hug/muninn/internal/config"
	"github.com/veridit/hug/muninn/internal/store"
)

// rjk44w: the CLI writes the trail (forget --reason, remember --supersedes
// and --reinforce) and hug mem stats --fixes reports it.
func TestCLIFixTrailAndStats(t *testing.T) {
	memHome(t)
	t.Setenv("JCODE_SESSION_ID", "session_otter_1_abc")
	ids := rememberN(t, 3)
	if _, _, code := runCLI(t, "", "index"); code != 0 {
		t.Fatal("index failed")
	}
	// Recall showed ids[0] to this session a minute ago.
	db, err := store.Open(config.DBPath())
	if err != nil {
		t.Fatal(err)
	}
	u, err := db.Resolve(ids[0])
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO injections(session,unit_id,content_hash,harness,at_ns) VALUES('jcode:session_otter_1_abc',?,?,'jcode',?)`, u.ID, u.ContentHash, time.Now().Add(-time.Minute).UnixNano()); err != nil {
		t.Fatal(err)
	}
	db.Close()
	if out, errOut, code := runCLI(t, "", "forget", ids[0], "--reason", "superseded by the 2026 policy"); code != 0 {
		t.Fatalf("forget: %s%s", out, errOut)
	}
	if out, errOut, code := runCLI(t, "", "remember", "me51r2 quartz moved to the 2026 policy", "--supersedes", ids[1], "--new-reason", "a correction"); code != 0 {
		t.Fatalf("remember --supersedes: %s%s", out, errOut)
	}
	out, errOut, code := runCLI(t, "", "stats", "--fixes", "--json")
	if code != 0 {
		t.Fatalf("stats --fixes: %s%s", out, errOut)
	}
	var s app.FixSummary
	if err := json.Unmarshal([]byte(out), &s); err != nil {
		t.Fatalf("%v: %s", err, out)
	}
	today := time.Now().UTC().Format("2006-01-02")
	if len(s.Days) != 1 || s.Days[0].Day != today || s.Days[0].Forgets != 1 || s.Days[0].Supersedes != 1 || s.Days[0].Fixes != 2 || s.Days[0].OfShown != 1 || s.Days[0].Blocks < 1 {
		t.Fatalf("summary %+v", s)
	}
	if len(s.Recent) != 2 || s.Recent[0].Reason != "superseded by the 2026 policy" || s.Recent[0].Session != "jcode:session_otter_1_abc" || !s.Recent[0].Shown || s.Recent[0].Surface != "cli" {
		t.Fatalf("recent %+v", s.Recent)
	}
	// Text form: a header, one row per day, then the recent fixes.
	out, _, code = runCLI(t, "", "stats", "--fixes")
	for _, want := range []string{"day         blocks  fixes  of-shown  not-shown  forget  reinforce  supersede  undone  rescope", today + " ", "forget " + ids[0] + " shown 1m", "reason: superseded by the 2026 policy", "supersede " + ids[1] + " -> "} {
		if code != 0 || !strings.Contains(out, want) {
			t.Fatalf("text lacks %q:\n%s", want, out)
		}
	}
	// --since drops older days.
	out, _, code = runCLI(t, "", "stats", "--fixes", "--since", time.Now().Add(48*time.Hour).Format("2006-01-02"), "--json")
	if err := json.Unmarshal([]byte(out), &s); err != nil || code != 0 || len(s.Days) != 0 {
		t.Fatalf("--since: %d %v %s", code, err, out)
	}
	// Plain stats is unchanged: no fixes key.
	out, _, _ = runCLI(t, "", "stats")
	if strings.Contains(out, "fixes") {
		t.Fatalf("plain stats changed:\n%s", out)
	}
}

// rjk44w check 3: the text report over a fixture store, compared with a
// golden file (HUG_UPDATE_GOLDEN=1 rewrites it).
func TestFixReportGolden(t *testing.T) {
	day := time.Date(2026, 11, 16, 9, 0, 0, 0, time.UTC)
	s := app.FixSummary{
		Days: []app.FixDay{
			{Day: "2026-11-16", Blocks: 41, Fixes: 3, OfShown: 2, NotShown: 1, Forgets: 2, Reinforces: 1, Undone: 1},
			{Day: "2026-11-17", Blocks: 37, Fixes: 2, OfShown: 0, NotShown: 2, Supersedes: 1, Rescopes: 1},
		},
		Instances: []app.FixInstance{{Instance: "global", Fixes: 3, OfShown: 2}, {Instance: "seid", Fixes: 2}},
		Recent: []app.Fix{
			{At: day, Action: "forget", SID: "k7m2qx", Session: "jcode:session_crab_1_a", Instance: "global", Surface: "mcp", Shown: true, ShownAgo: 3 * time.Minute, Reason: "the reboot happened"},
			{At: day.Add(25 * time.Hour), Action: "supersede", SID: "3nfhgr", NewSID: "9xybgp", Surface: "cli", Instance: "seid"},
			{At: day.Add(26 * time.Hour), Action: "rescope", SID: "p4w8zd", OldScope: "global", NewScope: "project:seid-lang", Mode: "batch", Surface: "cli", Instance: "seid"},
		},
	}
	got := app.FixReport(s)
	path := filepath.Join("testdata", "fixes.golden")
	if os.Getenv("HUG_UPDATE_GOLDEN") == "1" {
		if err := os.MkdirAll("testdata", 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("%v (HUG_UPDATE_GOLDEN=1 writes it)", err)
	}
	if got != strings.ReplaceAll(string(want), "\r\n", "\n") { // a Windows checkout may turn LF into CRLF
		t.Fatalf("report differs from %s:\n%s", path, got)
	}
}
