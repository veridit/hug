package mcpserver

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/veridit/hug/muninn/internal/app"
	"github.com/veridit/hug/muninn/internal/ingest"
	"github.com/veridit/hug/muninn/internal/scopetest"
	"github.com/veridit/hug/muninn/internal/trigger"
)

// jrj1vq check 1: memory_remember takes when, when_not and case; a changed
// trigger is refused as a NOT STORED error naming supersedes; supersedes
// with clear_trigger clears it.
func TestMCPRememberTrigger(t *testing.T) {
	f := scopetest.New(t)
	cs := connect(t, f.App, f.Bind(t, "alpha"))
	text := "Zebrafish model choice: coding agents run at medium effort or higher"
	args := map[string]any{"content": text, "type": "rule", "when": "choosing a model for a coding subagent", "when_not": "picking a model for a search scout",
		"case": map[string]any{"tools": []string{"swarm"}, "paths": []string{".jcode/config.toml"}}}
	out, bad := call(t, cs, "memory_remember", args)
	if bad {
		t.Fatal(out)
	}
	var reply idOut
	if err := json.Unmarshal([]byte(out), &reply); err != nil {
		t.Fatal(out, err)
	}
	want := &trigger.Trigger{When: "choosing a model for a coding subagent", WhenNot: "picking a model for a search scout", Case: &trigger.Case{Tools: []string{"swarm"}, Paths: []string{".jcode/config.toml"}}}
	if got := mcpTrigger(t, reply.ID); !trigger.Equal(got, want) {
		t.Fatalf("stored %+v", got)
	}
	args["when"] = "reviewing a pull request"
	out, bad = call(t, cs, "memory_remember", args)
	if !bad || !strings.HasPrefix(out, "NOT STORED:") || !strings.Contains(out, "--supersedes "+reply.ID) {
		t.Fatalf("changed trigger: %v %s", bad, out)
	}
	out, bad = call(t, cs, "memory_remember", map[string]any{"content": text, "supersedes": reply.ID, "clear_trigger": true})
	if bad {
		t.Fatal(out)
	}
	if got := mcpTrigger(t, reply.ID); !trigger.Cleared(got) {
		t.Fatalf("clear: %+v", got)
	}
}

func mcpTrigger(t *testing.T, sid string) *trigger.Trigger {
	t.Helper()
	ms, err := ingest.LoadNative(app.NativePath())
	if err != nil {
		t.Fatal(err)
	}
	for _, m := range ms {
		if m.SID == sid {
			return m.Trigger
		}
	}
	t.Fatalf("%s missing", sid)
	return nil
}
