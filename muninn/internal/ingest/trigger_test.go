package ingest

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/veridit/hug/muninn/internal/trigger"
)

// jrj1vq check 1: the native file stores a trigger as one nested object; a
// plain memory's JSON has no trigger key, as before; reinforcing never
// silently swaps a trigger.
func TestNativeTriggerFormat(t *testing.T) {
	path := filepath.Join(t.TempDir(), "n.json")
	tr := &trigger.Trigger{When: "  choosing a model for a coding subagent ", Case: &trigger.Case{Tools: []string{"swarm"}}}
	m, re, err := RememberWith(path, Remembering{Content: "Coding agents run at medium effort or higher", Category: "fact", Type: "rule", Trigger: tr})
	if err != nil || re {
		t.Fatal(err, re)
	}
	if m.Trigger == nil || m.Trigger.When != "choosing a model for a coding subagent" {
		t.Fatalf("stored trigger not canonical: %+v", m.Trigger)
	}
	plain, _, err := RememberWith(path, Remembering{Content: "Plain note", Category: "fact"})
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := os.ReadFile(path)
	var f struct{ Memories []map[string]json.RawMessage }
	if err := json.Unmarshal(raw, &f); err != nil {
		t.Fatal(err)
	}
	for _, x := range f.Memories {
		var id string
		json.Unmarshal(x["id"], &id)
		_, has := x["trigger"]
		if id == plain.ID && has {
			t.Fatalf("plain memory got a trigger key: %s", raw)
		}
		var compact bytes.Buffer
		json.Compact(&compact, x["trigger"])
		if id == m.ID && compact.String() != `{"case":{"tools":["swarm"]},"when":"choosing a model for a coding subagent"}` {
			t.Fatalf("trigger json %s", compact.String())
		}
	}
	// Same text, same trigger (any order or spacing): a reinforce.
	if got, re, err := RememberWith(path, Remembering{Content: m.Content, Category: "fact", Trigger: &trigger.Trigger{When: "choosing a model for a coding subagent", Case: &trigger.Case{Tools: []string{"swarm", " "}}}}); err != nil || !re || got.Strength != 2 {
		t.Fatalf("same trigger: %+v %v %v", got, re, err)
	}
	// Same text without a trigger: a reinforce that keeps the trigger.
	if got, re, err := RememberWith(path, Remembering{Content: m.Content, Category: "fact"}); err != nil || !re || got.Trigger == nil {
		t.Fatalf("no trigger: %+v %v %v", got, re, err)
	}
	// Same text, another trigger: refused, nothing written (crab ruling b).
	before, _ := os.ReadFile(path)
	_, _, err = RememberWith(path, Remembering{Content: m.Content, Category: "fact", Trigger: &trigger.Trigger{When: "reviewing a pull request"}})
	var tc *TriggerChangeError
	if !errors.As(err, &tc) || tc.ID != m.ID || !strings.Contains(err.Error(), "--supersedes") {
		t.Fatalf("changed trigger: %v", err)
	}
	if after, _ := os.ReadFile(path); string(after) != string(before) {
		t.Fatal("a refused reinforce wrote")
	}
	// A plain memory may gain a trigger by being remembered with one.
	if got, re, err := RememberWith(path, Remembering{Content: "Plain note", Category: "fact", Trigger: &trigger.Trigger{When: "w"}}); err != nil || !re || got.Trigger == nil || got.Trigger.When != "w" {
		t.Fatalf("plain gains a trigger: %+v %v %v", got, re, err)
	}
	// Malformed: refused with a fix line before anything is written.
	if _, _, err := RememberWith(path, Remembering{Content: "x", Trigger: &trigger.Trigger{When: strings.Repeat("y", trigger.MaxText+1)}}); err == nil || !strings.Contains(err.Error(), "fix: ") {
		t.Fatalf("long when: %v", err)
	}
}

// jrj1vq check 1: MergeImport (import, sync) carries a trigger, and an
// item without one never clears it.
func TestImportCarriesTrigger(t *testing.T) {
	path := filepath.Join(t.TempDir(), "n.json")
	tr := &trigger.Trigger{WhenNot: "editing the README"}
	if _, err := MergeImport(path, []Imported{{Origin: "repo:x/a", ID: "a", Content: "Note A", Scope: "project:x", Active: true, Trigger: tr}}, false); err != nil {
		t.Fatal(err)
	}
	mems, _ := LoadNative(path)
	if len(mems) != 1 || !trigger.Equal(mems[0].Trigger, tr) {
		t.Fatalf("import dropped the trigger: %+v", mems)
	}
	if st, err := MergeImport(path, []Imported{{Origin: "repo:x/a", ID: "a", Content: "Note A", Scope: "project:x", Active: true}}, false); err != nil || st.Unchanged != 1 {
		t.Fatalf("an item without a trigger is unchanged: %v %v", st, err)
	}
	mems, _ = LoadNative(path)
	if !trigger.Equal(mems[0].Trigger, tr) {
		t.Fatal("an item without a trigger cleared it")
	}
	other := &trigger.Trigger{WhenNot: "editing the CHANGELOG"}
	if st, err := MergeImport(path, []Imported{{Origin: "repo:x/a", ID: "a", Content: "Note A", Scope: "project:x", Active: true, Trigger: other}}, false); err != nil || st.Updated != 1 {
		t.Fatalf("a changed trigger is an update: %v %v", st, err)
	}
	mems, _ = LoadNative(path)
	if !trigger.Equal(mems[0].Trigger, other) {
		t.Fatalf("update kept the old trigger: %+v", mems[0].Trigger)
	}
}
