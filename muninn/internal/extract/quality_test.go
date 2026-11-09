package extract

import (
	"context"
	"encoding/json"
	"math"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestTrivialMinimumWordsAndShortExceptions(t *testing.T) {
	for _, tc := range []struct {
		category, content string
		want              bool
	}{
		{"fact", "Tests pass at commit abc123.", false},
		{"decision", "User approved the plan.", false},
		{"fact", "Thanks for all your help.", false},
		{"fact", "One two three four five", false},
		{"fact", "One two three four five six", true},
		{"decision", "Keep nightly builds isolated from production systems.", true},
		{"preference", "Use tabs.", true},
		{"correction", "Use PostgreSQL.", true},
	} {
		raw, _ := json.Marshal(Candidate{Category: tc.category, Content: tc.content, Trust: "high"})
		got, reasons := parseDetailed(string(raw))
		if (len(got) == 1) != tc.want {
			t.Fatalf("%s: got=%+v reasons=%+v", tc.content, got, reasons)
		}
		if !tc.want && (len(reasons) != 1 || !strings.HasPrefix(reasons[0].Reason, "trivial:")) {
			t.Fatal(reasons)
		}
	}
	for _, s := range []string{"task progress snapshots", "user asked for a recap", "Keep settled decisions WITH their content", "same_as", "updates"} {
		if !strings.Contains(prompt, s) {
			t.Fatal("prompt missing", s)
		}
	}
}
func TestRelationshipSchemaAndKnownMembership(t *testing.T) {
	for _, suffix := range []string{`,"same_as":"abc123","updates":"def456"`, `,"same_as":"bad!"`, `,"updates":12`} {
		if got, n := Parse(`{"category":"fact","trust":"high","content":"Quartz deployments require a durable checklist forever"` + suffix + `}`); len(got) != 0 || n != 1 {
			t.Fatal("invalid relation accepted", got, n)
		}
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{"choices": []any{map[string]any{"message": map[string]string{"content": `{"category":"fact","trust":"medium","content":"Quartz deployments require a durable checklist forever","same_as":"def456"}`}, "finish_reason": "stop"}}})
	}))
	defer server.Close()
	c := Defaults()
	c.Provider = "ollama"
	c.BaseURL = server.URL
	out, err := Run(context.Background(), c, "synthetic transcript", []Known{{SID: "abc123", Content: "Existing synthetic rule"}})
	if err != nil || len(out.Candidates) != 0 || out.Rejected != 1 {
		t.Fatalf("%+v %v", out, err)
	}
}
func TestDedupeThresholdConfiguration(t *testing.T) {
	a, r := Defaults().DedupeThresholds()
	if a != .95 || r != .88 {
		t.Fatal(a, r)
	}
	c := Config{DedupeCosine: .97, ReviewCosine: .9}
	a, r = c.DedupeThresholds()
	if a != .97 || r != .9 {
		t.Fatal(a, r)
	}
	for _, bad := range []float64{-1, 0, 2, math.NaN(), math.Inf(1)} {
		c = Config{DedupeCosine: bad, ReviewCosine: bad}
		a, r = c.DedupeThresholds()
		if a != .95 || r != .88 {
			t.Fatal(a, r)
		}
	}
}
