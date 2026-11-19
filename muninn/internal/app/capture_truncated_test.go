package app_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/veridit/hug/muninn/internal/app"
	"github.com/veridit/hug/muninn/internal/extract"
	"github.com/veridit/hug/muninn/internal/harness"
	"github.com/veridit/hug/muninn/internal/ingest"
	"github.com/veridit/hug/muninn/internal/scopetest"
)

// truncatingLLM answers every chunk with one memory named after the chunk's
// marker, except that the chunk carrying "quartz-overflow" is cut at
// max_tokens (finish_reason length) while maxTokens <= limit. It records the
// max_tokens of every request for that chunk.
type truncatingLLM struct {
	mu          sync.Mutex
	limit       int
	overflow    []int
	transcripts []string
}

const partialMemory = "Quartz partial reply must never be saved as complete."

func (l *truncatingLLM) serve(t *testing.T, f *scopetest.Fixture) {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request struct {
			MaxTokens int `json:"max_tokens"`
			Messages  []struct {
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
		l.mu.Lock()
		l.transcripts = append(l.transcripts, input.Transcript)
		overflow := strings.Contains(input.Transcript, "quartz-overflow")
		if overflow {
			l.overflow = append(l.overflow, request.MaxTokens)
		}
		limit := l.limit
		l.mu.Unlock()
		reply, finish := "", "stop"
		for _, marker := range []string{"alpha", "overflow", "omega"} {
			if strings.Contains(input.Transcript, "quartz-"+marker) {
				reply = line("fact", "high", "Quartz "+marker+" deployments need a durable checklist.", "")
			}
		}
		if overflow && request.MaxTokens <= limit {
			// A complete first line and a cut second one: the shape of run 8449.
			reply = line("fact", "high", partialMemory, "") + "\n{\"category\":\"fact\",\"trust\":\"hi"
			finish = "length"
		}
		json.NewEncoder(w).Encode(map[string]any{"choices": []any{map[string]any{"message": map[string]string{"content": reply}, "finish_reason": finish}}, "usage": extract.Usage{PromptTokens: 100, CompletionTokens: request.MaxTokens, TotalTokens: 100 + request.MaxTokens}})
	}))
	t.Cleanup(server.Close)
	f.App.Cfg.Extract = extract.Defaults()
	f.App.Cfg.Extract.Provider = "ollama"
	f.App.Cfg.Extract.BaseURL = server.URL
	f.App.Cfg.Extract.HookChunkBytes = 1024
}

// truncatedOpts is a hook run whose three messages land in three chunks:
// alpha, the overflowing one, omega.
func truncatedOpts(session string) app.CaptureOptions {
	o := captureOpts(session)
	o.Trigger = "session_end"
	pad := strings.Repeat("Durable deployment detail. ", 25)
	o.Messages = []harness.CaptureMessage{
		{ID: "alpha", Role: "user", Text: "quartz-alpha " + pad},
		{ID: "overflow", Role: "assistant", Text: "quartz-overflow " + pad},
		{ID: "omega", Role: "user", Text: "quartz-omega " + pad},
	}
	return o
}

func savedContents(t *testing.T) map[string]bool {
	t.Helper()
	mems, err := ingest.LoadNative(app.NativePath())
	if err != nil {
		t.Fatal(err)
	}
	out := map[string]bool{}
	for _, m := range mems {
		out[m.Content] = true
	}
	return out
}

func seenCount(t *testing.T, f *scopetest.Fixture, table, message string) int {
	t.Helper()
	var n int
	if err := f.App.DB.QueryRow(`SELECT count(*) FROM `+table+` WHERE message=?`, message).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

// nmn6wf check 1: a hook call cut at max_tokens is retried once with room to
// finish, and the run's high-water mark moves past that chunk.
func TestCaptureTruncatedChunkRetriesAndMovesPast(t *testing.T) {
	f := scopetest.New(t)
	b := f.Bind(t, "alpha")
	l := &truncatingLLM{limit: 1024}
	l.serve(t, f)
	o := truncatedOpts("truncated-once")
	r, err := b.Capture(context.Background(), f.App, o)
	if err != nil || r.Processed != 3 {
		t.Fatalf("run stopped at the truncated chunk: processed=%d err=%v", r.Processed, err)
	}
	if len(l.overflow) != 2 || l.overflow[0] != 1024 || l.overflow[1] <= 1024 {
		t.Fatalf("overflow chunk requests max_tokens=%v, want one 1024 call then one larger retry", l.overflow)
	}
	if seenCount(t, f, "capture_seen", "overflow") != 1 || seenCount(t, f, "capture_seen", "omega") != 1 {
		t.Fatal("high-water mark did not move past the truncated chunk")
	}
	saved := savedContents(t)
	if !saved["Quartz overflow deployments need a durable checklist."] || !saved["Quartz omega deployments need a durable checklist."] {
		t.Fatalf("retried or later chunk not applied: %v", saved)
	}
	if saved[partialMemory] {
		t.Fatal("truncated reply applied as complete")
	}
	t.Logf("CAPTURE truncated: overflow chunk max_tokens=%v, processed=%d, partial reply not applied", l.overflow, r.Processed)
}

// nmn6wf check 2: a reply still truncated after the retry is recorded in
// capture_runs, kept for --backfill, and later chunks and hooks go on.
func TestCaptureStillTruncatedIsRecordedAndDoesNotBlock(t *testing.T) {
	f := scopetest.New(t)
	b := f.Bind(t, "alpha")
	l := &truncatingLLM{limit: 1 << 20}
	l.serve(t, f)
	o := truncatedOpts("truncated-always")
	r, err := b.Capture(context.Background(), f.App, o)
	if err != nil || r.Processed != 3 {
		t.Fatalf("still-truncated chunk blocked the run: processed=%d err=%v", r.Processed, err)
	}
	if len(l.overflow) != 2 {
		t.Fatalf("overflow chunk calls=%d, want exactly one retry", len(l.overflow))
	}
	saved := savedContents(t)
	if saved[partialMemory] {
		t.Fatal("truncated reply applied as complete")
	}
	if !saved["Quartz omega deployments need a durable checklist."] {
		t.Fatal("a later chunk was blocked by the truncated one")
	}
	if seenCount(t, f, "capture_seen", "overflow") != 1 || seenCount(t, f, "capture_backlog", "overflow") != 1 || seenCount(t, f, "capture_deferred", "overflow") != 1 {
		t.Fatal("still-truncated chunk not checkpointed and kept for the drain and --backfill")
	}
	var truncated int
	var deferred, runErr string
	if err := f.App.DB.QueryRow(`SELECT coalesce(json_extract(record,'$.report.truncated'),0), coalesce(json_extract(record,'$.report.deferred'),''), coalesce(json_extract(record,'$.error'),'') FROM capture_runs WHERE session=? ORDER BY id DESC LIMIT 1`, o.Session).Scan(&truncated, &deferred, &runErr); err != nil {
		t.Fatal(err)
	}
	if truncated != 1 || deferred != `["overflow"]` || runErr != "" {
		t.Fatalf("capture_runs does not record the still-truncated chunk: truncated=%d deferred=%s error=%q", truncated, deferred, runErr)
	}
	// The next hook neither re-sends it nor stops on it.
	before := len(l.transcripts)
	o.Messages = append(o.Messages, harness.CaptureMessage{ID: "later", Role: "user", Text: "quartz-alpha later turn"})
	r, err = b.Capture(context.Background(), f.App, o)
	if err != nil || r.Processed != 1 || len(l.transcripts) != before+1 {
		t.Fatalf("next hook: processed=%d calls=%d err=%v", r.Processed, len(l.transcripts)-before, err)
	}
	// A manual --backfill sends it again at the full cap, where it fits.
	l.mu.Lock()
	l.limit = 2048
	l.mu.Unlock()
	o.Trigger, o.Backfill = "", true
	if _, err := b.Capture(context.Background(), f.App, o); err != nil {
		t.Fatal(err)
	}
	if !savedContents(t)["Quartz overflow deployments need a durable checklist."] || seenCount(t, f, "capture_backlog", "overflow") != 0 || seenCount(t, f, "capture_deferred", "overflow") != 0 {
		t.Fatal("--backfill did not recover the deferred chunk")
	}
	t.Logf("CAPTURE still truncated: max_tokens=%v, capture_runs truncated=%d deferred=%s, later chunk applied, --backfill recovered it", l.overflow, truncated, deferred)
}

// crab's point (1) on nmn6wf: a session's last hook (session_end) has no
// next hook of its own, so a chunk it defers must be drained by a later
// hook of any session: DrainDeferred, at the full cap, once.
func TestCaptureSessionEndDeferredChunkIsDrainedByALaterHook(t *testing.T) {
	f := scopetest.New(t)
	b := f.Bind(t, "alpha")
	l := &truncatingLLM{limit: 2048}
	l.serve(t, f)
	o := truncatedOpts("ended")
	if r, err := b.Capture(context.Background(), f.App, o); err != nil || r.Truncated != 1 {
		t.Fatalf("session_end run=%+v err=%v", r, err)
	}
	if savedContents(t)["Quartz overflow deployments need a durable checklist."] {
		t.Fatal("overflow chunk applied before the drain")
	}
	reads := 0
	read := func(harnessName, home, session string) (string, []harness.CaptureMessage, error) {
		reads++
		if harnessName != o.Harness || home != o.Home || session != o.Session {
			t.Fatalf("drain read %s %s %s", harnessName, home, session)
		}
		return f.Cwd, o.Messages, nil
	}
	before := len(l.overflow)
	r, drained, err := f.App.DrainDeferred(context.Background(), read)
	if err != nil || !drained || r.Processed != 1 || reads != 1 {
		t.Fatalf("drain=%+v drained=%v reads=%d err=%v", r, drained, reads, err)
	}
	if len(l.overflow) != before+1 || l.overflow[len(l.overflow)-1] != extract.Defaults().MaxTokens {
		t.Fatalf("drain sent max_tokens=%v, want one call at the full cap", l.overflow[before:])
	}
	if !savedContents(t)["Quartz overflow deployments need a durable checklist."] || seenCount(t, f, "capture_deferred", "overflow") != 0 || seenCount(t, f, "capture_backlog", "overflow") != 0 {
		t.Fatal("drained chunk not applied or still queued")
	}
	if _, drained, err := f.App.DrainDeferred(context.Background(), read); drained || err != nil || reads != 1 {
		t.Fatalf("empty queue drained=%v err=%v reads=%d", drained, err, reads)
	}
	t.Logf("CAPTURE session_end deferred: hook max_tokens=%v, drained once at %d, applied, queue empty", l.overflow[:before], l.overflow[before])
}

// A chunk the drain still cannot finish leaves capture_deferred (no endless
// paid retries) and stays in capture_backlog for --backfill; a split
// message's part is drained by its fragment id.
func TestCaptureDeferredStillTruncatedLeavesQueueKeepsBacklog(t *testing.T) {
	f := scopetest.New(t)
	b := f.Bind(t, "alpha")
	l := &truncatingLLM{limit: 1 << 20}
	l.serve(t, f)
	o := truncatedOpts("ended-long")
	o.Messages[1].Text = "quartz-overflow " + strings.Repeat("Long deployment detail. ", 80)
	r, err := b.Capture(context.Background(), f.App, o)
	if err != nil || r.Truncated == 0 || r.Split != 1 {
		t.Fatalf("hook run=%+v err=%v", r, err)
	}
	var parts int
	f.App.DB.QueryRow(`SELECT count(*) FROM capture_deferred WHERE original='overflow'`).Scan(&parts)
	if parts == 0 || seenCount(t, f, "capture_backlog", "overflow") != 1 {
		t.Fatalf("deferred parts=%d, original not backlogged", parts)
	}
	read := func(string, string, string) (string, []harness.CaptureMessage, error) { return f.Cwd, o.Messages, nil }
	before := len(l.overflow)
	d, drained, err := f.App.DrainDeferred(context.Background(), read)
	if err != nil || !drained || d.Truncated != parts || len(l.overflow)-before != parts {
		t.Fatalf("drain=%+v calls=%d parts=%d err=%v", d, len(l.overflow)-before, parts, err)
	}
	var left int
	f.App.DB.QueryRow(`SELECT count(*) FROM capture_deferred`).Scan(&left)
	if left != 0 || seenCount(t, f, "capture_backlog", "overflow") != 1 || savedContents(t)[partialMemory] {
		t.Fatalf("still-truncated drain: deferred left=%d, backlog kept=%d", left, seenCount(t, f, "capture_backlog", "overflow"))
	}
	t.Logf("CAPTURE deferred still truncated: %d part(s) drained once each, queue empty, message kept in backlog, nothing partial applied", parts)
}
