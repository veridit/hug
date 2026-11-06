// Command capture-eval exposes production transcript filtering and candidate
// validation to offline experiments. It never writes memories. The known mode
// indexes into an explicitly supplied private MUNINN_HOME. Redirect output to
// private scratch storage, never the repository.
package main

import (
	"bufio"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/veridit/muninn/internal/app"
	"github.com/veridit/muninn/internal/embed"
	"github.com/veridit/muninn/internal/extract"
	"github.com/veridit/muninn/internal/harness"
	"github.com/veridit/muninn/internal/project"
)

func main() {
	home := flag.String("home", "", "read-only jcode instance home")
	session := flag.String("session", "", "read-only session id")
	mode := flag.String("mode", "transcript", "transcript, secrets, validate, known, or embed")
	flag.Parse()
	enc := json.NewEncoder(os.Stdout)
	switch *mode {
	case "embed":
		var texts []string
		if err := json.NewDecoder(io.LimitReader(os.Stdin, 64<<20)).Decode(&texts); err != nil {
			fail(err)
		}
		ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
		defer cancel()
		e := embed.NewOllama("http://127.0.0.1:11434", "nomic-embed-text", 0)
		vecs, err := e.Embed(ctx, texts)
		if err != nil {
			fail(fmt.Errorf("local embedding request failed"))
		}
		if err := enc.Encode(vecs); err != nil {
			fail(err)
		}
	case "known":
		if os.Getenv("MUNINN_HOME") == "" {
			fail(fmt.Errorf("known mode requires an explicit private MUNINN_HOME"))
		}
		a, err := app.Open(context.Background())
		if err != nil {
			fail(err)
		}
		defer a.Close()
		if _, err := a.Reindex(context.Background()); err != nil {
			fail(err)
		}
		sc := bufio.NewScanner(os.Stdin)
		sc.Buffer(make([]byte, 4096), 2<<20)
		for sc.Scan() {
			var q struct{ Home, Cwd, Query string }
			if err := json.Unmarshal(sc.Bytes(), &q); err != nil {
				fail(err)
			}
			b := a.Bound(q.Home, q.Cwd)
			if b.Err != nil {
				fail(b.Err)
			}
			opts := a.RecallOptions()
			opts.MinScore = 0
			opts.Limit = 12
			opts.Scopes, opts.ScopeFor = b.Scopes, b.ScopeFor
			opts.Kinds = []string{"memory"}
			query := q.Query
			if len(query) > 4000 {
				query = query[len(query)-4000:]
			}
			hits, err := a.Searcher.Search(context.Background(), query, opts)
			if err != nil {
				fail(err)
			}
			known := []extract.Known{}
			for _, r := range hits {
				if !project.LooksSecret(r.Unit.Body) {
					known = append(known, extract.Known{SID: r.Unit.SID, Content: r.Unit.Body})
				}
			}
			if err := enc.Encode(known); err != nil {
				fail(err)
			}
		}
		if err := sc.Err(); err != nil {
			fail(err)
		}
	case "transcript":
		t, err := harness.ReadCaptureSession(*home, *session)
		if err != nil {
			fail(err)
		}
		filterSecrets(&t)
		if err := enc.Encode(t); err != nil {
			fail(err)
		}
	case "secrets":
		var texts []string
		if err := json.NewDecoder(io.LimitReader(os.Stdin, 64<<20)).Decode(&texts); err != nil {
			fail(err)
		}
		blocked := make([]bool, len(texts))
		for i, text := range texts {
			blocked[i] = project.LooksSecret(text)
		}
		if err := enc.Encode(blocked); err != nil {
			fail(err)
		}
	case "validate":
		b, err := io.ReadAll(io.LimitReader(os.Stdin, 2<<20))
		if err != nil {
			fail(err)
		}
		cs, rejected := extract.Parse(string(b))
		if err := enc.Encode(struct {
			Candidates []extract.Candidate `json:"candidates"`
			Rejected   int                 `json:"rejected"`
		}{cs, rejected}); err != nil {
			fail(err)
		}
	default:
		fail(fmt.Errorf("unknown mode"))
	}
}

func filterSecrets(t *harness.CaptureTranscript) {
	for i := range t.Messages {
		if project.LooksSecret(t.Messages[i].Text) {
			t.Messages[i].Text = ""
		}
	}
}

func fail(err error) { fmt.Fprintln(os.Stderr, err); os.Exit(1) }
