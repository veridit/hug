---
id: chxk50
title: "Release v0.7.104: no stale lint finding after an upgrade"
state: todo
type: feature
priority: ""
assignee: []
labels: []
group: ""
rank: "zzzzzzzzzzzz"
created: 2026-11-22
updated: 2026-11-22
---

## Goal
v0.7.104 rebuilds the board cache once, so lint stops serving findings an older hug cached.

## Purpose
One release ticket per release. v0.7.104 = main 142a5b90, landed a4e05130 on exact-SHA CI 8/8 (run 45013140613).

## Principle
Nothing for Jørgen to try beyond lint; the one question is about restarting old hug processes.

## Details
Shipped 6q1d1t checks 1-2: SchemaVersion 31; a fingerprint test fails when what the cache holds changes without a bump (13 mutants). After brew upgrade at 10:51Z, hug task lint on this board: 'no findings' (was section-order on 9s0vj3 and 6q1d1t). Old hug tui/mcp processes from 11-17..11-19 still run their own cache-30.db; one writing set --details can add a second Relevant details section (hxqqdp).

## Depends on

## Acceptance
- [ ] Jørgen restarts his long-running hug tui and hug mcp sessions, or says to leave them

## Notes
