package app_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/veridit/hug/internal/timing"
	"github.com/veridit/hug/muninn/internal/extract"
	"github.com/veridit/hug/muninn/internal/scopetest"
)

// a6xm8n check 1: a hook whose remaining budget cannot cover one call does
// not start it. Runs 8467 and 8468 started their last call with too little
// left; it was cut and the whole run, 83 and 88 actions in, was recorded as
// failed ('context deadline exceeded', 'invalid extraction response').
func TestCaptureHookDefersWhenBudgetCannotCoverCall(t *testing.T) {
	timing.Budget(t)
	f := scopetest.New(t)
	b := f.Bind(t, "alpha")
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		// The extractor takes 5 s, more than the hook has left.
		select {
		case <-r.Context().Done():
		case <-time.After(5 * time.Second):
		}
	}))
	defer server.Close()
	f.App.Cfg.Extract = extract.Defaults()
	f.App.Cfg.Extract.Provider = "ollama"
	f.App.Cfg.Extract.BaseURL = server.URL
	o := captureOpts("budget-short")
	o.Trigger = "turn_end"
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()
	start := time.Now()
	r, err := b.Capture(ctx, f.App, o)
	if err != nil || r.Skipped != "remaining messages deferred to next hook (budget)" || r.Processed != 0 {
		t.Fatalf("report=%+v err=%v", r, err)
	}
	if n := calls.Load(); n != 0 {
		t.Fatalf("started %d calls the budget could not cover", n)
	}
	if elapsed := time.Since(start); elapsed > 2*time.Second {
		t.Fatalf("waited %v instead of deferring", elapsed)
	}
	if n := seenCount(t, f, "capture_seen", "one"); n != 0 {
		t.Fatalf("deferred message checkpointed %d", n)
	}
	runs, err := f.App.AuditRuns(o.Session, time.Now().Add(-time.Hour))
	if err != nil || len(runs) != 1 || runs[0].Error != "" || runs[0].ErrorClass != "" {
		t.Fatalf("audit=%+v err=%v", runs, err)
	}
	// The next hook, with its full budget, captures the message.
	m := fakeCapture(t, f, line("fact", "high", "Quartz deferred hook memory stays durable.", ""))
	r, err = b.Capture(context.Background(), f.App, o)
	if err != nil || r.Processed != 1 || m.calls.Load() != 1 {
		t.Fatalf("next hook=%+v err=%v", r, err)
	}
	t.Log("CAPTURE hook budget: 4 s left, 5 s extractor: no call started, skipped (budget), no error; the next hook captures it")
}

// a6xm8n: run 8468's last call got its headers in 417 ms and was cut by the
// run's deadline while its body was read. The run was recorded as
// 'invalid extraction response (body redacted)', error_class parse.
func TestCaptureBudgetCutMidBodyIsTimeoutNotParse(t *testing.T) {
	f := scopetest.New(t)
	b := f.Bind(t, "alpha")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"choices":[{"message":{"content":"`))
		w.(http.Flusher).Flush()
		<-r.Context().Done()
	}))
	defer server.Close()
	f.App.Cfg.Extract = extract.Defaults()
	f.App.Cfg.Extract.Provider = "ollama"
	f.App.Cfg.Extract.BaseURL = server.URL
	o := captureOpts("budget-mid-body")
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	_, err := b.Capture(ctx, f.App, o)
	if err == nil {
		t.Fatal("a cut call returned no error")
	}
	runs, aerr := f.App.AuditRuns(o.Session, time.Now().Add(-time.Hour))
	if aerr != nil || len(runs) != 1 || runs[0].ErrorClass != "timeout" {
		t.Fatalf("err=%v audit=%+v %v", err, runs, aerr)
	}
	t.Logf("CAPTURE budget cut mid-body: %q, error_class %s", runs[0].Error, runs[0].ErrorClass)
}
