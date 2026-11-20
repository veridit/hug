---
id: ygay9c
title: "hug refuses a board worktree that belongs to another repo"
state: "done"
type: bug
priority: "medium"
assignee: ["kestrel"]
labels: []
group: ""
rank: "zzzzzzzzzzm"
created: 2026-11-18
updated: 2026-11-18
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
done_at: "2026-11-18T04:41"
done_release: "v0.7.55"
---

## Goal
When .hug/tasks/.git points at a worktree inside a different repository than the one hug runs in, hug stops before reading or writing, names both paths, and gives a fix line that repairs the link (git worktree repair, or re-create the board worktree).

## Purpose
wyvern 2026-11-18 (w40cmk bench): an APFS copy of a repo kept .hug/tasks/.git as 'gitdir: <original>/.git/worktrees/tasks', so hug running in the copy read and committed the original's board (37 notes landed there), while git status in the copy showed 74 deletions. Copying or moving a repo is ordinary; writing into another repo's board silently breaks 'nothing writes silently'.

## Principle
The board hug writes must belong to the repo hug runs in. Check it once per process, cheaply (read one .git file, compare with git rev-parse --git-common-dir); refuse, never guess or repair silently.

## Relevant details
Code: huginn/gitbranch (board worktree discovery). Seen in ~/.jcode/scratch/w40cmk-bench/before (kept as evidence). Note for w40cmk: the new layout keeps the worktree at .hug/tasks, so this check applies there too.

## Depends on

## Acceptance
- [x] A test copies a repo with its board worktree; hug task note in the copy exits non-zero, writes nothing to either repo, and the fix line, when run, makes the copy's own board work (fails first on main)
- [x] The check adds no git process to a normal command (test counts git calls)

## Notes


Landed a90f4d44.
- Failing first on main: TestBoardOfAnotherRepoRefused (cli) fails with 'boardowner_test.go:40: note in the copy went through'.
- Refusal now: 'open: the board checkout ORIG/.hug/tasks belongs to another repository: its .git names ORIG/.git/worktrees/tasks, outside this repository's COPY/.git (a copied or moved repo keeps the original's link); hug writes nothing to it', then 'fix: git -C COPY worktree repair COPY/.hug/tasks'.
- The test runs that fix with sh. After it the copy's note lands on its own board, and the original's HEAD is unchanged throughout. Experiment: worktree repair rewrites both links (copy/.git/worktrees/tasks/gitdir and copy/.hug/tasks/.git); orig untouched.
- Check 2: TestGitProcessesPerCommand passes with unchanged counts (show 2, ls 2, note 4 ...). The check is one os.ReadFile plus EvalSymlinks.
- gitbranch: TestOpenRefusesAnotherRepositorysBoard (copy refused, then repaired; original and linked worktree still open) and TestOwnedThroughSymlink.
- Mutants: 6 of 6 killed (no check, always owned, inverted, fix repairs the other, within always, no symlink resolve).
- Race ok: gitbranch 25.1s, cli 151.7s.
Reflection (dismissed, 2026-11-18, kestrel): The copy's board link is two files (admin gitdir and checkout .git); git worktree repair from the copy fixes both, so the fix line is git's own command, not a hug repair.
landed a90f4d4 on main 2026-11-18: hug refuses a board checkout that belongs to another repository; the fix is git…
shipped in v0.7.55 (a90f4d4)
landed 6c66ee2 on main 2026-11-18: test: board-owner refusal names the original as a resolved folder (ygay9c)
shipped in v0.7.57 (6c66ee2)
Reflection 1 dismissed (tb0dra check 5 triage, crab 2026-11-19): G10, a lesson about one's own method, fixed in its own ticket; no instruction misled.
## History
- 2026-11-18T04:34 todo -> doing/active (take)
- 2026-11-18T04:41 doing/active -> done (done); was doing_since: 2026-11-18T04:34
