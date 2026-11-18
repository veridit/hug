---
id: 9ahvgt
title: "stale-owner lint misses owners whose sessions are gone"
state: "doing"
type: bug
priority: ""
assignee: ["iwazaru"]
labels: []
group: ""
rank: "zzzzzzzzzzz3"
created: 2026-11-18
updated: 2026-11-18
doing_since: "2026-11-18T13:44"
stage: "active"
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
---

## Goal
hug task lint reports stale-owner for every doing task whose assignee has had no live harness session for over a day, including workers spawned in other jcode homes and names with no session file at all.

## Purpose
retriever 2026-11-18 12:34Z: lint --code stale-owner said 'no findings' while 26+ doing tasks belonged to gone sessions: hatchling 6, nautilus 11 (journal last 07:46), kestrel 8 (no session file), wyvern 1. Requests and answers then reach nobody.

## Principle
A finding must reflect what hug can actually see; when it cannot see sessions (unreadable home) it says so once, rather than staying silent.

## Relevant details
Code: huginn/internal/task/owner.go:95 builds the finding; it reads ~/.jcode/sessions only when readable. Check: does it match assignee names to session names (session_<name>_<ts>), and how does it treat a name with no session file (kestrel)? Scratch: /Users/jhf/.jcode/scratch/board-orphans/REPORT.md.

## Depends on

## Acceptance
- [x] On hug's board, lint reports stale-owner for nautilus, kestrel and hatchling tasks; a test with a gone owner and with an owner that has no session file fails first on main.
- [x] A live worker's task is not reported (test).

## Notes


Diagnosis (iwazaru, from ~/.jcode/sessions and logs). (1) owner.go judges by mtime only, with a 24 h threshold. nautilus' newest file is 07:46 today and hatchling's 06:47, so neither is past 24 h. Yet both sessions are gone: the last journal line of each has meta status Closed, and jcode-2026-11-16.log:1492534 reads 'Reaped idle spawned swarm worker session_nautilus_1794905151412_33a1e4b1294b58f5 (idle > 1800s)'. hug never reads the status. Of the 75 journals, 53 end Closed and 22 end Active, and the 22 Active ones are the 11 live swarm workers plus others. (2) LivenessFrom reads only session_*.json, not the .journal.jsonl that is written later (nautilus json 07:07, journal 07:46). (3) kestrel has no session file in any home (global, brovs, GenderedGames, truth_tribe_and_treasure, seid, statbus). Session calf's title is 'You are kestrel, a worker on hug', so calf took bxqtkt --as kestrel, and OwnerFindings skips any name the session list does not know. 'engineer' (xfv9pw: my own take --as engineer) and 'crab-engineer' (4vxqp4, duckling) are the same shape.
Failing-first on origin/main 77d25855: TestOwnerFindingsGoneSessions 'owner_gone_test.go:74: zhvkmj not reported (its owner is gone); findings: map[]' (also tb0dra, bxqtkt); TestLintStaleOwnerSessions (cli) 'staleowner_test.go:64: nautilus's task gz0epr not reported; findings map[]' + kestrel, wyvern, and 'staleowner_test.go:89: unreadable sessions: warnings []'. Real board, read-only, branch build: hug task lint --code stale-owner gives 22 findings (hatchling 5: h00cs2 n7jbmx s6ff5k tb0dra z9q42h; nautilus 9: 0gztsy 1bbe96 4s3g5q 5z4zt4 ahejza qh9ep1 rxr4z8 tmz060 zhvkmj; kestrel 6 '(kestrel was session calf)': 41hhgn bxqtkt ggydj2 hbgybn mbdfzs z2vmte; wyvern 1 jr55s8; puppy 1 911j70). The installed hug v0.7.62 on the same board: 'no findings with code stale-owner'. Owners flagged: exactly hatchling, kestrel, nautilus, puppy, wyvern; none of the 11 live workers (macaque, tigress, duckling, kikazaru, iwazaru, bonehound, piglet, sabertooth, mizaru, crab).
## History
- 2026-11-18T13:44 todo -> doing/active (take)
