package gitbranch

import (
	"testing"

	"github.com/veridit/hug/internal/flock"
)

// TurnHeld is a try of the board turn (46ek1f): true while a writer holds
// it, false once released, and it never takes the turn itself.
func TestTurnHeld(t *testing.T) {
	b := BackgroundAt(t.TempDir(), "hug/tasks")
	if b.TurnHeld() {
		t.Fatal("held before any writer took it")
	}
	release, err := b.Turn()
	if err != nil {
		t.Fatal(err)
	}
	if !b.TurnHeld() {
		t.Fatal("not held while a writer holds it")
	}
	release()
	if b.TurnHeld() {
		t.Fatal("still held after the release")
	}
	// Asking left the turn free: a writer gets it at once.
	again, ok, err := flock.TryAcquire(b.base + ".turn")
	if err != nil || !ok {
		t.Fatalf("the turn is not free after TurnHeld: ok %v, %v", ok, err)
	}
	again()
}
