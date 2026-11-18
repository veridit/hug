---
id: rk5q7r
title: "When an open reflection is older than 3 days the coordinator's board cue names…"
state: todo
type: feature
priority: ""
assignee: []
labels: []
group: ""
rank: "zzzzzzzzzzzu"
created: 2026-11-18
updated: 2026-11-18
---

## Goal
When an open reflection is older than 3 days the coordinator's board cue names the count and hug task reflections --by-instruction (test)

Promoted from item 4 of task:2k68eg.

## Purpose
Since tb0dra landed (2026-11-17 12:35) agents have written 56 reflections on three boards (hug 49, Seid 5, Statbus 2) and none is applied or dismissed. 10 of the 28 instruction fields on hug's board are fragments cut at an apostrophe ('s TUI golden line said HUGINN_UPDATE_GOLDEN', 's help says '), so the list cannot be scanned by instruction. Statbus closed 10 tasks with only 2 answers (8 closed in one batch at 03:31), and a plain done leaves no trace, so we cannot tell 'nothing to report' from 'never asked'. The same gap (check takes no evidence) was reported 3 times before anyone filed it.

## Principle
Agents propose, a human approves instruction changes (tb0dra principle). No form, no forced answer: done still succeeds without one. Grouping is by quoted text and named command, not by a model. Nudges go to the coordinator at a break (x325pp), never to the agent mid-task.

## Relevant details
Parser: huginn/internal/task/reflect.go var quoted, the third alternative '([^']{6,})' matches from a possessive apostrophe. Garbled ids: fkpyg3 scrxak n2hq8p x1ra0t cvbpdy t3fqaw ce3dmq wdkqzx ygay9c f5vx4x. Counts and quotes: /Users/jhf/.jcode/scratch/s6ff5k-feedback/REPORT.md clusters B and C2. Feeds tb0dra check 5.

## Depends on

## Acceptance
- [ ] When an open reflection is older than 3 days the coordinator's board cue names the count and hug task reflections --by-instruction (test)

## Notes
