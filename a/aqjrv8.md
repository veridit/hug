---
id: aqjrv8
title: "Release hug v0.7.59"
state: "dropped"
type: feature
priority: ""
assignee: ["crab"]
labels: []
group: ""
rank: "zzzzzzzzzzml"
created: 2026-11-18
updated: 2026-11-18
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
dropped_reason: "Obsolete: v0.7.59 release failed and was superseded by v0.7.60 and v0.7.61 (Jørgen's request 1)."
dropped_at: "2026-11-18T12:20"
---

## Goal
Publish and install all-green main348133a3 as v0.7.59 before layout3

## Purpose
Deliver bounded forget work and lower memory use with measured limitations

## Principle
No sorted recall behavior, no layout3 or machine migration, publish success before install, retain1.07s tail goal limitation

## Relevant details
MainCI44262023787 all6green; migration277 normalized responses identical. Worktree rel-7d600ff.

## Depends on

## Acceptance
- [ ] Tag/main pushed, release workflow success, installed hug version v0.7.59

## Requests
- 2026-11-18T12:19 request 1 by jhf:
  Isn't this obsolete now?
  resolved 2026-11-18T12:20 by crab:
    Yes, obsolete: the v0.7.59 tag's release failed and v0.7.60 superseded it (then v0.7.61). Dropped.

## Notes


Release44264072826 failed before build/publish: TestSettledWaitsForSlowChild sync_test.go193: settled returned before the pass pushed; remote head hug task new atj0m4. Installed remains hug v0.7.58. Installation chain stopped correctly. Wyvern assigned deterministic recurrence investigation; no fishing rerun, tag unchanged. Release task remains open.
## History
- 2026-11-18T09:05 todo -> doing/active (take)
- 2026-11-18T09:45 doing/active -> doing/waiting (wait)
- 2026-11-18T12:19 request 1 by jhf
- 2026-11-18T12:20 resolved request 1 by crab
- 2026-11-18T12:20 doing/waiting -> dropped (drop); was doing_since: 2026-11-18T09:05; waiting_asked_by: crab; waiting_on: v0.7.59 publication failed before build/publish; tag retained unchanged. Recovery v0.7.60 is tracked as 6np3a9; no successful v0.7.59 install claimed.; waiting_since: 2026-11-18T09:45
