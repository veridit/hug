---
id: 3m6byj
title: "Release v0.7.94: / search lands on the task you typed"
state: todo
type: chore
priority: ""
assignee: []
labels: []
group: ""
rank: "zzzzzzzzzzzy"
created: 2026-11-21
updated: 2026-11-21
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

## Notes
