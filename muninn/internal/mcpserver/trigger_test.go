package mcpserver

import (
	"context"
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

// jrj1vq check 2 (1): memory_remember asks the agent for when/when_not on a
// rule or pattern, and every field its words name is in its input schema.
func TestMCPRememberAsksForTrigger(t *testing.T) {
	f := scopetest.New(t)
	cs := connect(t, f.App, f.Bind(t, "alpha"))
	res, err := cs.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, tool := range res.Tools {
		if tool.Name != "memory_remember" {
			continue
		}
		d := tool.Description
		for _, want := range []string{"rule or pattern", "when (", "when_not (", "case (", "supersedes"} {
			if !strings.Contains(d, want) {
				t.Errorf("description lacks %q: %s", want, d)
			}
		}
		raw, _ := json.Marshal(tool.InputSchema)
		var schema struct {
			Properties map[string]struct {
				Description string `json:"description"`
			} `json:"properties"`
		}
		json.Unmarshal(raw, &schema)
		for _, field := range []string{"when", "when_not", "case", "clear_trigger", "supersedes", "content"} {
			if _, ok := schema.Properties[field]; !ok {
				t.Errorf("the words name %q, the schema has no such field", field)
			}
		}
		if !strings.Contains(schema.Properties["when"].Description, "rule or pattern") || !strings.Contains(schema.Properties["content"].Description, "THEN") {
			t.Errorf("field words: when=%q content=%q", schema.Properties["when"].Description, schema.Properties["content"].Description)
		}
		return
	}
	t.Fatal("memory_remember not listed")
}
