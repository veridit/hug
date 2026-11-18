---
id: ygay9c
title: "hug refuses a board worktree that belongs to another repo"
state: "doing"
type: bug
priority: "medium"
assignee: ["kestrel"]
labels: []
created: 2026-11-18
updated: 2026-11-18
doing_since: "2026-11-18T04:34"
stage: "active"
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
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
- [ ] A test copies a repo with its board worktree; hug task note in the copy exits non-zero, writes nothing to either repo, and the fix line, when run, makes the copy's own board work (fails first on main)
- [ ] The check adds no git process to a normal command (test counts git calls)

## Notes


## History
- 2026-11-18T04:34 todo -> doing/active (take)
