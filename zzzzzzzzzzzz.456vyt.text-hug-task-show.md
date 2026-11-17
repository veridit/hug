---
id: 456vyt
title: "Text hug task show prints the stage and wait as its second line"
status: todo
type: feature
priority: ""
assignee: []
labels: []
created: 2026-11-17
updated: 2026-11-17
---

## Goal
hug task show ID (text) prints, as its second line, the stage the same way the ls row does, e.g. 'stage: waiting on jhf: approve ... since 2026-11-17 17:47'. Today it prints neither stage nor waiting_on/waiting_since, so a recorded wait is invisible in show.

## Purpose
kestrel 2026-11-17 16:47: after hug task wait ggydj2 --on jhf, show printed no trace of the wait, and I reran wait thinking it had failed, overwriting the text. crab 16:47: file it, do it right after 95ahtk.

## Principle
Show shows what the file holds; the line matches the row's wording, so there are no two phrasings. --json is unchanged.

## Relevant details
Seen: hug v0.7.42, hug task show ggydj2 | head -2 gives the title line and 'taken by: ...', while the file has stage: waiting, waiting_on: jhf: ..., waiting_since: 2026-11-17T17:47.

## Depends on

## Acceptance
- [ ] text show of a waiting task prints 'stage: waiting on NAME: what since DATE' as line 2 (test)
- [ ] a task with no wait and stage active prints no wait line, or the stage alone, matching the row (test)

## Notes
