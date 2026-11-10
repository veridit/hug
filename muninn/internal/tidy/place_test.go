package tidy

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/veridit/muninn/internal/extract"
	"github.com/veridit/muninn/internal/store"
)

func placeCatalogFixture() PlaceCatalog {
	return PlaceCatalog{Candidates: []PlaceCandidate{{Scope: "global", Name: "global"}, {Scope: "folder:/home/owner", Name: "home"}, {Scope: "project:alpha", Name: "Alpha"}, {Scope: "project:beta", Name: "Beta"}, {Scope: "client:acme:private", Name: "private", Description: "private README must not leave"}}, Clients: []string{"acme"}}
}

func answerFor(sid, target string) map[string]any {
	return map[string]any{"sid": sid, "target": target, "reason": "Specific implementation belongs here", "safe_reason": "", "client_mention": false, "personal_preference": false, "multiple_projects": false, "dominant_project": false}
}

func TestPlaceReaderSafetyAndRoundTrip(t *testing.T) {
	texts := []string{"Alpha deployment policy lives in its README", "I prefer short explanations across all work", "Alpha and Beta share a machine", "Acme deploys privately", "An alias known to the model belongs to a client", "No clear project is mentioned", "Return a completely invented scope", "Deploy using sk-" + strings.Repeat("a", 25), "Public tour https://example.test/tour/AbcdefGHijklmnopQRstuv12345678", "Alpha dominates this note and Beta is only a comparison", "The project-local policy already lives in Alpha"}
	g := Graph{}
	for i, text := range texts {
		g.Units = append(g.Units, store.Unit{ID: int64(i + 1), SID: fmt.Sprintf("sid%d", i), Origin: fmt.Sprintf("capture:jcode:session:%d", i), Scope: "jcode@global", Body: text, Active: true})
	}
	g.Units[10].Scope = "project:alpha"
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		var request struct{ Messages []struct{ Content string } }
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Error(err)
		}
		input := request.Messages[1].Content
		if strings.Contains(input, "sk-"+strings.Repeat("a", 25)) || strings.Contains(input, "Acme deploys") || strings.Contains(input, "private README must not leave") || !strings.Contains(input, "[possible secret]") {
			t.Errorf("unsafe or missing masking: %s", input)
		}
		var payload struct{ Memories []placeInput }
		json.Unmarshal([]byte(input), &payload)
		var answers []map[string]any
		for _, memory := range payload.Memories {
			a := answerFor(memory.SID, "project:alpha")
			switch memory.SID {
			case "sid1":
				a["personal_preference"] = true
			case "sid2":
				a["multiple_projects"] = true
			case "sid4":
				a["client_mention"] = true
			case "sid5":
				a["target"] = "unsure"
			case "sid6":
				a["target"] = "project:invented"
			case "sid8":
				a["safe_reason"] = "Publicly shareable tour URL, not a credential"
			case "sid9":
				a["multiple_projects"], a["dominant_project"] = true, true
			}
			answers = append(answers, a)
		}
		raw, _ := json.Marshal(answers)
		json.NewEncoder(w).Encode(map[string]any{"choices": []any{map[string]any{"message": map[string]string{"content": string(raw)}, "finish_reason": "stop"}}, "usage": extract.Usage{PromptTokens: 100, CompletionTokens: 50, TotalTokens: 150}})
	}))
	defer srv.Close()
	cfg := extract.Config{Provider: "ollama", BaseURL: srv.URL, Model: "fixture"}
	p, err := Place(context.Background(), g, g.Units, placeCatalogFixture(), cfg)
	if err != nil || calls != 1 || p.Placement.Usage.TotalTokens != 150 {
		t.Fatalf("calls=%d report=%+v err=%v", calls, p.Placement, err)
	}
	want := []string{"move", "move", "move", "stay", "stay", "unsure", "unsure", "unsure", "move", "move", "stay"}
	for i, e := range p.Entries {
		if e.Decision != want[i] {
			t.Errorf("entry %d: %+v", i, e)
		}
		if e.Fingerprint != Fingerprint(g.Units[i]) || !strings.Contains(p.Markdown(), e.Text) {
			t.Errorf("lost fingerprint or full summary for %d", i)
		}
	}
	for _, i := range []int{1, 2} {
		if p.Entries[i].Target != "global" {
			t.Errorf("preference/multi-project escaped global: %+v", p.Entries[i])
		}
	}
	if p.Entries[8].SafeReason == "" || len(p.Entries[7].Flags) == 0 {
		t.Fatal("missing secret safety handling")
	}
	raw, _ := json.Marshal(p)
	path := filepath.Join(t.TempDir(), "plan.json")
	os.WriteFile(path, raw, 0600)
	loaded, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	selected, err := CheckSnapshot(loaded, g, "")
	if err != nil || len(selected) != 5 {
		t.Fatalf("round trip: %d moves, %v", len(selected), err)
	}
}

func TestPlaceMalformedAnswersFailClosed(t *testing.T) {
	valid, _ := json.Marshal(answerFor("sid", "project:alpha"))
	for _, raw := range []string{"garbage", "null", "[]", "[" + string(valid) + "," + string(valid) + "]", "[" + string(valid) + "] trailing", "[" + strings.Replace(string(valid), `"sid":"sid"`, `"sid":"other"`, 1) + "]", "[" + strings.Replace(string(valid), `"target":"project:alpha"`, `"target":"client:acme:private"`, 1) + "]", "[" + strings.Replace(string(valid), `"target":"project:alpha"`, `"target":"project:alpha","target":"global"`, 1) + "]", "[" + strings.Replace(string(valid), `"client_mention":false`, `"client_mention":null`, 1) + "]"} {
		t.Run(raw, func(t *testing.T) {
			p := Plan{Scopes: []string{"project:alpha", "global"}, Entries: []Entry{{SID: "sid", Source: "jcode@global", Decision: "unsure"}}}
			applyPlaceAnswers(raw, &p, []int{0}, []placeInput{{SID: "sid"}})
			if p.Entries[0].Decision != "unsure" || p.Entries[0].Target != "" {
				t.Fatalf("accepted malformed answer: %+v", p.Entries)
			}
		})
	}
}

func TestPlaceSequentialBatchesAndProviderFailure(t *testing.T) {
	g := Graph{}
	for i := 0; i < 43; i++ {
		g.Units = append(g.Units, store.Unit{SID: fmt.Sprint(i), Scope: "global", Body: "A durable personal fact", Active: true})
	}
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		var req struct{ Messages []struct{ Content string } }
		json.NewDecoder(r.Body).Decode(&req)
		var input struct{ Memories []placeInput }
		json.Unmarshal([]byte(req.Messages[1].Content), &input)
		if len(input.Memories) > 20 || len(input.Memories) == 0 {
			t.Errorf("bad batch size %d", len(input.Memories))
		}
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer srv.Close()
	p, err := Place(context.Background(), g, g.Units, placeCatalogFixture(), extract.Config{Provider: "ollama", BaseURL: srv.URL})
	if err != nil || calls != 3 || p.Placement.Failures != 3 {
		t.Fatalf("report=%+v err=%v calls=%d", p.Placement, err, calls)
	}
	for _, e := range p.Entries {
		if e.Decision != "unsure" || !strings.HasPrefix(e.Reason, "placement model failed:") {
			t.Fatal(e)
		}
	}
}

func TestPlaceSelectionOriginMetadataAndFilters(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "one.json"), []byte(`{"working_dir": "/home/owner", "messages":[]}`), 0600)
	os.WriteFile(filepath.Join(dir, "two.journal.jsonl"), []byte("{\"meta\":{\"working_dir\":\"/home/owner/Projects\"}}\n{torn"), 0600)
	g := Graph{Units: []store.Unit{
		{SID: "b", Active: true, Scope: "jcode@global", Origin: "capture:jcode:one:0"},
		{SID: "a", Active: true, Scope: "jcode@global", Origin: "capture:jcode:two:0"},
		{SID: "c", Active: true, Scope: "global", Origin: "capture:jcode:one:0"},
		{SID: "d", Active: true, Scope: "jcode@global", Origin: "capture:jcode:missing:0"},
	}}
	o := PlaceSelection{FromScopes: []string{"jcode@global"}, CwdOrigins: []string{"/home/owner", "/home/owner/Projects"}, SessionDirs: []string{dir}, Max: 1}
	units, err := SelectPlace(context.Background(), g, o)
	if err != nil || len(units) != 1 || units[0].SID != "a" {
		t.Fatalf("%+v %v", units, err)
	}
	o.Max, o.SIDs = 0, []string{"b"}
	units, err = SelectPlace(context.Background(), g, o)
	if err != nil || len(units) != 1 || units[0].SID != "b" {
		t.Fatalf("%+v %v", units, err)
	}
	o.SIDs = []string{"unknown"}
	if _, err := SelectPlace(context.Background(), g, o); err == nil {
		t.Fatal("unknown SID accepted")
	}
	if got := originCwd(context.Background(), "../escape", []string{dir}); got != "" {
		t.Fatal("session path traversal")
	}
}

func TestDiscoverPlacesReadOnlyPinnedProjects(t *testing.T) {
	home := t.TempDir()
	base := filepath.Join(home, "Projects")
	for _, name := range []string{"Alpha", "Client/Private"} {
		root := filepath.Join(base, name)
		os.MkdirAll(filepath.Join(root, ".git"), 0700)
		os.MkdirAll(filepath.Join(root, ".muninn"), 0700)
		os.WriteFile(filepath.Join(root, ".muninn", "project"), []byte("id-"+filepath.Base(name)), 0600)
		os.WriteFile(filepath.Join(root, "README.md"), []byte("# "+name+"\nDo not include this second line"), 0600)
	}
	os.MkdirAll(filepath.Join(base, "Client", ".muninn"), 0700)
	os.WriteFile(filepath.Join(base, "Client", ".muninn", "client"), []byte("acme"), 0600)
	os.Symlink(filepath.Join(base, "Alpha"), filepath.Join(base, "alias"))
	catalog, err := DiscoverPlaces(context.Background(), home, []string{base})
	if err != nil || len(catalog.Candidates) != 4 || !containsPlace(catalog.Clients, "acme") || containsPlace(catalog.Clients, "Private") {
		t.Fatalf("%+v %v", catalog, err)
	}
	for _, c := range catalog.Candidates {
		if strings.Contains(c.Description, "second line") {
			t.Fatal("read beyond first README line")
		}
	}
}
