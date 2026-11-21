---
id: 4ddcwa
title: "Detail page frame: ↑ on the top rule, a bottom rule with ↓ and the page status"
state: "doing"
type: feature
priority: "high"
assignee: ["ladybug"]
labels: []
group: ""
rank: "zzzzzzzzzzz7"
created: 2026-11-20
updated: 2026-11-20
doing_since: "2026-11-21T17:35"
stage: "asked"
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
asked_since: "2026-11-21T18:44"
asked_by: "ladybug"
asked_check: "1"
asked_what: "Open the four attached frames at 100x30 (top, mid-page, end, and mid-page with a message): do the ↑ on the rule under the title, the bottom rule '↓── Task 1 of 2 · 29% · next · previous · Esc back ───' right on the keys, and a message like '• prtwm2 moved to IN PROGRESS' taking the status's place on that rule (↓ kept) read right? Your answer also settles xt0qvf's layout for the task page."
asked_for: "jhf"
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
- [ ] The rule under the title shows ↑ at its left end exactly when the page is scrolled down (golden)
- [ ] A bottom rule sits right above the global keys, starts with ↓ when more is below, and carries the page status; no blank line between page, rule and keys (golden 100x30 and 160x45)
- [ ] Jørgen opens a long task after release and sees where it ends and that there is more

## Questions
- 2026-11-21T18:44 question 1 by ladybug to jhf (check 1):
  Open the four attached frames at 100x30 (top, mid-page, end, and mid-page with a message): do the ↑ on the rule under the title, the bottom rule '↓── Task 1 of 2 · 29% · next · previous · Esc back ───' right on the keys, and a message like '• prtwm2 moved to IN PROGRESS' taking the status's place on that rule (↓ kept) read right? Your answer also settles xt0qvf's layout for the task page.

## Notes


file:frame-1-top-100x30.ans (4089 bytes, sha256:5265319c) 4ddcwa check 1, 100x30, qpp30y opened at the top: plain top rule, bottom rule '↓── Task 1 of 2 · 4% …' right on the keys
file:frame-2-mid-100x30.ans (3698 bytes, sha256:77c3628f) 4ddcwa check 1, 100x30, qpp30y scrolled mid-page (J J): ↑ on the top rule, ↓ on the bottom rule
file:frame-3-end-100x30.ans (4447 bytes, sha256:d479a93f) 4ddcwa check 1, 100x30, qpp30y at the end (>): ↑ on the top rule, no ↓: '─── Task 1 of 2 · 100% …'
file:frame-4-mid-message-100x30.ans (3897 bytes, sha256:c8601952) 4ddcwa check 1, 100x30, a long task mid-page after t Enter: the message '• prtwm2 moved to IN PROGRESS' takes the status's place on the bottom rule, the ↓ stays (xt0qvf option A for the task page)
file:frame-1-top-100x30.ans (3953 bytes, sha256:9943caff) replaces sha256:5265319c 4ddcwa check 1, 100x30, qpp30y at the true top (opened, then <): no ↑ on the top rule, DESCRIPTION and PURPOSE in view, bottom rule '↓── Task 1 of 2 · 0% …' right on the keys (recaptured 17:49Z: the first capture was the opened page, which focus had scrolled 1 line)
file:frame-2-mid-100x30.ans (3688 bytes, sha256:1bdcb20c) replaces sha256:77c3628f recaptured 17:49Z in the same run as frame-1-top; same layout as before
## History
- 2026-11-21T17:35 todo -> doing/active (take)
- 2026-11-21T18:44 doing/active -> doing/asked (ask)
