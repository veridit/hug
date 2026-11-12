package app_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"strings"
	"testing"

	"github.com/veridit/hug/muninn/internal/app"
	"github.com/veridit/hug/muninn/internal/embed"
	"github.com/veridit/hug/muninn/internal/extract"
	"github.com/veridit/hug/muninn/internal/ingest"
	"github.com/veridit/hug/muninn/internal/scopetest"
)

type bandVectors struct{ score float64 }

func (bandVectors) Model() string { return "synthetic-duplicate-bands" }
func (v bandVectors) Embed(_ context.Context, texts []string) ([][]float32, error) {
	out := make([][]float32, 0, len(texts))
	for _, s := range texts {
		switch {
		case strings.Contains(s, "candidate-band"):
			out = append(out, []float32{float32(v.score), float32(math.Sqrt(1 - v.score*v.score))})
		case strings.Contains(s, "Matrix(Alpha)"):
			out = append(out, []float32{1, 0})
		default:
			out = append(out, []float32{-1, 0})
		}
	}
	return out, nil
}
func useBands(t *testing.T, f *scopetest.Fixture, score float64) {
	t.Helper()
	v := bandVectors{score}
	f.App.Embedder = v
	f.App.Indexer.Embedder = v
	f.App.Searcher.Embedder = v
	if _, err := f.App.Reindex(context.Background()); err != nil {
		t.Fatal(err)
	}
}

const bandContent = "Quartz candidate-band policy remains durable across deployments."

type recipeVectors struct {
	bandVectors
	inputs []string
}

func (*recipeVectors) Model() string { return "builtin:snowflake-arctic-embed-s" }
func (v *recipeVectors) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	v.inputs = append(v.inputs, texts...)
	return v.bandVectors.Embed(ctx, texts)
}
func TestDedupeUsesDocumentRecipeAndFailsClosedDuringMigration(t *testing.T) {
	f := scopetest.New(t)
	v := &recipeVectors{bandVectors: bandVectors{score: .96}}
	f.App.Indexer.Embedder = v
	f.App.Searcher.Embedder = v
	if _, err := f.App.Reindex(context.Background()); err != nil {
		t.Fatal(err)
	}
	v.inputs = nil
	b := f.Bind(t, "alpha")
	if _, re, err := b.Remember(context.Background(), f.App, bandContent, "fact", "", nil); err != nil || !re {
		t.Fatal(re, err)
	}
	for _, s := range v.inputs {
		if strings.HasPrefix(s, embed.PromptsFor(v.Model()).Query) {
			t.Fatal("dedupe used retrieval query prefix", s)
		}
	}
	if len(v.inputs) == 0 {
		t.Fatal("dedupe did not embed candidate")
	}
	if err := f.App.DB.SetMeta("pipeline_hash", "previous-recipe"); err != nil {
		t.Fatal(err)
	}
	v.inputs = nil
	o := f.App.RecallOptions()
	o.MinScore = 0
	o.DocumentQuery = true
	if _, err := f.App.Searcher.Search(context.Background(), bandContent, o); err != nil {
		t.Fatal(err)
	}
	if len(v.inputs) != 0 {
		t.Fatal("dedupe compared mixed embedding recipes")
	}
}

func TestRememberDuplicateBandsAndQuestionAnswers(t *testing.T) {
	for _, score := range []float64{.99, .97, .95001, .9499, .92, .90, .88001, .8799, .85} {
		t.Run(fmt.Sprint(score), func(t *testing.T) {
			f := scopetest.New(t)
			useBands(t, f, score)
			b := f.Bind(t, "alpha")
			before, _ := os.ReadFile(app.NativePath())
			u, re, err := b.Remember(context.Background(), f.App, bandContent, "fact", "", nil)
			switch {
			case score >= .95:
				if err != nil || !re || u.SID != f.Units["Alpha"].SID {
					t.Fatalf("u=%+v reinforced=%v err=%v", u, re, err)
				}
				var record string
				if err = f.App.DB.QueryRow(`SELECT record FROM capture_actions ORDER BY run_id DESC LIMIT 1`).Scan(&record); err != nil {
					t.Fatal(err)
				}
				if !strings.Contains(record, u.SID) || !strings.Contains(record, "cosine") {
					t.Fatal("missing audit target/score", record)
				}
			case score >= .88:
				var question *app.DuplicateCheckError
				if !errors.As(err, &question) || question.SID != f.Units["Alpha"].SID || !strings.Contains(err.Error(), "Same fact?") {
					t.Fatalf("question=%+v err=%v", question, err)
				}
				after, _ := os.ReadFile(app.NativePath())
				if string(before) != string(after) {
					t.Fatal("question stored a memory")
				}
				u, re, err = b.RememberPlaced(context.Background(), f.App, bandContent, "fact", "", nil, "", "", "", app.RememberDecision{NewReason: "Synthetic distinction: this is a separate policy dimension."})
				if err != nil || re || u.SID == question.SID {
					t.Fatalf("override u=%+v re=%v err=%v", u, re, err)
				}
				ms, _ := ingest.LoadNative(app.NativePath())
				found := false
				for _, m := range ms {
					if m.ID == u.Key {
						found = m.NewReason != ""
					}
				}
				if !found {
					t.Fatal("new_reason not persisted")
				}
			default:
				if err != nil || re {
					t.Fatalf("new memory re=%v err=%v", re, err)
				}
			}
		})
	}
	t.Run("explicit-reinforce", func(t *testing.T) {
		f := scopetest.New(t)
		useBands(t, f, .90)
		b := f.Bind(t, "alpha")
		old := f.Units["Alpha"]
		u, re, err := b.RememberPlaced(context.Background(), f.App, bandContent, "fact", "", nil, "", "", "", app.RememberDecision{Reinforce: old.SID})
		if err != nil || !re || u.SID != old.SID || u.Body != old.Body {
			t.Fatalf("%+v %v %v", u, re, err)
		}
		if _, _, err = b.RememberPlaced(context.Background(), f.App, bandContent, "fact", "", nil, "", "", "", app.RememberDecision{Reinforce: f.Units["Beta"].SID}); err == nil {
			t.Fatal("reinforced invisible target")
		}
	})
}

func TestCaptureSameCallJudgmentsAndUpdateTrust(t *testing.T) {
	for _, tc := range []struct {
		name, category, trust, link, want string
		score                             float64
		active                            bool
	}{
		{"automatic", "fact", "medium", "", "reinforce", .97, true},
		{"review-same", "fact", "medium", "same_as", "reinforce", .92, true},
		{"review-new", "fact", "medium", "", "add", .92, true},
		{"below-review", "fact", "medium", "", "add", .87, true},
		{"update-observed", "fact", "medium", "updates", "add", .99, true},
		{"update-user-fact", "fact", "high", "updates", "add", .99, true},
		{"correction-medium", "correction", "medium", "updates", "add", .99, true},
		{"correction-high", "correction", "high", "updates", "supersede", .99, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := scopetest.New(t)
			useBands(t, f, tc.score)
			old := f.Units["Alpha"]
			b := f.Bind(t, "alpha")
			c := extract.Candidate{Content: bandContent, Category: tc.category, Trust: tc.trust}
			if tc.link == "same_as" {
				c.SameAs = old.SID
			}
			if tc.link == "updates" {
				c.Updates = old.SID
			}
			raw, _ := json.Marshal(c)
			model := fakeCapture(t, f, string(raw))
			report, err := b.Capture(context.Background(), f.App, captureOpts(tc.name))
			if err != nil || len(report.Actions) != 1 || report.Actions[0].Action != tc.want || model.calls.Load() != 1 {
				t.Fatalf("report=%+v err=%v calls=%d input=%s", report, err, model.calls.Load(), model.input)
			}
			current, _ := f.App.DB.GetUnit(old.ID)
			if current.Active != tc.active {
				t.Fatal("wrong retirement state")
			}
			ms, _ := ingest.LoadNative(app.NativePath())
			for _, m := range ms {
				if m.SID == old.SID && m.Content != old.Body {
					t.Fatal("dated text rewritten")
				}
				if m.Content == bandContent && tc.link == "updates" && m.Updates != old.SID {
					t.Fatal("update link lost")
				}
			}
		})
	}
}
