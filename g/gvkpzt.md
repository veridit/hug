---
id: gvkpzt
title: "Release v0.7.87"
state: "doing"
type: feature
priority: ""
assignee: ["crab"]
labels: []
group: ""
rank: "zzzzzzzzzzzj"
created: 2026-11-20
updated: 2026-11-20
doing_since: "2026-11-20T06:14"
doing_with: "p277bp"
stage: "asked"
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
asked_by: "crab"
asked_what: "z6cb03: on statbus or sql_saga, where v0.7.86 added many long-Goal advice rows, open hug tui (restart it first: your TUIs there run 0.7.69 and 0.7.68). Are the broken tasks at the top of LINT, in view?"
asked_for: "jhf"
asked_since: "2026-11-20T06:14"
---

## Goal
v0.7.87 (broken lint before advice) is published, installs via brew, and Jørgen says whether broken tasks stay in view.

## Purpose
Release-as-group trial (75mxf3): one ticket groups what ships and asks one question per user-facing change.

## Principle
Shipped tickets are children; one ask per user-facing change.

## Relevant details
main 51e36090 (CI 44677696260, 8/8), CHANGELOG 57113899, merge 6fa18387, release.yml 44678753259 success. Ships z6cb03: hug tui's LINT section and lint view list broken tasks first, rank within each; needed after v0.7.86 added goal-too-long advice.

## Depends on

## Acceptance
- [x] release.yml green and brew upgrade gives hug version 0.7.87 with links into the new Cellar

## Questions
- 2026-11-20T06:14 question 1 by crab to jhf:
  z6cb03: on statbus or sql_saga, where v0.7.86 added many long-Goal advice rows, open hug tui (restart it first: your TUIs there run 0.7.69 and 0.7.68). Are the broken tasks at the top of LINT, in view?

## Notes


check 1: release.yml 44678753259 success; brew upgrade: hug v0.7.87, hug/muninn/huginn -> ../Cellar/hug/0.7.87/bin/*
## History
- 2026-11-20T06:14 todo -> doing/active (take with p277bp)
- 2026-11-20T06:14 doing/active -> doing/asked (ask)
