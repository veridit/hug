---
id: 3z65k1
title: "git segfaults under hug's test load on the owner's Mac (3 today)"
state: "doing"
type: bug
priority: ""
assignee: ["kestrel"]
labels: []
created: 2026-11-17
updated: 2026-11-17
doing_since: "2026-11-18T01:18"
stage: "active"
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
---

## Goal
Know why git crashes with SIGSEGV during hug's cli tests on the owner's machine, and whether real hug users can hit it; hug names the crash as git's, with a retry or fix line.

## Purpose
Three local -race runs of huginn/internal/cli failed on 2026-11-17 with git dying of SIGSEGV, never on CI: TestLintHidesClosedTasks 'git add: signal: segmentation fault'; TestEightWritersMixedCommandsAllLand 'diff --cached --name-only: signal: segmentation fault' and 'diff --name-status -z --find-renames ... : signal: segmentation fault' (then '159 commits for 160 writes'). A write that fails because git crashed looks like hug's bug.

## Principle
Observe first: no retry loop until the cause is known. Do not touch the owner's git install.

## Relevant details
git: /usr/bin/git 2.54.0 (Apple Git-157); /opt/homebrew/bin/git also on PATH. No git crash report in ~/Library/Logs/DiagnosticReports. All three ran under nice -n 19, GOMAXPROCS=2, with other agents' test loads at the same time (machine load 286-323 per crab 22:06). Logs: /Users/jhf/.jcode/scratch/kestrel-syncstall-out/run-9.txt, run-12.txt, /Users/jhf/.jcode/scratch/kestrel-b6jgpn-race3.txt.

## Depends on

## Acceptance
- [ ] the crash is reproduced outside hug's tests or tied to a cause (memory pressure, a git bug, a hook), with evidence in a note
- [ ] hug reports a git that died of a signal as git's crash, with a fix: line to retry

## Notes

kestrel 23:10, standalone repro (crab 22:53): /Users/jhf/.jcode/scratch/kestrel-3z65k1-loop.sh, 8 parallel workers for 600 s, each looping git add -A / diff --cached --name-only / commit / status --porcelain / rev-parse --verify HEAD / diff --name-status -z --find-renames, under nice -n 19. Result: 'failures: 0; segv (exit 139): 0', iterations per worker 1319-1365 (about 64,000 git processes). Load average before '{ 372,53 358,15 344,16 }', after '{ 608,63 516,81 432,84 }'. Gap: sh resolved git to /opt/homebrew/bin/git (git version 2.54.0), while the hug tests' Go exec.LookPath may find /usr/bin/git (Apple Git-157) first depending on PATH; this run says nothing about Apple's git yet.


## History
- 2026-11-18T01:18 todo -> doing/active (take)
