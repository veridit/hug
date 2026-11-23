---
id: psd8gz
title: "Release v0.7.109: a big task's page answers keys at once"
state: todo
type: feature
priority: ""
assignee: []
labels: []
group: ""
rank: "zzzzzzzzzzzz"
created: 2026-11-23
updated: 2026-11-23
---

## Goal
v0.7.109 ships 1tjq11: scrolling and typing in a big task's page answer at once; Jørgen tries it on p39w4a in Seid.

## Purpose
One ticket per release so Jørgen reviews what shipped in one place.

## Principle
Shipped tasks are children; one question at a time.

## Details
Main 3125fd29 (Release v0.7.109, merge of 8c16709c, CI 8/8 on that exact SHA). release.yml run 45098166000 success; brew gives ../Cellar/hug/0.7.109/bin/hug, hug version v0.7.109. crab's own check in a real terminal (tmux 180x50, copy of seid-lang's board, p39w4a, load ~10): one j took 210-472 ms on v0.7.108 and 14-39 ms on v0.7.109 (time until the screen changed, tmux polling included). monkey's numbers: generated 700 KB task, j 0.36 s -> ~2 ms, a typed character 0.29 s -> ~2 ms; 16 invalidation tests, 16 of 16 mutants caught; goldens unchanged. Not changed: opening a big board still takes seconds (the board load), outside this task.

## Depends on

## Acceptance
- [ ] Jørgen scrolls p39w4a in Seid and types an answer there, and finds it quick

## Notes
