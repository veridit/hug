---
id: hya1st
title: "Task page reads small to large: Goal first, a short Details shown open"
state: "doing"
type: feature
priority: "high"
assignee: ["rhino"]
labels: []
group: ""
rank: "zzzzzzzzzzzw"
created: 2026-11-24
updated: 2026-11-24
parent: "9a479v"
doing_since: "2026-11-25T09:31"
stage: "active"
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
---

## Goal
A task's page opens on its shortest section, the Goal, and grows toward the long ones; a short Details shows open, not folded.

## Purpose
Jørgen 2026-11-24 18:15Z on statbus kkj0mn (screenshot): 'the ordering seems off, it starts big and then shrinks, it should start small first? And DETAILS can show if not big? I.e. start auto expanded if small'.

## Principle
Small before large, so the page reads like a funnel. A section folds only when it is long enough to push the rest away; positions stay predictable from task to task.

## Details
Picked by Jørgen 18:23Z: GOAL, PURPOSE, PRINCIPLE, DETAILS, then ACCEPTANCE, the other short sections, then NOTES and HISTORY folded (the AGENTS.md writing order). Details opens when it fits in 6 wrapped lines or fewer at the current width (provisional), else it folds to one line with its count as today. Today (8a0ybb, shipped): PURPOSE, PRINCIPLE, GOAL, ACCEPTANCE, short sections, then DETAILS, NOTES, HISTORY always folded. On kkj0mn: Purpose 4 lines, Principle 2, Goal 1, and DETAILS (1) folded although it is 1 line. Supersedes 8a0ybb check 12.

## Depends on

## Acceptance
- [x] Jørgen picks the order from ANSI frames of kkj0mn and 8a0ybb at 100x30 (variants A and B, a short Details open, a long one folded)
- [ ] The page shows the picked order; Details is open when it fits in the threshold and folded with its count when longer (golden 100x30 and 160x45, test at the threshold)
- [ ] Jørgen confirms on his board after release

## Notes

check 1: Jørgen 18:23Z, without frames: 'Goal, Purpose, Principles, Details'. A fixed order, not variant A or B. Acceptance follows Details, as in AGENTS.md's writing order (Goal, Purpose, Principle, Relevant details, then Acceptance).

Jørgen 18:24Z, why this order: 'I.e. What, Why, How, Notes (to explain those over)'. So Goal is the what, Purpose the why, Principle the how, and Details the notes that explain them. The section names stay as they are; this is only the reason for the order.


failing-first 9e8b4875 (branch hya1st, on main ed882297): TestPageReadsGoalPurposePrincipleDetails 'hya1st_test.go:98: headings PURPOSE|PRINCIPLE|GOAL|ACCEPTANCE|DETAILS (1) want GOAL|PURPOSE|PRINCIPLE|DETAILS|ACCEPTANCE'; TestDetailsOpenAtSixLinesFoldedAtSeven 'hya1st_test.go:139: headings PURPOSE|PRINCIPLE|GOAL|ACCEPTANCE|DETAILS (4), want GOAL|PURPOSE|PRINCIPLE|DETAILS (4)|ACCEPTANCE'; TestTabFollowsGoalPurposePrincipleDetails 'hya1st_test.go:163: stops'; TestNextPreviousKeepTheDetailsRule FAIL; TestKkj0mnPageGolden 'hya1st_test.go:243: headings PURPOSE|PRINCIPLE|GOAL|ACCEPTANCE|DETAILS (1)'
file:hya1st-page-100x30.ans (3669 bytes, sha256:fe42ea46) hya1st check 2: the new page at 100x30, built binary from branch hya1st e033dee2, tmux capture-pane -e -p (COLORFGBG=15;0), on a copy of statbus kkj0mn with a 1-line Details (kkj0ms): GOAL, PURPOSE, PRINCIPLE, DETAILS open, ACCEPTANCE
file:hya1st-long-details-folded-100x30.ans (3598 bytes, sha256:7e492f92) hya1st check 2: a long Details (4 paragraphs, kkj0mp) folded to '› DETAILS (4)' after PRINCIPLE, 100x30, same binary and capture
file:hya1st-long-details-opened-100x30.ans (3943 bytes, sha256:c41071fd) hya1st check 2: Shift-Tab from the checks to DETAILS, Enter opens it in place, ACCEPTANCE after its text
file:mutants.md (2964 bytes, sha256:45ed8863) hya1st + 32ym01 mutants: 20 rules broken by hand, 20 killed (2 needed a test or a sharper mutant: see rows 9 and 15)
## History
- 2026-11-25T09:31 todo -> doing/active (take)
