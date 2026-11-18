---
id: 4c7327
title: "Secret check flags 'task-900' in a file path as a credential (sk-900)"
state: "doing"
type: bug
priority: ""
assignee: ["hatchling"]
labels: []
created: 2026-11-18
updated: 2026-11-18
doing_since: "2026-11-18T06:32"
stage: "active"
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
---

## Goal
hug's secret check does not flag ordinary words that contain 'sk-' followed by digits, such as 'task-900-...' in a path, while still catching real keys (sk-..., sk-ant-..., sk-proj-...).

## Purpose
crab 00:39 local, attaching seid-upgrade-dry-run.md to asr4t2: refused with 'credential-shaped text near sk-900...'. The match was 'task-900-task979-ownership-reconciliation-2026-08-17.md', a file name. A false refusal teaches agents to pass --safe-reason by habit, which defeats the check.

## Principle
Catch real keys, never ordinary text. A key starts at a word boundary. A test with both kinds fails first.

## Relevant details
secretHint in huginn/internal/cli (attach). The same pattern may exist in muninn's remember secret check: hatchling checks both.

## Depends on

## Acceptance
- [x] attach of a file holding 'task-900-task979-x.md' succeeds with no secret warning; a file holding 'sk-' plus 40 base62 characters is still refused (test, fails on main)
- [x] The memory secret check gets the same test, or the note says it uses a different rule

## Notes


file:4c7327-failbefore.txt (605 bytes, sha256:63a053e5)
Landed d544f777 on main. Fail-before transcript attached: TestAttachSkInsideAWordIsNoSecret fails on b1063988 with credential-shaped text near sk-900. Attachment sk- matching now requires the same preceding boundary already used by memory write-time check and CI check-secrets.sh. Added memory cases and a temporary-repo test invoking the actual CI secrets script: path passes, real key fails. Focused tests and repository secrets script pass. Race results: ok github.com/veridit/hug/huginn/internal/cli 119.304s; ok github.com/veridit/hug/muninn/internal/project 6.754s; ok github.com/veridit/hug/internal/repocheck 1.273s. Main ancestor and CHANGELOG marker checks passed before fast-forward push.
## History
- 2026-11-18T06:32 todo -> doing/active (take)
