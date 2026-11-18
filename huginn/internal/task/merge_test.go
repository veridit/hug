package task

import (
	"strings"
	"testing"
)

// w40cmk check 9, crab 04:44: the catch-up's per-file merge. Append-only
// parts are a union, a value changed on both sides keeps the board's and
// puts ours in a note, only an unreadable file stops.

const mergeBase = `---
id: 8q6v7b
title: "Noted twice"
state: "todo"
rank: "h"
group: ""
created: 2026-11-18
updated: 2026-11-18
---

## Goal
G

## Acceptance
- [ ] one
- [ ] two

## Notes

2026-11-18T05:00 first

## History
- 2026-11-18T05:00 created
`

func edit(s string, pairs ...string) []byte {
	for i := 0; i+1 < len(pairs); i += 2 {
		if !strings.Contains(s, pairs[i]) {
			panic("no " + pairs[i])
		}
		s = strings.Replace(s, pairs[i], pairs[i+1], 1)
	}
	return []byte(s)
}

var side = MergeSide{Who: "kestrel", At: "2026-11-18T05:10", Commit: "3f2a1c9", Subject: "hug task note 8q6v7b"}

func TestMergeUnionsNotesAndHistoryInTimeOrder(t *testing.T) {
	ours := edit(mergeBase, "2026-11-18T05:00 first\n", "2026-11-18T05:00 first\n\n2026-11-18T05:20 from B\n",
		"- 2026-11-18T05:00 created\n", "- 2026-11-18T05:00 created\n- 2026-11-18T05:20 B line\n")
	theirs := edit(mergeBase, "2026-11-18T05:00 first\n", "2026-11-18T05:00 first\n\n2026-11-18T05:10 from A\n\n2026-11-18T05:30 later A\n",
		"- 2026-11-18T05:00 created\n", "- 2026-11-18T05:00 created\n- 2026-11-18T05:10 A line\n")
	got, kept, e := MergeTask("8/8q6v7b.md", []byte(mergeBase), ours, theirs, side)
	if e != nil || len(kept) != 0 {
		t.Fatalf("e=%v kept=%v", e, kept)
	}
	want := edit(mergeBase, "2026-11-18T05:00 first\n", "2026-11-18T05:00 first\n\n2026-11-18T05:10 from A\n\n2026-11-18T05:20 from B\n\n2026-11-18T05:30 later A\n",
		"- 2026-11-18T05:00 created\n", "- 2026-11-18T05:00 created\n- 2026-11-18T05:10 A line\n- 2026-11-18T05:20 B line\n")
	if string(got) != string(want) {
		t.Fatalf("merged:\n%s\nwant:\n%s", got, want)
	}
}

func TestMergeNotesWithoutTimeKeepBoardFirst(t *testing.T) {
	ours := edit(mergeBase, "2026-11-18T05:00 first\n", "2026-11-18T05:00 first\n\nnote from B\n")
	theirs := edit(mergeBase, "2026-11-18T05:00 first\n", "2026-11-18T05:00 first\n\nnote from A\n")
	got, kept, e := MergeTask("8/8q6v7b.md", []byte(mergeBase), ours, theirs, side)
	if e != nil || len(kept) != 0 {
		t.Fatalf("e=%v kept=%v", e, kept)
	}
	want := edit(mergeBase, "2026-11-18T05:00 first\n", "2026-11-18T05:00 first\n\nnote from A\n\nnote from B\n")
	if string(got) != string(want) {
		t.Fatalf("merged:\n%s\nwant:\n%s", got, want)
	}
}

func TestMergeRepeatedNotesKeepUnion(t *testing.T) {
	first := edit(mergeBase, "2026-11-18T05:00 first\n", "2026-11-18T05:00 first\n\n2026-11-18T05:20 B first\n")
	second := edit(string(first), "2026-11-18T05:20 B first\n", "2026-11-18T05:20 B first\n\n2026-11-18T05:30 B second\n")
	remote := edit(mergeBase, "2026-11-18T05:00 first\n", "2026-11-18T05:00 first\n\n2026-11-18T05:10 A note\n")
	merged, _, e := MergeTask("8/8q6v7b.md", []byte(mergeBase), first, remote, side)
	if e != nil {
		t.Fatal(e)
	}
	got, kept, e := MergeTask("8/8q6v7b.md", first, second, merged, side)
	if e != nil || len(kept) != 0 {
		t.Fatalf("e=%v kept=%+v\n%s", e, kept, got)
	}
	want := edit(mergeBase, "2026-11-18T05:00 first\n", "2026-11-18T05:00 first\n\n2026-11-18T05:10 A note\n\n2026-11-18T05:20 B first\n\n2026-11-18T05:30 B second\n")
	if string(got) != string(want) {
		t.Fatalf("repeated note replay:\n%s\nwant:\n%s", got, want)
	}
}

func TestMergeRepeatedDocumentNotesKeepUnion(t *testing.T) {
	note := func(raw []byte, text string) []byte {
		d := Parse("8/8q6v7b.md", raw)
		d.Note(text)
		return d.Bytes()
	}
	base := []byte(mergeBase)
	first := note(base, "B first")
	second := note(first, "B second")
	remote := note(base, "A note")
	merged, _, e := MergeTask("8/8q6v7b.md", base, first, remote, side)
	if e != nil {
		t.Fatal(e)
	}
	got, kept, e := MergeTask("8/8q6v7b.md", first, second, merged, side)
	if e != nil || len(kept) != 0 {
		t.Fatalf("e=%v kept=%+v\n%s", e, kept, got)
	}
	for _, text := range []string{"2026-11-18T05:00 first", "A note", "B first", "B second"} {
		if n := strings.Count(string(got), text); n != 1 {
			t.Fatalf("%q appears %d times:\n%s", text, n, got)
		}
	}
}

func TestMergeDeletionAndChangeUsesBoardAndNotesLocal(t *testing.T) {
	ours := edit(mergeBase, `state: "todo"`, `state: "doing"`)
	theirs := edit(mergeBase, "state: \"todo\"\n", "")
	got, kept, e := MergeTask("8/8q6v7b.md", []byte(mergeBase), ours, theirs, side)
	if e != nil || len(kept) != 1 || kept[0].Field != "state" || strings.Contains(string(got), "state: ") {
		t.Fatalf("e=%v kept=%+v\n%s", e, kept, got)
	}
}

func TestMergeNewFieldAfterDeletedAnchorIsKept(t *testing.T) {
	ours := edit(mergeBase, "state: \"todo\"\n", "state: \"todo\"\nmodel: \"local\"\n")
	theirs := edit(mergeBase, "state: \"todo\"\n", "")
	got, _, e := MergeTask("8/8q6v7b.md", []byte(mergeBase), ours, theirs, side)
	if e != nil || !strings.Contains(string(got), "model: \"local\"\n") {
		t.Fatalf("e=%v\n%s", e, got)
	}
}

func TestMergeDifferentChecksBothTicked(t *testing.T) {
	ours := edit(mergeBase, "- [ ] two", "- [x] two")
	theirs := edit(mergeBase, "- [ ] one", "- [x] one", "- [ ] two\n", "- [ ] two\n- [ ] three\n")
	got, kept, e := MergeTask("8/8q6v7b.md", []byte(mergeBase), ours, theirs, side)
	if e != nil || len(kept) != 0 {
		t.Fatalf("e=%v kept=%v", e, kept)
	}
	want := edit(mergeBase, "- [ ] one\n- [ ] two\n", "- [x] one\n- [x] two\n- [ ] three\n")
	if string(got) != string(want) {
		t.Fatalf("merged:\n%s\nwant:\n%s", got, want)
	}
}

func TestMergeBothChangedKeepsBoardAndNotesOurs(t *testing.T) {
	ours := edit(mergeBase, `state: "todo"`, `state: "doing"`, `rank: "h"`, `rank: "c"`, "updated: 2026-11-18", "updated: 2026-11-19")
	theirs := edit(mergeBase, `state: "todo"`, `state: "done"`, "## Goal\nG\n", "## Goal\nG from A\n")
	got, kept, e := MergeTask("8/8q6v7b.md", []byte(mergeBase), ours, theirs, side)
	if e != nil {
		t.Fatal(e)
	}
	if len(kept) != 1 || kept[0] != (Kept{ID: "8q6v7b", Field: "state", Ours: "doing", Theirs: "done"}) {
		t.Fatalf("kept: %+v", kept)
	}
	s := string(got)
	for _, w := range []string{
		`state: "done"`, `rank: "c"`, "updated: 2026-11-19", "## Goal\nG from A\n",
		"kestrel set state doing here at 2026-11-18T05:10 (commit 3f2a1c9 \"hug task note 8q6v7b\"); the board had done; kept done\n",
	} {
		if !strings.Contains(s, w) {
			t.Fatalf("merged lacks %q:\n%s", w, s)
		}
	}
	// The note sits in Notes, before History.
	if strings.Index(s, "kept done") > strings.Index(s, "## History") {
		t.Fatalf("note not in Notes:\n%s", s)
	}
}

func TestMergeSectionTextBothChangedKeptAsQuote(t *testing.T) {
	ours := edit(mergeBase, "## Goal\nG\n", "## Goal\nG from B\nline two\n")
	theirs := edit(mergeBase, "## Goal\nG\n", "## Goal\nG from A\n")
	got, kept, e := MergeTask("8/8q6v7b.md", []byte(mergeBase), ours, theirs, side)
	if e != nil || len(kept) != 1 || kept[0].Field != "## Goal" {
		t.Fatalf("e=%v kept=%+v", e, kept)
	}
	s := string(got)
	if !strings.Contains(s, "## Goal\nG from A\n") || !strings.Contains(s, "kestrel set ## Goal here at 2026-11-18T05:10 (commit 3f2a1c9 \"hug task note 8q6v7b\"); the board had changed it too; kept the board's. This clone's text was:\n> G from B\n> line two\n") {
		t.Fatalf("merged:\n%s", s)
	}
}

func TestMergeRequestsRenumbersOursAfterBoards(t *testing.T) {
	base := edit(mergeBase, "## Notes\n", "## Requests\n- 2026-11-18T04:00 request 1 by jhf:\n  old\n\n## Notes\n")
	ours := edit(string(base), "  old\n", "  old\n- 2026-11-18T05:20 request 2 by jhf:\n  from B\n")
	theirs := edit(string(base), "  old\n", "  old\n  resolved 2026-11-18T05:05 by crab:\n    done\n- 2026-11-18T05:10 request 2 by sam:\n  from A\n")
	got, kept, e := MergeTask("8/8q6v7b.md", base, ours, theirs, side)
	if e != nil || len(kept) != 0 {
		t.Fatalf("e=%v kept=%+v", e, kept)
	}
	want := edit(string(base), "  old\n", "  old\n  resolved 2026-11-18T05:05 by crab:\n    done\n- 2026-11-18T05:10 request 2 by sam:\n  from A\n- 2026-11-18T05:20 request 3 by jhf:\n  from B\n")
	if string(got) != string(want) {
		t.Fatalf("merged:\n%s\nwant:\n%s", got, want)
	}
}

func TestMergeResolutionFollowsRenumberedRequest(t *testing.T) {
	base := []byte(mergeBase)
	ours := edit(mergeBase, "## Notes\n", "## Requests\n- 2026-11-18T05:20 request 1 by jhf:\n  B's request\n\n## Notes\n")
	theirs := edit(mergeBase, "## Notes\n", "## Requests\n- 2026-11-18T05:10 request 1 by sam:\n  A's request\n\n## Notes\n")
	merged, _, e := MergeTask("8/8q6v7b.md", base, ours, theirs, side)
	if e != nil {
		t.Fatal(e)
	}
	resolved := edit(string(ours), "  B's request\n", "  B's request\n  resolved 2026-11-18T05:30 by kestrel:\n    B's outcome\n")
	got, kept, e := MergeTask("8/8q6v7b.md", ours, resolved, merged, side)
	if e != nil || len(kept) != 0 {
		t.Fatalf("e=%v kept=%+v\n%s", e, kept, got)
	}
	reqs := Parse("8/8q6v7b.md", got).Task.Requests
	if len(reqs) != 2 || reqs[0].N != 1 || reqs[0].Resolved != nil || reqs[1].N != 2 || reqs[1].Text != "B's request" || reqs[1].Resolved == nil || reqs[1].Resolved.Text != "B's outcome" {
		t.Fatalf("resolution followed a number rather than its request: %+v\n%s", reqs, got)
	}
}

func TestMergeOneSideOnlyIsThatSide(t *testing.T) {
	ours := edit(mergeBase, `state: "todo"`, `state: "doing"`, "updated: 2026-11-18", "updated: 2026-11-19\nmodel: \"x\"")
	got, kept, e := MergeTask("8/8q6v7b.md", []byte(mergeBase), ours, []byte(mergeBase), side)
	if e != nil || len(kept) != 0 || string(got) != string(ours) {
		t.Fatalf("e=%v kept=%v\n%s", e, kept, got)
	}
	got, _, _ = MergeTask("8/8q6v7b.md", []byte(mergeBase), []byte(mergeBase), ours, side)
	if string(got) != string(ours) {
		t.Fatalf("board side:\n%s", got)
	}
}

func TestMergeUnreadableFileStops(t *testing.T) {
	// Both sides changed it and one cannot be read: the only stop.
	theirs := edit(mergeBase, "G\n", "G from A\n")
	_, _, e := MergeTask("8/8q6v7b.md", []byte(mergeBase), []byte("no front matter\n"), theirs, side)
	if e == nil || !strings.Contains(e.Error(), "hug cannot read this file") {
		t.Fatalf("e=%v", e)
	}
	// A fast path must not introduce an unreadable task either.
	_, _, e = MergeTask("8/8q6v7b.md", []byte(mergeBase), []byte("no front matter\n"), []byte(mergeBase), side)
	if e == nil || !strings.Contains(e.Error(), "hug cannot read this file") {
		t.Fatalf("unreadable single-sided edit: e=%v", e)
	}
}
