---
id: 5sc5rw
title: "hug mem: each project memory has a twin, and forget retires only one"
state: todo
type: feature
priority: ""
assignee: []
labels: []
created: 2026-11-17
updated: 2026-11-17
---

## Goal
A project memory exists once on this machine, so recall shows it once and hug mem forget retires it fully. Today every capture in a repo project also appears as an imp_ row with origin repo:SCOPE/cap_ID: same scope, same text (2651 of 2651 pairs on this machine, 2571 both active).

## Purpose
66ex4y daily check 2026-11-17: forget xgxqs6 left its twin 49ckwr active, and search still returned it first (score 225.99); 4 twins outlived their source (ny39q0->z8mwnw, ce6say->80hygq retired by others; p48p3d->45m9zx, 6w0sn4->kda2dx by me) until I forgot them by hand. A retired wrong memory keeps being recalled.

## Principle
No capture-code change from the daily check; this is muninn's project-notes import. Fix the identity, not the symptom: a repo-notes row that came from this machine's own capture is that capture, not a new memory. Don't rewrite the store by hand.

## Relevant details
Store: ~/Library/Application Support/muninn/memories.json; imp_ ids from muninn/internal/ingest/importer.go importID(origin). Seid scope project:rc-3557a74eab224a4c has 1647 pairs, hug 555, ssb 449. hug mem search also ranks a memory with its twin, crowding out a distinct third result.

## Depends on

## Acceptance
- [ ] a test: capture a memory in a project, sync the notes, re-import: the store holds one active row and search returns it once (fails before)
- [ ] hug mem forget ID retires the memory and any copy of it from the same origin; a test fails before
- [ ] on this machine, after hug upgrade or the next sync, 0 active pairs of a cap_ row and an imp_ row whose origin names it

## Notes
