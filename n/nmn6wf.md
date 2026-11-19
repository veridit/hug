---
id: nmn6wf
title: "Capture stuck when one hook call's output hits 1024 tokens"
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
A hook capture whose extraction call returns finish_reason length (output cut at the hook's 1024-token cap) still makes progress: the chunk is retried smaller or with room to finish, so the session's high-water mark moves past it instead of failing the same way on every later hook.

## Purpose
After v0.7.75 fixed 8jcj3k, crab_1793783281662's first capture (capture_runs 8449, 2026-11-19T02:09:29Z) processed 441 messages over 7 calls, then failed 'extraction response truncated': call 6 used completion_tokens 1024 of max_tokens 1024. Temperature is 0, so the same chunk truncates the same way on every retry: the 8jcj3k pattern again (capture stops for good), now on output size. Truncation errors: 2026-11-08 2, 11-09 5, 11-10 3, 11-17 2, 11-18 1, 11-19 1.

## Principle
No silent loss: a truncated reply is never applied as if complete. No paid call beyond the retry of that chunk. Keep the 1024 cap's purpose (bounded hook latency) unless the evidence says otherwise.

## Relevant details
muninn/internal/app/capture.go:353-356 caps hook calls at max_tokens 1024; muninn/internal/extract/extract.go:506-508 returns 'extraction response truncated' on finish_reason length, and the run returns without advancing the high-water mark. Options: halve the chunk and retry (the split helper from 8jcj3k exists), or one retry with a larger cap for that chunk. The 6 calls before it in run 8449 produced 29 actions; check whether they were applied or rolled back.

## Depends on

## Acceptance
- [ ] a test: a fake extractor that returns finish_reason length for a chunk on the first call makes the run retry that chunk (smaller or with a larger cap), and the high-water mark moves past it (fails before)
- [ ] a reply that is still truncated after the retry is recorded in capture_runs and does not block later chunks forever (test)
- [ ] on the live store after release, crab_1793783281662 has a capture_runs row with no error (count in a note)

## Notes
