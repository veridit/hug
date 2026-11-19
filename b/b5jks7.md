---
id: b5jks7
title: "Capture planning takes most of the hook budget"
state: "doing"
type: feature
priority: ""
assignee: ["sunflower"]
labels: []
group: ""
rank: "zzzzzzzzzzzy"
created: 2026-11-19
updated: 2026-11-19
doing_since: "2026-11-19T07:00"
stage: "active"
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
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
- [x] a note splits plan_ms of a real hook run into search time and per-candidate plan time, with counts
- [ ] after a change, plan_ms per applied action on live turn_end hooks falls, with before/after medians over at least 20 runs each

## Notes


Phase 1 measured (replay of capture_runs 8476 on a RECALL4-recipe HUG_HOME copy rolled back to 8476's start, origin/main 1ce66c8a, no paid calls; report /Users/jhf/.jcode/scratch/b5jks7/REPORT.md). Replay plan 115,295 ms (recorded 103,958) = chunk Searcher.Search 62,908 ms (14 x, mean 4,493) + per-candidate plan 52,386 ms (65 x, mean 806: writes.exact 0.2 ms, binding.refresh 17.9 ms x68, memoryMatches exact-body SQL 272 ms, memoryMatches Search 515 ms). Biggest single cost: concepts 'term LIKE %w%' full scan, 3,654 calls x ~14.6 ms = 53.2 s (46%); 1,168 of those (17.1 s) repeat a word already looked up. ollama embeds 79 calls = 4.3 s (3.7%). Exact-body SQL 17.7 s (15%, SCAN units). Proposal (not done): one in-memory LIKE-equivalent pass over deduped words (measured 1.7 s vs 52.8 s on the same 79 queries; '_' wildcard semantics must be kept: 4/79 queries differ with plain Contains). Est. plan 115 s -> ~64-67 s.
check 1: REPORT.md split: chunk search 62,908 ms/14 + per-candidate 52,386 ms/65 of 115,295 ms replay of run 8476, with sub-steps and counts (note above)
## History
- 2026-11-19T07:00 todo -> doing/active (take)
