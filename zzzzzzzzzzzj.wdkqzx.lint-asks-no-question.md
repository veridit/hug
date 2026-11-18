---
id: wdkqzx
title: "Lint goes to the agent writing the task, not to the person it asks"
state: "doing"
type: bug
priority: "high"
assignee: ["hatchling"]
labels: []
created: 2026-11-17
updated: 2026-11-17
stage: "active"
model: "claude-opus-5-5/high"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
doing_since: "2026-11-18T01:43"
---

## Goal
A task that waits on a person may still have lint findings, but the agent sees them while it writes the task, not the person afterwards. Every hug task write command (CLI and MCP: take, set, check, note, wait, ask, review, design) ends with that task's own findings and their fix: lines. Asking a person refuses while the task has a broken finding the agent can fix, so a person is never handed a broken task. Every fix: line is one hug accepts.

## Purpose
Jørgen 2026-11-17 23:20 local: 6ephqk was in WAITING and LINT at once; 'waiting for me is not a lint issue'. 23:48: 'is this lint truly clear?' (its fix 'hug task done 6ephqk' is refused, check 9 open). 23:51, superseding the suppress plan: 'you can lint on a task that waits for me, but the lint should have been returned to the agent that tried to modify the task to fix it up in the interaction with the task.'

## Principle
Guide the agent at the moment of the slip, not the person later. Recomputed from state, nothing stored. A refusal names the finding and ends with a fix: line hug accepts. Tests fail first.

## Relevant details
Code: huginn/internal/task/questions.go noteQuestion (shipped-not-done fix is 'hug task done ID' even with open checks); lint per task exists for hug task lint ID. Write paths: internal/cli and internal/mcp task_update. Ask path: 'hug task wait --on' today, 'hug task ask' after 41hhgn. shipped-not-done must not deadlock the ask: when every open check is the one being asked about (asked_check), the release question is the ask itself, so no finding. hatchling proposes the exact output and refusal text on the ticket before building.

## Depends on

## Acceptance
- [ ] ~~a doing task with a 'shipped in' note and waiting_for: jhf gives no shipped-not-done; the same task active does (test, fails on main)~~ (not relevant: Jørgen 23:51: lint is not suppressed on a task waiting on a person; it goes to the writing agent (checks 4-6), crab, 2026-11-17)
- [ ] ~~the overview shows 6ephqk in WAITING and not in LINT (golden or test)~~ (not relevant: Jørgen 23:51: superseded; a person is never handed a broken task (check 5), crab, 2026-11-17)
- [x] shipped-not-done never offers a fix that hug refuses: with open checks its fix names the first open check ('hug task check ID N', or strike it), not 'hug task done ID' (test: 8/9 settled, active, gives 'fix: hug task check ID 9')
- [x] Every hug task write command (take, set, check, note, wait, review, design; ask after 41hhgn) and MCP task_update end with the task's own findings and fix: lines when it has any, and print nothing extra when clean (test per command family, fails on main)
- [x] Asking a person (wait --on today, ask after 41hhgn) refuses while the task has a broken finding, naming it with a fix: line; after the fix the ask goes through (test)
- [x] shipped-not-done raises nothing when every open check is the one the person is asked about (test with 6ephqk's shape: 8/9, shipped, asked about check 9)
- [x] shipped-not-done is not raised on a task whose stage says why it isn't done (doing/waiting, doing/asked, review); the same task in doing/active is (test, fails on main)

## Notes


crab 23:48 local, from Jørgen's 'is this lint truly clear?': besides the waiting case, the fix line 'hug task done 6ephqk' is wrong on its own. Check 9 is open, so hug task done refuses with unchecked-acceptance (verified on a scratch board). A fix: line must be one hug accepts.
file:wdkqzx-proposal.txt (5261 bytes, sha256:fb2a73dc)
file:wdkqzx-fail-before.txt (358 bytes, sha256:f7e842b0)
file:wdkqzx-fail-before-tui.txt (171 bytes, sha256:a582df56)
Proposal for crab before building: file:wdkqzx-proposal.txt (exact block, refusal and question texts; order: fix lines + block on every write first; the ask refusal with asked_check after 41hhgn, else 6ephqk deadlocks). The suppress build (WaitsOnPerson, schema 19) was never pushed: local branch ci/wdkqzx only, set aside. Its fail-before outputs are kept for the record: file:wdkqzx-fail-before.txt, file:wdkqzx-fail-before-tui.txt.
crab 00:05 local rulings on file:wdkqzx-proposal.txt: (1) order approved: A (block on every write) + C1-2 (shipped-not-done fix lines) ship now; B (ask refusal) + C3 (no finding for the asked check) after 41hhgn lands asked_check; never parse 'check N' from free text. (2) review --by and design --for count as handing to a person (B applies), except design --for is not blocked by incomplete-task/draft findings (Goal or Acceptance missing), since the design answer often fills them; test that a draft task can still get design --for. (3) done and drop get no block (done already refuses open checks; drop needs a reason).
file:wdkqzx-c3-fail-before.txt (496 bytes, sha256:6ef6fcb8)
file:wdkqzx-c4-fail-before.txt (693 bytes, sha256:67b3d3cd)
crab 00:46 local, ruling on kestrel's proposal (made from ggydj2): shipped-not-done asks 'is it done?'. A stage that already says why it isn't (doing/waiting with waiting_on, doing/asked, review) has answered that, so it raises no shipped-not-done. This widens C3 (which covered only the asked check) to those stages. doing/active and todo still get it. hatchling owns questions.go for this (not kestrel, to keep one writer). Test: a shipped task in waiting, asked and review gives no finding; the same task active does.
file:wdkqzx-stage-fail-before.txt (156 bytes, sha256:ba08d091)
file:wdkqzx-board-before.txt (1254 bytes, sha256:87b129af)
file:wdkqzx-board-after.txt (930 bytes, sha256:b5fa95ca)
landed a509c2f on main 2026-11-18: hug task: every write ends with the task's own lint; shipped-not-done names the…
Landed a509c2f3 on main (A + C1-2 + the stage widening, per crab's 00:05 and 00:49 rulings). Check 3: TestShippedQuestionFixIsAccepted (task/questions_test.go) fails on main: 'got ... (its note says so); nautilus: is it done?' (file:wdkqzx-c3-fail-before.txt); now 8/9 settled gives '... and check 9 is open; nautilus: does it hold, or strike it?' with 'fix: hug task check ID 9', 9/9 gives 'hug task done ID' and done then goes through. Check 4: TestEveryWriteEndsWithTheTasksLint (cli/writelint_test.go) runs take, note, check, strike, strike --undo, attach, detach, dep --add/--remove, set, wait, resume, ask, answer, review, answer and design on a task with a finding (each output has 'lint: 1 finding' + the finding + 'fix: hug task set ID --principle "..."') and on a clean one (no 'lint:' line); fails on main at 'note' (file:wdkqzx-c4-fail-before.txt). MCP: TestNewAndUpdateReturnFindingsAndOneFix now also checks update/note carries lint; fails on main ('update note: map[changed:true ...]' with no lint key). Stage widening: TestShippedQuestionNotWhileStageSaysWhy: asked of jhf, waiting on CI, review -> no question; resumed to active -> asked again; fails on main 'doing/waiting on jhf: want no question, got "shipped-not-done"' (file:wdkqzx-stage-fail-before.txt, written before 41hhgn landed; the test now uses ask). hug's board, v0.7.50 vs this build: 32 findings (4 broken, all shipped-not-done on doing/waiting 0sbvnq ggydj2 5z4zt4 jr55s8) -> 28 findings (0 broken) (file:wdkqzx-board-before.txt, file:wdkqzx-board-after.txt). -race niced: task, cli, mcp, cache ok; tui ok with -short (the tmux tests park in loadGuard at load 677; CI runs them). Cache schema 21. Left: checks 5 (ask refusal) and 6 (asked_check) now that 41hhgn landed ask/asked_check.
Back to todo for t01kqm (crab 00:38: t01kqm next after A+C). Checks 5 (ask/review/design refuse a broken task; design --for not blocked by incomplete/draft) and 6 (no shipped-not-done when every open check is asked_check) are left; 41hhgn's ask --check (asked_check) is on main now, so 6 needs no parsing.
Retaken 00:43 (crab: finish checks 5 and 6 first, then t01kqm).
landed fe4937f on main 2026-11-18: hug task: ask, review and design refuse to hand a person a task with a broken…
file:wdkqzx-c56-fail-before.txt (765 bytes, sha256:dbc00103)
Checks 5, 6 landed fe4937fa on main (CI 44212208934). ask --to, review --by and design --for call readyToHand (cli/lintdetail.go) before writing: a task with a broken finding is refused with exit 2, 'hug task CMD: ID is not ready to hand to PERSON: N broken finding(s): [code] problem; ...' and one combined fix; nothing written. design --for is a draft: missing-goal and no-acceptance do not hold it back (crab 00:05). wait is never held back. ask --check N on a shipped task where N is the only open check is not held back by shipped-not-done. Tests TestAskingAPersonRefusesABrokenTask and TestAskAboutTheOpenCheckOfAShippedTask (cli/askgate_test.go) fail on a509c2f3, see file:wdkqzx-c56-fail-before.txt; pass after. -race cli, mcp, task, tui -short: all ok (cli 194.2s, mcp 12.9s, task 29.1s, tui 110.0s); cli -race again after the rebase onto 6f7e8698: ok 151.9s.
Check 7 landed in a509c2f3 (CI 44209919112, 5/5 jobs success): stageSaysWhy (waiting, asked, review) means no shipped-not-done; TestShippedQuestionNotWhileStageSaysWhy fails before, see file:wdkqzx-stage-fail-before.txt.
## History
- 2026-11-17T23:22 todo -> doing/active (take)
- 2026-11-18T01:43 doing/active -> todo/ready (give back; was assignee: hatchling); was doing_since: 2026-11-17T23:22
- 2026-11-18T01:43 todo/ready -> doing/active (take)
