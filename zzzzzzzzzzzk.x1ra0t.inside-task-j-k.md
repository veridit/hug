---
id: x1ra0t
title: "Inside a task, J and K never move it: moving is a list action"
status: "done"
type: bug
priority: "high"
assignee: ["nautilus"]
labels: []
created: 2026-11-17
updated: 2026-11-17
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
done_at: "2026-11-17T17:06"
---

## Goal
In a task's detail view, J and K (and every other key that changes rank or state) do not act on the task silently. Moving a task by rank works only from a list, overview section or board row, where you see the rows move. In the detail, J and K scroll by half a page (Shift reads as 'more' than j/k). Any change from the detail view goes through a visible prompt or the key bar names it first.

## Purpose
Jørgen 2026-11-17 14:49, in zhvkmj's detail: he pressed Shift-J and Shift-K to see if they scroll; hug instead moved zhvkmj in the rank, with only a footer line 'moved zhvkmj above vzghj8, row 14 of 19 in IN PROGRESS'. Moving the open task from inside it is a surprise: you cannot see the list it moved in.

## Principle
A key in a view acts on what that view shows. The detail shows one task's content, so its keys read and scroll; ranks change where the ranks are visible. No silent change: anything that writes is either visible as it happens or asks first.

## Relevant details
huginn/internal/tui/tui.go:925 and :1074 run actions t, d, x, J, K, e in the detail as well as in lists; actions.go:15-37 and move.go:28-32 do the move. The detail key bar today: 'j/k check · Space tick · - strike out · n/p task · Tab section · u up · Esc back'. Check t, d and x in the detail too: they write to the task the user is reading; they stay, because their effect shows in the header, but they should be named in the key bar.

## Depends on

## Acceptance
- [x] J and K in the detail scroll half a page and never change the rank (test + golden)
- [x] J and K on a list, overview or board row still move the task (test)
- [x] every key that writes in the detail is named in its key bar; no key writes without showing it (test enumerates the detail keys)

## Notes


nautilus 16:06: checks 1-3 ticked. 1: TestDetailJKScrollAndNeverMove + tmux detail goldens. 2: TestRowJKStillMove. 3: TestDetailWritingKeysAreNamed at 80/100/140. landed on main c48b30c (stack d107f9a..c48b30c: 6ef55df 3jtfw9, 842b403 cue, 998438f+c48b30c x1ra0t); local go test tui/task/cli/mcp/cache/harness ok after the rebase; earlier CI 44111992156 green on 99edb5f; cue abb9b76 CI green per hatchling
Reflection (open, 2026-11-17, nautilus): hug task check's help says 'hug task check ID N' with no evidence argument, while the board instructions say 'hug task check ID N with evidence'; I passed evidence as a third argument and got 'expected 2 positional arguments, got 3' with fix 'hug task help', which names no command. The fix line could say: hug task note ID "evidence".
## History
- 2026-11-17T15:52 todo -> doing/active (take)
- 2026-11-17T16:30 doing/active -> doing/waiting (wait)
- 2026-11-17T17:06 doing/waiting -> done (done); was doing_since: 2026-11-17T15:52; waiting_asked_by: nautilus; waiting_on: CI: 9b3085b in CI run 44111299428, then the landing turn with 3jtfw9; waiting_since: 2026-11-17T16:30
