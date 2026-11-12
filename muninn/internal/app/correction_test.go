package app_test

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/veridit/hug/muninn/internal/app"
	"github.com/veridit/hug/muninn/internal/scopetest"
)

func TestOwnerCorrectionBindingAndRollback(t *testing.T) {
	f := scopetest.New(t)
	b := f.Bind(t, "alpha")
	before, err := os.ReadFile(app.NativePath())
	if err != nil {
		t.Fatal(err)
	}
	for _, ref := range f.HiddenRefs("alpha") {
		if _, _, err := b.Correct(context.Background(), f.App, ref, "A synthetic replacement policy."); err == nil {
			t.Fatal("hidden correction allowed")
		}
	}
	wide := b.View(true, nil)
	if _, _, err := wide.Correct(context.Background(), f.App, f.Units["Beta"].SID, "A synthetic replacement policy."); err == nil {
		t.Fatal("owner read override widened correction writes")
	}
	after, _ := os.ReadFile(app.NativePath())
	if !bytes.Equal(before, after) {
		t.Fatal("denied correction changed native file")
	}
	// Fail after the native append but before the suppression transaction. Both
	// original file bytes and the index's active historical row must recover.
	if _, err = f.App.DB.Exec(`CREATE TRIGGER fail_owner_correction BEFORE INSERT ON suppressed BEGIN SELECT RAISE(FAIL,'synthetic suppression failure'); END`); err != nil {
		t.Fatal(err)
	}
	_, backup, err := b.Correct(context.Background(), f.App, f.Units["Alpha"].SID, "A corrected synthetic quartz deployment policy.")
	if err == nil || !strings.Contains(err.Error(), "synthetic suppression failure") {
		t.Fatalf("missing failure: %v", err)
	}
	after, _ = os.ReadFile(app.NativePath())
	if !bytes.Equal(before, after) {
		t.Fatal("failed correction did not restore exact file bytes")
	}
	got, err := f.App.DB.Resolve(f.Units["Alpha"].SID)
	if err != nil || !got.Active {
		t.Fatal("failed correction retired original in index")
	}
	if _, err = os.Stat(filepath.Join(backup, "files.json")); err != nil {
		t.Fatal("recovery backup missing")
	}
}
func TestOwnerCorrectionDoesNotEditUpstream(t *testing.T) {
	f := scopetest.New(t)
	b := f.Bind(t, "global")
	u := f.Units["OutsideNote"]
	origin, err := f.App.Origin(u)
	if err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(origin.Path)
	if err != nil {
		t.Fatal(err)
	}
	next, backup, err := b.Correct(context.Background(), f.App, u.SID, "A corrected synthetic quartz deployment policy from an owner.")
	if err != nil {
		t.Fatal(err)
	}
	after, _ := os.ReadFile(origin.Path)
	if !bytes.Equal(before, after) {
		t.Fatal("correction edited upstream")
	}
	if next.SID == u.SID || next.Category != "correction" || next.Scope != u.Scope {
		t.Fatalf("bad correction: %+v", next)
	}
	if !f.App.DB.IsSuppressed(u.SID) {
		t.Fatal("upstream original not locally suppressed")
	}
	if backup == "" {
		t.Fatal("no backup")
	}
}
