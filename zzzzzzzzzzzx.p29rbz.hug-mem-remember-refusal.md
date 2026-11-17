---
id: p29rbz
title: "hug mem remember: a refusal says NOT STORED first and exits non-zero"
status: "doing"
type: bug
priority: "high"
assignee: ["hatchling"]
labels: []
created: 2026-11-17
updated: 2026-11-17
doing_since: "2026-11-17T16:14"
stage: "active"
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
---

## Goal
When hug mem remember refuses (near-duplicate, possible secret, project-specific), the output starts with 'NOT STORED:' and the exit code is non-zero (a distinct code, documented), on CLI and in the MCP reply (isError or a clear stored:false first).

## Purpose
Seid foreman (orangutan) requests 2026-11-17, file: seid-lang .jcode/scratch/TASK-2057/hug-agent-requests.md, read by crab 15:07, item 4. A Seid agent read the near-duplicate refusal as success, ran hug mem forget X right after, and lost both memories.

## Principle
A refusal never looks like success, in a script or to a reader. The fix: line still offers --reinforce / --new-reason.

## Relevant details
Messages: muninn/internal/app/dedupe.go:23, binding.go:559, places.go:163, all start 'not stored yet:'.

## Depends on

## Acceptance
- [ ] each of the three refusals: output starts 'NOT STORED:', exit code non-zero (test)
- [ ] the MCP remember reply for a refusal is marked as an error or stored:false first (test)

## Notes


hatchling 15:19: fix 0c72206 on ci/p29rbz (from main b8f9000), CI 44112858767. Tests fail before the fix (stash of app/*.go, refusal branch disabled): TestRememberRefusalSaysNotStoredAndFails 'notstored_test.go:70: exit 1, want 3:' for duplicate/secret/place with output 'hug mem: not stored yet: this looks like memory bkpk4q: ... Same fact?'; TestMCPRememberDuplicateQuestionAndAnswers 'dedupe_test.go:29: refusal must start NOT STORED: "not stored yet: this looks like memory bkpk4q...'. After: all PASS. Exit code 3 = refused, documented in hug mem remember --help and CHANGELOG; other errors stay 1. MCP: the SDK already sets IsError for the returned error; its text now starts NOT STORED:. The test uses a new openApp hook in run/main.go to give Main the fixture's synthetic embedder.
## History
- 2026-11-17T16:14 todo -> doing/active (take)
