---
id: 4ddcwa
title: "Detail page frame: ↑ on the top rule, a bottom rule with ↓ and the page status"
state: "doing"
type: feature
priority: "high"
assignee: ["crab"]
labels: []
group: ""
rank: "zzzzzzzzzzxv"
created: 2026-11-20
updated: 2026-11-20
doing_since: "2026-11-21T17:35"
stage: "active"
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
parent: "wg59vb"
---

## Goal
A task's page sits between two rules: ↑ on the top one, ↓ and the page status on the bottom one.

## Purpose
You see where the ticket ends and whether there is more, and the page's own status is kept apart from the global keys.

## Principle
The page owns its frame and its status; the last line stays the global keys, and no blank line is wasted.

## Relevant details
Layout (Jørgen 2026-11-20 10:32Z, request 1 on qpp30y, screenshot screen-1130.ans there), bottom up:
- last line: the global keys, as today.
- above it, one bottom rule: '↓──' (the ↓ only when more is below), then the status of the thing you are in (today's 'Task 3 of 14 · 0% · next · previous · Esc back'), then the rule runs on to the edge. It replaces today's status line plus the blank line under it.
- top: the rule under the title gets a small ↑ at its left end when the page is scrolled down.
Sketch:
  ↑─────────────────────────────────────────────
  ...page...
  ↓── Task 3 of 14 · 0% · next · previous · Esc back ─────────────
  j/k check · Space tick · ... (global keys)
Replaces 8a0ybb check 9 (↑/↓ in the corners). Related: xt0qvf (where the status message sits in the lists). His words: 'there should be a small arrow pointing up on the long line on the left part of the below the subject ... there should be a line on the bottom of the screen where the ticket ends ... an arrow as well pointing down on that line ... that line should separate from what is below ... the one line that starts with an arrow down, a piece of line, and then any inline status for this thing I'm in now. And then the next line is the global line. Because right now you can see that there is this extra space.'

## Depends on

## Acceptance
- [ ] Jørgen approves ANSI frames at 100x30 of a long task: at the top, scrolled mid-page, and at the end
- [x] The rule under the title shows ↑ at its left end exactly when the page is scrolled down (golden)
- [x] A bottom rule sits right above the global keys, starts with ↓ when more is below, and carries the page status; no blank line between page, rule and keys (golden 100x30 and 160x45)
- [ ] Jørgen opens a long task after release and sees where it ends and that there is more

## Questions
- 2026-11-21T18:44 question 1 by ladybug to jhf (check 1):
  Open the four attached frames at 100x30 (top, mid-page, end, and mid-page with a message): do the ↑ on the rule under the title, the bottom rule '↓── Task 1 of 2 · 29% · next · previous · Esc back ───' right on the keys, and a message like '• prtwm2 moved to IN PROGRESS' taking the status's place on that rule (↓ kept) read right? Your answer also settles xt0qvf's layout for the task page.
  answered 2026-11-25T10:20 by jhf:
    I like it

## Notes


file:frame-1-top-100x30.ans (4089 bytes, sha256:5265319c) 4ddcwa check 1, 100x30, qpp30y opened at the top: plain top rule, bottom rule '↓── Task 1 of 2 · 4% …' right on the keys
file:frame-2-mid-100x30.ans (3698 bytes, sha256:77c3628f) 4ddcwa check 1, 100x30, qpp30y scrolled mid-page (J J): ↑ on the top rule, ↓ on the bottom rule
file:frame-3-end-100x30.ans (4447 bytes, sha256:d479a93f) 4ddcwa check 1, 100x30, qpp30y at the end (>): ↑ on the top rule, no ↓: '─── Task 1 of 2 · 100% …'
file:frame-4-mid-message-100x30.ans (3897 bytes, sha256:c8601952) 4ddcwa check 1, 100x30, a long task mid-page after t Enter: the message '• prtwm2 moved to IN PROGRESS' takes the status's place on the bottom rule, the ↓ stays (xt0qvf option A for the task page)
file:frame-1-top-100x30.ans (3953 bytes, sha256:9943caff) replaces sha256:5265319c 4ddcwa check 1, 100x30, qpp30y at the true top (opened, then <): no ↑ on the top rule, DESCRIPTION and PURPOSE in view, bottom rule '↓── Task 1 of 2 · 0% …' right on the keys (recaptured 17:49Z: the first capture was the opened page, which focus had scrolled 1 line)
file:frame-2-mid-100x30.ans (3688 bytes, sha256:1bdcb20c) replaces sha256:77c3628f recaptured 17:49Z in the same run as frame-1-top; same layout as before
file:frame-4-mid-message-100x30.ans (3944 bytes, sha256:600fe7d3) replaces sha256:c8601952 recaptured 17:49Z in the same run as frame-1-top; same layout as before
frames recaptured 17:49Z (crab review): the first frame-1-top was the page as opened, not the top: opening qpp30y focuses check 1 and openTop/showMarked (detail.go:1065-1070, ykta5g check 4) scrolls the page 1 line so the whole check is in view, hence ↑ and 4%. The old note 'opened at the top' on sha256:5265319c is wrong; the replacement sha256:9943caff is the true top (opened, then <): no ↑, PURPOSE in view, 0%. frame-3-end was unchanged by the rerun. The % on the rule is the page position, viewport.ScrollPercent() = YOffset/(lines-height), as before 4ddcwa (render.go pageStatus).
ladybug: build on branch 4ddcwa, a19cd1d4 (ci/4ddcwa run 44933305676). render.go: pageTopRule (↑ when viewport.YOffset>0), pageBottomRule ('↓── ' while !AtBottom, else '─── '), pageFoot (footRows for the task page: input line only when it says something, then the bottom rule; the message replaces the status after a dim '• ', ↓ kept), pageHeight/fitPage (page = body less header, top rule and the lines under it). tui.go: syncDetail sizes the page with pageHeight; Update calls fitPage so a confirm/prompt takes a row and gives it back. Failing first on 626c81d0: 4ddcwa_test.go:81 'the row above the keys is not the bottom rule' (TestPageFrameArrowsAndStatus 100x30 and 160x45), :116 (ShortTask), :130 (MessageAndPrompt), :172 (DoesNotPulse). Goldens testdata/4ddcwa/page-frame-{100x30,160x45}.txt, message-100x30.txt, message-mid-page-{100x30,160x45}.txt. Lists' message line untouched (xt0qvf).
check 2: TestPageFrameArrowsAndStatus (4ddcwa_test.go:62): ↑ on the top rule exactly when viewport.YOffset>0, at the top/mid-page/end, 100x30 and 160x45; goldens testdata/4ddcwa/page-frame-{100x30,160x45}.txt; a19cd1d4, CI 44933305676 8/8 success
check 3: TestPageFrameArrowsAndStatus + ShortTask + HeightAfterResize + MessageReplacesOnlyTheStatus: bottom rule on the keys, ↓ exactly while !AtBottom, page status on it, no blank line, page = height - header - 4 rows; goldens page-frame-{100x30,160x45}.txt, message-mid-page-{100x30,160x45}.txt; a19cd1d4, CI 44933305676 8/8 success
landed 6e44eb3 on main 2026-11-21: tui: a task's page sits between two rules: ↑ on the rule under the title when…
shipped in v0.7.98 (6e44eb3)
crab 18:23Z: shipped in v0.7.98 (main 8611f2b6, brew-verified). Release ticket wg59vb. Check 1 (frames) still waits on Jørgen; if he asks for a change, it goes in a follow-up release.
check 2: a19cd1d4: TestPageFrameArrowsAndStatus :83 + goldens testdata/4ddcwa/page-frame-{100x30,160x45}; mutants M1/M2 caught; CI 8/8 on a19cd1d4; shipped v0.7.98
check 3: a19cd1d4: bottom rule on the keys, ↓ when more below, status on it; message replaces only the status after a dim • (TestPageFrameMessageReplacesOnlyTheStatus); mutants M3-M9 caught; 26 goldens reviewed; CI 8/8; shipped v0.7.98
## History
- 2026-11-21T17:35 todo -> doing/active (take)
- 2026-11-21T18:44 doing/active -> doing/asked (ask)
- 2026-11-21T20:14 handed to crab: ladybug's session ended (closed 2026-11-21 19:13)
- 2026-11-25T10:20 answered asked (check 1), by jhf, for ladybug (doing/asked -> doing/active):
  > Open the four attached frames at 100x30 (top, mid-page, end, and mid-page with a message): do the ↑ on the rule under the title, the bottom rule '↓── Task 1 of 2 · 29% · next · previous · Esc back ───' right on the keys, and a message like '• prtwm2 moved to IN PROGRESS' taking the status's place on that rule (↓ kept) read right? Your answer also settles xt0qvf's layout for the task page.
  I like it
- 2026-11-25T10:20 answer 1 delivered to crab (session_crab_1795558631109_253787e0f5d2bbbb)
