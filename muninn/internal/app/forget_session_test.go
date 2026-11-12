package app_test

import (
	"context"
	"testing"

	"github.com/veridit/hug/muninn/internal/recall"
	"github.com/veridit/hug/muninn/internal/scopetest"
)

// 2026-11-11: a memory recalled into a session could not be forgotten from
// that session when the command ran in another folder. Forget now honours
// the session's own recall log, and still refuses everything else.
func TestForgetHonoursMemoriesRecalledIntoTheSession(t *testing.T) {
	f := scopetest.New(t)
	ctx := context.Background()
	other := f.Units["OtherProject"]
	here := f.App.Bound("", f.Cwd) // project:matrix-project: OtherProject is not visible
	if _, err := here.Forget(ctx, f.App, other.SID); err == nil {
		t.Fatal("forget of an invisible memory with no session must fail")
	}
	// Record that recall showed OtherProject to session s1.
	if _, err := f.App.DB.Exec(`INSERT INTO injections(session, unit_id, content_hash, harness, at_ns) VALUES('jcode:s1', ?, 'h', 'jcode', 1)`, other.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := here.Forget(ctx, f.App, other.SID, "jcode:s2"); err == nil {
		t.Fatal("another session must not be able to forget it")
	}
	if _, err := here.Forget(ctx, f.App, other.SID, "jcode:s1"); err != nil {
		t.Fatalf("session that was shown the memory must forget it: %v", err)
	}
	all, err := f.App.Recall.Recall(ctx, recall.Request{Message: "Matrix(OtherProject) quartz deployment policy"})
	if err != nil {
		t.Fatal(err)
	}
	for _, h := range all.Hits {
		if h.SID == other.SID {
			t.Fatal("forgotten memory still recalled")
		}
	}
}
