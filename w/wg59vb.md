---
id: wg59vb
title: "Release v0.7.98: a task's page sits between two rules"
state: todo
type: feature
priority: ""
assignee: []
labels: []
group: ""
rank: "zzzzzzzzzzzb"
created: 2026-11-21
updated: 2026-11-21
design_question: "v0.7.98 is installed. Open a long task (for example qpp30y) and press J a few times: the rule under the title gets ↑, and the bottom rule right on the keys reads '↓── Task 1 of 2 · 29% · next · previous · Esc back'. Press t on a todo task, and the message '• ID moved to IN PROGRESS' takes the status's place. Does it read right, or what should change?"
design_for: "jhf"
design_since: "2026-11-21T19:23"
design_asked_by: "crab"
stage: "design"
---

## Goal
Jørgen tries v0.7.98 on a real board and says whether a task's page frame reads right.

## Purpose
One release ticket per release. v0.7.98 = main 8611f2b6, landed a19cd1d4 on exact-SHA CI 8/8, release.yml success, brew-verified (Cellar/hug/0.7.98, hug v0.7.98).

## Principle
Ask one thing; frames are on 4ddcwa; his answer there settles xt0qvf for the task page.

## Relevant details
Shipped: 4ddcwa checks 2 and 3 (goldens 100x30 and 160x45, 10 mutants caught). The status line and its blank line are gone, so the page gets two rows. Lists keep their message line (xt0qvf). TestTmuxAutoRefresh flaked once: filed as tfe9nr.

## Depends on

## Acceptance
- [ ] Jørgen opens a long task in v0.7.98 and says the frame reads right, or what to change

## Questions
- 2026-11-21T19:23 question 1 by crab to jhf (check 1):
  v0.7.98 is installed. Open a long task (for example qpp30y) and press J a few times: the rule under the title gets ↑, and the bottom rule right on the keys reads '↓── Task 1 of 2 · 29% · next · previous · Esc back'. Press t on a todo task, and the message '• ID moved to IN PROGRESS' takes the status's place. Does it read right, or what should change?

## Notes


## History
- 2026-11-21T19:23 todo -> todo/design (ask)
