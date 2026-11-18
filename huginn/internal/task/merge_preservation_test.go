package task

import (
	"strings"
	"testing"
)

func TestMergeRemoteTaskDeletionStops(t *testing.T) {
	ours := edit(mergeBase, `state: "todo"`, `state: "doing"`)
	got, _, err := MergeTask("8/8q6v7b.md", []byte(mergeBase), ours, nil, side)
	if err == nil || !strings.Contains(err.Error(), "8q6v7b") || !strings.Contains(err.Error(), "deleted") || got != nil {
		t.Fatalf("remote deletion silently resurrected: err=%v\n%s", err, got)
	}
}

func TestMergeCompetingResolutionIsLabelledAndIdempotent(t *testing.T) {
	base := edit(mergeBase, "## Notes\n", "## Requests\n- 2026-11-18T05:20 request 1 by jhf:\n  B's request\n\n## Notes\n")
	ours := edit(string(base), "  B's request\n", "  B's request\n  resolved 2026-11-18T05:30 by kestrel:\n    B's outcome\n    exact second line\n")
	theirs := edit(string(base), "request 1 by jhf", "request 2 by jhf", "  B's request\n", "  B's request\n  resolved 2026-11-18T05:31 by crab:\n    A's outcome\n")
	got, kept, err := MergeTask("8/8q6v7b.md", base, ours, theirs, side)
	if err != nil || len(kept) != 1 || !strings.Contains(kept[0].Field, "competing resolution") {
		t.Fatalf("unlabelled competing resolution: err=%v kept=%+v\n%s", err, kept, got)
	}
	for _, want := range []string{"Competing request resolution", "identity", "request 2", "resolved 2026-11-18T05:30 by kestrel:", "    B's outcome\n>     exact second line"} {
		if !strings.Contains(string(got), want) {
			t.Fatalf("missing %q:\n%s", want, got)
		}
	}
	reqs := Parse("8/8q6v7b.md", got).Task.Requests
	if len(reqs) != 1 || reqs[0].N != 2 || reqs[0].Resolved == nil || reqs[0].Resolved.Text != "A's outcome" {
		t.Fatalf("canonical remote resolution changed: %+v", reqs)
	}
	again, _, err := MergeTask("8/8q6v7b.md", base, ours, got, side)
	if err != nil || string(again) != string(got) {
		t.Fatalf("repeated merge changed retained resolution: %v\n%s\nfirst:\n%s", err, again, got)
	}
}

func TestMergeAdjacentUntimedAppendAlreadyPresentOnce(t *testing.T) {
	base := edit(mergeBase, "2026-11-18T05:00 first\n", "seed\n")
	ours := edit(string(base), "seed\n", "seed\nshared adjacent note\n")
	theirs := edit(string(base), "seed\n", "seed\nshared adjacent note\nremote next\n")
	got, kept, err := MergeTask("8/8q6v7b.md", base, ours, theirs, side)
	if err != nil || len(kept) != 0 || strings.Count(string(got), "shared adjacent note") != 1 || string(got) != string(theirs) {
		t.Fatalf("adjacent append duplicated: err=%v kept=%+v\n%s", err, kept, got)
	}
}
