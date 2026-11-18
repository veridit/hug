---
id: 55a7xx
title: "CI and the pre-push check refuse leftover merge conflict markers"
state: "doing"
type: chore
priority: "medium"
assignee: ["hatchling"]
labels: []
created: 2026-11-18
updated: 2026-11-18
doing_since: "2026-11-18T05:29"
stage: "active"
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
---

## Goal
No commit with a merge conflict marker line (<<<<<<<, |||||||, =======, >>>>>>> at line start) in any tracked text file reaches main: a test fails on it locally and in CI, naming the file and line.

## Purpose
wyvern 2026-11-18 04:38 local: CHANGELOG.md on main held a stray '||||||| parent of c9ad1b2e' line under v0.7.54 from a rebase (27643306, tn8bmn); every lane resolves CHANGELOG conflicts by hand several times a night, so it will recur. Removed in 4336e2b0.

## Principle
Catch it at the moment of the slip, for the agent who made it: one fast Go test in the normal suite (no new tool), with a fix line naming the file and line. Goldens and testdata that need such lines are listed by path in the test, not by pattern.

## Relevant details
A test under the repo root's test package that walks git ls-files. Exclude only named fixtures if any contain markers on purpose.

## Depends on

## Acceptance
- [ ] The test fails on 27643306's tree with 'CHANGELOG.md:NN: leftover conflict marker' and passes on main (fails first)
- [ ] It runs in go test ./... in under a second on this repo (counting: one git ls-files call)

## Notes


file:55a7xx-failbefore.txt (416 bytes, sha256:315575f8)
## History
- 2026-11-18T05:29 todo -> doing/active (take)
