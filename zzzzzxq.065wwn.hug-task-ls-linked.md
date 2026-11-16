---
id: 065wwn
title: "hug task ls in a linked worktree shows the main board and writes to it"
status: "done"
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
- [x] No second checkout: git worktree list shows hug/tasks once
- [x] MCP from a linked worktree reads and writes the main board
- [x] Remaining errors (missing or stale checkout) print a fix: that works when run
- [x] Full niced suite green: GOMAXPROCS=2 GOFLAGS=-p=2 nice -n 19 go test ./...

## Notes

2026-11-16 repro: in a linked worktree (git worktree add --detach <dir> <rev>), hug task ls fails: 'worktree add <dir>/.hug/tasks hug/tasks: ... fatal: 'hug/tasks' is already used by worktree at '/Users/jhf/Projects/hug/.hug/tasks'; exit status 128' and its fix: line repeats the same failing git worktree add. Wanted: every linked worktree uses the one hug/tasks checkout of the main worktree (git rev-parse --git-common-dir), reads and writes the same board; no second checkout. MCP and TUI too. Tagging by worktree stays in x1t1t1 (needs discussion with Jørgen).

check 1 (hare, built from HEAD+065wwn files): in linked worktree /tmp/065wwn.b5d0/l, 'hug task ls' printed '4b3j38  todo     feature  made-main    Made in main' (task made in main); 'hug task take 9xxy8y' in linked printed 'made-linked  Made in linked  9xxy8y  doing', and 'hug task ls' in main then showed '9xxy8y  doing    feature  made-linked  Made in linked'. Test: TestLinkedWorktreeUsesMainBoard (huginn/internal/cli/worktree_test.go) does new/ls/take/show across both.

check 2: fresh repo, 'hug task init' run in linked worktree /tmp/065wwn.b5d0/l printed 'tasks on branch hug/tasks, worktree /private/tmp/065wwn.b5d0/m/.hug/tasks, head bd101663, remote (none: tasks stay local until a remote exists)'; 'ls -a ../l' shows only '.', '..', '.git' (no .hug). Tests: TestLinkedWorktreeUsesMainBoard/init_from_linked, TestLinkedWorktreeSharesMainCheckout/linked_first.

check 3: 'git worktree list' after init/new/take from linked: '/private/tmp/065wwn.b5d0/m            09d08e2 [main]' / '/private/tmp/065wwn.b5d0/l            09d08e2 (detached HEAD)' / '/private/tmp/065wwn.b5d0/m/.hug/tasks 45efd63 [hug/tasks]' (3 entries, hug/tasks once). Tests assert 3 worktree entries and one 'branch refs/heads/hug/tasks'.

check 4: 'hug mcp' started in linked worktree, tools/call task_new 'Made over MCP in linked' then task_list; afterwards 'hug task ls' in main printed 'vrb4v8  todo     feature  made-over-mcp-linked  Made over MCP in linked'; 'git worktree list | wc -l' = 3. MCP and TUI both open through cli.App.open -> gitbranch.Open(..., WithSharedCheckout()) (huginn/run/run.go, hooks.go tui.Run(s.s)).

check 5: checkout folder removed: 'open: branch hug/tasks is registered as checked out at /private/tmp/065wwn.b5d0/m/.hug/tasks but that folder is gone' / 'fix: git -C '/private/tmp/065wwn.b5d0/m' worktree prune' (run: then ls works). Folder git no longer knows: 'fix: mv '.../m/.hug/tasks' '.../m/.hug/tasks.stale' && git -C '.../m' worktree prune && git -C '.../m' worktree add '.../m/.hug/tasks' 'hug/tasks'' ; running it printed 'HEAD is now at a02ef64 hug task new vrb4v8' and 'hug task ls' listed the 3 tasks. Test TestMissingCheckoutFixIsPrune runs both fix lines with sh -c.

check 6: tree = HEAD 6e47fca + the 8 files of fd928d0 (byte-identical, cmp), tui/ at HEAD because poodle's uncommitted tui.go does not build ('huginn/internal/tui/tui.go:97:34: m.loads undefined'). 'GOMAXPROCS=2 GOFLAGS=-p=2 nice -n 19 go test -count=1 ./...': every package ok, e.g. 'ok  	github.com/veridit/hug/huginn/internal/cli	44.633s', 'ok  	github.com/veridit/hug/huginn/gitbranch	14.585s', last line 'ok  	github.com/veridit/hug/muninn/run	11.889s'. Commit fd928d0.

memories (crab 13:41: same rule as tasks, no opt-in option): in linked worktree /tmp/065mem.ic9M/l, 'hug mem project' printed 'file:  /private/tmp/065mem.ic9M/m/.muninn-notes/memories.jsonl'; 'hug mem remember --project ...' then 'hug mem sync' printed '... file changed=true | branch muninn-notes at .muninn-notes | migrated=false'; 'git -C m show muninn-notes:memories.jsonl' has '"content":"Synthetic memory written in the linked worktree"' (commit '5574477 Update muninn-notes'); 'git worktree list' = main, linked, '/private/tmp/065mem.ic9M/m/.muninn-notes 5574477 [muninn-notes]'; linked has only .git. Test TestBranchSyncFromLinkedWorktreeUsesMainCheckout (main first / linked first) passes; on fd928d0's muninn it fails: 'File(): linked .../linked/.muninn-notes/memories.jsonl ... want .../main/.muninn-notes/memories.jsonl'. Full niced suite on HEAD 71876dd + 7 files: all ok, last 'ok  	github.com/veridit/hug/muninn/run	8.801s'.
