---
id: agh2h7
title: "Board cue routes checks-all-hold to the owner, else the coordinator"
state: "doing"
type: feature
priority: ""
assignee: ["duckling"]
labels: []
group: ""
rank: "zzzzzzzzzzz8"
created: 2026-11-18
updated: 2026-11-18
doing_since: "2026-11-18T14:40"
stage: "active"
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
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

## Acceptance
- [x] A checks-all-hold finding reaches its owner's session as a board line naming hug task done ID (test)
- [ ] With the owner gone (no live session), the board coordinator gets it instead (test)

## Notes

duckling 13:33Z: piglet confirmed not started (DM 13:31Z). Worktree scratch/agh2h7 branch agh2h7 = b7517169 + cherry-picks of gpyqq9 (9ba1bf54, 8183ded0) + bg1rd8 (4bfd6797), CHANGELOG conflicts resolved keeping all entries.

BAR (duckling 13:40Z, before any run). Rule: checks-all-hold (problem 'every check holds, but ') becomes ownerOnly; a session that is not the owner gets it only when it is the board coordinator and no assignee has had a live jcode session within 2h (no coordinator on the board: any session at a break, as requestFor). Replay: same 7 sets as 6fqhg9 (run.sh), board4 with the lint clock pinned to the turn time and liveness from session file start/mtime; arm base = harness commit without routing, arm new = with routing; same judge prompt as base was judged with (e66eeb65 rubric), judge only lines new to an arm. PASS iff (a) in new every checks-all-hold line goes to an assignee or, owner gone, to crab: 0 to others; (b) six-set false(new) <= false(base) and right(new) >= right(base) - 1; (c) fresh hug false not up; (d) every coordinator line in new judged correct by both of 2 judgings, or there are none (then (a) and unit tests carry the change, said so). Judge calls counted per arm.


duckling: removed dep bg1rd8 to take it; the code is stacked on bg1rd8's commit (cherry-pick 4bfd6797), and crab lands bg1rd8 before this (crab DM 13:33Z: z82qsy, b7517169 stack, gpyqq9, bg1rd8, qpp30y). Rebase on main after bg1rd8 lands and drop the cherry-picks.
BAR AMEND (duckling 13:45Z, still before any run): with no coordinator on the board the gone-owner line goes to nobody (not any session): closing another's task is the coordinator's call. The hug board got coordinator=crab at 2026-11-18 02:05, after every replay window, so both arms replay hug sets with board4 -coordinator crab (counterfactual, same for both arms); seid sets have no coordinator. Lint clock pinned to turn time in both arms (else checks-all-hold fires on real time).
RUN 1 (13:51Z): base vs new on the 7 pinned sets: byte-identical outputs, 217 lines each, 0 'every check holds' lines in either (windows end 2026-11-17 14:00; 44w1ev waited 11-17 09:00 to 11-18 13:02). Bar (b),(c) hold trivially, (a),(d) vacuous. ADDED before running it: supplementary set fh3 = hug sessions with turns 2026-11-17T14:00Z..2026-11-18T12:00Z, -coordinator crab, both arms; judge only lines that differ (2 judgings). Not one of the 6 sets; reported apart.
RUN 2 fh3 (supplementary, 618 hug sessions with turns 2026-11-17T14:00Z..11-18T12:00Z, -coordinator crab): 44 lines each arm; 2 differ. base gave 44w1ev's checks-all-hold to calf (06:45Z) and crab (07:20Z); judge A wrong 2/2, judge B wrong 2/2. new gives those turns' lint slot to other findings (zhvkmj, rxr4z8 'stage asked disagrees'): A correct 2/2, B wrong 2/2. new keeps 44w1ev for wyvern (3 lines, 16:07Z, 01:52Z, 08:44Z, both arms). Coordinator route fired 0 times in replay (wyvern stayed live in the replay's liveness until 44w1ev closed), so bar (d) is carried by TestBoardChecksHoldOwnerElseCoordinator (mutation-checked: dropping ownerOnly fails 3 cases, disabling pickGone fails 2). Judge calls: 4 (2 judgings x 1 batch). Bar: (a) 0 to others PASS; (b) six sets identical PASS; (c) fresh hug identical PASS; (d) no coordinator lines, said so.
## History
- 2026-11-18T14:40 todo -> doing/active (take)
