package mcpserver

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/veridit/hug/muninn/internal/scopetest"
	"github.com/veridit/hug/muninn/internal/testutil"
)

func TestMCPRememberDuplicateQuestionAndAnswers(t *testing.T) {
	f := scopetest.New(t)
	e := testutil.DuplicateEmbedder{Cosine: .9}
	f.App.Indexer.Embedder = e
	f.App.Searcher.Embedder = e
	if _, err := f.App.Reindex(context.Background()); err != nil {
		t.Fatal(err)
	}
	cs := connect(t, f.App, f.Bind(t, "alpha"))
	args := map[string]any{"content": "A synthetic-near durable quartz policy remains useful."}
	out, bad := call(t, cs, "memory_remember", args)
	if !bad || !strings.Contains(out, "Same fact?") {
		t.Fatal("missing duplicate question", out)
	}
	args["new_reason"] = "Synthetic distinction from the existing policy."
	out, bad = call(t, cs, "memory_remember", args)
	if bad {
		t.Fatal(out)
	}
	var reply idOut
	if err := json.Unmarshal([]byte(out), &reply); err != nil || reply.Reinforced {
		t.Fatal(out, err)
	}
	delete(args, "new_reason")
	args["reinforce"] = f.Units["Alpha"].SID
	out, bad = call(t, cs, "memory_remember", args)
	if bad {
		t.Fatal(out)
	}
	if err := json.Unmarshal([]byte(out), &reply); err != nil || !reply.Reinforced || reply.ID != f.Units["Alpha"].SID {
		t.Fatal(out, err)
	}
}
