---
id: 3m6byj
title: "Release v0.7.94: / search lands on the task you typed"
state: todo
type: chore
priority: ""
assignee: []
labels: []
group: ""
rank: "zzzzzzzzzzzs"
created: 2026-11-21
updated: 2026-11-21
design_since: "2026-11-21T11:43"
design_asked_by: "crab"
design_question: "v0.7.94: on statbus, / k5q120 now selects k5q120 in LINT; Enter shows the results and j/k walk every match; Esc returns to all tasks with k5q120 selected; Tab still keeps it as a filter. One change: Esc no longer clears filter chips (F clears all, ⌫ the last). Does search work the way you wanted, and is Esc keeping chips right?"
design_for: "jhf"
stage: "design"
---

## Goal
Jørgen confirms v0.7.94: typing a task id in / selects it, Enter shows the results to walk, Esc returns with it selected.

## Purpose
On his statbus board / k5q120 picked a task that only mentioned k5q120, the arrows did nothing, and Enter jumped away.

## Principle
One question per shipped task; frames attached so it can be judged without installing.

## Relevant details
Shipped: 5cjy88 (c05fb8d5 + 6ed006ff, CI 44894781379 8/8). Try it: brew upgrade veridit/tap/hug, then on statbus: / k5q120 (k5q120 is selected in LINT), Enter (results: 'found: / k5q120 · N matches · Esc all tasks', j/k walk every match across sections), Esc (all tasks, k5q120 still selected). Tab still keeps the search as a filter chip. One change to check: Esc no longer clears filter chips (/ or f); F clears all, ⌫ the last. Frames attached: before and after, typing, Enter, Esc.

## Depends on

## Acceptance
- [ ] Jørgen confirms the search flow and the Esc-keeps-chips change

## Questions
- 2026-11-21T11:43 question 1 by crab to jhf (check 1):
  v0.7.94: on statbus, / k5q120 now selects k5q120 in LINT; Enter shows the results and j/k walk every match; Esc returns to all tasks with k5q120 selected; Tab still keeps it as a filter. One change: Esc no longer clears filter chips (F clears all, ⌫ the last). Does search work the way you wanted, and is Esc keeping chips right?

## Notes

file:1-typing-k5q120.ansi (2602 bytes, sha256:52530af6) before (v0.7.93): typing k5q120 selects hm2dne

file:after-1-typing-k5q120.ansi (2481 bytes, sha256:8802fad2) after: 1-typing-k5q120

file:after-2-enter-results.ansi (2546 bytes, sha256:e4a0d203) after: 2-enter-results

file:after-3-esc-after-enter.ansi (3494 bytes, sha256:1310d9fe) after: 3-esc-after-enter


shipped: tag v0.7.94 (feedab54), release.yml 44895643755 success, brew -> ../Cellar/hug/0.7.94/bin/hug, hug v0.7.94
## History
- 2026-11-21T11:43 todo -> todo/design (ask)
