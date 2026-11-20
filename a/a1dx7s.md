---
id: a1dx7s
title: "seid-lang: restore epic links and work the lint list"
state: todo
type: chore
priority: ""
assignee: []
labels: []
group: ""
rank: "yq"
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

## Requests
- 2026-11-20T13:08 request 1 by jhf:
  Can you give me a file with an instruction to the other agent in this ticket so I can copy it and give it to that one?
- 2026-11-20T13:11 request 2 by jhf:
  And when you do, you can ask me a question to open up and give it to the other agent and I can check it off when it is done, so it happens to come into my queue

## Notes


## History
- 2026-11-20T13:08 request 1 by jhf
- 2026-11-20T13:08 request 1 delivered to crab (session_crab_1793783281662_44ffaee927a8a9d5)
- 2026-11-20T13:08 request 1 acknowledged by crab
- 2026-11-20T13:11 request 2 by jhf
- 2026-11-20T13:11 request 2 delivered to crab (session_crab_1793783281662_44ffaee927a8a9d5)
