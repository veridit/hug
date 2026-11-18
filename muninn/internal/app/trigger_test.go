package app_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/veridit/hug/muninn/internal/app"
	"github.com/veridit/hug/muninn/internal/harness"
	"github.com/veridit/hug/muninn/internal/ingest"
	"github.com/veridit/hug/muninn/internal/scopetest"
	"github.com/veridit/hug/muninn/internal/trigger"
)

func nativeTrigger(t *testing.T, sid string) *trigger.Trigger {
	t.Helper()
	mems, err := ingest.LoadNative(app.NativePath())
	if err != nil {
		t.Fatal(err)
	}
	for _, m := range mems {
		if m.SID == sid {
			return m.Trigger
		}
	}
	t.Fatalf("%s not in the native store", sid)
	return nil
}

// jrj1vq check 1: remember stores a trigger in the native file and the
// index; reinforce keeps it and refuses a different one (crab ruling b);
// supersede with the same text replaces it in place, and clearing it is a
// supersede that stores None; forget keeps it on the retired memory.
func TestTriggerRememberReinforceSupersedeForget(t *testing.T) {
	f := scopetest.New(t)
	ctx := context.Background()
	b := f.App.Bound("", f.Cwd)
	text := "Zebrafish tickets keep their history in notes, not the description"
	tr := &trigger.Trigger{When: "reviewing a zebrafish ticket's history", WhenNot: "writing a new ticket", Case: &trigger.Case{Tools: []string{"hug task"}}}
	u, re, err := b.RememberPlaced(ctx, f.App, text, "fact", "", nil, "", "", "", app.RememberDecision{Type: "rule", Trigger: tr})
	if err != nil || re {
		t.Fatal(err, re)
	}
	if got := nativeTrigger(t, u.SID); !trigger.Equal(got, tr) {
		t.Fatalf("native trigger %+v", got)
	}
	if idx, _ := f.App.DB.Resolve(u.SID); idx.Trigger != trigger.Encode(tr) {
		t.Fatalf("index trigger %q", idx.Trigger)
	}
	// The same memory again without a trigger: a reinforce that keeps it.
	if got, re, err := b.RememberPlaced(ctx, f.App, text, "fact", "", nil, "", "", "", app.RememberDecision{Reinforce: u.SID}); err != nil || !re || got.SID != u.SID {
		t.Fatal(got, re, err)
	}
	if !trigger.Equal(nativeTrigger(t, u.SID), tr) {
		t.Fatal("reinforce lost the trigger")
	}
	// Reinforcing it with another trigger is refused, with the supersede fix.
	other := &trigger.Trigger{When: "auditing zebrafish ticket history"}
	for _, d := range []app.RememberDecision{{Reinforce: u.SID, Trigger: other}, {Trigger: other}} {
		_, _, err := b.RememberPlaced(ctx, f.App, text, "fact", "", nil, "", "", "", d)
		var tc *ingest.TriggerChangeError
		if !errors.As(err, &tc) || !strings.HasPrefix(err.Error(), "NOT STORED:") || !strings.Contains(err.Error(), "fix: hug mem remember --supersedes "+u.SID) || !app.Refusal(err) {
			t.Fatalf("changed trigger %+v: %v", d, err)
		}
	}
	if !trigger.Equal(nativeTrigger(t, u.SID), tr) {
		t.Fatal("a refused reinforce changed the trigger")
	}
	// Supersede with the same text: the trigger is replaced in place, same sid,
	// one supersede row naming itself.
	got, re, err := b.RememberPlaced(ctx, f.App, text, "fact", "", nil, "", "", "", app.RememberDecision{Supersedes: u.SID, Trigger: other})
	if err != nil || re || got.SID != u.SID {
		t.Fatal(got, re, err)
	}
	if !trigger.Equal(nativeTrigger(t, u.SID), other) {
		t.Fatalf("supersede did not replace: %+v", nativeTrigger(t, u.SID))
	}
	if rows := fixRows(t, f, u.SID); len(rows) == 0 || rows[len(rows)-1].Action != "supersede" || rows[len(rows)-1].NewSID != u.SID {
		t.Fatalf("trail %+v", rows)
	}
	// Clearing it: a supersede with ClearTrigger stores None, the index none.
	if _, _, err := b.RememberPlaced(ctx, f.App, text, "fact", "", nil, "", "", "", app.RememberDecision{Supersedes: u.SID, ClearTrigger: true}); err != nil {
		t.Fatal(err)
	}
	if got := nativeTrigger(t, u.SID); !trigger.Cleared(got) {
		t.Fatalf("clear stored %+v", got)
	}
	if idx, _ := f.App.DB.Resolve(u.SID); idx.Trigger != "" {
		t.Fatalf("index trigger after clear %q", idx.Trigger)
	}
	// Clearing without saying which memory: refused with a fix.
	if _, _, err := b.RememberPlaced(ctx, f.App, text, "fact", "", nil, "", "", "", app.RememberDecision{ClearTrigger: true}); err == nil || !strings.Contains(err.Error(), "fix: ") {
		t.Fatalf("clear without supersedes: %v", err)
	}
	// After a clear a new trigger may be stated again by supersede.
	if _, _, err := b.RememberPlaced(ctx, f.App, text, "fact", "", nil, "", "", "", app.RememberDecision{Supersedes: u.SID, Trigger: tr}); err != nil || !trigger.Equal(nativeTrigger(t, u.SID), tr) {
		t.Fatal(err)
	}
	// A supersede with new text and a trigger is a new memory carrying it.
	n, _, err := b.RememberPlaced(ctx, f.App, "Zebrafish ticket history moved to the wiki in November", "fact", "", nil, "", "", "", app.RememberDecision{Supersedes: u.SID, Trigger: other, NewReason: "moved"})
	if err != nil || n.SID == u.SID || !trigger.Equal(nativeTrigger(t, n.SID), other) {
		t.Fatal(n, err)
	}
	// Forget keeps the trigger on the retired memory.
	if _, err := b.Forget(ctx, f.App, u.SID); err != nil {
		t.Fatal(err)
	}
	if !trigger.Equal(nativeTrigger(t, u.SID), tr) {
		t.Fatal("forget dropped the trigger")
	}
	// A malformed trigger fails with a fix line and stores nothing.
	if _, _, err := b.RememberPlaced(ctx, f.App, "Zebrafish other note", "fact", "", nil, "", "", "", app.RememberDecision{Trigger: &trigger.Trigger{When: strings.Repeat("w", trigger.MaxText+1)}}); err == nil || !strings.Contains(err.Error(), "\nfix: hug mem remember --when") {
		t.Fatalf("long when: %v", err)
	}
}

// jrj1vq check 1: a plain memory stays plain everywhere: no trigger key in
// the native file and an empty index column.
func TestPlainMemoryHasNoTrigger(t *testing.T) {
	f := scopetest.New(t)
	b := f.App.Bound("", f.Cwd)
	u, _, err := b.RememberPlaced(context.Background(), f.App, "Zebrafish plain note stays plain", "fact", "", nil, "", "", "")
	if err != nil {
		t.Fatal(err)
	}
	if got := nativeTrigger(t, u.SID); got != nil {
		t.Fatalf("plain memory got %+v", got)
	}
	if idx, _ := f.App.DB.Resolve(u.SID); idx.Trigger != "" {
		t.Fatalf("index %q", idx.Trigger)
	}
}

// twins stores two native memories with the same text in one scope, with
// the given triggers, and reindexes.
func twins(t *testing.T, f *scopetest.Fixture, a, b *trigger.Trigger) {
	t.Helper()
	ms, err := ingest.LoadNative(app.NativePath())
	if err != nil {
		t.Fatal(err)
	}
	at := time.Unix(1_700_000_000, 0)
	text := "Zebrafish twin note about the staging gateway"
	ms = append(ms,
		ingest.NativeMemory{ID: "twin_a", Content: text, Category: "fact", Scope: "global", Active: true, Strength: 1, Created: at, Updated: at, Trigger: a},
		ingest.NativeMemory{ID: "twin_b", Content: text, Category: "fact", Scope: "global", Active: true, Strength: 1, Created: at.Add(time.Hour), Updated: at.Add(time.Hour), Trigger: b})
	if err := ingest.SaveNative(app.NativePath(), ms); err != nil {
		t.Fatal(err)
	}
	if _, err := f.App.Reindex(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func byID(t *testing.T, id string) ingest.NativeMemory {
	t.Helper()
	ms, _ := ingest.LoadNative(app.NativePath())
	for _, m := range ms {
		if m.ID == id {
			return m
		}
	}
	t.Fatalf("%s missing", id)
	return ingest.NativeMemory{}
}

// jrj1vq check 1: dream never merges away a trigger. Twins whose triggers
// differ go to an agent's judgment; twins with the same trigger merge and
// the keeper keeps it.
func TestDreamKeepsTriggers(t *testing.T) {
	ctx := context.Background()
	w := &trigger.Trigger{When: "deploying to staging"}
	t.Run("differ", func(t *testing.T) {
		f := scopetest.New(t)
		twins(t, f, w, &trigger.Trigger{When: "debugging the gateway"})
		r, err := f.App.Dream(ctx, false)
		if err != nil {
			t.Fatal(err)
		}
		if !byID(t, "twin_a").Active || !byID(t, "twin_b").Active || r.Retired != 0 {
			t.Fatalf("merged twins with different triggers: %+v", r)
		}
		var n int
		f.App.DB.QueryRow(`SELECT count(*) FROM dream_items WHERE kind='dupes'`).Scan(&n)
		if n != 1 {
			t.Fatalf("dupes items %d", n)
		}
	})
	t.Run("same", func(t *testing.T) {
		f := scopetest.New(t)
		twins(t, f, w, &trigger.Trigger{When: "deploying   to staging"})
		r, err := f.App.Dream(ctx, false)
		if err != nil || r.Retired != 1 {
			t.Fatalf("%+v %v", r, err)
		}
		a, b := byID(t, "twin_a"), byID(t, "twin_b")
		keeper := b
		if !b.Active {
			keeper = a
		}
		if !keeper.Active || !trigger.Equal(keeper.Trigger, w) {
			t.Fatalf("keeper lost its trigger: a=%+v b=%+v", a, b)
		}
	})
}

// jrj1vq check 1: an owner correction replaces the note and keeps the
// situation it is for.
func TestCorrectionKeepsTrigger(t *testing.T) {
	f := scopetest.New(t)
	ctx := context.Background()
	b := f.App.Bound("", f.Cwd)
	tr := &trigger.Trigger{When: "deploying to staging"}
	u, _, err := b.RememberPlaced(ctx, f.App, "Zebrafish staging deploys need the canary flag", "fact", "", nil, "", "", "", app.RememberDecision{Trigger: tr})
	if err != nil {
		t.Fatal(err)
	}
	next, _, err := b.Correct(ctx, f.App, u.SID, "Zebrafish staging deploys need the canary flag and a ticket")
	if err != nil {
		t.Fatal(err)
	}
	if got := nativeTrigger(t, next.SID); !trigger.Equal(got, tr) {
		t.Fatalf("correction trigger %+v", got)
	}
}

// jrj1vq check 2: with capture triggers on, a captured rule keeps the
// trigger the extractor proposed; with them off the request carries no
// trigger instructions and nothing is stored.
func TestCaptureStoresProposedTrigger(t *testing.T) {
	for _, on := range []bool{false, true} {
		f := scopetest.New(t)
		b := f.Bind(t, "alpha")
		reply := `{"category":"decision","type":"rule","content":"Zebrafish coding agents run at medium effort or higher always","trust":"high","source":"user","when":"choosing a model for a coding subagent","case":{"tools":["swarm"]}}`
		m := fakeCaptureSystem(t, f, reply)
		f.App.Cfg.Extract.Triggers = on
		r, err := b.Capture(context.Background(), f.App, app.CaptureOptions{Harness: "stdin", Session: "trig", Messages: []harness.CaptureMessage{{ID: "one", Role: "user", Text: "Always run coding agents at medium effort or higher."}}})
		if err != nil || len(r.Actions) != 1 || r.Actions[0].Action != "add" {
			t.Fatalf("on=%v %+v %v", on, r, err)
		}
		if strings.Contains(m.system, `"when_not"`) != on {
			t.Fatalf("on=%v: trigger instructions in the request = %v", on, !on)
		}
		ms, _ := ingest.LoadNative(app.NativePath())
		var got *trigger.Trigger
		for _, x := range ms {
			if strings.HasPrefix(x.Content, "Zebrafish coding agents") {
				got = x.Trigger
			}
		}
		// Off: the extractor was not asked; a stray trigger in its reply is
		// still a proposal nobody asked for, and is not stored.
		if on && (got == nil || got.When != "choosing a model for a coding subagent" || got.Case.Tools[0] != "swarm") {
			t.Fatalf("on: stored %+v", got)
		}
		if !on && got != nil {
			t.Fatalf("off: stored %+v", got)
		}
	}
}
