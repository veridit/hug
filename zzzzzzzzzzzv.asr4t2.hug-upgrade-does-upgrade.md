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
- [ ] hug upgrade previews by default (changes nothing, ends fix: hug upgrade --apply, exits 5 with changes pending, 0 with none); --apply does it; --dry-run is gone; hug task upgrade the same (Jørgen 18:58, test fails on main first)
- [ ] ~~plain hug upgrade previews (nothing changes, ends fix: hug upgrade --apply, exit 5 pending, 0 none); --apply does it; --dry-run gone; hug task upgrade the same (test fails on main first)~~ (not relevant: duplicate of check 3 (added twice), nautilus, 2026-11-17)

## Notes

Jørgen 17:54Z: hug upgrade previews by default; acting needs an explicit flag. crab ruling: 'hug upgrade' prints what it would change per part and ends with 'fix: hug upgrade --apply'; exits 0 when nothing to do, and a distinct non-zero code when changes are pending so scripts notice; 'hug upgrade --apply' does the work. --dry-run is removed (no backward compatibility). The same for 'hug task upgrade'. Messages that say 'next: hug upgrade' stay correct (it is now the preview).

landed f4ee48b on main (dcb20c2..f4ee48b). check 1: TestUpgradeMigratesStateKey (old status: board: upgrade prints 'tasks: 1 tasks moved to state: (was status:)', commits 'hug upgrade: status: -> state: in 1 tasks') and TestUpgradeMovesWaitingFor, TestUpgradeReordersSections. check 2: same test, second run equals 'layout: nothing to do; tasks: nothing to do; variables: nothing to do; memory: nothing to do'. -race -count=1 huginn/internal/cli ok 135.9s. Seid copy: 2520 tasks, see 6ephqk file:seid-upgrade-dry-run.md.
