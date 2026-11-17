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
- [x] an unrelated lint finding is handed out only once the session's own doing task has no open finding (test)
- [ ] ~~replay (board4, blind judge): false lint lines on held-out fall from the stop-order arm's 1 and fresh lint lines stay at 13 of 13, counts in muninn/docs/BOARD4.md~~ (not relevant: baseline moved: fkpyg3 lint raised main's lint lines 49 -> 68 on the same sets; measured instead (BOARD4.md tpnd3n): held-out false lint 8 -> 7, fresh+fresh2 lint 14 -> 14 kept, right 11 -> 12; all sets false 23 -> 12, hatchling, 2026-11-17)
- [x] a finding withheld on a stop turn is offered at the session's next break or to the next session: in the fresh replay skunk's w96jpg long-title line (main arm 2026-11-11T18:13:45Z) arrives later instead of being lost (x325pp check 3)

## Notes

hatchling 12:42, from x325pp: measured arms in muninn/docs/BOARD4.md 'Stop orders, and lint between tasks'. Lint lines right/false: main 97: 87/7; stop order 95: 89/3; + between tasks 81: 73/5; + busy when editing 73: 65/5. fresh2 seid: 8: 8/0 -> between tasks 3: 0/3. Raw: $JCODE_SCRATCH_DIR/board4/x325pp/ (main/ stop/ new/ newb/, judgments.json, score2.py).

hatchling 12:58: carries x325pp check 3 too (crab ruling): the stop-order rule loses skunk's w96jpg finding in fresh, details in the x325pp note 12:58.


hatchling 14:29: replay (scratch/board4/x325pp, tpnd.sh, judge_board.py blind, judgments3.json) on today's boards, 6 sets: main 38ff464 (stop2) 128 lines, false 23 (lint 20), precision 0.811; tpnd3n rule as specified (tpnd) 127 lines, false 23 (lint 20): no change, the rule only reorders. Pre-fkpyg3 binary 206038a on the same boards (old): 109 lines, false 7 (lint 4), precision 0.934. So fkpyg3 (section lint) put +19 lint lines and +16 false on the replay: 'missing section Principle/Purpose' on unrelated todo tasks (dev: 8 false of 16 lint; ho-hug 3 of 4: s6ff5k/vk9td0 named while the user forbade other tasks). Testing variant: unrelated findings exclude 'missing section' (only on touched tasks).
hatchling 14:46: check 1: TestBoardUnrelatedLintWaitsForOwnTask (fails without the rule: 'unrelated finding while own task has one'). Added TestBoardMissingSectionOnlyToItsSession: without ownerOnly it fails 'boardcue_test.go:277: section finding to a passer-by: "Task board (return): s6ff5k: missing section Purpose (why)..."'. Check 3: TestBoardStopTurnKeepsFinding passes (stop turn writes nothing to board_lint, so the next break / next session gets w96jpg). Replay: see BOARD4.md 'Missing sections go to the task's own session': six sets lint main 68: 43 right / 20 false (5) -> 55: 45 / 9 (1); all lines false 23 -> 12. Rejudged the 42 main-only and 29 rule-only lines twice from empty caches: main 17 and 15 false, rule 0 and 0 false. Fresh hug (367 sessions after 2026-11-16 20:00, 38.8 h) main 4 lint (3 right, 1 unclear) -> 3 (3 right). Commit d708fb4 on ci/tpnd3n, CI 44106642855.
## History
- 2026-11-17T15:10 todo -> doing/active (take)
