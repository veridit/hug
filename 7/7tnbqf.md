---
id: 7tnbqf
title: "Release v0.7.108: short evidence refused, wait-for-merge hint, superseded checks"
state: todo
type: feature
priority: ""
assignee: []
labels: []
group: ""
rank: "zzzzzzzzzzzv"
created: 2026-11-22
updated: 2026-11-22
design_for: "jhf"
design_since: "2026-11-22T18:40"
design_asked_by: "crab"
design_question: "v0.7.108 is out. hug task check now refuses evidence under 8 characters, so 'x' or 'ok' no longer ticks a check (it says: fix: hug task check ID N \"what shows it holds\"). Is 8 characters the right minimum?"
stage: "design"
---

## Goal
v0.7.108 ships y8we58: three gaps agents hit (short evidence, take while awaiting merge, superseded checks) are fixed.

## Purpose
One ticket per release so Jørgen reviews what shipped in one place.

## Principle
Shipped tasks are children; one question at a time.

## Details
Main 6b8899ba (Release v0.7.108, merge of 570ac238, CI run 45048705221 8/8 on that exact SHA). release.yml run 45049911216 success; brew gives ../Cellar/hug/0.7.108/bin/hug, hug version v0.7.108. Shipped: (1) hug task check ID N 'x' refuses evidence under 8 characters (after trimming spaces), CLI and MCP, with fix: hug task check ID N "what shows it holds"; (2) a refused take while your doing task has every check settled names hug task wait ID "branch ID merged" first; (3) AGENTS.md and the hug setup section say: a ruling that supersedes a check on another task strikes that check there, naming the ruling task's id. Repos that ran hug setup see their section as out of date until they run it again.

## Depends on

## Acceptance
- [ ] Jørgen says whether 8 characters is the right minimum for evidence

## Questions
- 2026-11-22T18:40 question 1 by crab to jhf (check 1):
  v0.7.108 is out. hug task check now refuses evidence under 8 characters, so 'x' or 'ok' no longer ticks a check (it says: fix: hug task check ID N "what shows it holds"). Is 8 characters the right minimum?

## Notes


## History
- 2026-11-22T18:40 todo -> todo/design (ask)
