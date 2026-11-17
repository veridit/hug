---
id: x325pp
title: "Board lint nudges go to agents at a break in their own work"
status: todo
type: feature
priority: ""
assignee: ["guppy"]
labels: []
created: 2026-11-16
updated: 2026-11-16
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
- [ ] a turn whose message tells the agent to stop or not run commands gets no board line
- [ ] replay: lint precision >= 0.9 on held-out with no drop in lint lines on fresh data, counts in muninn/docs/BOARD4.md

## Notes
