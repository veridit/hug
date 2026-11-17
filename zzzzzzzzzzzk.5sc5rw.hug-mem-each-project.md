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
stage: "waiting"
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
waiting_asked_by: "hatchling"
waiting_for: "crab"
waiting_on: "v0.7.49 installed (release of 833eee1), then check 3 via hug upgrade"
waiting_since: "2026-11-17T23:20"
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
- [x] a test: capture a memory in a project, sync the notes, re-import: the store holds one active row and search returns it once (fails before)
- [x] hug mem forget ID retires the memory and any copy of it from the same origin; a test fails before
- [x] on this machine, after hug upgrade or the next sync, 0 active pairs of a cap_ row and an imp_ row whose origin names it

## Notes


file:5sc5rw-bar.txt (1668 bytes, sha256:01535731) recall bar, written before any run
file:fail-before-1.txt (1116 bytes, sha256:b3004737)
file:fail-before-2.txt (135 bytes, sha256:6acc7f79)
file:5sc5rw-ab.txt (2351 bytes, sha256:19f294f8)
file:e2e.txt (1673 bytes, sha256:0acf909c)
landed 833eee1 on main 2026-11-17: hug mem: a project memory exists once; forget retires its copy; hug mem upgrade…
Landed 833eee1 on main. Cause: sync's MergeImport keyed note rows by origin repo:SCOPE/ID, and a capture keeps its capture origin, so each sync re-imported this machine's own capture as imp_ (origin repo:SCOPE/cap_ID). Fix: Imported carries the note row's ID and MergeImport maps it to the memory with that id in the same scope; ForgetMany also retires a twin; hug mem upgrade (hug upgrade's memory: part) folds existing pairs after a backup. Check 1: TestSyncReadsOwnCaptureBackAsTheSameMemory (muninn/internal/project/twins_test.go) fails on origin/main 58fcd3c, file:fail-before-1.txt, passes at 833eee1. Check 2: TestForgetLeavesNoTwinSearchable (muninn/run/twins_test.go) fails before: 'after forget hzwm7m, search still finds [d2mn4c]', file:fail-before-2.txt. Also TestFoldTwinsKeepsTheLocalMemory, TestMemUpgradeFoldsTwins, TestUpgradeMemoryPartRunsMemUpgrade. -race niced on cli, task, ingest, project, muninn/run, cmd/hug: all ok after rebase.
Recall A/B vs the bar (file:5sc5rw-bar.txt, written before the run), judged blind, 0 unjudged: rel/msg all 277 A 0.751 B 0.751, delta +0.000 [+0.000, +0.000] (bar 1 pass); messages injecting a memory and its twin A 0 B 0, folded imp_ twins injected A 5 B 0 (bar 2 pass); relevant-in-A texts missing from B: 0 (bar 3 pass). Side check: frozen recall3 snapshots have 0 twins, injected hits identical in 277 of 277 (only .elapsed differs). file:5sc5rw-ab.txt. End to end on a scratch copy of today's live store (HUG_HOME, live sha a10ccb71 unchanged): preview exit 5 lists 449 + 555 + 1647; --apply folded 2651, backup written; after: 16310 -> 13659 memories, 0 pairs, active (scope,text) 13251 before and after, 0 lost; ids removed 2651, all imp_. file:e2e.txt
crab 22:20Z: check 3 runs through hug upgrade with the released v0.7.49 (crab says when it is installed, ~23:00Z if CI 44188866860 is green): preview lists 2651, then --apply, then verify 0 pairs and the active count.
CI 44188866860 on 833eee1: success, all 5 jobs (secrets, bench, test ubuntu/macos/windows).
shipped in v0.7.49 (833eee1)
file:check3.txt (1568 bytes, sha256:f9f39901)
Check 3 on the live store with brew hug v0.7.49 (23:02Z, no hug mem serve running). Before: memories 16310, active 15821, twin pairs 2651, both active 2571. 'hug upgrade' preview: 'memory: would fold 449 ... client:ssb:rc-898d04734041ab10', '555 ... project:rc-0ca4bf6be5d989c3', '1647 ... project:rc-3557a74eab224a4c', exit 5, store unchanged. 'hug upgrade --apply': folded 449/555/1647, backup memories.json.bak-upgrade-20261117T230203Z, exit 0. After: memories 13659, active 13250, twin pairs 0, both active 0; active (scope,text) 13250 before and after, 0 lost; ids removed 2651, all imp_, 0 added. Index: 13250 active native units, 0 not in memories.json. Second 'hug upgrade': 'memory: nothing to do', exit 0. file:check3.txt
## History
- 2026-11-17T22:24 todo -> doing/active (take)
- 2026-11-17T23:20 doing/active -> doing/waiting (wait)
