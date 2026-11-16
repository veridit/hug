---
id: ksv323
title: "TUI move (J/K) must move past the row you see"
status: "done"
type: bug
priority: "high"
assignee: ["poodle"]
labels: []
created: 2026-11-16
updated: 2026-11-16
---

## Goal
Pressing J or K on a task moves it past the next or previous row shown in the section you are looking at, and the screen shows the row in its new place

## Purpose
Jørgen 2026-11-16 pressed J on a1dx7s in the overview's NEXT UP three times. Each time the bar asked to confirm, said 'move a1dx7s', and the row stayed exactly where it was. Three commits were made (a655982, 37ec5ec, dfdc006) that renamed the file zzzzzu -> zzzzzx8 -> zzzzzu -> zzzzzx8, so the board was changed with nothing visible

## Principle
Move relative to what is on screen: the neighbour is the adjacent row of the visible list (after filters, in the focused section), never a hidden task. A move that changes nothing visible must not happen or must say why. The confirm line names the task it will move past

## Relevant details
Cause: huginn/internal/tui/actions.go lines 67-82 pick the neighbour from m.all (every task in rank order), i+1 or i-1. NEXT UP hides epics, taken tasks and done tasks, so the neighbour of a1dx7s (rank zzzzzx8) was the epic d634kr 'Recall quality' (zzzzzx) above and x1t1t1 below; moving after/before a hidden task leaves the visible order unchanged. Fix: take the neighbour from m.indices() (the focused pane's visible rows). Also: the message says 'move ID' for both directions; say 'moved ID below OTHER'. The key bar and help call this 'move down / up in global rank order'; check the wording still fits. Board view: J/K inside a status column should use that column's rows. Check that a filtered list moves relative to the filtered rows. Rank strings grew to 7 characters here; rebalance exists for that (rank.NeedsRebalance at more than 12)

## Depends on

## Acceptance
- [x] in the overview NEXT UP, J on a row swaps it with the row shown below it, and the screen shows the new order (test with a hidden epic ranked between them)
- [x] K does the same upward; at the first or last visible row the message says so and no commit is made
- [x] the confirm line names both tasks, e.g. 'Move a1dx7s below x1t1t1?', and the result message says where it went
- [x] with a filter active, moves are relative to the filtered rows (test)
- [x] tests reproduce Jørgen's case: three presses no longer produce commits with no visible change

## Notes

2026-11-16 Jørgen: ranks are sparse so a task can be inserted between two others, and the move must take that into account. Checked: Store.Move -> rankPlan (huginn/internal/task/ordering.go:20) already does the sparse insert correctly: it places the moved task right after (or before) the target in the full order and gives it a rank between its new neighbours, renaming as few files as possible. So the rank arithmetic is sound; the fault is only which target the TUI hands it (the adjacent task in the full list, m.all, which may be hidden). Fix in the TUI: pass the adjacent VISIBLE row as the target. Then 'after x1t1t1' lands a1dx7s between x1t1t1 and the next task in the full order (gem59b, a hidden epic), which shows as below x1t1t1. Second finding for this ticket or 44w1ev: new tasks are appended at the end, so ranks only grow (this board: 52 tasks, ranks up to 9 characters, the last six all 'zzzzzzz…'; rank.NeedsRebalance fires above 12). Appending should leave room (pick the midpoint between the last rank and the end, or rebalance automatically when a rank passes a length), otherwise every board drifts toward 'hug task rebalance'.

check 1 (c812523): TestMoveJPastHiddenEpicInNextUp: tasks Alpha, epic Gem (child Child), Xray, Child; NEXT UP shows Alpha Xray Child (epic not shown). J on Alpha -> confirm "Move <a> below <x> in NEXT UP: Alpha? Enter yes · Esc no"; after Enter NEXT UP is Xray Alpha Child, on screen in that order, selection stays on Alpha, full order Gem Xray Alpha Child (only Alpha renamed; rankPlan unchanged). Fix: huginn/internal/tui/move.go moveTarget picks the next/previous row of m.indices() (current section after filters), not m.all. On the old code this test fails with "▶ Confirm move down xpfs8f Alpha? Enter yes · Esc no".

check 2 (c812523): TestMoveKAndEdges: K on Child -> "Move <c> above <x> in NEXT UP: Child?", after Enter NEXT UP Alpha Child Xray, message "moved <c> above <x>, row 2 of 3 in NEXT UP", 1 commit. K on the first row -> no confirm, message "<a> is already the first row in NEXT UP; nothing moved"; J on the last row -> "<x> is already the last row in NEXT UP; nothing moved"; commits stay 1. Choice: an edge press asks nothing (no confirm), since there is nothing to confirm.

check 3 (c812523): confirm line (render.go confirmLine case J/K): "Move <id> below|above <target id> in <section>: <title>? Enter yes · Esc no", section = NEXT UP / IN PROGRESS / BLOCKED / LINT / the TODO column / the list / the lint list, plus " (filtered)" with a filter. Result message: "moved <id> below <target>, row N of M in <section>" (asserted verbatim in TestMoveJPastHiddenEpicInNextUp and TestMoveKAndEdges; wording also in TestFocusConfirmPromptsStandOut). Help hint now "J / K  move below / above the row shown next to it" (help goldens regenerated, only that line changed).

check 4 (c812523): TestMoveRelativeToFilteredRows: Alpha one, Beta, Gamma, Alpha two, Delta; list filtered "/Alpha" shows Alpha one, Alpha two. J on Alpha one -> "Move <a1> below <a2> in the list (filtered)", after: Alpha two, Alpha one; full order Beta, Gamma, Alpha two, Alpha one, Delta; J again -> "already the last row in the list (filtered)", no confirm. TestMoveRepairsTiesAndUsesGlobalOrder updated: with a filter showing one row J used to move past a hidden task; it now refuses with "... already the last row in the list (filtered); nothing moved" and the order is unchanged.

check 5 (c812523): TestMoveThreePressesEachVisible: Alpha, epic Gem + Gem child, done Old, dropped Gone, Xray, Yoke; NEXT UP Alpha, Gem child, Xray, Yoke. Three J+Enter: each changes NEXT UP (Gem child Alpha Xray Yoke -> Gem child Xray Alpha Yoke -> Gem child Xray Yoke Alpha), screen order matches, commits 1, 2, 3; a fourth J says "already the last row in NEXT UP", no commit. On the old code: "press 1: jkwqa4 t2enjf rehb4q g1pj70 -> jkwqa4 t2enjf rehb4q g1pj70, commits 1" (commit, no visible change). go test -v: "--- PASS: TestMoveJPastHiddenEpicInNextUp", "--- PASS: TestMoveKAndEdges", "--- PASS: TestMoveRelativeToFilteredRows", "--- PASS: TestMoveThreePressesEachVisible", "--- PASS: TestMoveRepairsTiesAndUsesGlobalOrder". Full suite GOMAXPROCS=2 GOFLAGS=-p=2 nice -n 19 go test -count=1 ./huginn/... ./cmd/...: 11 ok, 0 FAIL, last "ok  	github.com/veridit/hug/cmd/hug	0.230s". The second finding in the notes (append-only ranks drift) is not in this ticket and was not touched.
