package app_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/veridit/hug/muninn/internal/app"
	"github.com/veridit/hug/muninn/internal/scopetest"
)

// fixRows reads the trail rows for sid, oldest first.
func fixRows(t *testing.T, f *scopetest.Fixture, sid string) []app.Fix {
	t.Helper()
	fs, err := app.Fixes(context.Background(), f.App, time.Time{})
	if err != nil {
		t.Fatal(err)
	}
	var out []app.Fix
	for _, x := range fs {
		if x.SID == sid {
			out = append(out, x)
		}
	}
	return out
}

// rjk44w check 1: a forget leaves one trail row with the session, harness,
// instance, surface, the reason, and whether recall showed that memory to
// the session first and how long before. Recording never blocks the forget.
func TestForgetLeavesTrail(t *testing.T) {
	f := scopetest.New(t)
	ctx := context.Background()
	b := f.App.Bound("", f.Cwd)
	shown, hidden := f.Units["Shared"], f.Units["Project"]
	at := time.Now().Add(-90 * time.Second)
	if _, err := f.App.DB.Exec(`INSERT INTO injections(session, unit_id, content_hash, harness, at_ns) VALUES('jcode:s1', ?, ?, 'jcode', ?)`, shown.ID, shown.ContentHash, at.UnixNano()); err != nil {
		t.Fatal(err)
	}
	fc := app.FixContext{Sessions: []string{"jcode:s1"}, Harness: "jcode", Instance: "global", Surface: "mcp", Reason: "the reboot happened on 2026-11-15"}
	if _, err := b.Forget(app.WithFix(ctx, fc), f.App, shown.SID, "jcode:s1"); err != nil {
		t.Fatal(err)
	}
	rows := fixRows(t, f, shown.SID)
	if len(rows) != 1 {
		t.Fatalf("rows %+v", rows)
	}
	r := rows[0]
	if r.Action != "forget" || r.Session != "jcode:s1" || r.Harness != "jcode" || r.Instance != "global" || r.Surface != "mcp" || r.Reason != fc.Reason || !r.Shown || r.ShownAgo < 89*time.Second || r.ShownAgo > time.Hour || r.ContentHash != shown.ContentHash {
		t.Fatalf("row %+v", r)
	}
	// Not shown in this session: recorded, shown=false, no reason.
	if _, err := b.Forget(app.WithFix(ctx, app.FixContext{Sessions: []string{"jcode:s1"}, Harness: "jcode", Surface: "cli"}), f.App, hidden.SID); err != nil {
		t.Fatal(err)
	}
	if rows := fixRows(t, f, hidden.SID); len(rows) != 1 || rows[0].Shown || rows[0].Reason != "" || rows[0].Surface != "cli" || rows[0].ShownAgo != 0 {
		t.Fatalf("hidden rows %+v", rows)
	}
	// A forget with no context at all (an old caller) is still recorded.
	if _, err := b.Forget(ctx, f.App, f.Units["Personal"].SID); err != nil {
		t.Fatal(err)
	}
	if rows := fixRows(t, f, f.Units["Personal"].SID); len(rows) != 1 || rows[0].Session != "" || rows[0].Action != "forget" {
		t.Fatalf("bare rows %+v", rows)
	}
	// A failed forget (unknown id) records nothing.
	before, _ := app.Fixes(ctx, f.App, time.Time{})
	if _, err := b.Forget(ctx, f.App, "zzzzzz"); err == nil {
		t.Fatal("unknown id forgot")
	}
	if after, _ := app.Fixes(ctx, f.App, time.Time{}); len(after) != len(before) {
		t.Fatalf("failed forget recorded: %d -> %d", len(before), len(after))
	}
}

// rjk44w check 2: a remember that names an old id records the same row:
// reinforce=ID, and supersedes=ID (which only records; it forgets nothing).
func TestRememberNamingOldIDLeavesTrail(t *testing.T) {
	f := scopetest.New(t)
	ctx := context.Background()
	b := f.Bind(t, "alpha")
	old := f.Units["Alpha"]
	if _, err := f.App.DB.Exec(`INSERT INTO injections(session, unit_id, content_hash, harness, at_ns) VALUES('mcp:abc', ?, ?, 'mcp', ?)`, old.ID, old.ContentHash, time.Now().UnixNano()); err != nil {
		t.Fatal(err)
	}
	fc := app.WithFix(ctx, app.FixContext{Sessions: []string{"mcp:abc"}, Harness: "mcp", Surface: "mcp", Reason: "still holds"})
	if _, _, err := b.RememberPlaced(fc, f.App, old.Body, "fact", "", nil, "", "", "", app.RememberDecision{Reinforce: old.SID}); err != nil {
		t.Fatal(err)
	}
	rows := fixRows(t, f, old.SID)
	if len(rows) != 1 || rows[0].Action != "reinforce" || !rows[0].Shown || rows[0].Reason != "still holds" || rows[0].Session != "mcp:abc" {
		t.Fatalf("reinforce rows %+v", rows)
	}
	u, _, err := b.RememberPlaced(app.WithFix(ctx, app.FixContext{Surface: "cli"}), f.App, "Alpha quartz deployment moved to Thursdays in 2026", "correction", "", nil, "", "", "", app.RememberDecision{Supersedes: old.SID, NewReason: "a correction"})
	if err != nil {
		t.Fatal(err)
	}
	rows = fixRows(t, f, old.SID)
	if len(rows) != 2 || rows[1].Action != "supersede" || rows[1].NewSID != u.SID || rows[1].Shown {
		t.Fatalf("supersede rows %+v", rows)
	}
	// supersedes records only: the old memory stays active.
	if got, err := f.App.DB.Resolve(old.SID); err != nil || !got.Active {
		t.Fatalf("supersedes retired the old memory: %+v %v", got, err)
	}
	// A plain remember names no old id and records nothing.
	n := len(rows)
	if _, _, err := b.RememberPlaced(ctx, f.App, "Alpha quartz audit happens yearly on a separate policy page", "fact", "", nil, "", "", "", app.RememberDecision{NewReason: "different"}); err != nil {
		t.Fatal(err)
	}
	if all, _ := app.Fixes(ctx, f.App, time.Time{}); len(all) != n {
		t.Fatalf("plain remember recorded: %+v", all)
	}
	// supersedes must name a visible memory.
	if _, _, err := b.RememberPlaced(ctx, f.App, "Alpha quartz again", "fact", "", nil, "", "", "", app.RememberDecision{Supersedes: "zzzzzz", NewReason: "x"}); err == nil || !strings.Contains(err.Error(), "fix: ") {
		t.Fatalf("unknown supersedes: %v", err)
	}
}

// A correction is near its old memory by nature: naming that memory in
// supersedes answers the duplicate question, any other near memory still asks.
func TestSupersedesAnswersDuplicateQuestion(t *testing.T) {
	f := scopetest.New(t)
	useBands(t, f, .92) // the question band: asks unless answered
	b := f.Bind(t, "alpha")
	ctx := context.Background()
	if _, _, err := b.Remember(ctx, f.App, bandContent, "fact", "", nil); err == nil || !strings.Contains(err.Error(), "Same fact?") {
		t.Fatalf("band did not ask: %v", err)
	}
	if _, _, err := b.RememberPlaced(ctx, f.App, bandContent, "fact", "", nil, "", "", "", app.RememberDecision{Supersedes: f.Units["Project"].SID}); err == nil || !strings.Contains(err.Error(), "Same fact?") {
		t.Fatalf("superseding another memory skipped the question: %v", err)
	}
	u, re, err := b.RememberPlaced(ctx, f.App, bandContent, "fact", "", nil, "", "", "", app.RememberDecision{Supersedes: f.Units["Alpha"].SID})
	if err != nil || re || u.SID == f.Units["Alpha"].SID {
		t.Fatalf("u=%+v reinforced=%v err=%v", u, re, err)
	}
	if rows := fixRows(t, f, f.Units["Alpha"].SID); len(rows) != 1 || rows[0].NewSID != u.SID {
		t.Fatalf("rows %+v", rows)
	}
}

// rjk44w check 4: a forget whose memory is named again (reinforced, or
// superseded naming it) within 7 days counts as undone in the summary.
func TestFixSummaryCountsUndone(t *testing.T) {
	f := scopetest.New(t)
	ctx := context.Background()
	b := f.App.Bound("", f.Cwd)
	day := time.Date(2026, 11, 17, 10, 0, 0, 0, time.UTC)
	for i, sid := range []string{f.Units["Shared"].SID, f.Units["Project"].SID, f.Units["Personal"].SID} {
		if _, err := b.Forget(app.WithFix(ctx, app.FixContext{Surface: "cli", Now: day.Add(time.Duration(i) * time.Minute)}), f.App, sid); err != nil {
			t.Fatal(err)
		}
	}
	// Shared comes back active two days later: undone.
	if err := app.RecordFix(ctx, f.App, app.Fix{Action: "reinforce", SID: f.Units["Shared"].SID, At: day.Add(48 * time.Hour)}); err != nil {
		t.Fatal(err)
	}
	// Project is named again after 8 days: not undone.
	if err := app.RecordFix(ctx, f.App, app.Fix{Action: "supersede", SID: f.Units["Project"].SID, At: day.Add(8 * 24 * time.Hour)}); err != nil {
		t.Fatal(err)
	}
	s, err := app.SummarizeFixes(ctx, f.App, time.Time{})
	if err != nil {
		t.Fatal(err)
	}
	var d *app.FixDay
	for i := range s.Days {
		if s.Days[i].Day == "2026-11-17" {
			d = &s.Days[i]
		}
	}
	if d == nil || d.Forgets != 3 || d.Undone != 1 {
		t.Fatalf("summary %+v", s)
	}
}
