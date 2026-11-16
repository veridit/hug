---
id: a1dx7s
title: "seid-lang: restore epic links and work the lint list"
status: todo
type: chore
priority: ""
assignee: []
labels: []
created: 2026-11-16
updated: 2026-11-16
---

## Goal
seid-lang's tasks have their epic links back and its lint list shrinks kind by kind

## Purpose
The 2026-11-15 import lost 294 parent links and left 2,400+ lint findings; epics and a clean lint make next and the board useful for the seid agents

## Principle
The seid agent runs the commands itself in seid-lang; hug only supplies the commands and codes. One lint code at a time, highest value first; never bulk-rewrite titles without reading them

## Relevant details
Commands for the seid agent (hug >= 0.7.7): 1) hug task import backlog .backlog (restores parents; on a copy 282 of 294 came back) 2) hug task lint --code parent-too-deep (20, each with its fix) 3) hug task lint --code long-title (827 open titles > 80 chars) 4) hug task import files .backlog/evidence --cite .backlog/evidence (task files). Copy for testing: ~/.jcode/scratch/seidcopy/repo.

## Depends on

## Acceptance
- [ ] import backlog rerun in seid-lang; hug task ls --jsonl shows about 280 tasks with a parent
- [ ] lint --code parent-too-deep is empty
- [ ] evidence files imported as task files

## Notes
