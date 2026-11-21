---
id: kka3b1
title: "Release v0.7.91: r sends one request for the marked tasks"
state: todo
type: chore
priority: ""
assignee: []
labels: []
group: ""
rank: "zzzzzzzzzzzo"
created: 2026-11-20
updated: 2026-11-20
design_question: "v0.7.91: with rows marked, r opens one request box under the list and Enter sends it for exactly those rows (frames attached; try it on your 4 LINT rows). Is that what you wanted?"
design_for: "jhf"
design_since: "2026-11-20T18:22"
design_asked_by: "crab"
stage: "design"
---

## Goal
Jørgen confirms v0.7.91: r on marked rows opens one request for exactly that set, box under the list.

## Purpose
He marked four LINT rows and had no way to ask about them together.

## Principle
One question per shipped task; frames attached so it can be judged without installing.

## Relevant details
Shipped: ec6z6j checks 4 and 5 (d3b79155, CI 44780705119 8/8). Try it: brew upgrade veridit/tap/hug, hug task tui, Space on 2-3 rows in the lint view (!), then r, write, Enter. Frames attached: before and after at 80x24.

## Depends on

## Acceptance
- [ ] Jørgen confirms r on marked rows works as he wanted

## Questions
- 2026-11-20T18:22 question 1 by crab to jhf (check 1):
  v0.7.91: with rows marked, r opens one request box under the list and Enter sends it for exactly those rows (frames attached; try it on your 4 LINT rows). Is that what you wanted?

## Notes

file:before-lint-3marked-marked.ans (5739 bytes, sha256:231e255d) before (v0.7.90): 3 rows marked, no request in the footer

file:after-lint-3marked-marked.ans (5796 bytes, sha256:41c28c0b) after: footer offers request

file:after-lint-3marked-r.ans (5076 bytes, sha256:a6ca6dea) after: r opened the box under the list, marked rows in view


shipped: tag v0.7.91 (00a06a61), release.yml 44782605940 success, brew upgrade -> ../Cellar/hug/0.7.91/bin/hug, hug v0.7.91
## History
- 2026-11-20T18:22 todo -> todo/design (ask)
