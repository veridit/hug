---
id: kbc99j
title: "Windows: a parallel note fails with 'file is being used by another process'"
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
done_at: "2026-11-17T15:22"
---

## Goal
On Windows, 20 parallel hug task note commands on one task all land, as on macOS and Linux: no command fails because another one has the task file open or is renaming over it.

## Purpose
Main CI run 44065882953 (517ed97), windows: TestParallelNotesWithoutRemoteAllLand failed with 'parallel note: open C:\...\h.wkaavb.parallel.md: The process cannot access the file because it is being used by another process' and 'parallel_test.go:37: 19 of 20 notes in the task file'. Kestrel saw the same on sfa8w4's CI. It breaks the rule that concurrent writes wait and never fail.

## Principle
Find the cause and prove it with a test that fails first. Waiting, not retry-and-hope: if a read can race a rename on Windows, the read happens inside the board turn or opens with sharing that allows the rename, and the turn covers every task-file read that precedes a write.

## Relevant details
Windows refuses to rename over a file another process holds open without FILE_SHARE_DELETE; Go's os.Open on Windows does not pass FILE_SHARE_DELETE. Suspects: a read of the task file outside the board turn (wyvern's step 8 moved open's read inside the turn, c66e317), or the cache hashing files while a write renames. Test: huginn/internal/cli parallel_test.go:34.

## Depends on

## Acceptance
- [x] a test reproduces the failure on Windows CI before the fix (run id), and passes after
- [x] TestParallelNotesWithoutRemoteAllLand passes in a Windows -count=20 run (run id)

## Notes

hatchling 12:57, two more: main CI 44084323148 (206038a) windows TestParallelNotesWithoutRemoteAllLand 'being used by another process', 19 of 20; ci/fkpyg3 CI 44083418033 (c555895) same. The same 206038a passed on ci/v3k6cp (44082371484) and 5817fd7 passed (44085256434): a flake, about 2 in the last 6 Windows runs I watched.

kestrel 13:11, cause and fix. Probe 4a2c487 (TestAtomicWriteRacesReads: 4 readers with os.ReadFile, 1 writer with AtomicWrite, 2 s) on ci/kbc99j, windows CI 44086864807, before the fix: 'atomic_race_test.go:65: 11646 reads, 169 writes' and 'atomic_race_test.go:73: 6 of 11646 reads failed (first: open ...\h.race.md: The process cannot access the file because it is being used by another process.), 165 of 169 writes failed (first: rename ...\.huginn-2869333018 ...\h.race.md: Access is denied.), 0 torn reads'. macOS and ubuntu passed the same probe. Cause: Go's os.Open on Windows shares read and write but not delete (syscall_windows.go: sharemode FILE_SHARE_READ|FILE_SHARE_WRITE), and os.Rename is MoveFileEx(REPLACE_EXISTING), so a read and a rename over the same task file fail each other; the board turn covers writers, but readers are outside it. The ticket's suspect 'read outside the turn' is that, and AtomicWrite's rename is the other half. Fix: internal/sharefile, where Open shares delete (CreateFile with FILE_SHARE_DELETE) and Rename uses os.Root.Rename, which is FILE_RENAME_POSIX_SEMANTICS on Windows. The task-file readers (cache, store, sections, files, reflect, tui detail and tui, cli) and AtomicWrite use it. Fix commit on ci/kbc99j, CI 44090177314 with a temporary -count=20 Windows step.

kestrel 13:29, CI 44090177314 on dd3e1ff (fix a45bec0 + TEMP step) green on all 5 jobs. The Windows step 'go test -count=20 -run TestParallelNotesWithoutRemoteAllLand|TestAtomicWriteRacesReads|TestRenameOverOpenFile' printed: 'ok  github.com/veridit/hug/huginn/internal/cli 282.515s', 'ok  github.com/veridit/hug/huginn/internal/task 40.599s', 'ok  github.com/veridit/hug/internal/sharefile 0.121s'. Before the fix, the same probe failed on windows CI 44086864807 (note 13:11). The TEMP step is dropped before landing.


Landed: 38ff464..4ecfa25 on main (fix 2d10706, probe 6a1c641, changelog), CI 44099384892 on 4ecfa25 green on all 5 jobs.
## History
- 2026-11-17T14:33 doing -> doing/waiting (wait)
- 2026-11-17T15:22 doing/waiting -> doing/active (resume); was waiting_asked_by: calf; waiting_on: landing turn: checks 1-2 ticked; combined with 4vcwxk on kestrel/land-kbc99j-4vcwxk, queued after hatchling (crab 13:29); waiting_since: 2026-11-17T14:33
- 2026-11-17T15:22 doing/active -> done (done)
