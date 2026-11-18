---
id: v3k6cp
title: "Board nudges when the assignee is not the agent doing the work"
state: "done"
type: bug
priority: ""
assignee: ["hatchling"]
labels: []
group: ""
rank: "01"
created: 2026-11-17
updated: 2026-11-17
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
---

## Goal
The assignee on a doing task is always the agent actually working it. When a session works a task (it took it, or works in its recorded worktree or branch) but the assignee names someone else, the recall hook nudges once with the fix: hug task set ID --assignee SESSION. take and MCP task_take default the assignee to the session's own name from the harness, and --as with a different name is accepted but noted. Lint flags a doing task whose assignee has had no live session for over a day.

## Purpose
Jørgen 2026-11-17 10:02: after the overnight reaping, the respawned workers (wyvern, nautilus, hatchling) worked 44w1ev, zhvkmj, tzc9hb and others for an hour while the board still named humpback, vole, guppy and dragon; he had to point it out. The board nudges (qawmnb, BOARD4.md) cover take, work, done and lint, but none covers a wrong owner: 44w1ev was already doing, so no rule fired. The coordinator made it worse by spawning wyvern with 'You are humpback'.

## Principle
The board says who is working, so it must be true without anyone remembering. The session's name comes from the harness hook payload, never guessed, never from the model. One line, with the exact command, under the existing nudge rates. Coordinators never give a worker a name other than its session name.

## Relevant details
Nudge rules: muninn/docs/BOARD4.md; 'a task is the session's when it took it, its name is the assignee, or the recorded worktree or branch is where it works'. New kind 'owner': the task is the session's by take, worktree or branch, but the assignee is another name. Liveness for the lint rule: the harness's session list (jcode: ~/.jcode/sessions/*.json) when available, else skip the rule. Related: 805p6y (current-work line), 5dxzxe (own harness).

## Depends on

## Acceptance
- [x] a session working a doing task assigned to another name gets one 'owner' line ending in hug task set ID --assignee NAME (replayed on this morning's wyvern session: fires on 44w1ev)
- [x] hug task take without --as assigns the harness session name; with a different --as it says so (test)
- [x] hug task lint reports a doing task whose assignee has no live session for over 24 h, when the harness's session list is readable (test)
- [x] the replay precision of all nudge kinds stays at or above 0.88 (BOARD4 sets)

## Notes

guppy (session hatchling) 2026-11-17 12:33: ci/v3k6cp ec11186. (1) muninn/internal/recall/boardcue.go kind 'owner' (TestBoardOwnerLine); replay $JCODE_SCRATCH_DIR/board4/v3k6cp/wyvern.jsonl on session_wyvern_1794905939974_6ffa78e419a994c9: 1 line, 2026-11-17T09:46:55Z 'Task board: you are working on 44w1ev but the board says humpback. If it is yours now: hug task set 44w1ev --assignee wyvern'. (2) take.go warning when --as/HUG_USER differs from JCODE_SESSION_ID's name; TestTakeUsesJcodeSessionName checks no warning for the session's own name and 'warning: assignee humpback is not this session (wyvern); if you are wyvern: hug task set ID --assignee wyvern'. (3) task/owner.go stale-owner from JCODE_HOME/sessions mtimes, TestOwnerFindings (stale > 24 h, a newer session keeps a name live, persons/models never judged, unreadable list says nothing). (4) six BOARD4 sets, DeepSeek Flash blind judge: main 97 lines 87 right 7 false 0.926; with owner 109 lines 99 right 7 false 0.934; owner 14 lines 13 right (false: poodle rx74bp, told it was crab's). Counts in BOARD4.md.

hatchling 12:48: rebased on main 6d88d74 (help.go conflict: kept main's done/reflections text, re-applied the stale-owner lint line), ci/v3k6cp b1213ea, fresh CI 44077138292 (44075111188 was ec11186, pre-rebase). Local go test ./huginn/... ./cmd/... ./muninn/internal/recall/ green. Push to main when 44077138292 is green on all three.

hatchling 12:34: on main 206038a (ff from c9ea6ed), CI 44082371484 green ubuntu/macos/windows.
