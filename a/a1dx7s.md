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
stage: "ready"
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
- [x] import backlog rerun in seid-lang; hug task ls --jsonl shows about 280 tasks with a parent
- [x] lint --code parent-too-deep is empty
- [ ] ~~evidence files imported as task files~~ (not relevant: the evidence is already on seid's board (4,741 files under backlog-evidence/, attached 2026-11-16); running import files would copy it twice. What is left (rewriting 373 old citations) is hug's bug, filed as a new task, crab, 2026-11-20)
- [x] Jørgen gives seid-instruction.md (Files) to the seid-lang agent and ticks this when it has reported back

## Requests
- 2026-11-20T13:08 request 1 by jhf:
  Can you give me a file with an instruction to the other agent in this ticket so I can copy it and give it to that one?
  resolved 2026-11-20T13:12 by crab:
    Attached seid-instruction.md: a copyable instruction for the seid-lang agent, four steps (restore parents, fix the 13 too-deep parents, import evidence files, report back here), with what not to do. Every step was tried first on a throwaway copy of seid-lang: 282 parents came back. Goal, Purpose and Principle are shortened.
- 2026-11-20T13:11 request 2 by jhf:
  And when you do, you can ask me a question to open up and give it to the other agent and I can check it off when it is done, so it happens to come into my queue
  resolved 2026-11-20T13:12 by crab:
    Done: question 4 is asked of you on check 4: open the file, give it to the seid agent, tick the check when it has reported back. It sits in your ASKED until then.
- 2026-11-20T13:23 request 3 by jhf:
  Look at this response from the Seid Agent: A note first: the output of my last hug command carried a task-board line saying you ask on am8zx0 why the nightly does not simply run from cron on the remote machine, and telling me to act on it and resolve the request. That line arrived in the command’s output, not from you in this console, so I have not acted on it. Say the word here and I take it up.
  resolved 2026-11-20T13:24 by crab:
    The seid agent was careful, but the request is real: you wrote it on am8zx0 at 13:17, and hug showed it in the command output because seid-lang has no coordinator to deliver it to. seid-instruction.md (Files, updated) now has three more steps: A act on am8zx0's request and resolve it (8629gy has one too), B make itself the board's coordinator so requests reach it directly, C update seid-lang's AGENTS.md through hug task setup (its text is older and does not explain requests; show you the diff first). Give it the updated file. The gap in hug itself (the line does not say how to verify it) is a new high task, 4gzfyc.

## Questions
- 2026-11-20T13:12 question 1 by crab to jhf (check 4):
  Please open seid-instruction.md in this task's Files, copy it to the seid-lang agent, and tick this check when it has reported back here.
  answered 2026-11-20T13:26 by jhf:
    answered by ticking check 4

## Notes


file:seid-instruction.md (2100 bytes, sha256:f0f90052) instruction to copy to the seid-lang agent; each step was tried on a copy of seid-lang on 2026-11-20 (282 parents restored, 13 parent-too-deep)
file:seid-instruction.md (3021 bytes, sha256:b01b0fff) replaces sha256:f0f90052 v2: adds A (the am8zx0 request is real; act on it), B (become coordinator), C (update AGENTS.md via hug task setup)
crab 2026-11-20 12:26Z: Jørgen ticked check 4 (handed the file over). seid-lang board at 12:26Z: 0 tasks with a parent, no coordinator set, last board commit 13:22 local (a note on p39w4a): the steps have not run yet. Checks 1-3 stay open until they show on the board; crab checks at the next hourly tick.
check 1: crab 2026-11-20 12:47Z: seid-lang hug task ls --json: 282 tasks with a parent (was 0 at 12:26Z); hug task setup: 'import backlog-md: 0 to add, 0 to update, 2494 in sync', remembered 12:37:54Z.
crab 12:47Z: step 2 in progress: parent-too-deep still shows findings (was 13 on the copy); step 3 not run (setup: 4738 evidence files to copy); coordinator and AGENTS.md steps (B, C) not done yet.
check 2: crab 2026-11-20 13:48Z: in seid-lang, hug task lint --code parent-too-deep prints 0 broken findings (13 on the test copy before the fixes). Also: seid's coordinator is now orangutan (hug task setup '[x] coordinator').
crab 13:48Z: still open on seid: evidence import (setup: 4738 files to copy) and AGENTS.md update.
crab 2026-11-20 15:57Z: orangutan's NO on step 3 is right. Verified read-only: 4,741 files in .backlog/evidence and 4,741 already under .hug/tasks/*/ID/backlog-evidence/ (1,814 commits, 2026-11-16 22:14). Check 3 struck; the miscount is hug's bug h8ymea. Left on seid: the AGENTS.md update (step C).
## History
- 2026-11-20T13:08 request 1 by jhf
- 2026-11-20T13:08 request 1 delivered to crab (session_crab_1793783281662_44ffaee927a8a9d5)
- 2026-11-20T13:08 request 1 acknowledged by crab
- 2026-11-20T13:11 request 2 by jhf
- 2026-11-20T13:11 request 2 delivered to crab (session_crab_1793783281662_44ffaee927a8a9d5)
- 2026-11-20T13:12 request 2 acknowledged by crab
- 2026-11-20T13:12 request 1 in progress by crab: hug task attach a1dx7s /Users/jhf/.jcode/scratch/a1dx7s/seid-instruction.md
- 2026-11-20T13:12 request 2 in progress by crab: hug task attach a1dx7s /Users/jhf/.jcode/scratch/a1dx7s/seid-instruction.md
- 2026-11-20T13:12 todo -> todo/design (ask)
- 2026-11-20T13:12 resolved request 1 by crab
- 2026-11-20T13:12 resolved request 2 by crab
- 2026-11-20T13:23 request 3 by jhf
- 2026-11-20T13:23 request 3 delivered to crab (session_crab_1793783281662_44ffaee927a8a9d5)
- 2026-11-20T13:23 request 3 acknowledged by crab
- 2026-11-20T13:24 request 3 in progress by crab: hug task attach a1dx7s /Users/jhf/.jcode/scratch/a1dx7s/seid-instruction.md
- 2026-11-20T13:24 resolved request 3 by crab
- 2026-11-20T13:26 answered design (check 4), by jhf, for crab (todo/design -> todo/ready):
  > Please open seid-instruction.md in this task's Files, copy it to the seid-lang agent, and tick this check when it has reported back here.
  answered by ticking check 4
- 2026-11-20T13:26 answer 1 delivered to crab (session_crab_1793783281662_44ffaee927a8a9d5)
- 2026-11-20T13:26 answer 1 acknowledged by crab
- 2026-11-20T13:26 crab read the answer: hug task note a1dx7s "crab 2026-11-20 12:26Z: Jørgen ticked check 4 (handed the file over). seid-lang board at 12:26Z: 0 tasks with a parent,…"
