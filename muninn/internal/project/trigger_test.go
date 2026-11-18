package project

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/veridit/hug/muninn/internal/ingest"
	"github.com/veridit/hug/muninn/internal/trigger"
)

// jrj1vq check 1: a trigger travels through the hug/memory file to another
// clone; a plain memory's line is byte-identical to the old format; a line
// written by a hug that does not know triggers never clears one.
func TestSyncCarriesTrigger(t *testing.T) {
	repo := t.TempDir()
	p := Project{ID: "trig-demo", Label: "demo", Root: repo}
	alice := filepath.Join(t.TempDir(), "alice.json")
	bob := filepath.Join(t.TempDir(), "bob.json")
	tr := &trigger.Trigger{When: "reviewing a ticket's history", WhenNot: "writing the ticket text", Case: &trigger.Case{Paths: []string{"docs/x.md"}, Tools: []string{"git"}}}
	if _, _, err := ingest.RememberWith(alice, ingest.Remembering{Content: "Ticket history lives in the notes, not the body", Category: "fact", Scope: p.Scope(), Trigger: tr}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := ingest.RememberWith(alice, ingest.Remembering{Content: "Run make check before pushing", Category: "fact", Scope: p.Scope()}); err != nil {
		t.Fatal(err)
	}
	if _, err := Sync(p, alice, false); err != nil {
		t.Fatal(err)
	}
	raw, _ := os.ReadFile(p.File())
	lines := strings.Split(strings.TrimSpace(string(raw)), "\n")
	var withT, plain string
	for _, l := range lines {
		if strings.Contains(l, "make check") {
			plain = l
		} else {
			withT = l
		}
	}
	if strings.Contains(plain, "trigger") {
		t.Fatalf("a plain memory's line changed: %s", plain)
	}
	if !strings.Contains(withT, `"trigger":{"case":{"paths":["docs/x.md"],"tools":["git"]},"when":"reviewing a ticket's history","when_not":"writing the ticket text"}`) {
		t.Fatalf("line without its trigger: %s", withT)
	}
	if _, err := Sync(p, bob, false); err != nil {
		t.Fatal(err)
	}
	if got := triggerOf(t, bob, "Ticket history"); !trigger.Equal(got, tr) {
		t.Fatalf("bob's copy has trigger %+v", got)
	}
	// An older hug rewrites the file without the key (it drops fields it
	// does not know): bob keeps his trigger and writes it back.
	old := strings.Replace(string(raw), withT, dropTrigger(t, withT), 1)
	if err := os.WriteFile(p.File(), []byte(old), 0o644); err != nil {
		t.Fatal(err)
	}
	st, err := Sync(p, bob, false)
	if err != nil {
		t.Fatal(err)
	}
	if got := triggerOf(t, bob, "Ticket history"); !trigger.Equal(got, tr) {
		t.Fatalf("an old line cleared the trigger: %+v", got)
	}
	if b, _ := os.ReadFile(p.File()); !st.FileChanged || !strings.Contains(string(b), `"when":"reviewing a ticket's history"`) {
		t.Fatalf("sync did not write the trigger back: %v %s", st, b)
	}
	// A changed trigger on the file reaches the other clone.
	b2, _ := os.ReadFile(p.File())
	changed := strings.Replace(string(b2), "reviewing a ticket's history", "auditing a ticket's history", 1)
	if err := os.WriteFile(p.File(), []byte(changed), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Sync(p, alice, false); err != nil {
		t.Fatal(err)
	}
	if got := triggerOf(t, alice, "Ticket history"); got == nil || got.When != "auditing a ticket's history" {
		t.Fatalf("alice did not take the changed trigger: %+v", got)
	}
}

// jrj1vq (crab): clearing a trigger is a local act that travels as "replace
// with none" ("trigger":{}), never as a missing key: clone A clears, clone
// B syncs, and the trigger stays cleared on both and on the file.
func TestSyncCarriesTriggerClear(t *testing.T) {
	p := Project{ID: "trig-clear", Label: "demo", Root: t.TempDir()}
	alice := filepath.Join(t.TempDir(), "alice.json")
	bob := filepath.Join(t.TempDir(), "bob.json")
	tr := &trigger.Trigger{When: "deploying the gateway"}
	m, _, err := ingest.RememberWith(alice, ingest.Remembering{Content: "Gateway deploys need the canary flag", Category: "fact", Scope: p.Scope(), Trigger: tr})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Sync(p, alice, false); err != nil {
		t.Fatal(err)
	}
	if _, err := Sync(p, bob, false); err != nil {
		t.Fatal(err)
	}
	if got := triggerOf(t, bob, "Gateway"); !trigger.Equal(got, tr) {
		t.Fatalf("bob's copy: %+v", got)
	}
	// Alice clears it on purpose.
	if _, found, err := ingest.ReplaceTrigger(alice, m.ID, nil); err != nil || !found {
		t.Fatal(found, err)
	}
	if _, err := Sync(p, alice, false); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(p.File()); !strings.Contains(string(b), `"trigger":{}`) {
		t.Fatalf("the clear is not on the file: %s", b)
	}
	// Bob syncs: his copy is cleared, and the file keeps the clear.
	if _, err := Sync(p, bob, false); err != nil {
		t.Fatal(err)
	}
	if got := triggerOf(t, bob, "Gateway"); !trigger.Cleared(got) {
		t.Fatalf("bob kept the cleared trigger: %+v", got)
	}
	if b, _ := os.ReadFile(p.File()); strings.Contains(string(b), "deploying the gateway") {
		t.Fatalf("bob wrote the cleared trigger back: %s", b)
	}
	// And alice's next sync keeps it cleared too.
	if _, err := Sync(p, alice, false); err != nil {
		t.Fatal(err)
	}
	if got := triggerOf(t, alice, "Gateway"); !trigger.Cleared(got) {
		t.Fatalf("alice got the trigger back: %+v", got)
	}
}

func triggerOf(t *testing.T, path, prefix string) *trigger.Trigger {
	t.Helper()
	mems, err := ingest.LoadNative(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, m := range mems {
		if strings.HasPrefix(m.Content, prefix) {
			return m.Trigger
		}
	}
	t.Fatalf("%q missing from %s", prefix, path)
	return nil
}

func dropTrigger(t *testing.T, line string) string {
	t.Helper()
	i := strings.Index(line, `,"trigger":`)
	if i < 0 {
		t.Fatalf("no trigger in %s", line)
	}
	depth, j := 0, i+len(`,"trigger":`)
	for ; j < len(line); j++ {
		switch line[j] {
		case '{':
			depth++
		case '}':
			depth--
		}
		if depth == 0 {
			break
		}
	}
	return line[:i] + line[j+1:]
}
