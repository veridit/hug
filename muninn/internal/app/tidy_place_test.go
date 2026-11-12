package app_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/veridit/hug/muninn/internal/extract"
	"github.com/veridit/hug/muninn/internal/scopetest"
	"github.com/veridit/hug/muninn/internal/store"
	"github.com/veridit/hug/muninn/internal/tidy"
)

func TestPlacePlanReusesJcodeCopyAndSuppress(t *testing.T) {
	f := scopetest.New(t)
	u := f.Units["OutsideNote"]
	// Exercise the exact non-native jcode kind through the shared apply path.
	if _, err := f.App.DB.Exec(`UPDATE sources SET kind='jcode' WHERE id=?`, u.SourceID); err != nil {
		t.Fatal(err)
	}
	var path string
	f.App.DB.QueryRow(`SELECT path FROM sources WHERE id=?`, u.SourceID).Scan(&path)
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := json.Marshal([]map[string]any{{"sid": u.SID, "target": "project:matrix-project", "reason": "The implementation belongs to the project", "safe_reason": "", "client_mention": false, "personal_preference": false, "multiple_projects": false, "dominant_project": false}})
		json.NewEncoder(w).Encode(map[string]any{"choices": []any{map[string]any{"message": map[string]string{"content": string(raw)}, "finish_reason": "stop"}}})
	}))
	defer srv.Close()
	g := tidy.Graph{Units: []store.Unit{u}}
	p, err := tidy.Place(context.Background(), g, g.Units, tidy.PlaceCatalog{Candidates: []tidy.PlaceCandidate{{Scope: "global"}, {Scope: "project:matrix-project"}}}, extract.Config{Provider: "ollama", BaseURL: srv.URL})
	if err != nil {
		t.Fatal(err)
	}
	dry, err := f.App.ApplyTidy(context.Background(), p, "", true)
	if err != nil || len(dry.Moves) != 1 {
		t.Fatalf("dry=%+v err=%v", dry, err)
	}
	result, err := f.App.ApplyTidy(context.Background(), p, "", false)
	if err != nil || len(result.Moves) != 1 {
		t.Fatalf("result=%+v err=%v", result, err)
	}
	copy, err := f.App.DB.Resolve(result.Moves[0].Ref)
	if err != nil || copy.Body != u.Body || copy.Origin != "tidy:"+u.Origin || copy.Scope != "project:matrix-project" || copy.SID == u.SID || !f.App.DB.IsSuppressed(u.SID) || f.App.DB.IsSuppressed(copy.SID) {
		t.Fatalf("copy=%+v err=%v", copy, err)
	}
	after, _ := os.ReadFile(path)
	if string(before) != string(after) {
		t.Fatal("upstream jcode source changed")
	}
}
