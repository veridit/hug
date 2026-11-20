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
seid-lang's tasks get their epic links back, and the seid agent works the lint list one kind at a time.

## Purpose
Epics and a clean lint make hug task next and the board useful for the seid agents.

## Principle
The seid agent runs the commands in seid-lang; hug supplies a tested instruction and the fix lines.

## Relevant details
Commands for the seid agent (hug >= 0.7.7): 1) hug task import backlog .backlog (restores parents; on a copy 282 of 294 came back) 2) hug task lint --code parent-too-deep (20, each with its fix) 3) hug task lint --code long-title (827 open titles > 80 chars) 4) hug task import files .backlog/evidence --cite .backlog/evidence (task files). Copy for testing: ~/.jcode/scratch/seidcopy/repo.

## Depends on

## Acceptance
- [ ] import backlog rerun in seid-lang; hug task ls --jsonl shows about 280 tasks with a parent
- [ ] lint --code parent-too-deep is empty
- [ ] evidence files imported as task files
- [ ] Jørgen gives seid-instruction.md (Files) to the seid-lang agent and ticks this when it has reported back

## Requests
- 2026-11-20T13:08 request 1 by jhf:
  Can you give me a file with an instruction to the other agent in this ticket so I can copy it and give it to that one?
- 2026-11-20T13:11 request 2 by jhf:
  And when you do, you can ask me a question to open up and give it to the other agent and I can check it off when it is done, so it happens to come into my queue

## Notes


file:seid-instruction.md (2100 bytes, sha256:f0f90052) instruction to copy to the seid-lang agent; each step was tried on a copy of seid-lang on 2026-11-20 (282 parents restored, 13 parent-too-deep)
## History
- 2026-11-20T13:08 request 1 by jhf
- 2026-11-20T13:08 request 1 delivered to crab (session_crab_1793783281662_44ffaee927a8a9d5)
- 2026-11-20T13:08 request 1 acknowledged by crab
- 2026-11-20T13:11 request 2 by jhf
- 2026-11-20T13:11 request 2 delivered to crab (session_crab_1793783281662_44ffaee927a8a9d5)
- 2026-11-20T13:12 request 2 acknowledged by crab
- 2026-11-20T13:12 request 1 in progress by crab: hug task attach a1dx7s /Users/jhf/.jcode/scratch/a1dx7s/seid-instruction.md
- 2026-11-20T13:12 request 2 in progress by crab: hug task attach a1dx7s /Users/jhf/.jcode/scratch/a1dx7s/seid-instruction.md
