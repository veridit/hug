---
id: 8a0ybb
title: "Detail view: the question first, links that jump, short sections before long"
state: todo
type: feature
priority: ""
assignee: []
labels: []
group: ""
rank: "zzzzzzzzzzzx"
created: 2026-11-19
updated: 2026-11-19
stage: "ready"
---

## Goal
An asked task opens on its question, which jumps to its target in one key, and reads short to long.

## Purpose
Jørgen 2026-11-19 11:23Z (crab console, screenshot of p87ngv in hug tui v0.7.81): the page opened on MOCK.md inlined at the top (asked_show) with no question above it, so it looked like content from later in the ticket; the question was only found by scrolling; the long description came before purpose and principle ('where am I getting the value for the money'). 11:24Z (second screenshot, check 5 with its question and 'answer'): the question names MOCK.md but offers no way to jump to it; he had to press down many times to reach FILES and then the file.

## Principle
The person is asked first, then oriented, then given detail. A question links, never inlines. Links use the mark syntax (p87ngv): .target:(.File .name:"MOCK.md"), .target:(.Section .name:"Acceptance" .item:5), .target:(.File .name:"tui-100x30.txt" .line:63); asked_show maps to .target:(.File ...). One jump key everywhere (Enter on a link). Folded sections show their one-line summary (FILES (6), NOTES (12), HISTORY (40)). Calm rules (qh9ep1 check 9).

## Relevant details
Opening a task that asks you something shows, under the title, a FOR YOU section with each open question; each names its target (a file, a section, a check, a line in a file) and Enter on it jumps straight there, a answers. The target file is shown only when you follow the link, never inlined at the top. Below that the ticket reads short to long: Purpose, Principle, Goal, Acceptance, then Details, Files, Notes, History as one-line folded sections until opened. Code: huginn/internal/tui/detail.go (sections around 530-600 and 819-830, markedBlock 1060-1100), picture.go:248 questionPictureOf (asked_show inline). jnaspy set Goal, Purpose, Principle first; this changes the order to Purpose, Principle, Goal and folds reference sections. Open for Jørgen (asked on this task): whether Goal stays as long as today or is cut to 1-2 sentences with the rest in Details. Mock first (frames at 100x30 and 160x45 of p87ngv as it is now), approved before build. Order (Jørgen 11:31Z): the DESCRIPTION heading goes; the sections are FOR YOU, Purpose, Principle, Goal, Acceptance, then the short sections (Files, Depends on, Parent and children, open requests), then the long ones: Relevant details, Notes, History.

## Depends on

## Acceptance
- [ ] Jørgen approves a mock of p87ngv's detail with FOR YOU first, the link, and the short-to-long order (frames attached)
- [ ] Enter on a question's target opens that file in the file frame, or scrolls to that section or check, in one key (test per target kind)
- [ ] A task with asked_show no longer inlines the file above the question; the question shows its target as a link (golden)
- [ ] Details, Files, Notes and History are folded to one line until opened; Purpose, Principle, Goal, Acceptance show in that order (golden 100x30 and 160x45)
- [ ] Jørgen opens an asked task in hug tui and reaches the thing he is asked about in one key
- [ ] No DESCRIPTION heading; order is FOR YOU, Purpose, Principle, Goal, Acceptance, short sections (Files, Depends on, Parent/children, requests), then Relevant details, Notes, History; long sections fold to one line until opened (golden 100x30 and 160x45)

## Questions
- 2026-11-19T12:24 question 1 by crab to jhf:
  Should the Goal stay as long as today, or be cut to one or two sentences with the rest moved to Details?
  answered 2026-11-19T12:26 by jhf:
    (console 11:26Z) Yes, the Goal should be shorter, not long. It is wrong today: AI tends to bloat, and that loses focus. We have to fight for focus.

## Notes


## History
- 2026-11-19T12:24 todo -> todo/design (ask)
- 2026-11-19T12:26 answered design, by jhf, for crab (todo/design -> todo/ready):
  > Should the Goal stay as long as today, or be cut to one or two sentences with the rest moved to Details?
  (console 11:26Z) Yes, the Goal should be shorter, not long. It is wrong today: AI tends to bloat, and that loses focus. We have to fight for focus.
- 2026-11-19T12:26 answer 1 delivered to crab (session_crab_1793783281662_44ffaee927a8a9d5)
- 2026-11-19T12:26 crab read the answer: hug task set 8a0ybb
- 2026-11-19T12:26 answer 1 acknowledged by crab
