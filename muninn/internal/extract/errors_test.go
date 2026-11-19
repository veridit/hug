package extract

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestFailureClassNeverIncludesProviderDetails(t *testing.T) {
	for _, tc := range []struct {
		err  error
		want string
	}{
		{context.DeadlineExceeded, "timeout"},
		{fmt.Errorf("extraction timeout or cancellation: %w", context.DeadlineExceeded), "timeout"},
		{errors.New("extraction HTTP status 429 (body redacted)"), "provider_http_429"},
		{errors.New("extraction credential is not configured"), "no_key"},
		{errors.New("extraction HTTP request failed (details redacted)"), "provider_network"},
		{errors.New("invalid extraction response (body redacted)"), "parse"},
		{errors.New("capture session binding changed, start a new session"), "scope"},
		{errors.New("constraint failed: UNIQUE constraint failed: neighbors.unit_id, neighbors.nbr_id (1555)"), "index"},
		{errors.New("index: synthetic storage failure"), "index"},
		{errors.New("context deadline exceeded"), "timeout"},
		{errors.New("context canceled"), "timeout"},
		{errors.New("unexpected synthetic private text"), "internal"},
	} {
		if got := FailureClass(tc.err); got != tc.want {
			t.Fatalf("class=%s want=%s", got, tc.want)
		}
	}
}

// a6xm8n: a deadline that cuts the reply body mid-read is a timeout, never a
// parse error (capture_runs 8468: 'invalid extraction response', parse).
func TestDeadlineMidBodyIsTimeout(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"choices":[{"message":{"content":"`))
		w.(http.Flusher).Flush()
		<-r.Context().Done()
	}))
	defer s.Close()
	c := Defaults()
	c.Provider = "ollama"
	c.BaseURL = s.URL
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	_, err := Run(ctx, c, "user: hello", nil)
	if !errors.Is(err, context.DeadlineExceeded) || FailureClass(err) != "timeout" {
		t.Fatalf("err=%v class=%s", err, FailureClass(err))
	}
}
