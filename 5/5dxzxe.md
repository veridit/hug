---
id: 5dxzxe
title: "Own agent harness: persistent workers, messages and wakes on the board"
state: todo
type: spike
priority: ""
assignee: []
labels: []
group: ""
rank: "zzzzzzzzzz28"
created: 2026-11-17
updated: 2026-11-17
---

## Goal
We decide whether hug grows its own swarm layer: persistent workers, messages never lost, and waits that wake agents.

## Purpose
Jørgen 2026-11-17 morning: overnight, jcode reaped all four workers after 30 idle minutes (logs 22:18-22:44: 'Reaped idle spawned swarm worker ... idle > 1800s'), each was waiting on a turn on main or a reply, and nothing landed for 10 hours. The coordinator's todos either nag (open) or go silent (waiting), DMs were lost several times on 2026-11-16, and turns on main were passed by hand in DMs. Workers are easy to spawn and easy to lose, so knowledge workers cannot stay around.

## Principle
The board is the truth: queues, turns, waits and messages live in hug, not in the harness's memory. A wait names its condition (16dxsr: task done, branch merged, CI green, date) and whoever it wakes. A worker that goes quiet is resumed from its ticket, not lost. Spike first: no build until Jørgen approves a design.

## Relevant details
Decide whether hug grows its own swarm layer, likely on a Go agent harness such as a Go port of Pi: workers that persist across idle time and restarts, messages that are never lost, and waits that are real conditions on the board which wake the right agent when they become true.

Evidence: ~/.jcode/logs (2026-11-16 22:18:26 vole, 22:30:24 dragon, 22:34:24 humpback, 22:44:24 guppy reaped). Related: 16dxsr (conditions as dependencies), tb0dra (reflection), bxqtkt (contributions), m8shdj/0hkaa5 (write turns and background sync, the base for a main-push turn). Candidate pieces: hug task wait ID --until CONDITION, a main-push turn like the board turn, hug msg on the hug/tasks branch, a supervisor that resumes a worker from its taken ticket.

## Depends on

## Acceptance
- [ ] a design note compares extending jcode, building on a Go harness, and a thin hug supervisor, with the cost of each; Jørgen picks one
- [ ] the chosen design is split into build tickets with acceptance

## Notes
