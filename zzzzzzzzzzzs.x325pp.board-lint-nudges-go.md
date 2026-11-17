---
id: x325pp
title: "Board lint nudges go to agents at a break in their own work"
status: "doing"
type: feature
priority: ""
assignee: ["hatchling"]
labels: []
created: 2026-11-16
updated: 2026-11-16
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
---

## Goal
A lint finding from the recall hook reaches an agent when it can act on it without leaving its task: never to an agent told to stop, and an unrelated finding only when the agent is between tasks

## Purpose
In the qawmnb replay 5 of the 7 false board lines were lint findings handed to agents busy with other work (muninn/docs/BOARD4.md), one of them just told to stop

## Principle
Measure on replayed real sessions before shipping (board4 + blind judge); keep cleaning the board over time, do not silence lint; no paid calls in the hook

## Relevant details
muninn/internal/recall/boardcue.go pickFinding/BoardCue; replay muninn/cmd/board4, judge muninn/scripts/judge_board.py; false lint lines: hog p39w4a (stop order), orangutan zjedy0, pawprint 2n92r7, piglet n0fzz1, sunflower qaxx25

## Depends on

## Acceptance
- [ ] an unrelated lint finding is handed out only when the session has no doing task of its own or just closed one
- [x] a turn whose message tells the agent to stop or not run commands gets no board line
- [ ] replay: lint precision >= 0.9 on held-out with no drop in lint lines on fresh data, counts in muninn/docs/BOARD4.md

## Notes

guppy 2026-11-17 10:49: ci/x325pp 1a527be ships check 2 only. Replay (board4, DeepSeek Flash judge blind to arm; raw in $JCODE_SCRATCH_DIR/board4/x325pp/: main/ stop/ new/ newb/ final/, judgments.json, score2.py) over dev, overnight, held-out hug, held-out seid, fresh, fresh2 (20 seid sessions after 2026-11-16 20:00, 21.5 h). Lint lines right/false: main 51 (44/5); stop order 49 (46/1); stop + between tasks (check 1) 35 (30/3); + busy when editing 27 (22/3). All lines: main 97 (87/7), stop 95 (89/3), between 81 (73/5). Check 1 cut fresh lint lines 14 -> 8 and fresh2's remaining 3 were judged false (sessions editing with no doing task; re-judges 3/3 and 0/3: the judge is unstable there), so check 1 fails check 3's 'no drop in lint lines on fresh data' and is not shipped; code removed. The stop rule's held-out gain beyond hog is rotation reshuffle (14 breaks got a different finding), recorded as luck in BOARD4.md. Remaining false lint line: piglet n0fzz1 (another agent's doing task). Question for the coordinator (also tmp/lanes/morning.md): drop check 1, or rewrite it?
