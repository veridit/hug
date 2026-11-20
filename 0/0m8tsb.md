---
id: 0m8tsb
title: "session_end capture fails before extraction on jcode sessions"
state: todo
type: feature
priority: ""
assignee: []
labels: []
group: ""
rank: "zzzzzzzzzzzv"
created: 2026-11-20
updated: 2026-11-20
---

## Goal
A jcode session_end capture runs extraction, or records a cause that names what failed.

## Purpose
crab 2026-11-20 05:45Z (read-only, live store): capture_runs session_end 'capture failed before extraction (details withheld)', error_class internal, harness jcode: 95 since 11-15 (11-15 23, 11-16 38, 11-18 14, 11-19 19), vs 21 OK on 11-19. Session ends are where the most context sits, so lost captures lose the most.

## Principle
Reproduce from the record first; the error must say what failed (fix: line), not withhold it; measure the rate before and after on the live store, read-only.

## Relevant details
Example: 2026-11-19T16:53:02Z, session_badger_1795107151750_18ab71c4418d696b, home /Users/jhf/Projects/Brovs/.jcode, model deepseek-flash, all timings 0 (fails before extract). Query: sqlite3 'file:$HOME/Library/Application Support/muninn/muninn.db?mode=ro' with json_extract(record,'$.error') like 'capture failed before%'. Code: muninn/internal/harness, muninn/run capture path.

## Depends on

## Acceptance
- [ ] The cause of the 'capture failed before extraction' runs is named in a note with the record and code line
- [ ] A day after the fix ships, session_end failures before extraction are 0 on the live store (count in a note)

## Notes
