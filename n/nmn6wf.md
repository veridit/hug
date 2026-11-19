---
id: nmn6wf
title: "Capture stuck when one hook call's output hits 1024 tokens"
state: "doing"
type: feature
priority: ""
assignee: ["mushroom"]
labels: []
group: ""
rank: "zzzzzzzzzzzz"
created: 2026-11-19
updated: 2026-11-19
doing_since: "2026-11-19T04:01"
stage: "active"
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
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
- [x] a test: a fake extractor that returns finish_reason length for a chunk on the first call makes the run retry that chunk (smaller or with a larger cap), and the high-water mark moves past it (fails before)
- [x] a reply that is still truncated after the retry is recorded in capture_runs and does not block later chunks forever (test)
- [ ] on the live store after release, crab_1793783281662 has a capture_runs row with no error (count in a note)

## Notes


crab 03:23Z, premise corrected from capture_runs (read-only): not stuck for good. 8449 (02:09Z) truncated at call 7 after 6 calls and 29 adds. 8453 (03:02Z) re-sent that chunk, which passed at 780 completion tokens (its known memories had changed), made 51 actions over 7 calls (780,608,437,615,566,717,1024 tokens), and truncated on a later chunk. 8456 (03:23Z) skipped: capture already running. Real cost: each truncation wastes a hook and delays the rest; the goal stands, but the 'stuck for good' wording in the purpose overstates it.
crab 03:25Z, my 03:23 correction was itself wrong; mushroom is right, reproduced: crab's capture IS stuck while Known is unchanged: 8459 (03:23:43Z) and 8460 (03:24:00Z) truncated with processed 0 on the same chunk 8453 cut. Only the cut chunk is re-sent (earlier chunks checkpoint). Truncations by trigger: session_end 13, turn_end 8; a session_end truncation loses that session's tail for good.
Run 8449 (live store, read-only): the 29 actions of calls 1-6 were APPLIED, not rolled back. Each chunk saves and checkpoints on its own: all 441 messages before the cut are in capture_seen (441/441 for session hash 66b1a53c...), and all 29 origins are in memories.json (29/29, all 'add'). Only call 7's chunk (62 messages, 8168 bytes, 4077 prompt + 1024 completion tokens, 14 reply lines with the 14th cut) was lost. The next run 8453 started at 8449's message #441 (the cut chunk): only the cut chunk is re-sent, never the ones before it. It passed at 780 tokens because Known had changed after the 29 adds. 8453 then cut on a later chunk, which 8459 (03:23:43Z) and 8460 (03:24:00Z) re-sent byte-identical (same hash of transcript+known), 1024/1024 each time, processed=0: stuck while Known is unchanged. Duplicate cost per truncation on main: the cut chunk only, once per hook while it stays cut. With the fix: one more call for that chunk at 2048 (prompt again, ~4k tokens), and at most one drain call at 4096 if it is still cut; chunks before it are never re-sent. Retry cap sizing: of 1905 audited hook calls with max_tokens 1024, p50 367, p90 641, p95 740, p99 1024 completion tokens; 21 (1.1%) hit 1024, 37 >= 900, 81 >= 768. The 1884 complete replies had at most 19 lines (p99 13); the 21 cut ones were 11-17 lines in at ~62-98 tokens a line, so 19 lines is ~1400-1900 tokens: 2048 covers them on that estimate (not proven per chunk: their real length is unknown). Output rate p10/p50/p90 108/138/175 tok/s, so 1024 more tokens is 6-10 s. 13 of 21 truncations were session_end, never captured again; the fix drains them on a later hook (DrainDeferred).
check 1: TestCaptureTruncatedChunkRetriesAndMovesPast (commit e98a12ac; failed first at 01b6f80a: 'capture_truncated_test.go:123: run stopped at the truncated chunk: processed=1 err=extraction response truncated'). Now: 'CAPTURE truncated: overflow chunk max_tokens=[1024 2048], processed=3, partial reply not applied'. CI run 44456503911 on e98a12ac: success, all 8 jobs; 'ok github.com/veridit/hug/muninn/internal/app' on ubuntu, macos, windows.
check 2: TestCaptureStillTruncatedIsRecordedAndDoesNotBlock + TestCaptureSessionEndDeferredChunkIsDrainedByALaterHook + TestCaptureDeferredStillTruncatedLeavesQueueKeepsBacklog (e98a12ac): 'CAPTURE still truncated: max_tokens=[1024 2048 4096], capture_runs truncated=1 deferred=["overflow"], later chunk applied, --backfill recovered it'; 'CAPTURE session_end deferred: hook max_tokens=[1024 2048], drained once at 4096, applied, queue empty'; 'CAPTURE deferred still truncated: 1 part(s) drained once each, queue empty, message kept in backlog, nothing partial applied'. CI run 44456503911: success.
## History
- 2026-11-19T04:01 todo -> doing/active (take)
