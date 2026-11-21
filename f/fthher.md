---
id: fthher
title: "hug task ls cannot list the questions waiting for someone"
state: todo
type: feature
priority: ""
assignee: []
labels: []
group: ""
rank: "zzzzzzzzzzzz"
created: 2026-11-21
updated: 2026-11-21
---

## Goal
One shell command lists every open question asked of a person, with task, check and question.

## Purpose
crab 2026-11-21 19:03Z told Jørgen 'hug task ls --asked lists the ones waiting for you'; the flag does not exist (unknown flag --asked). Only the TUI's ASKED section and '? N for you' show them. An agent guessed the flag, so a human will too.

## Principle
Reuse ls: a filter, not a new command. Its error for --asked must point to it.

## Relevant details
Candidate: hug task ls --asked [--to NAME] (default: the caller), one line per question: id, check, first line of the question, asked since.

## Depends on

## Acceptance
- [ ] hug task ls --asked lists the open questions asked of the caller, one per line with id and check (test)
- [ ] Jørgen runs it and gets his list

## Notes
