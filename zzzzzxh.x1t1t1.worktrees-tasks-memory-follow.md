---
id: x1t1t1
title: "Worktrees: record branch and worktree on take; branch as a condition"
status: "doing"
type: "feature"
priority: "high"
assignee: ["kestrel"]
labels: ["needs-discussion"]
created: 2026-11-16
updated: 2026-11-16
model: "claude-opus-5-5"
waiting_on: "jhf: check 7: a Statbus task taken from a linked worktree (git worktree add) via CLI and MCP, or strike it; only separate clones are in use (14:40)"
waiting_since: "2026-11-17T15:39"
waiting_asked_by: "calf"
stage: "waiting"
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
- [x] show, ls --json and the TUI detail show the branch; in-progress rows show it when it is not the main branch (test and golden)
- [x] memories captured or remembered in a worktree record its branch (test)
- [x] lint flags a doing task whose branch is merged into main or deleted, with fix: hug task done ID (test)
- [ ] tried in practice: agents in two Statbus worktrees (one CLI, one MCP) take tasks and the board shows the right branch for each (evidence in a note)

## Notes

check 2 (dragon, 3b2c0e3): TestTakeRecordsBranchAndWorktree: in a linked worktree on feat/x, 'hug task take ID' (no flags) -> task branch feat/x, worktree <linked dir>; file 'branch: "feat/x"' and 'worktree: "<dir>"'; show 'taken by: tester · test-model · feat/x'; ls row 'Worktree work  [tester · test-model · feat/x]'; ls --json has "branch":"feat/x"; take on main records branch main but the ls row stays '[tester · test-model]'; --branch/--worktree override. task: TestTakeAtRecordsBranchAndWorktree (move on retake, kept when no place). 'go test ./huginn/internal/cli/ ./cmd/hug/' -> 'ok  	github.com/veridit/hug/huginn/internal/cli	52.667s'

check 3 (dragon, 3b2c0e3): TestTakeRequiresCwdAndRecordsBranch: task_take without cwd, with 'relative/dir', and with a temp folder outside git each fail with a fix naming cwd (e.g. 'task_take with cwd set to your working directory (absolute path), ...'); with cwd=<linked worktree on feat/x> the server (running in the main checkout) records branch feat/x and that worktree. 'go test ./huginn/internal/mcp/' -> 'ok  	github.com/veridit/hug/huginn/internal/mcp	9.945s'. setup/AGENTS.md: 'task_take (model and cwd required: your working folder, its branch is recorded; agent optional)'.

check 5 (dragon, 3b2c0e3): TestMemoriesRecordWorktreeBranch: Binding.Remember from a linked worktree on feat/x -> memories.json branch "feat/x"; from the main checkout on main -> none; Capture from the worktree -> new memory branch "feat/x"; outside git -> none. '--- PASS: TestMemoriesRecordWorktreeBranch (2.25s)'; app/ingest/run/mcpserver ok. Not yet: MCP memory_remember uses the server's binding folder (no per-call cwd), so an MCP server started in the main checkout records no branch.

check 6 (dragon, 817c7b0 on ci/x1t1t1): task.BranchFindings, outside task.Lint (not cached, per humpback), 2 git calls per lint (for-each-ref + branch --merged), doing tasks with a non-main branch only. TestLintFlagsDoingTaskOnMergedOrDeletedBranch: 'v.d2bbg2.gone-work.md:1: [branch-gone] branch feat/gone is deleted, but d2bbg2 is still doing / fix: hug task done d2bbg2' and the same for feat/merged ('is merged into main'); unmerged feat/open not flagged (exactly 2 [branch-gone]); after done the finding goes. 'go test ./huginn/internal/cli/' -> 'ok  	github.com/veridit/hug/huginn/internal/cli	75.061s', task ok 3.881s.

check 4 partial (dragon): show 'taken by: NAME · MODEL · BRANCH', ls rows add the branch when not main, ls --json carries branch/worktree (TestTakeRecordsBranchAndWorktree); TUI detail header 'assigned:dragon · claude-opus-5-5 · feat/x' (TestDetailHeaderShowsBranch, ea4a33c, tui ok 43.900s, no golden change). Open: in-progress rows in the TUI overview (render.go sectionDoing), after vole's z6kz20 lands (vole 17:45), then regenerate goldens. Check 7 (Statbus trial) after that.

landed on main 2250330..9d1f382 (ff), CI run 43924391679: ubuntu/macos/windows success. Open: check 4 TUI rows (after vole z6kz20), check 7 Statbus trial.

Released in v0.7.31 (checks 2,3,5,6). Open: 4 (TUI in-progress rows show branch, after vole's z6kz20) and 7 (Statbus trial).

check 4 evidence (dragon, fd25eb0 on ci/x1t1t1-tui, on v0.7.34): show/ls --json/TUI detail already carried branch (cli.go takenBy, tui/detail.go owner line). New: TUI IN PROGRESS rows, m.rowOwner = owner then ' · branch' unless Options.MainBranch (gitbranch.DefaultBranch). Test huginn/internal/tui/owner_branch_test.go TestInProgressRowShowsBranch: 'poodle · feat/x', main hidden, 80 cols 'alice · feat/te'. Goldens: GOMAXPROCS=2 HUGINN_UPDATE_GOLDEN=1 go test -run TestTmuxGolden -> 4 overview.txt files changed; 140x40 row: '> ◑ 1/2 000002   doing   bug     critical Bug task 02: make the terminal workflow observable                    alice · feat/terminal-000002'; 80x24: 'alice · feat/te…'. Local: ok github.com/veridit/hug/huginn/internal/tui 45.524s. CI 43943254468 pending; tick after green and on main.

Jørgen 2026-11-16: check 7 = observe real use in Statbus (live production), read-only, together with 0hkaa5 check 6.

2026-11-17 09:00 crab: check 4 landed as 8e9df0b (CI 44049093502 green), released in v0.7.38. Open: check 7, Statbus observation.

kestrel 2026-11-17 11:02, Statbus observation (read-only, nothing written there): no hug writes since v0.7.38. Installed: /opt/homebrew/Cellar/hug/0.7.39 (hug v0.7.39, v0.7.38 tagged 2026-11-17 09:59). ~/ssb/statbus: hug/tasks last commit 2eb713fd8 2026-11-16 18:26:30 'hug task note hmgadt', origin/hug/tasks same; no .git/hug/sync/ directory (no syncer has ever run). ~/ssb/statbus_test: last commit 2375234a8 2026-11-16 22:41:26 'hug task note 00rnq5', origin/hug/tasks 031c91eb2 (1 local commit unpushed from before background sync); no .git/hug/sync/. statbus_rel_2026113 does not exist (git worktree list: statbus, statbus-db-seed, .hug/tasks; statbus_test has 3 scratch worktrees + .hug/tasks). Nothing to measure yet; check again later.

kestrel 2026-11-17 12:35, Statbus worktree observation (read-only). Board: one doing task records a branch: 00rnq5 'field-s-timeline-shows' assignee rabbit, model claude-opus-5-5, branch "statbus-628-all-forms", worktree "/Users/jhf/ssb/statbus_test" (file ~/ssb/statbus/.hug/tasks/0000000005p.00rnq5.field-s-timeline-shows.md). statbus_test is a separate clone (not a linked worktree) on branch statbus-628-all-forms, so this is the separate-clone case; its own linked worktrees (.jcode/scratch/wt-edit-fix on statbus-edit-empty, two detached livedb-upgrade checkouts) have taken no task. statbus_rel_2026113 does not exist. Not yet seen: an MCP take with cwd in a linked worktree, and a CLI take in a second worktree. Sync between the two clones: see 0hkaa5 note 12:35 (0 errors, 3-6 s commit to push). Check again later.

kestrel 12:40, check 7 waits on: real use of a LINKED worktree in Statbus (git worktree add) where an agent takes a task, once via CLI and once via MCP task_take with cwd. So far (12:35) Statbus has used only a separate clone (statbus_test, branch statbus-628-all-forms, 00rnq5 recorded with that branch and worktree, correct). No linked statbus worktree has taken a task; statbus_rel_2026113 does not exist. Nothing for hug to do until that happens; recheck the boards read-only then.


kestrel 14:40, Statbus observed read-only (nothing written there). ~/ssb/statbus has one linked worktree, statbus-db-seed (branch db-seed); no task records it. ~/ssb/statbus_test is a separate clone (git rev-parse --git-common-dir prints .git), not a linked worktree. In it, rabbit (claude-opus-5-5) took 6 tasks today and each records branch 'statbus-628-all-forms' and worktree '/Users/jhf/ssb/statbus_test' correctly (ezp3h0, hzyejk, qtx2xm, mtw92d, s83by9, 00rnq5). Check 7 needs an agent to take a task from a linked Statbus worktree via CLI and via MCP. That only happens when Jørgen's work uses one, so we can't produce it (crab 14:39). Waiting on jhf: use a linked worktree for a Statbus task, or strike check 7 with a reason.
## History
- 2026-11-17T14:33 doing -> doing/waiting (wait)
