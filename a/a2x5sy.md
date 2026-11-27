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

## Notes
