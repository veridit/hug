---
id: h9ymk8
title: "AGENTS.md names HUGINN_UPDATE_GOLDEN; the tests read HUG_UPDATE_GOLDEN"
state: "done"
type: docs
priority: ""
assignee: ["crab"]
labels: []
group: ""
rank: "zzzzzzzzzzw3"
created: 2026-11-19
updated: 2026-11-19
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
done_at: "2026-11-20T04:32"
done_release: "v0.7.86"
parent: "ddpxs8"
---

## Goal
AGENTS.md and any doc naming the golden update variable give the one the tests read.

## Purpose
tulip 2026-11-19 (a82j3n): the task and AGENTS.md said HUGINN_UPDATE_GOLDEN=1, the TUI tests read HUG_UPDATE_GOLDEN; tulip set both to update goldens.

## Principle
One name. Check with git grep which the code reads; fix the docs, no alias unless the code reads both.

## Relevant details
AGENTS.md 'TUI goldens' line; git grep -n 'UPDATE_GOLDEN'.

## Depends on

## Acceptance
- [x] git grep UPDATE_GOLDEN shows docs and code naming the same variable (note with output)

## Notes

Add to AGENTS.md with this fix (bat's wacc33 reflection): a new HUG_* variable, test ones too, goes on internal/env Known, or TestEveryHUGNameIsKnown fails.


check 1: Already fixed on main by 38ecd7d7 (fkpyg3). git grep UPDATE_GOLDEN on origin/main 91d0a014: AGENTS.md:25 'HUG_UPDATE_GOLDEN=1 go test -run TestTmuxGolden', huginn/internal/tui/README.md:106 HUG_UPDATE_GOLDEN=1, internal/env/env.go:69 {"HUG_UPDATE_GOLDEN", Test, ...}, every test reads HUG_UPDATE_GOLDEN; the one other match is huginn/internal/task/reflect_test.go:10 oldGolden, a fixture quoting old board text. The HUGINN_ name only survives in the owner's stale checkout /Users/jhf/Projects/hug (main at e6a135ae, 2026-11-16, 87 changed files), which I did not touch. Added the env Known rule (bat's wacc33 reflection) to AGENTS.md in c3d90499.
Landed on main c3d90499 by fast-forward (CI 44667474508 8/8 green on that exact SHA). Docs-only.
Reflection (open, 2026-11-20, crab): The task's premise was already fixed on main by 38ecd7d7 (fkpyg3); the stale owner checkout at e6a135ae still shows the old name, which is how the report came about.
landed c3d9049 on main 2026-11-20: AGENTS.md: a new HUG_* variable goes on internal/env Known (h9ymk8)
shipped in v0.7.86 (c3d9049)
## History
- 2026-11-20T04:24 todo -> doing/active (take with p277bp)
- 2026-11-20T04:32 doing/active -> done (done); was doing_since: 2026-11-20T04:24; doing_with: p277bp
