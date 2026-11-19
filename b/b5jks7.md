---
id: b5jks7
title: "Capture planning takes most of the hook budget"
state: "doing"
type: feature
priority: ""
assignee: ["crab"]
labels: []
group: ""
rank: "zzzzzzzzzzz7"
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
Phase 2 on branch b5jks7 (base 1ce66c8a): 66065f03 dedup, d3acde6a one-read Go LIKE (port of SQLite patternCompare, terms read per Search call, no cache), 849a56d1 LIKE-equality tests (8476's _-wildcard words, odd words, 200x300 seeded random, FuzzLike: 671,821 execs clean; a '_'-as-literal mutation fails 139 lines), e2e1b037 CHANGELOG. Gate: 79 searches of run 8476 byte-identical (cmp base.tsv new.tsv: 79 searches, 2248 result rows, 0 differences, cached query vectors); recall4 default 277 msgs: dev 200 msgs/253 hits 0 differences, heldout 77/131 0 differences. Harness plan time 115,801 ms -> 66,279 ms (chunk search 63,805 -> 20,296 ms, per candidate 799.9 -> 707.4 ms mean); dedup alone 97,782 ms. CI run 44476979919. Check 2 waits for live hooks after release.
crab 07:08Z, gate reproduced by crab from sunflower's files: (1) 79 replay searches: gate/base.tsv vs gate/new.tsv, 2327 lines each, cmp byte-identical (sha256 339c8dd6...). (2) recall4 replay, recall4_cmp.py (every field except timing): development 'messages 200 vs 200, hits compared 253, messages with injections 111, differences 0'; heldout 'messages 77 vs 77, hits compared 131, messages with injections 50, differences 0'. (3) harness plan_ms 115801.5 ms -> 66278.7 ms (chunk search 63805.4 -> 20295.8 ms), same 65 actions (57 add, 5 add similar, 3 reinforce). Code reviewed at e2e1b037.
crab: landed as main 75cf6431 (restack of e2e1b037 on v0.7.78, CODE-DIFF and CHANGELOG IDENTICAL; CI 44478091776 success 8/8); released in v0.7.79. Check 2: compare plan_ms per applied action, median over >= 20 live turn_end hooks before (v0.7.77/78) vs after v0.7.79.
landed 6936349 on main 2026-11-19: search: look up each distinct query word and bigram once (b5jks7)
landed c79c22f on main 2026-11-19: search: match long query words against concepts in one read (b5jks7)
landed fa9733d on main 2026-11-19: search: pin the Go LIKE matcher to SQLite's LIKE (b5jks7)
shipped in v0.7.79 (6936349)
crab 19:30Z, check 2 measured (read-only, live store, trigger turn_end, report.actions > 0; v0.7.79 brew at 07:29:19Z per gn7tsm): before 05:37:40Z-07:29:19Z n=7, median plan_ms 102210, median plan_ms/action 1419; after n=22, median plan_ms 80420, median plan_ms/action 834. 184 of 213 turn_end rows had no actions. The bar (>= 20 runs each side) cannot be met: the before window is fixed and closed with 7 runs, and earlier runs are v0.7.76 or older with a6xm8n's different budget checks, so widening it would confound. The direction agrees with check 1's replay (115.8 s -> 66.3 s, byte-identical output) but n=7 is too few to claim it live.
## History
- 2026-11-19T07:00 todo -> doing/active (take)
- 2026-11-19T09:56 handed to crab: sunflower's session ended (closed 2026-11-19 08:15)
