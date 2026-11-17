package run

import (
	"bytes"
	"context"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/veridit/hug/muninn/internal/app"
	"github.com/veridit/hug/muninn/internal/scopetest"
	"github.com/veridit/hug/muninn/internal/testutil"
)

// mainOutput runs Main and returns its exit code with stdout and stderr.
func mainOutput(t *testing.T, args ...string) (int, string) {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	so, se := os.Stdout, os.Stderr
	os.Stdout, os.Stderr = w, w
	done := make(chan string)
	go func() { var b bytes.Buffer; io.Copy(&b, r); done <- b.String() }()
	code := Main(args)
	os.Stdout, os.Stderr = so, se
	w.Close()
	return code, <-done
}

// A refusal never looks like success (p29rbz): each of remember's three
// questions starts "NOT STORED:" and exits with ExitNotStored, so a script
// stops and a reader does not take it for a stored memory. A Seid agent read
// "not stored yet: ... Same fact?" as success, forgot the original, and lost
// both.
func TestRememberRefusalSaysNotStoredAndFails(t *testing.T) {
	f := scopetest.New(t)
	e := testutil.DuplicateEmbedder{Cosine: .9}
	f.App.Indexer.Embedder = e
	f.App.Searcher.Embedder = e
	if _, err := f.App.Reindex(context.Background()); err != nil {
		t.Fatal(err)
	}
	f.Bind(t, "alpha")
	t.Chdir(f.Cwd)
	// Commands open their own app on the fixture's store, with the same
	// synthetic embedder, so the duplicate question fires as for a user.
	openApp = func(ctx context.Context) (*app.App, error) {
		a, err := app.Open(ctx)
		if err == nil {
			a.Embedder, a.Indexer.Embedder, a.Searcher.Embedder = e, e, e
		}
		return a, err
	}
	t.Cleanup(func() { openApp = app.Open })
	for _, tc := range []struct{ name, text, want string }{
		{"duplicate", "A synthetic-near durable quartz policy remains useful.", "Same fact?"},
		{"secret", "The deploy token is sk-proj-" + strings.Repeat("q", 32), "secrets"},
		{"place", "Synthetic work repository rule.", "applies beyond it"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			args := []string{"remember", tc.text}
			if tc.name == "place" {
				args = append(args, "--place", "global")
			}
			code, out := mainOutput(t, args...)
			if code != ExitNotStored {
				t.Fatalf("exit %d, want %d:\n%s", code, ExitNotStored, out)
			}
			if !strings.HasPrefix(out, "NOT STORED:") || !strings.Contains(out, tc.want) {
				t.Fatalf("output must start NOT STORED: and ask its question:\n%s", out)
			}
			if strings.Contains(out, "remembered ") {
				t.Fatalf("refusal printed a success line:\n%s", out)
			}
		})
	}
}
