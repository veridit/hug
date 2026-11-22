---
id: 95ahtk
title: "Task writers wait their turn: no index or temp-file races"
state: "done"
type: bug
priority: "high"
assignee: ["kestrel"]
labels: []
group: ""
rank: "zzzzzzzzzzcq"
created: 2026-11-17
updated: 2026-11-17
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
done_at: "2026-11-17T18:53"
done_release: "v0.7.45"
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
Reflection (dismissed, 2026-11-17, calf): The ticket's 'huginn has no file lock today (no flock/LockFile in huginn)' was out of date: m8shdj (896ca11) had already added the board turn for CLI writers, and an 8-writer CLI test passed on main. Grepping for the turn's callers found the real gap (the TUI). Also 'hug task show' printed no stage, so I could not see my own wait on ggydj2 and reran it (filed 456vyt).
landed edec38c on main 2026-11-17: hug tui: every write holds the board turn, as CLI writes do (95ahtk)
landed dcb20c2 on main 2026-11-17: Merge v0.7.44 into main (456vyt, 95ahtk stay under Unreleased)
shipped in v0.7.45 (edec38c)
Size change (0sbvnq, crab 22:02): TestEightWritersMixedCommandsAllLand now runs 8 writers x 3 rounds (24 writes; each writer runs note, check and set once, all at once), was 8 x 20 = 160. It cost 88-101 s of the Windows cli package (paired run 44180555871). Mutation, board turn removed (session.turn returns a no-op), 10 runs each on macOS: 1 round 10 of 10 fail, 2 rounds 10/10, 3 rounds 10/10, 6 rounds 10/10, 20 rounds 10/10. Failures: 'N commits for M writes' (e.g. '5 commits for 24 writes', '33 commits for 160 writes'), index.lock 'File exists', 'index already contains staged changes'. With the turn, 6 rounds passed 3 of 3. The test's end checks now follow rounds (items checked, the last set). Same size on every OS.
landed 7592dc8 on main 2026-11-18: hug task writes start 4 git processes, was 6: worktree list from the files, no…
TestTUIWriteAndCLINoteAtOnce size change (0sbvnq, 1c40426d): 10 toggles beside 10 notes, was 30+30 (36-46 s on Windows). Mutation, the TUI's board turn removed (Options.Turn a no-op), 10 runs each on macOS: 5+5 10 of 10 fail, 10+10 10/10, 30+30 10/10. Failures: 'N of M note commits', index.lock 'Unable to create', 'N of M TUI writes on the branch'. With the turn, 10+10 passed 5 of 5.
shipped in v0.7.52 (7592dc8)
## History
- 2026-11-17T17:47 todo -> doing/active (take)
- 2026-11-17T18:53 doing/active -> done (done); was doing_since: 2026-11-17T17:47
