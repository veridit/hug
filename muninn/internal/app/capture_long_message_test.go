package app_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/veridit/hug/muninn/internal/extract"
	"github.com/veridit/hug/muninn/internal/harness"
	"github.com/veridit/hug/muninn/internal/scopetest"
)

// 8jcj3k: one message longer than the chunk limit stopped a session's capture
// for good ("capture message exceeds 8192 bytes, refusing silent loss"), and
// every later hook retried and failed again. A long message is cut into parts
// under the limit, in order; every part reaches the extractor, the split is
// recorded in capture_runs, and the turns after it are captured.
func TestCaptureLongMessageIsSplitNotRefused(t *testing.T) {
	for _, tc := range []struct {
		name, trigger string
		size, limit   int
	}{
		{"hook 20000 bytes", "turn_end", 20000, 8192},
		{"session end 8192 limit exact multiple", "session_end", 3 * 8192, 8192},
		{"manual 60000 bytes", "", 60000, 48 << 10},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := scopetest.New(t)
			b := f.Bind(t, "alpha")
			var mu sync.Mutex
			var transcripts []string
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var request struct {
					Messages []struct {
						Content string `json:"content"`
					} `json:"messages"`
				}
				if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
					t.Error(err)
					return
				}
				var input struct {
					Transcript string `json:"transcript"`
				}
				json.Unmarshal([]byte(request.Messages[1].Content), &input)
				mu.Lock()
				transcripts = append(transcripts, input.Transcript)
				mu.Unlock()
				json.NewEncoder(w).Encode(map[string]any{"choices": []any{map[string]any{"message": map[string]string{"content": ""}, "finish_reason": "stop"}}})
			}))
			defer server.Close()
			f.App.Cfg.Extract = extract.Defaults()
			f.App.Cfg.Extract.Provider = "ollama"
			f.App.Cfg.Extract.BaseURL = server.URL

			// ASCII, so no rune boundary shortens a part below the limit.
			var long strings.Builder
			for i := 0; long.Len() < tc.size; i++ {
				long.WriteString("line ")
				long.WriteString(strings.Repeat("x", i%7))
				long.WriteString(" of the long report\n")
			}
			text := long.String()[:tc.size]
			o := captureOpts("long-" + tc.trigger)
			o.Trigger = tc.trigger
			o.Messages = []harness.CaptureMessage{
				{ID: "before", Role: "user", Text: "Short question before the report."},
				{ID: "long", Role: "assistant", Text: text},
				{ID: "after", Role: "user", Text: "I prefer quartz deployment policy after the report."},
			}
			r, err := b.Capture(context.Background(), f.App, o)
			if err != nil {
				t.Fatalf("capture refused a %d-byte message: %v (report %+v)", tc.size, err, r)
			}
			if r.Split != 1 {
				t.Fatalf("report.Split = %d, want 1 (the long message)", r.Split)
			}

			mu.Lock()
			all := strings.Join(transcripts, "")
			mu.Unlock()
			for i, tr := range transcripts {
				if len(tr) > tc.limit {
					t.Fatalf("call %d sent %d bytes, limit %d", i, len(tr), tc.limit)
				}
			}
			// The text's own newlines stay inside a part, so rebuild from the
			// transcripts' assistant parts rather than by line.
			got := assistantText(all)
			if got != text {
				t.Fatalf("long message reached the extractor changed or incomplete: got %d bytes, want %d", len(got), len(text))
			}
			if !strings.Contains(all, "user: I prefer quartz deployment policy after the report.") {
				t.Fatal("the turn after the long message never reached the extractor")
			}
			if strings.Index(all, "Short question before") > strings.Index(all, "quartz deployment policy after") {
				t.Fatal("parts out of order")
			}

			for _, id := range []string{"before", "long", "after"} {
				var n int
				// capture_seen keys sessions by a hash; this fixture holds one session.
				if err := f.App.DB.QueryRow(`SELECT count(*) FROM capture_seen WHERE message=?`, id).Scan(&n); err != nil || n != 1 {
					t.Fatalf("%s not checkpointed: %d %v", id, n, err)
				}
			}
			runs, err := f.App.AuditRuns(o.Session, time.Now().Add(-time.Hour))
			if err != nil {
				t.Fatal(err)
			}
			parts, split := 0, 0
			for _, run := range runs {
				split += run.Report.Split
				for _, m := range run.Messages {
					if m.OriginalID == "long" {
						parts++
					}
				}
			}
			if want := (tc.size + tc.limit - 1) / tc.limit; parts < want || split != 1 {
				t.Fatalf("capture_runs records %d parts of 'long' (want >= %d) and split=%d (want 1)", parts, want, split)
			}

			// A second hook replays nothing.
			before := len(transcripts)
			r, err = b.Capture(context.Background(), f.App, o)
			if err != nil || r.Processed != 0 || len(transcripts) != before {
				t.Fatalf("second run replayed: %+v %v", r, err)
			}
		})
	}
}

// assistantText joins the assistant parts of a transcript stream. A part is
// written as "assistant: " + part + "\n"; parts are cut at byte limits, so a
// part may itself hold newlines and the next line may not start with a role.
func assistantText(all string) string {
	var out strings.Builder
	rest := all
	for {
		i := strings.Index(rest, "assistant: ")
		if i < 0 {
			return out.String()
		}
		rest = rest[i+len("assistant: "):]
		end := len(rest)
		for _, marker := range []string{"\nuser: ", "\nassistant: "} {
			if j := strings.Index(rest, marker); j >= 0 && j < end {
				end = j
			}
		}
		part := rest[:end]
		if end == len(rest) {
			part = strings.TrimSuffix(part, "\n")
		}
		out.WriteString(part)
		rest = rest[end:]
	}
}
