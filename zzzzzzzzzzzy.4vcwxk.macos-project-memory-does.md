---
id: 4vcwxk
title: "macOS: a project memory does not reach the other clone within 10 s"
status: "doing"
type: bug
priority: "high"
assignee: ["kestrel"]
labels: []
created: 2026-11-17
updated: 2026-11-17
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
---

## Goal
In TestBackgroundSyncTwoClonesRealBinary, a project memory remembered in clone A shows up in clone B within the test's limit on every OS, as the task note does; the cause of the miss is found and fixed, not the limit raised.

## Purpose
Main CI macOS failed twice the same way: run 44077251884 (893cd50) 'clone B: note after 8.758308208s, memory after -1ns (-1: not within 10s)' and run 44068175565 (c6e0bff) 'note after 9.325147584s, memory after -1ns'. The note arrives (late), the memory never does. 0hkaa5 promised project memories sync in the background within seconds.

## Principle
A wait that never ends is a bug, not a flake: find why the muninn-notes push or fetch does not happen, prove it with logs from .git/hug/sync/, fix it. Raise a limit only if the cause is the runner's speed and the measurement shows it.

## Relevant details
Test: sync_test.go:120. The note at 8.7-9.3 s is also near the 10 s limit; ubuntu and Windows pass. Sync logs: .git/hug/sync/. The memory sync runs on the muninn-notes branch (0hkaa5); the quiet() fix 18203c2 covered the cleanup race, not this.

## Depends on

## Acceptance
- [x] the cause is named with evidence (sync log lines or a trace) in a note
- [ ] a test that fails before the fix passes after; macOS CI green on three consecutive main runs (run ids)

## Notes

kestrel 12:53, cause (check 1): the test polled the memory only after the note had arrived, against the same start and 10 s limit. Clone B fetches project memories only when a hug mem read finds the last pass stale, so the first memory fetch came after the note. Local run with a 30 s limit (DIAG, not committed): note 11.53 s, memory 18.06 s, and B's muninn-notes.log shows its first pass at 12:30:06, after the note had shown. Fix 7aa4600 polls both in every round from the start, limits unchanged: 4 of 4 local passes (note/memory 9.02/7.17, 3.93/4.56, 3.66/3.84, 4.83/9.06 s). | Where the note's seconds go (crab 12:50), three instrumented local runs, times from start: A's note command returns at +0.22 s. A's pass settles 1.5 s quiet (Settle), then the push pass takes 0.3 s, so the note is on the remote at +0.5 / +2.2 / +2.2 s. B's reads nudge a fetch only when its last pass is older than FetchAfter = 5 s (huginn/internal/cli/sync.go:23); B's last pass was in setup, so its fetch began at +3.9 / +3.9 / +3.7 s (logs: B passes at 12:51:58.08 then 12:52:03.53, 12:52:08.25 then 13.88, 12:52:19.87 then 25.40). That gate kept B waiting 1.5 to 3.4 s with the note already on the remote. The fetch pass takes 0.2 s, then the next show at a 0.4 s poll: note seen at 3.96 / 4.27 / 4.07 s. B's task show costs 57-160 ms. Worst case by design on this machine: 1.5 settle + 0.3 push + 5 FetchAfter + 0.2 fetch + 0.4 poll = 7.4 s. macOS CI's 8.7-9.3 s is that worst case on a slower runner. So the time is in sync's two constants (FetchAfter 5 s, then Settle 1.5 s), not in the test. Memories use the same two constants (muninn/internal/project/background.go). To get 'within seconds' as a bound rather than an average, FetchAfter is the lever; that is a design call (each lowering adds one network fetch per reading clone per interval) for crab/Jørgen.
