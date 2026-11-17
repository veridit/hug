---
id: tpnd3n
title: "an unrelated lint finding is handed out only when the session has no doing task…"
status: todo
type: feature
priority: ""
assignee: []
labels: []
created: 2026-11-17
updated: 2026-11-17
---

## Goal
an unrelated lint finding is handed out only when the session has no doing task of its own or just closed one

Promoted from item 1 of task:x325pp.

## Purpose
In the qawmnb replay 5 of the 7 false board lines were lint findings handed to agents busy with other work (muninn/docs/BOARD4.md), one of them just told to stop

## Principle
Measure on replayed real sessions before shipping (board4 + blind judge); keep cleaning the board over time, do not silence lint; no paid calls in the hook

## Relevant details
muninn/internal/recall/boardcue.go pickFinding/BoardCue; replay muninn/cmd/board4, judge muninn/scripts/judge_board.py; false lint lines: hog p39w4a (stop order), orangutan zjedy0, pawprint 2n92r7, piglet n0fzz1, sunflower qaxx25

## Depends on

## Acceptance
- [ ] an unrelated lint finding is handed out only when the session has no doing task of its own or just closed one

## Notes

hatchling 12:42, from x325pp: measured arms in muninn/docs/BOARD4.md 'Stop orders, and lint between tasks'. Lint lines right/false: main 97: 87/7; stop order 95: 89/3; + between tasks 81: 73/5; + busy when editing 73: 65/5. fresh2 seid: 8: 8/0 -> between tasks 3: 0/3. Raw: $JCODE_SCRATCH_DIR/board4/x325pp/ (main/ stop/ new/ newb/, judgments.json, score2.py).
