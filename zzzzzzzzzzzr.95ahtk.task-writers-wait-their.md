---
id: 95ahtk
title: "Task writers wait their turn: no index or temp-file races"
status: "done"
type: bug
priority: "high"
assignee: ["kestrel"]
labels: []
created: 2026-11-17
updated: 2026-11-17
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
done_at: "2026-11-17T18:53"
---

## Goal
Any number of hug task writers on one board at the same moment all succeed: they serialise on a per-board lock before touching the index, temp files or the commit, as hug mem does since me51r2.

## Purpose
Seid foreman (orangutan) requests 2026-11-17, file: seid-lang .jcode/scratch/TASK-2057/hug-agent-requests.md, read by crab 15:07, item 3b. Two agents writing at once in Seid raced on the git index and on temp files; Seid now batches writes, one write per ticket per batch, to avoid it. Rule: concurrent writes wait, never fail.

## Principle
Reproduce first with N parallel writers in a test that fails today; then a lock that waits, with no timeout that turns into a failure. Readers never wait for the lock.

## Relevant details
me51r2 (9af390c) fixed this for hug mem. huginn has no file lock today (no flock/LockFile in huginn). Check the cache.sync BEGIN IMMEDIATE fix from 44w1ev too.

## Depends on

## Acceptance
- [x] a test runs 8 writers (note, check, set) at once on one board, 20 rounds: every write succeeds and is on the branch; it fails on main before the fix
- [x] readers (ls, show) do not wait on a held writer lock (test)

## Notes


Landed in main edec38c (65fe3db..edec38c). Cause: CLI writers already serialise on the board turn since m8shdj. The TUI did not: its store writes (check, uncheck, strike, answer, give back, done, drop, new, move, the S screen attach) and its exit commit 'hug task tui' ran outside the turn. Seid evidence: .jcode/scratch/TASK-2057/a71qfj/docs-attach/commands.jsonl, 2026-11-17T16:49:23 attach rpjd4z failed with "Unable to create '.../.git/worktrees/-huginn/index.lock': File exists", the same second as a 'hug task tui' commit on Seid's board. Fix: tui.Options.Turn = session turn, held from file change through commit; the editor commit and the exit commit hold it too. Test TestTUIWriteAndCLINoteAtOnce (TUI toggles a check 30x while hug task note adds 30 notes on the same task): on main's locking 3 of 3 runs fail, e.g. 'hug task note beside the TUI: --literal-pathspecs add --all -- .: fatal: unable to stat '.huginn-52152001': No such file or directory', '29 of 30 note commits', '25 of 30 TUI writes have their own commit'. With the fix 3 of 3 pass. Check 1: TestEightWritersMixedCommandsAllLand (8 writers x 20 rounds note/check/set = 160 commits). It passes on main because m8shdj already covers CLI writers. With the turn removed it fails: '33 commits for 160 writes' plus index.lock and .huginn-* errors. The failing-first case for this ticket is the TUI test above (crab 17:04). Check 2: TestReadersDoNotWaitOnHeldTurn (ls, show, next, search answer while the turn is held). -race before push: cli ok 150.954s, tui ok 838.752s (load ~105).
Reflection (open, 2026-11-17, calf): The ticket's 'huginn has no file lock today (no flock/LockFile in huginn)' was out of date: m8shdj (896ca11) had already added the board turn for CLI writers, and an 8-writer CLI test passed on main. Grepping for the turn's callers found the real gap (the TUI). Also 'hug task show' printed no stage, so I could not see my own wait on ggydj2 and reran it (filed 456vyt).
## History
- 2026-11-17T17:47 todo -> doing/active (take)
- 2026-11-17T18:53 doing/active -> done (done); was doing_since: 2026-11-17T17:47
