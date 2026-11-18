---
id: asr4t2
title: "hug upgrade does the upgrade and says what changed, per part"
state: "done"
type: bug
priority: "medium"
assignee: ["nautilus"]
labels: []
group: ""
rank: "zzzzzzzzzze"
created: 2026-11-17
updated: 2026-11-17
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
done_at: "2026-11-17T19:25"
done_release: "v0.7.45"
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
- [x] hug upgrade previews by default (changes nothing, ends fix: hug upgrade --apply, exits 5 with changes pending, 0 with none); --apply does it; --dry-run is gone; hug task upgrade the same (Jørgen 18:58, test fails on main first)
- [ ] ~~plain hug upgrade previews (nothing changes, ends fix: hug upgrade --apply, exit 5 pending, 0 none); --apply does it; --dry-run gone; hug task upgrade the same (test fails on main first)~~ (not relevant: duplicate of check 3 (added twice), nautilus, 2026-11-17)

## Notes

Jørgen 17:54Z: hug upgrade previews by default; acting needs an explicit flag. crab ruling: 'hug upgrade' prints what it would change per part and ends with 'fix: hug upgrade --apply'; exits 0 when nothing to do, and a distinct non-zero code when changes are pending so scripts notice; 'hug upgrade --apply' does the work. --dry-run is removed (no backward compatibility). The same for 'hug task upgrade'. Messages that say 'next: hug upgrade' stay correct (it is now the preview).

landed f4ee48b on main (dcb20c2..f4ee48b). check 1: TestUpgradeMigratesStateKey (old status: board: upgrade prints 'tasks: 1 tasks moved to state: (was status:)', commits 'hug upgrade: status: -> state: in 1 tasks') and TestUpgradeMovesWaitingFor, TestUpgradeReordersSections. check 2: same test, second run equals 'layout: nothing to do; tasks: nothing to do; variables: nothing to do; memory: nothing to do'. -race -count=1 huginn/internal/cli ok 135.9s. Seid copy: 2520 tasks, see 6ephqk file:seid-upgrade-dry-run.md.


check 3 landed 317f5c2 on main (f4ee48b..317f5c2). TestUpgradePreviewsThenApplies failed on main first: 'plain upgrade: exit 0, want 5 ending fix: hug upgrade --apply'; now ok. Binary run on a scratch board: plain 'hug upgrade' prints 'tasks: would: 1 tasks moved to state: (was status:)' + diff, ends 'fix: hug upgrade --apply', exit 5; 'hug task upgrade' exit 5; '--dry-run' exit 2 (unknown flag); '--apply' exit 0, 'tasks: 1 tasks moved...'; again: nothing to do x4, exit 0. -race -count=1 cli ok 131.9s; cmd/hug, mcp, task ok.
landed 317f5c2 on main 2026-11-17: hug upgrade previews by default: it changes nothing, ends with fix: hug upgrade…
Reflection (applied, 2026-11-17, nautilus): hug task set --accept 'VALUE' added the check but the reply ended with a lint fix about a long title, which read as if the set had failed, so I added the check twice and struck the duplicate. A set that wrote should say what it wrote first, before lint advice.
shipped in v0.7.45 (f4ee48b)
file:seid-upgrade-dry-run.md (93062 bytes, sha256:2e7b25b6) copied from 6ephqk (crab, missing-file)
Reflection 1 applied (tb0dra check 5 triage, crab 2026-11-19): G1, a reply hides what the command did: task zdgw60 filed (set says what it wrote before lint advice).
## History
- 2026-11-17T19:12 todo -> doing/active (take)
- 2026-11-17T19:25 doing/active -> done (done); was doing_since: 2026-11-17T19:12
