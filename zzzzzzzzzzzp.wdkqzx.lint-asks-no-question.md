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
doing_since: "2026-11-17T23:22"
stage: "active"
model: "claude-opus-5-5/high"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
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
- [ ] shipped-not-done never offers a fix that hug refuses: with open checks its fix names the first open check ('hug task check ID N', or strike it), not 'hug task done ID' (test: 8/9 settled, active, gives 'fix: hug task check ID 9')
- [ ] Every hug task write command (take, set, check, note, wait, review, design; ask after 41hhgn) and MCP task_update end with the task's own findings and fix: lines when it has any, and print nothing extra when clean (test per command family, fails on main)
- [ ] Asking a person (wait --on today, ask after 41hhgn) refuses while the task has a broken finding, naming it with a fix: line; after the fix the ask goes through (test)
- [ ] shipped-not-done raises nothing when every open check is the one the person is asked about (test with 6ephqk's shape: 8/9, shipped, asked about check 9)

## Notes


crab 23:48 local, from Jørgen's 'is this lint truly clear?': besides the waiting case, the fix line 'hug task done 6ephqk' is wrong on its own. Check 9 is open, so hug task done refuses with unchecked-acceptance (verified on a scratch board). A fix: line must be one hug accepts.
file:wdkqzx-proposal.txt (5261 bytes, sha256:fb2a73dc)
file:wdkqzx-fail-before.txt (358 bytes, sha256:f7e842b0)
file:wdkqzx-fail-before-tui.txt (171 bytes, sha256:a582df56)
Proposal for crab before building: file:wdkqzx-proposal.txt (exact block, refusal and question texts; order: fix lines + block on every write first; the ask refusal with asked_check after 41hhgn, else 6ephqk deadlocks). The suppress build (WaitsOnPerson, schema 19) was never pushed: local branch ci/wdkqzx only, set aside. Its fail-before outputs are kept for the record: file:wdkqzx-fail-before.txt, file:wdkqzx-fail-before-tui.txt.
crab 00:05 local rulings on file:wdkqzx-proposal.txt: (1) order approved: A (block on every write) + C1-2 (shipped-not-done fix lines) ship now; B (ask refusal) + C3 (no finding for the asked check) after 41hhgn lands asked_check; never parse 'check N' from free text. (2) review --by and design --for count as handing to a person (B applies), except design --for is not blocked by incomplete-task/draft findings (Goal or Acceptance missing), since the design answer often fills them; test that a draft task can still get design --for. (3) done and drop get no block (done already refuses open checks; drop needs a reason).
file:wdkqzx-c3-fail-before.txt (496 bytes, sha256:6ef6fcb8)
file:wdkqzx-c4-fail-before.txt (693 bytes, sha256:67b3d3cd)
## History
- 2026-11-17T23:22 todo -> doing/active (take)
