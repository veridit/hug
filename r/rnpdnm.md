---
id: rnpdnm
title: "Memory placement: extractor places personal/other projects, sync gate rule"
state: "doing"
type: feature
priority: "high"
assignee: ["bonehound"]
labels: []
group: ""
rank: "zzzzzzzzzzzc"
created: 2026-11-18
updated: 2026-11-18
doing_since: "2026-11-18T21:36"
stage: "active"
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
---

## Goal
A sharing repo pushes only memories both A+ and B pass (gav9nw, Jørgen 19:40Z). A+: the capture extractor places a memory as personal, project:<other> or client:<name> in its existing call (no extra paid call), from the catalog hug mem tidy place builds. B: a local, model-free gate at sync, beside LooksSecret (sync.go), holds any line that names a home path, another known project or client, the person, or is a preference/correction. Its term list lives in local config, never in the repo.

## Purpose
On hug's 868 public memories, 58 (6.7%) should not have been pushed. A+ and B together leaked 0 of 699 in the measurement (gav9nw OPTIONS.md), with 13.7% held back. A pushed memory can't be taken back, so a held one is the cheap mistake.

## Principle
Pushed only when both say push. B also covers memories written before it. A held line is released by rescope or a local exception, never by loosening the default. Confirm on fresh captures before any repo opts in (34r1ft).

## Relevant details
OPTIONS: /Users/jhf/.jcode/scratch/gav9nw/OPTIONS.md (scripts, labels, logs alongside). The A+ row was estimated by a labeler, not measured on the extractor itself. Term list: local config (hug's global config), seeded from the place catalog plus the person's names from [people]. hug mem sync reports how many it held and why, one line per reason.

## Depends on

## Acceptance
- [x] B: sync holds a line naming /Users/, ~/, a cataloged other project/client, or the person, and a preference/correction; the rest of the project scope pushes (table test); the term list is read from local config, not the repo (test)
- [ ] A+: the extractor's place field accepts personal, project:<other>, client:<name>; a captured personal memory is never in memories.jsonl (test with a recorded extraction)
- [ ] On fresh captures (not the 868 used to choose), A+ and B leak at most 1 in 200 pushed, judged blind; withheld rate reported with its interval
- [x] hug mem sync prints how many were held, by reason, and how to release one (fix: line)

## Notes

crab 21:36 local: dependency on gav9nw removed. The decision is written (8a1b0c05, DESIGN.md 'Personal vs shared memories', CI 44386929067 green) and accepted; it lands in the next stack.


B (checks 1, 4): a3790d1d on 8bf24c02, CI 44397313515 8/8 success. muninn/internal/project/gate.go Gate.Hold beside LooksSecret in syncFile: home folder (/Users/, ~/, /home/, C:\Users\), configured term as whole word with ø/æ/å folded, preference/correction. Terms in local muninn config.json sync_gate.terms (additive), seeded once from store places + hug.toml [people]; never printed. Sync output: 'held N: REASON' per reason, history note once for lines already pushed, 'fix: hug mem sync --held' (read-only; release named as a sentence for a person). Tests: project TestGateHoldsByRule (table: Jørgen's/Jorgen/jhf:/the user/C:\Users\//home/ hold; research, jhfx, internal/home/x.go, own name push), TestSyncGateWritesOnlyWhatPasses (fails without the gate call); run TestCLISyncGateHoldsAndReports (terms read from local config, not in output or git log -p hug/memory; fails with Gate nil), TestCLISyncSaysHeldLinesStayInHistory. Measured on the 868 (shipped Go gate, real seed, 0 paid calls): leak 13/705=1.8% [1.1,3.1], withheld 118/810=14.6% [12.3,17.2].
## History
- 2026-11-18T21:36 todo -> doing/active (take)
