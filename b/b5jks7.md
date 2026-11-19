---
id: b5jks7
title: "Capture planning takes most of the hook budget"
state: todo
type: feature
priority: ""
assignee: []
labels: []
group: ""
rank: "zzzzzzzzzzzz"
created: 2026-11-19
updated: 2026-11-19
---

## Goal
A hook capture spends most of its time budget on extraction calls, not on planning: the memory search per chunk and the per-candidate plan in capturePlan take a small share of the 180 s run.

## Purpose
a6xm8n (cactus): 8467 used 48.4 s extract + 115.6 s plan + 12.1 s apply = 176.1 s of the 180 s budget; 8468 52.1 + 112.0 + 12.0. Planning is about 65% of each hook run, so fewer messages are captured per hook and the budget cut lands sooner.

## Principle
Measure first: name where plan_ms goes (search vs capturePlan, per candidate) before changing anything. No change to what is extracted or applied.

## Relevant details
plan_ms = per-chunk Searcher.Search + capturePlan per candidate (muninn/internal/app/capture.go). Evidence: capture_runs 8467, 8468 record JSON (extract_ms, plan_ms, apply_ms).

## Depends on

## Acceptance
- [ ] a note splits plan_ms of a real hook run into search time and per-candidate plan time, with counts
- [ ] after a change, plan_ms per applied action on live turn_end hooks falls, with before/after medians over at least 20 runs each

## Notes
