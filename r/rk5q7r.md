---
id: rk5q7r
title: "When an open reflection is older than 3 days the coordinator's board cue names…"
state: "doing"
type: feature
priority: ""
assignee: ["duckling"]
labels: []
group: ""
rank: "zzzzzzzzzzz1"
created: 2026-11-18
updated: 2026-11-18
doing_since: "2026-11-18T15:25"
stage: "active"
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
---

## Goal
When an open reflection is older than 3 days, the coordinator's board cue (muninn/internal/recall/boardcue.go) names how many there are and the command hug task reflections --by-instruction, at a break in the coordinator's work, so reflections reach a reviewer instead of piling up.

## Purpose
Split from 2k68eg check 4 (crab 2026-11-18 13:35): boardcue.go is being changed by duckling and piglet now. 0 of 56 reflections on three boards were resolved by 2026-11-18; 2k68eg makes them readable (quote parsed, grouped, unanswered counted) but nothing yet tells anyone to read them.

## Principle
Nudges go to the coordinator at a break (x325pp), never to an agent mid-task. Counts from hug task reflections, no model. One cue line, with a runnable command.

## Relevant details
Parser: huginn/internal/task/reflect.go var quoted, the third alternative '([^']{6,})' matches from a possessive apostrophe. Garbled ids: fkpyg3 scrxak n2hq8p x1ra0t cvbpdy t3fqaw ce3dmq wdkqzx ygay9c f5vx4x. Counts and quotes: /Users/jhf/.jcode/scratch/s6ff5k-feedback/REPORT.md clusters B and C2. Feeds tb0dra check 5.

## Depends on

## Acceptance
- [ ] When an open reflection is older than 3 days the coordinator's board cue names the count and hug task reflections --by-instruction (test)

## Notes


BAR (duckling 14:27Z, before any run): (1) six sets + fh3 replayed with agh2h7 (c89178f9) as base and rk5q7r as new, same flags (-coordinator crab on hug sets): outputs byte-identical except added lines of the new kind; (2) every new line is in a session named crab (the coordinator) at a break turn; 0 elsewhere; (3) judge calls <=10, spent only on new lines (2 judgings, if any). Count source: main's ParseReflections (open state), dated > 3 days before the turn. Report the count on today's boards under main's parser and under 2k68eg's (origin/2k68eg).
## History
- 2026-11-18T15:25 todo -> doing/active (take)
