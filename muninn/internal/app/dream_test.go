package app_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/veridit/hug/muninn/internal/app"
	"github.com/veridit/hug/muninn/internal/recall"
	"github.com/veridit/hug/muninn/internal/scopetest"
)

func TestDreamQueuesHandsOutScopedAndAutoResolves(t *testing.T) {
	f := scopetest.New(t)
	ctx := context.Background()
	here := f.App.Bound("", f.Cwd)
	client := f.App.Bound("", f.ClientCwd)
	// Two judgement items: one in this project, one sealed in client acme.
	proj, other := f.Units["Project"], f.Units["Shared"]
	acme, acme2 := f.Units["Client"], f.Units["ClientProject"]
	ins := `INSERT INTO dream_items(kind,key,sids,scope,instruction,created_ns) VALUES('dupes',?,?,?,'decide',1)`
	if _, err := f.App.DB.Exec(ins, "k1", proj.SID+" "+other.SID, "project:matrix-project"); err != nil {
		t.Fatal(err)
	}
	if _, err := f.App.DB.Exec(ins, "k2", acme.SID+" "+acme2.SID, "client:acme"); err != nil {
		t.Fatal(err)
	}
	if n := here.OpenDreamCount(f.App); n != 1 {
		t.Fatalf("project session sees %d items, want 1 (client item sealed)", n)
	}
	items, err := here.DreamItems(f.App, "jcode:s1", 5)
	if err != nil || len(items) != 1 || !strings.Contains(strings.Join(items[0].Memories, "\n"), proj.SID) {
		t.Fatalf("hand-out: %+v %v", items, err)
	}
	for _, m := range items[0].Memories {
		if strings.Contains(m, acme.SID) {
			t.Fatal("client memory leaked into a non-client session")
		}
	}
	if got, _ := client.DreamItems(f.App, "jcode:c1", 5); len(got) == 0 {
		t.Fatal("client session must get its client item")
	}
	// The agent forgets one copy by hand: the item closes itself.
	if _, err := here.Forget(ctx, f.App, other.SID); err != nil {
		t.Fatal(err)
	}
	if got, _ := here.DreamItems(f.App, "jcode:s1", 5); len(got) != 0 {
		t.Fatalf("item with one memory left must auto-resolve: %+v", got)
	}
	if err := client.ResolveDream(f.App, items[0].ID, "x"); err == nil {
		t.Fatal("resolving an already-resolved item must fail")
	}
	r, err := f.App.Dream(ctx, true)
	if err != nil || !r.DryRun {
		t.Fatalf("dry dream: %+v %v", r, err)
	}
}

func TestDreamCueOnlyAtBreaks(t *testing.T) {
	f := scopetest.New(t)
	ctx := context.Background()
	b := f.App.Bound("", f.Cwd)
	open := func() int { return 2 }
	ask := func(session, msg string) recall.Response {
		r, err := f.App.Recall.Recall(ctx, recall.Request{Session: session, Harness: "jcode", Message: msg, Scopes: b.Scopes, ScopeFor: b.ScopeFor, DreamOpen: open})
		if err != nil {
			t.Fatal(err)
		}
		return r
	}
	first := ask("jcode:b1", "let us look at the deployment pipeline today")
	if first.Break != "start" || !strings.Contains(first.Block, "memory_dream") {
		t.Fatalf("first turn is a break with a cue: %+v", first)
	}
	mid := ask("jcode:b1", "now refactor the parser module")
	if mid.Break != "" || strings.Contains(mid.Block, "memory_dream") {
		t.Fatalf("mid-task turn must not cue: %+v", mid)
	}
	// Back after a long break: a break, but cued only once per CueEvery.
	if _, err := f.App.DB.Exec(`UPDATE session_turns SET last_ns=? WHERE session='jcode:b1'`, time.Now().Add(-2*recall.BreakGap).UnixNano()); err != nil {
		t.Fatal(err)
	}
	back := ask("jcode:b1", "ok where were we with the parser")
	if back.Break != "return" || strings.Contains(back.Block, "memory_dream") {
		t.Fatalf("return is a break but the session was cued recently: %+v", back)
	}
	wrap := ask("jcode:b2", "fix the lint warnings please")
	wrap = ask("jcode:b2", "thanks, that's all for now")
	if wrap.Break != "wrap-up" {
		t.Fatalf("wrap-up not detected: %+v", wrap)
	}
	none := func() int { return 0 }
	r, _ := f.App.Recall.Recall(ctx, recall.Request{Session: "jcode:b3", Harness: "jcode", Message: "start of something", DreamOpen: none})
	if strings.Contains(r.Block, "memory_dream") {
		t.Fatal("no open items, no cue")
	}
	var _ app.DreamItem
}

// 2026-11-12 (user): an unrecalled memory is vetted, never archived. Rare
// knowledge is often the most critical.
func TestDreamVetsUnrecalledNeverArchives(t *testing.T) {
	f := scopetest.New(t)
	ctx := context.Background()
	old := time.Now().Add(-90 * 24 * time.Hour).UnixNano()
	if _, err := f.App.DB.Exec(`UPDATE units SET updated_ns=?`, old); err != nil {
		t.Fatal(err)
	}
	recalled := f.Units["Shared"]
	// The fixture's own setup recalls everything; start from a clean log.
	if _, err := f.App.DB.Exec(`DELETE FROM injections`); err != nil {
		t.Fatal(err)
	}
	// Recall logging began 90 days ago; Shared was recalled, the rest never.
	if _, err := f.App.DB.Exec(`INSERT INTO injections(session,unit_id,content_hash,harness,at_ns) VALUES('jcode:x',?,'h','jcode',?)`, recalled.ID, old); err != nil {
		t.Fatal(err)
	}
	before := 0
	f.App.DB.QueryRow(`SELECT count(*) FROM units WHERE active=1`).Scan(&before)
	r, err := f.App.Dream(ctx, false)
	if err != nil || r.Queued == 0 {
		t.Fatalf("vetting items expected: %+v %v", r, err)
	}
	after := 0
	f.App.DB.QueryRow(`SELECT count(*) FROM units WHERE active=1`).Scan(&after)
	if after != before {
		t.Fatalf("dream must never archive unrecalled memories: %d -> %d", before, after)
	}
	var vetRecalled int
	f.App.DB.QueryRow(`SELECT count(*) FROM dream_items WHERE kind='vet' AND sids=?`, recalled.SID).Scan(&vetRecalled)
	if vetRecalled != 0 {
		t.Fatal("a recalled memory must not be vetted")
	}
	here := f.App.Bound("", f.Cwd)
	items, _ := here.DreamItems(f.App, "jcode:v", 5)
	var vet app.DreamItem
	for _, it := range items {
		if it.Kind == "vet" {
			vet = it
		}
	}
	if vet.ID == 0 || len(vet.SIDs) != 1 || !strings.Contains(vet.Instruction, "rare knowledge") {
		t.Fatalf("vet item not handed out with its instruction: %+v", items)
	}
	// Later passes drain the backlog VetPerPass at a time, never twice per memory.
	for i := 0; i < 3; i++ {
		if _, err := f.App.Dream(ctx, false); err != nil {
			t.Fatal(err)
		}
	}
	var dupes int
	f.App.DB.QueryRow(`SELECT count(*) FROM (SELECT sids FROM dream_items WHERE kind='vet' GROUP BY sids HAVING count(*)>1)`).Scan(&dupes)
	if dupes != 0 {
		t.Fatalf("%d memories got more than one vet item", dupes)
	}
}
