package web

import (
	"context"
	"encoding/json"
	"strings"
	"sync"
	"testing"

	"github.com/veridit/muninn/internal/scopetest"
	"github.com/veridit/muninn/internal/testutil"
)

func TestWebRememberDuplicateQuestionAndAnswers(t *testing.T) {
	f := scopetest.New(t)
	e := testutil.DuplicateEmbedder{Cosine: .9}
	f.App.Indexer.Embedder = e
	f.App.Searcher.Embedder = e
	if _, err := f.App.Reindex(context.Background()); err != nil {
		t.Fatal(err)
	}
	h := (&Server{App: f.App, Mu: &sync.Mutex{}, Binding: f.Bind(t, "alpha")}).Handler()
	args := map[string]any{"content": "A synthetic-near durable quartz policy remains useful."}
	request := func() (string, int) {
		raw, err := json.Marshal(args)
		if err != nil {
			t.Fatal(err)
		}
		return matrixRequest(h, "POST", "/api/remember", string(raw))
	}
	out, status := request()
	if status != 400 || !strings.Contains(out, "Same fact?") {
		t.Fatal("missing duplicate question", status, out)
	}
	args["new_reason"] = "Synthetic distinction from the existing policy."
	out, status = request()
	if status != 200 {
		t.Fatal(status, out)
	}
	var reply struct {
		ID         string
		Reinforced bool
	}
	if err := json.Unmarshal([]byte(out), &reply); err != nil || reply.Reinforced {
		t.Fatal(out, err)
	}
	delete(args, "new_reason")
	args["reinforce"] = f.Units["Alpha"].SID
	out, status = request()
	if status != 200 {
		t.Fatal(status, out)
	}
	if err := json.Unmarshal([]byte(out), &reply); err != nil || !reply.Reinforced || reply.ID != f.Units["Alpha"].SID {
		t.Fatal(out, err)
	}
}
