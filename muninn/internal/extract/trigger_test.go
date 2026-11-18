package extract

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"testing"
)

// promptSHA is sha256 of the capture system prompt (without places) on
// origin/main e6853a7c, before jrj1vq. With capture triggers off the prompt
// must stay byte-identical (crab, jrj1vq check 2): the default capture is
// not changed by shipping the switch.
const promptSHA = "fecb156969d4ce194a08a9daa1e23665c9f81416b5ef0e8fd4c3b3f3076ed0fd"

func sha(s string) string { h := sha256.Sum256([]byte(s)); return hex.EncodeToString(h[:]) }

func TestCapturePromptOffIsByteIdentical(t *testing.T) {
	if Defaults().Triggers {
		t.Fatal("capture triggers must be off by default")
	}
	if got := sha(SystemPrompt(Defaults(), nil)); got != promptSHA {
		t.Fatalf("off prompt changed: sha256 %s, want %s", got, promptSHA)
	}
	places := []Place{{"here", "this project"}}
	if SystemPrompt(Defaults(), places) != prompt+placementPrompt(places) {
		t.Fatal("off prompt with places changed")
	}
	on := Defaults()
	on.Triggers = true
	p := SystemPrompt(on, places)
	if !strings.HasPrefix(p, prompt) || p == prompt+placementPrompt(places) || !strings.Contains(p, `"when"`) || !strings.Contains(p, `"when_not"`) || !strings.Contains(p, `"case"`) {
		t.Fatalf("on prompt: %s", p[len(prompt):])
	}
}

// With triggers the extractor may add when/when_not/case to a rule or
// preference; on another type they are dropped (the memory is kept);
// malformed ones drop only the trigger.
func TestParseTriggers(t *testing.T) {
	in := strings.Join([]string{
		`{"category":"fact","type":"rule","content":"Coding agents run at medium effort or higher always","trust":"high","source":"user","when":"choosing a model for a coding subagent","when_not":"picking a scout model","case":{"tools":["swarm"]}}`,
		`{"category":"fact","type":"fact","content":"The gateway listens on port 9443 in staging","trust":"medium","source":"assistant","when":"debugging the gateway"}`,
		`{"category":"preference","type":"preference","content":"Short answers please","trust":"high","source":"user","when":"` + strings.Repeat("w", 600) + `"}`,
		`{"category":"fact","type":"rule","content":"Never push to the hug/memory branch by hand","trust":"high","source":"user","case":{"tools":["git"],"colour":["x"]}}`,
	}, "\n")
	cs, rejected := Parse(in)
	if len(cs) != 4 || rejected != 0 {
		t.Fatalf("got %d candidates, %d rejected: %+v", len(cs), rejected, cs)
	}
	if tr := cs[0].Trigger(); tr == nil || tr.When != "choosing a model for a coding subagent" || tr.Case == nil || tr.Case.Tools[0] != "swarm" {
		t.Fatalf("rule trigger %+v", tr)
	}
	if cs[1].Trigger() != nil {
		t.Fatal("a fact keeps no trigger")
	}
	if cs[2].Trigger() != nil || cs[2].Content != "Short answers please" {
		t.Fatal("an over-long when drops only the trigger")
	}
	// An unknown key inside case never costs the memory.
	if tr := cs[3].Trigger(); tr == nil || tr.Case == nil || tr.Case.Tools[0] != "git" {
		t.Fatalf("lenient case %+v", cs[3])
	}
	// A string case (not an object) drops only the trigger.
	bad, rej := Parse(`{"category":"fact","type":"rule","content":"Never push to the hug/memory branch by hand","trust":"high","case":"git"}`)
	if len(bad) != 1 || rej != 0 || bad[0].Trigger() != nil {
		t.Fatalf("string case: %+v %d", bad, rej)
	}
}
