package task

import (
	"strings"
	"testing"
)

// 8zwy6r check 3: state set on both sides keeps the board's state whole.
// The local move's node fields (dropped_reason, dropped_at on a drop)
// belong to the state that lost; left in the front matter they lint
// broken [stray-node-field] on the board's doing task. They go to
// History, as a move puts them, and the local state is still in its note.
func TestMergeStateBothSidesMovesLosingNodeFields(t *testing.T) {
	ours := edit(mergeBase, `state: "todo"`, "state: \"dropped\"\ndropped_reason: \"not needed here\"\ndropped_at: \"2026-11-18T05:09\"")
	theirs := edit(mergeBase, `state: "todo"`, "state: \"doing\"\ndoing_since: \"2026-11-18T05:08\"\nstage: \"active\"")
	got, kept, e := MergeTask("8/8q6v7b.md", []byte(mergeBase), ours, theirs, side)
	if e != nil {
		t.Fatal(e)
	}
	if len(kept) != 1 || kept[0].Field != "state" {
		t.Fatalf("kept: %+v", kept)
	}
	d := Parse("8/8q6v7b.md", got)
	if d.Task.State != "doing" {
		t.Fatalf("state %q:\n%s", d.Task.State, got)
	}
	for _, f := range Lint([]*Document{d}) {
		if FindingCode(f.Problem) == "stray-node-field" {
			t.Fatalf("merged task lints %s: %s\n%s", f.Code, f.Problem, got)
		}
	}
	s := string(got)
	for _, w := range []string{"dropped_reason: not needed here", "dropped_at: 2026-11-18T05:09", "kestrel set state dropped here", `doing_since: "2026-11-18T05:08"`} {
		if !strings.Contains(s, w) {
			t.Fatalf("merged lacks %q:\n%s", w, s)
		}
	}
	if h := strings.Index(s, "## History"); strings.Index(s, "dropped_reason: not needed here") < h {
		t.Fatalf("dropped_reason is not in History:\n%s", s)
	}
}
