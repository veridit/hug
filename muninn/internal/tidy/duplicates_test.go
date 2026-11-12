package tidy

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/veridit/hug/muninn/internal/extract"
	"github.com/veridit/hug/muninn/internal/store"
)

func dupeUnit(id int64, scope string) store.Unit {
	return store.Unit{ID: id, SID: fmt.Sprintf("sid%d", id), Origin: fmt.Sprintf("synthetic:%d", id), Scope: scope, Kind: "memory", Body: fmt.Sprintf("Synthetic dated note %d remains as stored.", id), Updated: time.Unix(id, 0), Active: true}
}
func TestDuplicateClusteringBandsChainsAndBoundaries(t *testing.T) {
	g := Graph{MemoryMeta: map[int64]MemoryMeta{1: {Native: true, Strength: 3}, 2: {Native: true, Strength: 4}, 3: {Native: false, Strength: 8}}, EffectiveScopes: map[int64]string{8: "client:a", 9: "client:b"}}
	for i := int64(1); i <= 9; i++ {
		scope := "jcode@alpha"
		if i == 6 {
			scope = "jcode@beta"
		}
		if i == 7 {
			scope = "client:a:project-two"
		}
		if i == 8 || i == 9 {
			scope = "global"
		}
		g.Units = append(g.Units, dupeUnit(i, scope))
	}
	g.Edges = []Edge{{1, 2, .95}, {2, 3, .96}, {1, 3, .84}, {3, 4, .9499}, {4, 5, .88}, {5, 6, .99}, {5, 7, .99}, {8, 9, .99}, {2, 1, .97}}
	p, err := ScanDuplicates(g, .95, .88)
	if err != nil || len(p.Duplicates) != 1 || len(p.Duplicates[0].Retire) != 2 || len(p.DuplicatePairs) != 4 {
		t.Fatalf("%+v %v", p, err)
	}
	if p.Duplicates[0].Keeper.SID != "sid2" {
		t.Fatal("native/recency keeper order", p.Duplicates[0])
	}
	if _, err = SelectedDuplicates(p, g); err != nil {
		t.Fatal(err)
	}
	// A newer strength breaks ties only after source preference and recency.
	g.Units[0].Updated = g.Units[1].Updated
	g.MemoryMeta[1] = MemoryMeta{Native: true, Strength: 9}
	p, _ = ScanDuplicates(g, .95, .88)
	if p.Duplicates[0].Keeper.SID != "sid1" {
		t.Fatal("strength tie break", p.Duplicates[0])
	}
	g.Units[0].Body += " changed"
	if _, err = SelectedDuplicates(p, g); err == nil {
		t.Fatal("stale plan accepted")
	}
}
func TestDuplicateExactTextWithoutNeighbors(t *testing.T) {
	a, b := dupeUnit(1, "global"), dupeUnit(2, "global")
	b.Body = a.Body
	p, err := ScanDuplicates(Graph{Units: []store.Unit{a, b}}, .95, .88)
	if err != nil || len(p.Duplicates) != 1 || p.DuplicatePairs[0].Cosine != 1 {
		t.Fatalf("%+v %v", p, err)
	}
	b.Scope = "personal"
	p, _ = ScanDuplicates(Graph{Units: []store.Unit{a, b}}, .95, .88)
	if len(p.Duplicates) != 0 {
		t.Fatal("exact match crossed scopes")
	}
}
func TestDuplicateJudgeBoundedStrictAndTransitive(t *testing.T) {
	g := Graph{Units: []store.Unit{dupeUnit(1, "global"), dupeUnit(2, "global"), dupeUnit(3, "global"), dupeUnit(4, "global")}, Edges: []Edge{{1, 2, .96}, {2, 3, .92}, {3, 4, .90}}}
	bad := false
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		var request struct {
			Messages []struct {
				Content string `json:"content"`
			} `json:"messages"`
		}
		json.NewDecoder(r.Body).Decode(&request)
		var pairs []struct{ A, B string }
		if err := json.Unmarshal([]byte(request.Messages[1].Content), &pairs); err != nil {
			t.Error(err)
		}
		answers := []map[string]string{}
		for _, p := range pairs {
			a := p.A
			if bad {
				a = "invented"
			}
			answers = append(answers, map[string]string{"a": a, "b": p.B, "decision": "same", "reason": "Same synthetic rule reworded"})
		}
		raw, _ := json.Marshal(answers)
		json.NewEncoder(w).Encode(map[string]any{"choices": []any{map[string]any{"message": map[string]string{"content": string(raw)}, "finish_reason": "stop"}}})
	}))
	defer srv.Close()
	cfg := extract.Defaults()
	cfg.Provider = "ollama"
	cfg.BaseURL = srv.URL
	p, _ := ScanDuplicates(g, .95, .88)
	if err := JudgeDuplicates(context.Background(), &p, g, cfg, 1); err != nil {
		t.Fatal(err)
	}
	if calls != 1 || len(p.Duplicates) != 1 || len(p.Duplicates[0].Retire) != 2 || p.Duplicates[0].Band != "judged" {
		t.Fatalf("calls=%d %+v", calls, p)
	}
	n := 0
	for _, pair := range p.DuplicatePairs {
		if pair.Judged {
			n++
		}
	}
	if n != 1 {
		t.Fatal("max judge not obeyed", n)
	}
	bad = true
	p, _ = ScanDuplicates(g, .95, .88)
	if err := JudgeDuplicates(context.Background(), &p, g, cfg, 1); err == nil {
		t.Fatal("invented judge identity accepted")
	}
	p, _ = ScanDuplicates(g, .95, .88)
	before := calls
	if err := JudgeDuplicates(context.Background(), &p, g, cfg, 0); err != nil || calls != before {
		t.Fatal("max judge zero called provider")
	}
}
