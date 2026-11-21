---
id: ta6fy6
title: "Release v0.7.99: H and L stop on no column past either end"
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
Jørgen tries v0.7.99 and says whether H and L stepping onto no column past either end feels right.

## Purpose
One release ticket per release. v0.7.99 = main a5910763, landed 1d864738 on exact-SHA CI 8/8 (run 44960662438).

## Principle
Ask one thing; the frame is on hktdgs.

## Relevant details
Shipped: hktdgs check 2 (failing-first test in overview, list and board; 7 mutants killed; 5 help goldens changed, one line each). An order set with o stays; o and the sort picker's c say to mark a column first while none is marked.

## Depends on

## Acceptance
- [ ] Jørgen steps past both ends with H and L in v0.7.99 and says it reads right, or what to change

## Notes

crab 23:33Z: shipped v0.7.99 (main a5910763, landed 1d864738, CI run 44960662438 8/8). release.yml success; brew-verified: Cellar/hug/0.7.99, hug version = v0.7.99.
