---
id: kbc99j
title: "Windows: a parallel note fails with 'file is being used by another process'"
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
On Windows, 20 parallel hug task note commands on one task all land, as on macOS and Linux: no command fails because another one has the task file open or is renaming over it.

## Purpose
Main CI run 44065882953 (517ed97), windows: TestParallelNotesWithoutRemoteAllLand failed with 'parallel note: open C:\...\h.wkaavb.parallel.md: The process cannot access the file because it is being used by another process' and 'parallel_test.go:37: 19 of 20 notes in the task file'. Kestrel saw the same on sfa8w4's CI. It breaks the rule that concurrent writes wait and never fail.

## Principle
Find the cause and prove it with a test that fails first. Waiting, not retry-and-hope: if a read can race a rename on Windows, the read happens inside the board turn or opens with sharing that allows the rename, and the turn covers every task-file read that precedes a write.

## Relevant details
Windows refuses to rename over a file another process holds open without FILE_SHARE_DELETE; Go's os.Open on Windows does not pass FILE_SHARE_DELETE. Suspects: a read of the task file outside the board turn (wyvern's step 8 moved open's read inside the turn, c66e317), or the cache hashing files while a write renames. Test: huginn/internal/cli parallel_test.go:34.

## Depends on

## Acceptance
- [ ] a test reproduces the failure on Windows CI before the fix (run id), and passes after
- [ ] TestParallelNotesWithoutRemoteAllLand passes in a Windows -count=20 run (run id)

## Notes

hatchling 12:57, two more: main CI 44084323148 (206038a) windows TestParallelNotesWithoutRemoteAllLand 'being used by another process', 19 of 20; ci/fkpyg3 CI 44083418033 (c555895) same. The same 206038a passed on ci/v3k6cp (44082371484) and 5817fd7 passed (44085256434): a flake, about 2 in the last 6 Windows runs I watched.
