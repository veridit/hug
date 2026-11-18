---
id: agh2h7
title: "Board cue routes checks-all-hold to the owner, else the coordinator"
state: todo
type: feature
priority: ""
assignee: []
labels: []
group: ""
rank: "zzzzzzzzzzzx"
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

## Acceptance
- [ ] A checks-all-hold finding reaches its owner's session as a board line naming hug task done ID (test)
- [ ] With the owner gone (no live session), the board coordinator gets it instead (test)

## Notes
