---
id: qpp30y
title: "Answer box shows at once, with the only moving cursor on screen"
state: "doing"
type: bug
priority: ""
assignee: ["crab"]
labels: []
group: ""
rank: "zzzzzzzzzzt1"
created: 2026-11-18
updated: 2026-11-18
doing_since: "2026-11-18T13:34"
stage: "asked"
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
asked_what: "Answer a question in hug tui: does the answer box appear at once, and do the box and its cursor read clearly?"
asked_for: "jhf"
asked_since: "2026-11-20T09:26"
asked_by: "crab"
asked_check: "4"
---

## Goal
Pressing a or r draws the answer box at once, with a blinking cursor as the only motion on screen.

## Purpose
Jørgen 2026-11-18: pressing a changed the footer to 'Enter send' but drew no box; it appeared only after the first keystroke, so nothing showed where to type.

## Principle
One consistent 'you are here' indicator: an underline-style cursor that blinks or pulses gently, shared with the pending view-key prefix (v then o/l/b/!). Nothing else on the screen moves or flashes.

## Relevant details
Pressing a (answer) or r (request) draws the empty box under the question immediately, with a blinking underline cursor where the text goes; that cursor is the only motion on the otherwise calm screen.

Cause: answerbox.go openBox() sets m.prompt, resets and focuses the textarea, but never calls m.syncDetail(), so the detail page is not re-rendered with the box; updateAnswer() calls syncDetail() on each key, which is why it appears after typing. Cursor is set static at answerbox.go:30 (cursor.CursorStatic), drawn as a block. Stale 'request qtj177' under the page is z82qsy.

## Depends on

## Acceptance
- [x] Pressing a or r draws the box with its label and cursor in the same frame, before any keystroke; a test fails first on main.
- [x] The cursor in the answer and request box is an underline that blinks (or pulses), and no other element on the screen animates (test of the cursor mode plus golden).
- [x] The pending view-key prefix uses the same underline cursor style.
- [ ] Jørgen answers a question in hug tui and says the box and cursor read clearly.

## Questions
- 2026-11-20T09:26 question 1 by crab to jhf (check 4):
  Answer a question in hug tui: does the answer box appear at once, and do the box and its cursor read clearly?

## Notes


check 1: commit aee5b13e on branch qpp30y. openBox now calls syncDetail after Focus, so the box is drawn in the frame a or r is pressed. TestBoxShowsAtOnce (80 and 140, a and r) failed first at qpp30y_test.go:49 ('at 80 a drew no box before a key') and passes now.
check 2: theme.cursor = accent #00bfff/#006c9c, Underline+Bold (render.go), applied to the box through a Cursor.Style transform, because bubbles forces Reverse on the cell. Bytes on: '\x1b[0m\x1b[38;2;0;191;255;4m \x1b[0m', off: plain space (golden testdata/detail/answer-box.txt, 80 and 140). The cursor is CursorBlink at bubbles' 530ms. cursor.BlinkMsg goes to blinkBox, which updates the open box and redraws its page. A closed box returns no command. textinput is set to CursorStatic. TestBoxCursorBlinksUnderline failed first at qpp30y_test.go:70 ('box cursor mode static speed 530ms, want blink at 530ms') and passes now. Full suite: 34 ok, 0 FAIL ('ok github.com/veridit/hug/huginn/internal/tui 105.662s'). CI 44308766736 on ci/qpp30y.
CI run 44308766736 on ci/qpp30y (aee5b13e) finished completed success, all 6 jobs. Check 3 was left to puppy's grid by crab's ruling, and the reuse plan is in the report. Check 4 needs Jørgen. Report: /Users/jhf/.jcode/scratch/qpp30y/REPORT.md
crab review 13:16Z: answerbox.go diff read (syncDetail in openBox, CursorBlink, Transform redraws the cell in theme.cursor over bubbles' reverse, blinks only while the box is open). Exact SHA aee5b13e CI 44308766736 6/6. Bold dropped on an empty cell is fine: underline plus accent still reads as the cursor. Lands after z82qsy, gpyqq9, bg1rd8 (rebase plus exact CI). Check 3 waits on 911j70 (Jørgen picks the grid), check 4 on Jørgen after release.
Landed on main via 5a8b14da (stack bg1rd8 f7d607e4, qpp30y 3d74229a, 2k68eg 7fafab16, vk9td0 5a8b14da; CI 44328375780 exact SHA 6/6, ubuntu passed on rerun after an unrelated muninn/internal/project flake, see ybbfsa). Ships in the next release.
landed 3d74229 on main 2026-11-18: TUI answer/request box: drawn at once, blinking underline cursor (qpp30y)
shipped in v0.7.65 (3d74229)
check 3: 911j70 check 3 (macaque 89e280c9): after v, the slot after 'view:' shows theme.cursor (the qpp30y style, #00bfff underline) blinking at 530 ms via bubbles cursor.Model; golden testdata/header/view-cursor.txt 'after v' shows ESC[38;2;0;191;255;4m. Lands with 911j70 check 3.
crab 20:20 local: check 3 was ticked on 89e280c9; that lands as 897d4810 + dd1830f0 (qh9ep1 check 9, quieter cues). Same cursor behaviour.
## History
- 2026-11-18T13:34 todo -> doing/active (take)
- 2026-11-18T14:15 doing/active -> doing/waiting (wait)
- 2026-11-19T04:01 handed to crab: macaque's session ended (closed 2026-11-19 02:14)
- 2026-11-20T09:26 doing/waiting -> doing/asked (ask); was waiting_asked_by: macaque; waiting_on: crab review; check 3 with puppy's grid, check 4 Jørgen; waiting_since: 2026-11-18T14:15
