package cli

import (
	"strings"
	"testing"
)

// sfa8w4 checks 1-3 through the CLI: strike writes the decision, show
// prints [-] with it, --jsonl carries the state, done accepts ticked or
// struck, --undo restores.
func TestStrikeThroughCLI(t *testing.T) {
	today := pinClock(t, lastTick) // the strike's date is the board clock's (jsyksd)
	a := app(t)
	execute(t, a, "init")
	id := execute(t, a, "new", "Strike CLI", "--goal", "G", "--purpose", "P", "--principle", "Q", "--accept", "holds", "--accept", "no longer applies")["task"].(Reply)["id"].(string)
	if code, out := runText(a, "strike", id, "2"); code != 2 || !strings.Contains(out, "needs a reason") || !strings.HasSuffix(out, "fix: hug task strike "+id+" 2 --reason \"why it no longer applies\"\n") {
		t.Fatalf("no reason: %d\n%s", code, out)
	}
	execute(t, a, "strike", id, "2", "--reason", "the API changed", "--as", "kestrel")
	if f := taskFile(t, a, id); !strings.Contains(f, "\n- [ ] ~~no longer applies~~ (not relevant: the API changed, kestrel, "+today+")\n") {
		t.Fatalf("file:\n%s", f)
	}
	if _, out := runText(a, "show", id); !strings.Contains(out, "2. [-] ~~no longer applies~~ (not relevant: the API changed, kestrel, "+today+")\n") || !strings.Contains(out, "1. [ ] holds\n") {
		t.Fatalf("show:\n%s", out)
	}
	if _, out := runText(a, "ls", "--jsonl"); !strings.Contains(out, `"state":"struck"`) || !strings.Contains(out, `"reason":"the API changed"`) || !strings.Contains(out, `"state":"open"`) {
		t.Fatalf("ls --jsonl:\n%s", out)
	}
	execute(t, a, "take", id)
	if code, out := runText(a, "done", id); code != 3 || !strings.Contains(out, "1. holds") || strings.Contains(out, "2. no longer") || !strings.HasSuffix(out, "fix: hug task check "+id+" 1\n") {
		t.Fatalf("done with 1 open: %d\n%s", code, out)
	}
	execute(t, a, "check", id, "1")
	execute(t, a, "done", id)
	execute(t, a, "strike", id, "2", "--undo")
	if f := taskFile(t, a, id); !strings.Contains(f, "\n- [ ] no longer applies\n") {
		t.Fatalf("undo:\n%s", f)
	}
	// Now done with an open check: lint judges open tasks only (t01kqm).
	if _, out := runText(a, "lint", id); out != id+" is done: lint judges open tasks only\nnext: hug task show "+id+"\n" {
		t.Fatalf("lint:\n%s", out)
	}
}
