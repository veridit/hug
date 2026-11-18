package app_test

import (
	"context"
	"encoding/json"
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/veridit/hug/muninn/internal/app"
	"github.com/veridit/hug/muninn/internal/extract"
	"github.com/veridit/hug/muninn/internal/ingest"
	"github.com/veridit/hug/muninn/internal/project"
	"github.com/veridit/hug/muninn/internal/scopetest"
)

// wideReply is a recorded extraction (rnpdnm A+): the extractor placed one
// memory personal, one other (about Statbus) and one here.
func wideReply() string {
	var lines []string
	for _, c := range []extract.Candidate{
		{Content: "Synthetic: the person keeps the release laptop on battery saver", Category: "fact", Trust: "high", Place: "personal", PlaceReason: "about the person's machine"},
		{Content: "Synthetic: Statbus deploys run only from its release branch", Category: "fact", Trust: "high", Place: "other", PlaceName: "Statbus", PlaceReason: "another project"},
		{Content: "Synthetic: this repository's tests run niced with bounded parallelism", Category: "fact", Trust: "high", Place: "here", PlaceReason: "this project"},
	} {
		b, _ := json.Marshal(c)
		lines = append(lines, string(b))
	}
	return strings.Join(lines, "\n")
}

func storedScope(t *testing.T, content string) ingest.NativeMemory {
	t.Helper()
	mems, err := ingest.LoadNative(app.NativePath())
	if err != nil {
		t.Fatal(err)
	}
	for _, m := range mems {
		if m.Content == content {
			return m
		}
	}
	t.Fatalf("%q not stored", content)
	return ingest.NativeMemory{}
}

// rnpdnm check 2: with wide places on, the extractor's personal and other
// land in the personal scope (other tagged about:NAME, crab: never written
// into another project's scope), here stays the project's, and a project
// sync never writes the personal ones to memories.jsonl.
func TestCaptureWidePlacesPersonalNeverSynced(t *testing.T) {
	f := scopetest.New(t)
	b := f.Bind(t, "global")
	m := fakeCapture(t, f, wideReply())
	f.App.Cfg.Extract.WidePlaces = true
	r, err := b.Capture(context.Background(), f.App, captureOpts("wide-places"))
	if err != nil || len(r.Actions) != 3 {
		t.Fatalf("%+v %v", r, err)
	}
	if !strings.Contains(m.system, "\npersonal: ") || !strings.Contains(m.system, "Choose personal for") {
		t.Fatalf("wide places not offered: %s", m.system)
	}
	if strings.Contains(m.system, "Statbus") || strings.Contains(m.system, "project:matrix-project") {
		t.Fatal("a scope or another place's name reached the model")
	}
	personal := storedScope(t, "Synthetic: the person keeps the release laptop on battery saver")
	other := storedScope(t, "Synthetic: Statbus deploys run only from its release branch")
	here := storedScope(t, "Synthetic: this repository's tests run niced with bounded parallelism")
	if personal.Scope != app.PersonalScope || other.Scope != app.PersonalScope || here.Scope != "project:matrix-project" {
		t.Fatalf("scopes: personal=%s other=%s here=%s", personal.Scope, other.Scope, here.Scope)
	}
	if !slices.Equal(other.Tags, []string{"about:Statbus"}) || len(personal.Tags) != 0 {
		t.Fatalf("tags: other=%v personal=%v", other.Tags, personal.Tags)
	}
	p, err := project.Find(f.Cwd, false)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := project.Sync(p, app.NativePath(), false); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(p.File())
	if err != nil {
		t.Fatal(err)
	}
	file := string(raw)
	if !strings.Contains(file, "tests run niced") {
		t.Fatalf("here memory not synced: %s", file)
	}
	if strings.Contains(file, "battery saver") || strings.Contains(file, "Statbus") {
		t.Fatalf("a personal capture reached memories.jsonl: %s", file)
	}
}

// Off (the default), the same recorded extraction places as before:
// personal and other are unknown names and map to here. A client session
// never gets the wide places: personal is out of its reach.
func TestCaptureWidePlacesOffAndClient(t *testing.T) {
	f := scopetest.New(t)
	b := f.Bind(t, "global")
	m := fakeCapture(t, f, wideReply())
	if _, err := b.Capture(context.Background(), f.App, captureOpts("wide-off")); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(m.system, "\npersonal: ") || strings.Contains(m.system, "Choose personal for") {
		t.Fatal("off: wide places offered")
	}
	if s := storedScope(t, "Synthetic: the person keeps the release laptop on battery saver").Scope; s != "project:matrix-project" {
		t.Fatalf("off: personal mapped to %s, want here", s)
	}

	g := scopetest.New(t)
	c := g.Bind(t, "client")
	m = fakeCapture(t, g, wideReply())
	g.App.Cfg.Extract.WidePlaces = true
	if _, err := c.Capture(context.Background(), g.App, captureOpts("wide-client")); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(m.system, "\npersonal: ") || strings.Contains(m.system, "Choose personal for") {
		t.Fatal("client: wide places offered")
	}
	s := storedScope(t, "Synthetic: the person keeps the release laptop on battery saver").Scope
	if s == app.PersonalScope || !strings.HasPrefix(s, "client:acme") {
		t.Fatalf("client: personal mapped to %s", s)
	}
}
