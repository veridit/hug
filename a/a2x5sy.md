---
id: a2x5sy
title: "Release v0.7.117: hug names the step due now, for each role"
state: todo
type: feature
priority: ""
assignee: []
labels: []
group: ""
rank: "zzzzzzzzzzzx"
created: 2026-11-27
updated: 2026-11-27
design_question: "v0.7.117 is installed. After a review, hug now tells the coordinator to ship once the reviewer's answer approves. Today an answer approves when all it says is 'looks right', 'ok', 'lgtm', 'yes' or similar; anything more ('looks right, but rename X') counts as a change request. Keep it like that, or have reviewers type an explicit 'hug task approve ID'? Reply 'words' or 'approve'."
design_for: "jhf"
design_since: "2026-11-27T06:43"
design_asked_by: "crab"
stage: "design"
---

## Goal
Jørgen confirms the step hug names after a review answer is the right one, and how an approval is given.

## Purpose
One release, one question: erv680 (self-guiding board) shipped alone.

## Principle
One question per shipped release, asked here only; erv680's design question is settled from this answer.

## Details
Shipped: erv680 (raccoon ebebcaab; crab review fix bb1518d1: 'looks right, but rename X' had counted as approval; CI 45945304510 8/8; main c54b3324; release.yml 45946502724; brew hug v0.7.117). What changed: CLI next:, MCP next and the board cue name one step from the task's state: write checks (task with none), tick check N, reviewer answers, after approval 'ship it, as hug.toml says: LINE' ([board] ship) else 'as your AGENTS.md says', then done. An approval today is a review answer whose every clause is one of: looks right, looks good, approved, approve, lgtm, ok, okay, yes, ship it. 5yh7hr (review starts on the last tick) is in progress.

## Depends on

## Acceptance
- [ ] Jørgen chooses how a reviewer approves: the answer words as today, or an explicit hug task approve ID

## Questions
- 2026-11-27T06:43 question 1 by crab to jhf (check 1):
  v0.7.117 is installed. After a review, hug now tells the coordinator to ship once the reviewer's answer approves. Today an answer approves when all it says is 'looks right', 'ok', 'lgtm', 'yes' or similar; anything more ('looks right, but rename X') counts as a change request. Keep it like that, or have reviewers type an explicit 'hug task approve ID'? Reply 'words' or 'approve'.

## Notes


## History
- 2026-11-27T06:43 todo -> todo/design (ask)
