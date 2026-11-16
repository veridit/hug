package app_test

import (
	"context"
	"testing"
	"time"

	"github.com/veridit/hug/muninn/internal/scopetest"
	"github.com/veridit/hug/muninn/internal/store"
)

// m8shdj check 3: capture's database writes take the write turn. When
// another process takes it while the model answers (a second handle on
// the same database here), capture's save and checkpoint wait, and land
// once the turn is free. The turn is never held across the model call.
func TestCaptureWritesWaitForTheWriteTurn(t *testing.T) {
	f := scopetest.New(t)
	b := f.Bind(t, "alpha")
	m := fakeCapture(t, f, line("fact", "high", "Synthetic turn-taking fact about basalt.", ""))
	other, err := store.Open(f.App.DB.Path)
	if err != nil {
		t.Fatal(err)
	}
	defer other.Close()
	// While the model answers, another process takes the write turn.
	held := make(chan func(), 1)
	m.beforeReply.Store(func() {
		_, release, err := other.WriteTurn(context.Background())
		if err != nil {
			panic(err)
		}
		held <- release
	})
	done := make(chan error, 1)
	go func() {
		_, err := b.Capture(context.Background(), f.App, captureOpts("turn"))
		done <- err
	}()
	var release func()
	select {
	case release = <-held:
	case err := <-done:
		t.Fatalf("capture finished before the model was asked: %v", err)
	case <-time.After(20 * time.Second):
		t.Fatal("capture never reached the model")
	}
	// Nothing is checkpointed until the turn is free.
	time.Sleep(500 * time.Millisecond)
	select {
	case err := <-done:
		t.Fatalf("capture finished while another process held the write turn: %v", err)
	default:
	}
	var n int
	if err := f.App.DB.QueryRow(`SELECT count(*) FROM capture_seen`).Scan(&n); err != nil || n != 0 {
		t.Fatalf("capture_seen has %d rows while the turn is held elsewhere (%v)", n, err)
	}
	release()
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(20 * time.Second):
		t.Fatal("capture did not finish after the turn was released")
	}
	if err := f.App.DB.QueryRow(`SELECT count(*) FROM capture_seen`).Scan(&n); err != nil || n != 1 {
		t.Fatalf("capture_seen has %d rows after capture, want 1 (%v)", n, err)
	}
}
