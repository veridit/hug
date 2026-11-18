---
id: agh2h7
title: "Board cue routes checks-all-hold to the owner, else the coordinator"
state: todo
type: feature
priority: ""
assignee: []
labels: []
group: ""
rank: "zzzzzzzzzzzt"
created: 2026-11-18
updated: 2026-11-18
---

## Goal
The recall board cue gives a checks-all-hold finding (bg1rd8: a waiting task whose every check and child is done) to the task's owner, or to the board coordinator when the owner has no live session, with its fix hug task done ID.

## Purpose
bg1rd8 check 2, split off: the lint finding exists (branch bg1rd8) but only reaches a session through the generic lint line; 44w1ev sat a day in waiting because nobody was told.

## Principle
Build on 4vxqp4 (duckling, ci/4vxqp4: ownerOnlyPrefixes in boardcue.go ~line 616, d.mention). Measured by board4 replay like other board-cue changes; no change ships without a CI-clear gain.

## Relevant details
Add the prefix 'every check holds, but ' to ownerOnlyPrefixes; the coordinator fallback is new logic in pickFinding (requestFor() is the closest pattern). Agreed with duckling 2026-11-18 12:15Z: duckling does not add it in 4vxqp4.

## Depends on

- bg1rd8
## Acceptance
- [ ] A checks-all-hold finding reaches its owner's session as a board line naming hug task done ID (test)
- [ ] With the owner gone (no live session), the board coordinator gets it instead (test)

## Notes

duckling 13:33Z: piglet confirmed not started (DM 13:31Z). Worktree scratch/agh2h7 branch agh2h7 = b7517169 + cherry-picks of gpyqq9 (9ba1bf54, 8183ded0) + bg1rd8 (4bfd6797), CHANGELOG conflicts resolved keeping all entries.

BAR (duckling 13:40Z, before any run). Rule: checks-all-hold (problem 'every check holds, but ') becomes ownerOnly; a session that is not the owner gets it only when it is the board coordinator and no assignee has had a live jcode session within 2h (no coordinator on the board: any session at a break, as requestFor). Replay: same 7 sets as 6fqhg9 (run.sh), board4 with the lint clock pinned to the turn time and liveness from session file start/mtime; arm base = harness commit without routing, arm new = with routing; same judge prompt as base was judged with (e66eeb65 rubric), judge only lines new to an arm. PASS iff (a) in new every checks-all-hold line goes to an assignee or, owner gone, to crab: 0 to others; (b) six-set false(new) <= false(base) and right(new) >= right(base) - 1; (c) fresh hug false not up; (d) every coordinator line in new judged correct by both of 2 judgings, or there are none (then (a) and unit tests carry the change, said so). Judge calls counted per arm.
