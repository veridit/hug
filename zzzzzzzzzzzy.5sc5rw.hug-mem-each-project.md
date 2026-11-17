---
id: 5sc5rw
title: "hug mem: each project memory has a twin, and forget retires only one"
state: "doing"
type: feature
priority: "high"
assignee: ["hatchling"]
labels: []
created: 2026-11-17
updated: 2026-11-17
doing_since: "2026-11-17T22:24"
stage: "active"
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
---

## Goal
A project memory exists once on this machine, so recall shows it once and hug mem forget retires it fully. Today every capture in a repo project also appears as an imp_ row with origin repo:SCOPE/cap_ID: same scope, same text (2651 of 2651 pairs on this machine, 2571 both active).

## Purpose
66ex4y daily check 2026-11-17: forget xgxqs6 left its twin 49ckwr active, and search still returned it first (score 225.99); 4 twins outlived their source (ny39q0->z8mwnw, ce6say->80hygq retired by others; p48p3d->45m9zx, 6w0sn4->kda2dx by me) until I forgot them by hand. A retired wrong memory keeps being recalled.

## Principle
No capture-code change from the daily check; this is muninn's project-notes import. Fix the identity, not the symptom: a repo-notes row that came from this machine's own capture is that capture, not a new memory. Don't rewrite the store by hand.

## Relevant details
Store: ~/Library/Application Support/muninn/memories.json; imp_ ids from muninn/internal/ingest/importer.go importID(origin). Seid scope project:rc-3557a74eab224a4c has 1647 pairs, hug 555, ssb 449. While both are active, search showed one of the pair (fxa8yb shown, twin pq5c8b not, for 'Seid backlog retirement tracker clerk steps 1-3'), so the harm is after a forget: the twin takes its place. Found and filed by hatchling for crab.

## Depends on

## Acceptance
- [ ] a test: capture a memory in a project, sync the notes, re-import: the store holds one active row and search returns it once (fails before)
- [ ] hug mem forget ID retires the memory and any copy of it from the same origin; a test fails before
- [ ] on this machine, after hug upgrade or the next sync, 0 active pairs of a cap_ row and an imp_ row whose origin names it

## Notes


file:5sc5rw-bar.txt (1668 bytes, sha256:01535731) recall bar, written before any run
file:fail-before-1.txt (1116 bytes, sha256:b3004737)
file:fail-before-2.txt (135 bytes, sha256:6acc7f79)
file:5sc5rw-ab.txt (2351 bytes, sha256:19f294f8)
file:e2e.txt (1673 bytes, sha256:0acf909c)
landed 833eee1 on main 2026-11-17: hug mem: a project memory exists once; forget retires its copy; hug mem upgrade…
## History
- 2026-11-17T22:24 todo -> doing/active (take)
