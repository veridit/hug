---
id: z82qsy
title: "File viewer: a rule under the header; no stale action message"
state: "done"
type: bug
priority: ""
assignee: ["macaque"]
labels: []
group: ""
rank: "zzzzzzzzzzna"
created: 2026-11-18
updated: 2026-11-18
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
done_at: "2026-11-18T18:40"
---

## Goal
In a task's file viewer, a horizontal rule separates the header (path, size, n of m, title) from the file's contents, and the message line never shows a leftover result from an earlier action on another task.

## Purpose
Jørgen 2026-11-18 viewing 5z4zt4/detail-stage.md: the contents start with no visual boundary, and the bottom line read 'answer vzghj8', the result of answering a different task earlier, which looks like part of this view.

## Principle
The message line reports the action just taken and clears when the person moves on (opens another task or file, changes view). One rule style shared with the detail header's existing rules.

## Relevant details
Message comes from actions.go finish(): m.message = action + ' ' + ID, never cleared on navigation. Message also reads as a command ('answer vzghj8') rather than a result; prefer 'answered vzghj8'. Header lines from files.go (size via fmt KiB). Screenshot in conversation 11:37Z.

## Depends on

## Acceptance
- [x] The file viewer draws a horizontal rule between its header lines and the contents, in overview, list and board (golden).
- [x] After an action on one task, opening another task or file clears the message line; a test fails first on main.
- [x] Action results read as past tense with the id ('answered vzghj8'), not as a command (test).
- [x] Viewing a file, everything else goes away: the top shows only the crumb (task id > file path, n of m), the bottom only the keys (next/previous file, back), and a full-width rule at the top and bottom frames the file so a screenshot's own lines read as inside it (golden; Jørgen's 5z4zt4 request 2)

## Notes


check 1: commit d24bfd34 on branch z82qsy. render.go draws the detail header's dim rule under the file header line too, and glanceChrome=3 keeps the screen height. TestFileViewerRuleUnderHeader opens 000001 by Enter on its row in overview, list and board (the breadcrumbs read 'HUG TASKS · overview › 000001', '... list ...', '... board ...'), then shows bench/README.md. It failed first at z82qsy_test.go:58 ('under the file header "Bench..." / "   ...", want the rule then the contents') and passes now. New golden testdata/detail/file-rule.txt. glance.txt and markdown-file.txt changed by +8 rule lines and -8 blank padding lines, nothing else.
check 2: finish() records result and resultAt (view + open task path + file head). clearStaleResult in afterUpdate clears the message when the place changes, and leaves it alone once another message has replaced it. TestActionMessageClearsOnMovingOn does t Enter on 000001, j keeps the result, then opening 000002, a file, or v b clears it. It failed first on main at z82qsy_test.go:75 ('another task: the old result stays: "take 000001"') and passes now.
check 3: pastTense maps take->took, give back->gave back, done->marked done, drop->dropped, wait->parked, answer->answered, request->requested, strike->struck, restore->restored, check->checked, uncheck->unchecked, new->created, move->moved. TestActionResultIsPastTense failed first ('t Enter: "take 000001", want "took 000001"') and passes now. Updated 4 existing tests (answer, asked, request, strike) and review_tmux_test waits. Full suite: 34 ok, 0 FAIL ('ok github.com/veridit/hug/huginn/internal/tui 120.358s'). CI 44302950954 on ci/z82qsy.
check 4: commit ef9c060f on branch z82qsy. Viewing a file, the screen holds: crumb 'HUG TASKS · list › 000001 › compiler/000001/screen.ans · 80 B · 7 of 7', then the attachment note dim ('screen capture from hug tui' or 'detail mock', parsed from the task's 'file:NAME (N bytes, sha256:H) WHY' note, with no line when there is no note and never the .md's H1), then a full-width rule, the file, a full-width rule, then the message line, input line and keys outside the frame. Chips, dates, lint and task title are gone. The 'File · %' footer is gone (the keys are on the bar). TestFileViewerFramesOnlyTheFile (80 and 140, .ans plus .md attached via Store.Attach) failed first at z82qsy_test.go:172 (crumb 'HUG TASKS · list › 000001', no file), and TestFileViewerNoNoteNoLine failed first at :202 (chips under the crumb). Both pass now. Golden testdata/detail/file-frame.txt is new. glance.txt, markdown-file.txt and file-rule.txt are rewritten to the frame, diffs reviewed. TestGlanceStepsAndScrolls runs at 80x9 so escapes.log still scrolls in the taller frame. Full suite: 34 ok, 0 FAIL ('ok github.com/veridit/hug/huginn/internal/tui 102.014s'). CI for commit 1 (d24bfd34) is run 44302950954, completed success. CI for this commit is 44305041418.
CI run 44305041418 on ci/z82qsy (ef9c060f) finished completed success, all 6 jobs. Report: /Users/jhf/.jcode/scratch/z82qsy/REPORT.md
crab review 12:52Z: read file-frame.txt golden (crumb with path, size, n of m; dim attachment note; rules frame the file; keys alone at the bottom). Matches Jørgen's 5z4zt4 request 2. d24bfd34 CI 44302950954 and ef9c060f CI 44305041418 both 6/6. Lands after xfv9pw (rebase plus exact-SHA CI). Gap: a .png opens full screen via imageView, outside the frame; follow-up task filed.
landed afaa5df on main 2026-11-18: TUI file viewer: rule under the file header; results past tense, cleared on…
landed 161381f on main 2026-11-18: TUI file viewer: only the file, framed by rules (z82qsy check 4)
shipped in v0.7.63 (afaa5df)
landed 161381f on main 2026-11-18 (with ef9c060 rebased as 161381f3, afaa5df2): TUI file viewer frame (z82qsy); in v0.7.63.
Reflection (unanswered, 2026-11-18, crab)
## History
- 2026-11-18T12:53 todo -> doing/active (take)
- 2026-11-18T13:34 doing/active -> doing/waiting (wait)
- 2026-11-18T18:40 doing/waiting -> done (done); was doing_since: 2026-11-18T12:53; waiting_asked_by: macaque; waiting_on: crab review; waiting_since: 2026-11-18T13:34
