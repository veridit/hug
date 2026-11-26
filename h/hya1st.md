---
id: hya1st
title: "Task page reads small to large: Goal first, a short Details shown open"
state: "doing"
type: feature
priority: "high"
assignee: ["crab"]
labels: []
group: ""
rank: "zzzzzzzzzzzk"
created: 2026-11-24
updated: 2026-11-24
parent: "a2k4r5"
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
- [x] The page shows the picked order; Details is open when it fits in the threshold and folded with its count when longer (golden 100x30 and 160x45, test at the threshold)
- [ ] Jørgen confirms on his board after release

## Questions
- 2026-11-25T11:19 question 1 by crab to jhf (check 3):
  Jørgen confirms on his board after release
  answered 2026-11-25T13:01 by crab:
    Withdrawn by crab 2026-11-25: asked once, on release ticket a2k4r5.

## Notes

check 1: Jørgen 18:23Z, without frames: 'Goal, Purpose, Principles, Details'. A fixed order, not variant A or B. Acceptance follows Details, as in AGENTS.md's writing order (Goal, Purpose, Principle, Relevant details, then Acceptance).

Jørgen 18:24Z, why this order: 'I.e. What, Why, How, Notes (to explain those over)'. So Goal is the what, Purpose the why, Principle the how, and Details the notes that explain them. The section names stay as they are; this is only the reason for the order.


failing-first 9e8b4875 (branch hya1st, on main ed882297): TestPageReadsGoalPurposePrincipleDetails 'hya1st_test.go:98: headings PURPOSE|PRINCIPLE|GOAL|ACCEPTANCE|DETAILS (1) want GOAL|PURPOSE|PRINCIPLE|DETAILS|ACCEPTANCE'; TestDetailsOpenAtSixLinesFoldedAtSeven 'hya1st_test.go:139: headings PURPOSE|PRINCIPLE|GOAL|ACCEPTANCE|DETAILS (4), want GOAL|PURPOSE|PRINCIPLE|DETAILS (4)|ACCEPTANCE'; TestTabFollowsGoalPurposePrincipleDetails 'hya1st_test.go:163: stops'; TestNextPreviousKeepTheDetailsRule FAIL; TestKkj0mnPageGolden 'hya1st_test.go:243: headings PURPOSE|PRINCIPLE|GOAL|ACCEPTANCE|DETAILS (1)'
file:hya1st-page-100x30.ans (3669 bytes, sha256:fe42ea46) hya1st check 2: the new page at 100x30, built binary from branch hya1st e033dee2, tmux capture-pane -e -p (COLORFGBG=15;0), on a copy of statbus kkj0mn with a 1-line Details (kkj0ms): GOAL, PURPOSE, PRINCIPLE, DETAILS open, ACCEPTANCE
file:hya1st-long-details-folded-100x30.ans (3598 bytes, sha256:7e492f92) hya1st check 2: a long Details (4 paragraphs, kkj0mp) folded to '› DETAILS (4)' after PRINCIPLE, 100x30, same binary and capture
file:hya1st-long-details-opened-100x30.ans (3943 bytes, sha256:c41071fd) hya1st check 2: Shift-Tab from the checks to DETAILS, Enter opens it in place, ACCEPTANCE after its text
file:mutants.md (2964 bytes, sha256:45ed8863) hya1st + 32ym01 mutants: 20 rules broken by hand, 20 killed (2 needed a test or a sharper mutant: see rows 9 and 15)
check 2: c8b22551 on branch hya1st, CI run 45591665551 8/8 success (secrets, bench, test ubuntu/macos/windows, test-windows-cli sync/rest/shards). Page reads GOAL, PURPOSE, PRINCIPLE, DETAILS, ACCEPTANCE: TestPageReadsGoalPurposePrincipleDetails; Details open at 6 wrapped lines, folded '› DETAILS (N)' at 7, counted at the page width: TestDetailsOpenAtSixLinesFoldedAtSeven (six/seven/wide-at-100/wide-at-160); Tab/Shift-Tab, Enter on folded DETAILS, ring page-DETAILS-checks: TestTabFollowsGoalPurposePrincipleDetails; n/p: TestNextPreviousKeepTheDetailsRule; goldens testdata/hya1st/kkj0mn-100x30.txt (folded: kkj0mn's 1 paragraph wraps to 8 lines at 100) and kkj0mn-160x45.txt (open, 4 lines); 8a0ybb pins rewritten. Constant detailsOpenLines = 6 (detail.go). Mutants: mutants.md, 20/20 killed. Frames: hya1st-page-100x30.ans, hya1st-long-details-folded-100x30.ans, hya1st-long-details-opened-100x30.ans
Built on branch hya1st (not merged, no tag): final SHA c8b22551, CI run 45591665551 on ci/hya1st 8/8 success. Failing-first 9e8b4875, fix e033dee2, ring pin c8b22551. Goldens changed (32): 24403z/breadcrumb-100x30, 4ddcwa/{message-100x30,message-mid-page-100x30,message-mid-page-160x45,page-frame-100x30,page-frame-160x45}, 8a0ybb/page-{100x30,160x45}, cd63d1/page-100x30, detail/{answer-100,answer-box,asked-block,asked-check,bands,check-ids,find,manyrequest,p87ngv-five-questions-100x30,p87ngv-five-questions-160x45,p87ngv-old-answer-100x30,p87ngv-old-answer-160x45,picture-framed,picture,sections-000001-100,several-questions-100,tab-ids}, palette/context, rows/dated-wait-120, tmux/{80x24,140x40}-{color,no-color}/detail; new hya1st/kkj0mn-{100x30,160x45}. Each reviewed: reorder only (Goal moved up), a short Details now open, or the lint text (how: ...). Found: kkj0mn's own 1-paragraph Details is 8 lines at 100 cols (glamour leaves '(fall' and 'and' alone on lines, its two wrap passes disagree by one column), so it folds at 100x30 and opens at 160x45; the frame hya1st-page-100x30.ans uses a 1-line copy (kkj0ms). Ready for release; check 3 is Jørgen's after it.
Frames correction: the binary for the three hya1st-*.ans frames was built at 10:01Z from the worktree at c8b22551 (not e033dee2 as their notes say); c8b22551 changes only hya1st_test.go, so the code is the same.
landed e033dee on main 2026-11-25: A task's page reads Goal, Purpose, Principle, Details (open at 6 lines or…
shipped in v0.7.112 (e033dee)
crab: shipped in v0.7.112 (main c1056b46, merge of c8b22551, CI 45591665551 8/8, release.yml 45593553415, brew hug v0.7.112). Release ticket a2k4r5.
crab 2026-11-25: the last check is confirmed through the release ticket's one question; crab ticks it from that answer.
crab: check 3 holds; asked once on release ticket a2k4r5 check 1 (one question per release); tick from that answer.
## History
- 2026-11-25T09:31 todo -> doing/active (take)
- 2026-11-25T11:19 doing/active -> doing/asked (ask)
- 2026-11-25T12:55 handed to crab: rhino's session ended (closed 2026-11-25 11:17)
- 2026-11-25T13:01 answered asked (check 3), by crab, for crab (doing/asked -> doing/active):
  > Jørgen confirms on his board after release
  Withdrawn by crab 2026-11-25: asked once, on release ticket a2k4r5.
