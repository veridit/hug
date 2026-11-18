---
id: qpp30y
title: "Answer box shows at once, with the only moving cursor on screen"
state: todo
type: bug
priority: ""
assignee: []
labels: []
group: ""
rank: "zzzzzzzzzzzz"
created: 2026-11-18
updated: 2026-11-18
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
- [ ] Pressing a or r draws the box with its label and cursor in the same frame, before any keystroke; a test fails first on main.
- [ ] The cursor in the answer and request box is an underline that blinks (or pulses), and no other element on the screen animates (test of the cursor mode plus golden).
- [ ] The pending view-key prefix uses the same underline cursor style.
- [ ] Jørgen answers a question in hug tui and says the box and cursor read clearly.

## Notes
