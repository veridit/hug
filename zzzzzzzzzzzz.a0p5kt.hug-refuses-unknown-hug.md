---
id: a0p5kt
title: "hug refuses an unknown HUG_* variable instead of ignoring it"
state: "doing"
type: bug
priority: "high"
assignee: ["wyvern"]
labels: []
created: 2026-11-18
updated: 2026-11-18
doing_since: "2026-11-18T03:41"
stage: "active"
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
---

## Goal
Any hug command run with an environment variable named HUG_* that hug does not know stops before doing anything, names the variable, and gives the closest real one as a fix line.

## Purpose
wyvern 2026-11-18 03:40 local: a forget benchmark set HUG_NO_SYNC=1 (not a real variable; the real one is HUG_SYNC=off), so hug ignored it and synced 11 forgets from a store copy into origin/muninn-notes between 03:19 and 03:37 local. Repaired by hand (muninn-notes 76b153c6). A misspelt safety switch must never fail open; Jørgen's rule: nothing writes silently.

## Principle
Fail closed on a misspelt switch: unknown HUG_* is a usage error (exit 2) with a fix line. The list of known variables lives in one place in the code and is what hug help prints; tests and the release check that every HUG_ name in the code is on it. No allowlist of extras.

## Relevant details
Known today (git grep on main): HUG_HOME, HUG_SYNC, HUG_MODEL, HUG_USER, HUG_NO_EMBED, HUG_FULL_SPEED, HUG_BIN_DIR, HUG_SERVE, HUG_CPUPROFILE, HUG_BENCH, HUG_MCP_RESUME, HUG_TUI_STATE, HUG_TUI_ARTIFACTS, HUG_MIGRATION_COPY, HUG_REPAIR_COPY, HUG_UPDATE_GOLDEN, plus HUG_TEST_* and HUG_CAPTURE_KILL_* used by tests. Check at startup in cmd/hug before dispatch (also hug mcp and hooks). Example: 'hug mem forget: HUG_NO_SYNC is not a hug variable' then 'fix: HUG_SYNC=off hug mem forget ...'.

## Depends on

## Acceptance
- [ ] HUG_NO_SYNC=1 hug mem forget X exits 2, writes nothing, and the fix line names HUG_SYNC=off (test, fails first on main)
- [ ] A test greps the code for every HUG_ name and fails if one is missing from the known list, so a new variable cannot be forgotten
- [ ] hug help lists every user-facing variable with one line each

## Notes


## History
- 2026-11-18T03:41 todo -> doing/active (take)
