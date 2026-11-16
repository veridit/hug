---
id: x1t1t1
title: "Worktrees: record branch and worktree on take; branch as a condition"
status: "doing"
type: "feature"
priority: "high"
assignee: ["dragon"]
labels: ["needs-discussion"]
created: 2026-11-16
updated: 2026-11-16
model: "claude-opus-5-5"
---

## Goal
Every git worktree already shares one board and one memory store (065wwn). On top of that, hug records which branch and worktree work happens in: take stores branch and worktree on the task next to the model; memories captured in a worktree store the branch. A merged or deleted branch becomes a condition: lint flags a doing task on it, and a memory tied to it goes to a dream item. Nothing is deleted automatically and there is no discard command.

## Purpose
Agents create worktrees for isolation, then lose sight of the board or create tasks that never reach the main checkout; a discarded experiment should take its tasks and memories with it

## Principle
Decided with Jørgen 2026-11-16: detect when hug can, require it when hug cannot. The CLI detects branch and worktree from the current directory (git rev-parse); --branch/--worktree override. MCP always requires cwd (decided with Jørgen 2026-11-16, one habit for agents): task_take without cwd fails with a fix: line, like the model; hug detects branch and worktree from that cwd. The CLI detects from its own directory. Then test in practice on Statbus with worktrees before calling it done. The branch is what matters; the worktree folder is only where to look now. Show prints 'taken by: NAME · MODEL · BRANCH'; the TUI shows the branch on in-progress rows when it is not the main branch. Ties into ez50rb: 'holds while BRANCH is unmerged'.

## Relevant details
Today: hug task finds .hug/tasks via the repository root; a linked worktree has a different root and may create its own .hug/tasks worktree or fail. Memory binds by folder/project id (rc- root commit), so worktrees of one repo already share the project scope. Needed: (1) resolve .hug/tasks through git rev-parse --git-common-dir to the main checkout; (2) record origin worktree (path + branch) on tasks and memories created there; (3) hug worktree keep|discard NAME lists and applies

## Depends on

## Acceptance
- [x] done when task:065wwn is done (hug task ls in a linked worktree shows the main board and writes to it)
- [x] hug task take in a linked worktree on branch feat/x records branch feat/x and the worktree folder on the task, without flags (test)
- [x] MCP task_take always requires cwd: without it, it fails with a fix: line; with it, branch and worktree are detected and recorded (test)
- [ ] show, ls --json and the TUI detail show the branch; in-progress rows show it when it is not the main branch (test and golden)
- [x] memories captured or remembered in a worktree record its branch (test)
- [x] lint flags a doing task whose branch is merged into main or deleted, with fix: hug task done ID (test)
- [ ] tried in practice: agents in two Statbus worktrees (one CLI, one MCP) take tasks and the board shows the right branch for each (evidence in a note)

## Notes

check 2 (dragon, 3b2c0e3): TestTakeRecordsBranchAndWorktree: in a linked worktree on feat/x, 'hug task take ID' (no flags) -> task branch feat/x, worktree <linked dir>; file 'branch: "feat/x"' and 'worktree: "<dir>"'; show 'taken by: tester · test-model · feat/x'; ls row 'Worktree work  [tester · test-model · feat/x]'; ls --json has "branch":"feat/x"; take on main records branch main but the ls row stays '[tester · test-model]'; --branch/--worktree override. task: TestTakeAtRecordsBranchAndWorktree (move on retake, kept when no place). 'go test ./huginn/internal/cli/ ./cmd/hug/' -> 'ok  	github.com/veridit/hug/huginn/internal/cli	52.667s'

check 3 (dragon, 3b2c0e3): TestTakeRequiresCwdAndRecordsBranch: task_take without cwd, with 'relative/dir', and with a temp folder outside git each fail with a fix naming cwd (e.g. 'task_take with cwd set to your working directory (absolute path), ...'); with cwd=<linked worktree on feat/x> the server (running in the main checkout) records branch feat/x and that worktree. 'go test ./huginn/internal/mcp/' -> 'ok  	github.com/veridit/hug/huginn/internal/mcp	9.945s'. setup/AGENTS.md: 'task_take (model and cwd required: your working folder, its branch is recorded; agent optional)'.

check 5 (dragon, 3b2c0e3): TestMemoriesRecordWorktreeBranch: Binding.Remember from a linked worktree on feat/x -> memories.json branch "feat/x"; from the main checkout on main -> none; Capture from the worktree -> new memory branch "feat/x"; outside git -> none. '--- PASS: TestMemoriesRecordWorktreeBranch (2.25s)'; app/ingest/run/mcpserver ok. Not yet: MCP memory_remember uses the server's binding folder (no per-call cwd), so an MCP server started in the main checkout records no branch.

check 6 (dragon, 817c7b0 on ci/x1t1t1): task.BranchFindings, outside task.Lint (not cached, per humpback), 2 git calls per lint (for-each-ref + branch --merged), doing tasks with a non-main branch only. TestLintFlagsDoingTaskOnMergedOrDeletedBranch: 'v.d2bbg2.gone-work.md:1: [branch-gone] branch feat/gone is deleted, but d2bbg2 is still doing / fix: hug task done d2bbg2' and the same for feat/merged ('is merged into main'); unmerged feat/open not flagged (exactly 2 [branch-gone]); after done the finding goes. 'go test ./huginn/internal/cli/' -> 'ok  	github.com/veridit/hug/huginn/internal/cli	75.061s', task ok 3.881s.

check 4 partial (dragon): show 'taken by: NAME · MODEL · BRANCH', ls rows add the branch when not main, ls --json carries branch/worktree (TestTakeRecordsBranchAndWorktree); TUI detail header 'assigned:dragon · claude-opus-5-5 · feat/x' (TestDetailHeaderShowsBranch, ea4a33c, tui ok 43.900s, no golden change). Open: in-progress rows in the TUI overview (render.go sectionDoing), after vole's z6kz20 lands (vole 17:45), then regenerate goldens. Check 7 (Statbus trial) after that.
