---
id: qpp30y
title: "Answer box shows at once, with the only moving cursor on screen"
state: "doing"
type: bug
priority: ""
assignee: ["macaque"]
labels: []
group: ""
rank: "zzzzzzzzzzyh"
created: 2026-11-18
updated: 2026-11-18
doing_since: "2026-11-18T13:34"
stage: "waiting"
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
waiting_on: "crab review; check 3 with puppy's grid, check 4 Jørgen"
waiting_since: "2026-11-18T14:15"
waiting_asked_by: "macaque"
---

## Goal
Pressing a (answer) or r (request) draws the empty box under the question immediately, with a blinking underline cursor where the text goes; that cursor is the only motion on the otherwise calm screen.

## Purpose
Jørgen 2026-11-18: pressing a changed the footer to 'Enter send' but drew no box; it appeared only after the first keystroke, so nothing showed where to type.

## Principle
One consistent 'you are here' indicator: an underline-style cursor that blinks or pulses gently, shared with the pending view-key prefix (v then o/l/b/!). Nothing else on the screen moves or flashes.

## Relevant details
Cause: answerbox.go openBox() sets m.prompt, resets and focuses the textarea, but never calls m.syncDetail(), so the detail page is not re-rendered with the box; updateAnswer() calls syncDetail() on each key, which is why it appears after typing. Cursor is set static at answerbox.go:30 (cursor.CursorStatic), drawn as a block. Stale 'request qtj177' under the page is z82qsy.

## Depends on

## Acceptance
- [x] Pressing a or r draws the box with its label and cursor in the same frame, before any keystroke; a test fails first on main.
- [x] The cursor in the answer and request box is an underline that blinks (or pulses), and no other element on the screen animates (test of the cursor mode plus golden).
- [ ] The pending view-key prefix uses the same underline cursor style.
- [ ] Jørgen answers a question in hug tui and says the box and cursor read clearly.

## Notes


check 1: commit aee5b13e on branch qpp30y. openBox now calls syncDetail after Focus, so the box is drawn in the frame a or r is pressed. TestBoxShowsAtOnce (80 and 140, a and r) failed first at qpp30y_test.go:49 ('at 80 a drew no box before a key') and passes now.
check 2: theme.cursor = accent #00bfff/#006c9c, Underline+Bold (render.go), applied to the box through a Cursor.Style transform, because bubbles forces Reverse on the cell. Bytes on: '\x1b[0m\x1b[38;2;0;191;255;4m \x1b[0m', off: plain space (golden testdata/detail/answer-box.txt, 80 and 140). The cursor is CursorBlink at bubbles' 530ms. cursor.BlinkMsg goes to blinkBox, which updates the open box and redraws its page. A closed box returns no command. textinput is set to CursorStatic. TestBoxCursorBlinksUnderline failed first at qpp30y_test.go:70 ('box cursor mode static speed 530ms, want blink at 530ms') and passes now. Full suite: 34 ok, 0 FAIL ('ok github.com/veridit/hug/huginn/internal/tui 105.662s'). CI 44308766736 on ci/qpp30y.
CI run 44308766736 on ci/qpp30y (aee5b13e) finished completed success, all 6 jobs. Check 3 was left to puppy's grid by crab's ruling, and the reuse plan is in the report. Check 4 needs Jørgen. Report: /Users/jhf/.jcode/scratch/qpp30y/REPORT.md
## History
- 2026-11-18T13:34 todo -> doing/active (take)
- 2026-11-18T14:15 doing/active -> doing/waiting (wait)
