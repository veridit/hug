---
id: tpnd3n
title: "An unrelated lint finding waits until the session's own task has none"
status: "doing"
type: feature
priority: ""
assignee: ["hatchling"]
labels: []
created: 2026-11-17
updated: 2026-11-17
doing_since: "2026-11-17T15:10"
stage: "active"
model: "claude-opus-5-5"
branch: "ci/tpnd3n"
worktree: "/Users/jhf/.jcode/scratch/guppy-lint"
---

## Goal
An unrelated lint finding waits until the session's own task has no open finding (instead of until the session is between tasks), and fresh lint lines do not drop. Rewrite of x325pp check 1 (crab ruling B, 2026-11-17). Promoted from item 1 of task:x325pp.

## Purpose
In the qawmnb replay 5 of the 7 false board lines were lint findings handed to agents busy with other work (muninn/docs/BOARD4.md), one of them just told to stop

## Principle
Measure on replayed real sessions before shipping (board4 + blind judge); keep cleaning the board over time, do not silence lint; no paid calls in the hook

## Relevant details
muninn/internal/recall/boardcue.go pickFinding/BoardCue; replay muninn/cmd/board4, judge muninn/scripts/judge_board.py; false lint lines: hog p39w4a (stop order), orangutan zjedy0, pawprint 2n92r7, piglet n0fzz1, sunflower qaxx25

## Depends on

## Acceptance
- [ ] an unrelated lint finding is handed out only once the session's own doing task has no open finding (test)
- [ ] replay (board4, blind judge): false lint lines on held-out fall from the stop-order arm's 1 and fresh lint lines stay at 13 of 13, counts in muninn/docs/BOARD4.md
- [ ] a finding withheld on a stop turn is offered at the session's next break or to the next session: in the fresh replay skunk's w96jpg long-title line (main arm 2026-11-11T18:13:45Z) arrives later instead of being lost (x325pp check 3)

## Notes

hatchling 12:42, from x325pp: measured arms in muninn/docs/BOARD4.md 'Stop orders, and lint between tasks'. Lint lines right/false: main 97: 87/7; stop order 95: 89/3; + between tasks 81: 73/5; + busy when editing 73: 65/5. fresh2 seid: 8: 8/0 -> between tasks 3: 0/3. Raw: $JCODE_SCRATCH_DIR/board4/x325pp/ (main/ stop/ new/ newb/, judgments.json, score2.py).

hatchling 12:58: carries x325pp check 3 too (crab ruling): the stop-order rule loses skunk's w96jpg finding in fresh, details in the x325pp note 12:58.


## History
- 2026-11-17T15:10 todo -> doing/active (take)
