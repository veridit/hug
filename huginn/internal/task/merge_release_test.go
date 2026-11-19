package task

import (
	"strings"
	"testing"
)

// j0zvee (CI 44412479085, iwazaru's niced run): two clones deduce the same
// release tag offline and each closes the task by it, each with its own
// clock. The History lines then differ in their minute only, and the merge
// kept both: "2 release lines". One transition by one release happened
// once, whatever minute each clone stamped.

const releaseBase = `---
id: g47zmg
title: "Shared"
state: "doing"
stage: "waiting"
rank: "h"
group: ""
created: 2026-11-18
updated: 2026-11-18
---

## Acceptance
- [x] C

## Notes

## History
- 2026-11-18T23:41 todo -> doing/active (take)
- 2026-11-18T23:42 doing/active -> doing/waiting (wait)
`

// closedAt is the base closed by release v0.1.1 at minute at, as
// closeShipped writes it.
func closedAt(at string) []byte {
	return edit(releaseBase,
		"state: \"doing\"\nstage: \"waiting\"\n", "state: \"done\"\ndone_at: \""+at+"\"\ndone_release: \"v0.1.1\"\n",
		"- 2026-11-18T23:42 doing/active -> doing/waiting (wait)\n",
		"- 2026-11-18T23:42 doing/active -> doing/waiting (wait)\n- "+at+" doing/waiting -> done ("+ReleaseLine("v0.1.1")+"); was waiting_since: 2026-11-18T23:42\n")
}

func TestMergeSameReleaseCloseInAnotherMinuteCountsOnce(t *testing.T) {
	theirs := closedAt("2026-11-18T23:43") // the board: the clone that pushed first
	ours := closedAt("2026-11-18T23:44")   // this clone, a minute later
	got, _, e := MergeTask("g/g47zmg.md", []byte(releaseBase), ours, theirs, side)
	if e != nil {
		t.Fatal(e)
	}
	if n := strings.Count(string(got), "by release v0.1.1"); n != 1 {
		t.Fatalf("%d release lines, want 1:\n%s", n, got)
	}
	if !strings.Contains(string(got), "- 2026-11-18T23:43 doing/waiting -> done (") {
		t.Fatalf("the board's line (23:43) is not the one kept:\n%s", got)
	}
}

// What counts once is the same transition by the same release: a close by
// another release, or another transition in the same minutes, stays.
func TestMergeDifferentTransitionsInHistoryAllStay(t *testing.T) {
	theirs := closedAt("2026-11-18T23:43")
	ours := edit(string(closedAt("2026-11-18T23:44")), "by release v0.1.1", "by release v0.1.2")
	got, _, e := MergeTask("g/g47zmg.md", []byte(releaseBase), ours, theirs, side)
	if e != nil {
		t.Fatal(e)
	}
	for _, want := range []string{"by release v0.1.1", "by release v0.1.2"} {
		if strings.Count(string(got), want) != 1 {
			t.Fatalf("want one %q:\n%s", want, got)
		}
	}
	// Two takes a minute apart on two clones are two events: both stay.
	theirs = edit(releaseBase, "- 2026-11-18T23:42 doing/active -> doing/waiting (wait)\n", "- 2026-11-18T23:42 doing/active -> doing/waiting (wait)\n- 2026-11-18T23:50 doing/waiting -> doing/active (resume)\n")
	ours = edit(releaseBase, "- 2026-11-18T23:42 doing/active -> doing/waiting (wait)\n", "- 2026-11-18T23:42 doing/active -> doing/waiting (wait)\n- 2026-11-18T23:51 doing/waiting -> doing/active (resume)\n")
	got, _, e = MergeTask("g/g47zmg.md", []byte(releaseBase), ours, theirs, side)
	if e != nil {
		t.Fatal(e)
	}
	if strings.Count(string(got), "(resume)") != 2 {
		t.Fatalf("two resumes on two clones, want both:\n%s", got)
	}
}
