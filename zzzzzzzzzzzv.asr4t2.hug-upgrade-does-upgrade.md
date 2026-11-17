---
id: asr4t2
title: "hug upgrade does the upgrade: runs hug task upgrade and hug mem's, then says what changed"
status: todo
type: bug
priority: "medium"
assignee: ["nautilus"]
labels: []
created: 2026-11-17
updated: 2026-11-17
---

## Goal
hug upgrade runs every mechanical migration itself (hug task upgrade, and any hug mem migration), prints what it changed per board ('moved 12 tasks to state', or 'nothing to do'), and only then points at hug setup for choices.

## Purpose
Seid foreman (orangutan) requests 2026-11-17, file: seid-lang .jcode/scratch/TASK-2057/hug-agent-requests.md, read by crab 15:07, item 6. hug upgrade printed only 'hug v0.7.41' and 'next: hug setup'; the work was in hug task upgrade, so it looked as if the upgrade had run.

## Principle
One command for one intent; output says what was done. hug upgrade is mechanical only (ruling 2026-11-17).

## Relevant details
cmd/hug/main.go:62 dispatches setup, tui, upgrade. 6ephqk step 4 (status -> state) adds a migration that must run through hug upgrade.

## Depends on

## Acceptance
- [x] on a board in the old layout, hug upgrade migrates it and prints what changed (test)
- [x] on a current board it prints 'nothing to do' per part (test)

## Notes

Jørgen 17:54Z: hug upgrade previews by default; acting needs an explicit flag. crab ruling: 'hug upgrade' prints what it would change per part and ends with 'fix: hug upgrade --apply'; exits 0 when nothing to do, and a distinct non-zero code when changes are pending so scripts notice; 'hug upgrade --apply' does the work. --dry-run is removed (no backward compatibility). The same for 'hug task upgrade'. Messages that say 'next: hug upgrade' stay correct (it is now the preview).
