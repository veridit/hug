---
id: 557s7n
title: "Setup's remembered checks drop a failed write silently"
state: "doing"
type: bug
priority: ""
assignee: ["kestrel"]
labels: []
created: 2026-11-17
updated: 2026-11-17
doing_since: "2026-11-17T22:48"
stage: "active"
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
---

## Goal
hug setup's cache writes (setup-backlog.json, setup-files-*.json in the git common dir) replace the file the way task files are replaced (internal/sharefile rename, POSIX semantics on Windows, a temp file of its own), and a failed write says so on stderr once per command instead of vanishing.

## Purpose
Windows CI 44172854884 (e971d24): 'TestSetupFilesStepRemembered: remembered files check: []' with a valid key and a clean board: writeAtomic (backlogcheck.go) wrote path.tmp then os.Rename and ignored both errors. crab 21:47: a swallowed rename error is a bug whatever a stress run shows.

## Principle
A remembered answer is an optimization: a failed write never fails setup, but it is never silent. Reads go through sharefile.ReadFile so they never block a rename on Windows.

## Relevant details
Two writers shared the fixed name path.tmp, so one's rename found the other's temp gone. kbc99j has the same Windows failure for task files (Access is denied, 165 of 169 renames).

## Depends on

## Acceptance
- [ ] a test races 2 writers and 4 readers on one cache file (the kbc99j probe pattern): no write or read fails; it fails before the fix
- [ ] a test makes the cache folder unwritable: setup still answers, and stderr names the failed write once, with a fix: line

## Notes


## History
- 2026-11-17T22:48 todo -> doing/active (take)
