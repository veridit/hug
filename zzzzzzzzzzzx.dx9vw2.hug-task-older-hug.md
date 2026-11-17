---
id: dx9vw2
title: "hug task: an older hug says the board is newer, with an upgrade fix"
state: "doing"
type: feature
priority: ""
assignee: ["hatchling"]
labels: []
created: 2026-11-17
updated: 2026-11-17
doing_since: "2026-11-17T21:13"
stage: "active"
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
---

## Goal
When the hug binary is older than the board it reads (a task file says state:, the binary knows only status:), every command says so once, with fix: brew upgrade veridit/tap/hug (or hug upgrade from a source build), instead of failing on an empty state.

## Purpose
2026-11-17 19:20: hatchling's hug (built 14:36) ran 'hug task wait s6ff5k' on the board after abcf36b (status: -> state: in 126 tasks) and got 'not-allowed s6ff5k:  -> doing/waiting is not a move a command makes' (empty state) with fix 'hug task take', which sent it the wrong way. Note and attach from the old binary landed, so the wrong state was silent elsewhere.

## Principle
One clear line naming both sides (board layout vs this hug's version), not a lint per task. A newer hug reading an older board keeps working (old-layout lint already covers that side).

## Relevant details
Reflection on ce3dmq (2026-11-17). Possible signal: a board-level layout version (hug.toml or a marker on hug/tasks) that hug compares to the one it knows; or a task file with an unknown front-matter key that is a known future key. Crab 19:41: small bug, after typq1e.

## Depends on

## Acceptance
- [ ] test: an old-layout binary (simulated layout version) on a newer board: a write command fails with 'this board is newer than this hug' and fix: brew upgrade veridit/tap/hug, before any write (test fails before)
- [ ] read commands warn once on stderr and still print

## Notes


file:fail-before.txt (700 bytes, sha256:87239cef) TestOlderHug* (checks 1, 2 only) on origin/main 4fb0c92: note exits 0 and writes, ls warns 0 times
## History
- 2026-11-17T21:13 todo -> doing/active (take)
