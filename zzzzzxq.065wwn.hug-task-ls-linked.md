---
id: 065wwn
title: "hug task ls in a linked worktree shows the main board and writes to it"
status: "doing"
type: "bug"
priority: "high"
assignee: ["hare"]
labels: []
created: 2026-11-16
updated: 2026-11-16
---

## Goal
hug task ls in a linked worktree shows the main board and writes to it

Promoted from item 1 of task:x1t1t1.

## Depends on

## Acceptance
- [x] hug task ls in a linked worktree shows the main board and writes to it
- [x] A board created from a linked worktree of a fresh repo lands at the main worktree .hug/tasks
- [ ] No second checkout: git worktree list shows hug/tasks once
- [ ] MCP from a linked worktree reads and writes the main board
- [ ] Remaining errors (missing or stale checkout) print a fix: that works when run
- [ ] Full niced suite green: GOMAXPROCS=2 GOFLAGS=-p=2 nice -n 19 go test ./...

## Notes

2026-11-16 repro: in a linked worktree (git worktree add --detach <dir> <rev>), hug task ls fails: 'worktree add <dir>/.hug/tasks hug/tasks: ... fatal: 'hug/tasks' is already used by worktree at '/Users/jhf/Projects/hug/.hug/tasks'; exit status 128' and its fix: line repeats the same failing git worktree add. Wanted: every linked worktree uses the one hug/tasks checkout of the main worktree (git rev-parse --git-common-dir), reads and writes the same board; no second checkout. MCP and TUI too. Tagging by worktree stays in x1t1t1 (needs discussion with Jørgen).

check 1 (hare, built from HEAD+065wwn files): in linked worktree /tmp/065wwn.b5d0/l, 'hug task ls' printed '4b3j38  todo     feature  made-main    Made in main' (task made in main); 'hug task take 9xxy8y' in linked printed 'made-linked  Made in linked  9xxy8y  doing', and 'hug task ls' in main then showed '9xxy8y  doing    feature  made-linked  Made in linked'. Test: TestLinkedWorktreeUsesMainBoard (huginn/internal/cli/worktree_test.go) does new/ls/take/show across both.

check 2: fresh repo, 'hug task init' run in linked worktree /tmp/065wwn.b5d0/l printed 'tasks on branch hug/tasks, worktree /private/tmp/065wwn.b5d0/m/.hug/tasks, head bd101663, remote (none: tasks stay local until a remote exists)'; 'ls -a ../l' shows only '.', '..', '.git' (no .hug). Tests: TestLinkedWorktreeUsesMainBoard/init_from_linked, TestLinkedWorktreeSharesMainCheckout/linked_first.

check 3: 'git worktree list' after init/new/take from linked: '/private/tmp/065wwn.b5d0/m            09d08e2 [main]' / '/private/tmp/065wwn.b5d0/l            09d08e2 (detached HEAD)' / '/private/tmp/065wwn.b5d0/m/.hug/tasks 45efd63 [hug/tasks]' (3 entries, hug/tasks once). Tests assert 3 worktree entries and one 'branch refs/heads/hug/tasks'.
