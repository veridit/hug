---
id: 9gpaqr
title: "Release v0.7.70"
state: "done"
type: chore
priority: ""
assignee: ["crab"]
labels: []
group: ""
rank: "zzzzzzzzzzwn"
created: 2026-11-18
updated: 2026-11-18
model: "claude-opus-5-5"
effort: "high"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
done_at: "2026-11-18T22:10"
---

## Goal
v0.7.70 is published and installed on Jørgen's machine

## Purpose
Hourly release cadence: ship green main to Jørgen for live testing

## Principle
Exact-SHA all-green content only; never edit a released CHANGELOG section

## Relevant details
Main d5c66a1f (stack CI 44393153152 8/8): 34r1ft, 41hhgn c11, 0n4rnx replay + offline test, zhvkmj c5/c6, 33v71x, 05r3pg, yf39ya c2-c4, q2adr8. CHANGELOG c4eb6fed (zhvkmj and 05r3pg lines added in the release commit), Release merge 8a45b0eb, tag v0.7.70 on c4eb6fed.

## Depends on

## Acceptance
- [x] release.yml succeeds for v0.7.70
- [x] brew upgrade veridit/tap/hug; hug version reports v0.7.70

## Notes


check 1: release.yml 44396400817 completed success (first tag push started no run; re-pushing the same tag c4eb6fed did)
check 2: brew upgrade veridit/tap/hug; hug version: hug v0.7.70
Reflection (unanswered, 2026-11-18, crab)
## History
- 2026-11-18T22:10 todo -> doing/active (take)
- 2026-11-18T22:10 doing/active -> done (done); was doing_since: 2026-11-18T22:10
