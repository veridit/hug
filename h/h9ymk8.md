---
id: h9ymk8
title: "AGENTS.md names HUGINN_UPDATE_GOLDEN; the tests read HUG_UPDATE_GOLDEN"
state: todo
type: docs
priority: ""
assignee: []
labels: []
group: ""
rank: "zzzzzzzzzzzh"
created: 2026-11-19
updated: 2026-11-19
---

## Goal
AGENTS.md and any doc naming the golden update variable give the one the tests read.

## Purpose
tulip 2026-11-19 (a82j3n): the task and AGENTS.md said HUGINN_UPDATE_GOLDEN=1, the TUI tests read HUG_UPDATE_GOLDEN; tulip set both to update goldens.

## Principle
One name. Check with git grep which the code reads; fix the docs, no alias unless the code reads both.

## Relevant details
AGENTS.md 'TUI goldens' line; git grep -n 'UPDATE_GOLDEN'.

## Depends on

## Acceptance
- [ ] git grep UPDATE_GOLDEN shows docs and code naming the same variable (note with output)

## Notes
