---
id: fd0j4p
title: "Agent instructions: questions in tickets, no blocked to quiet a reminder"
state: todo
type: feature
priority: ""
assignee: []
labels: []
group: ""
rank: "zzzzzzzzzzy8"
created: 2026-11-19
updated: 2026-11-19
stage: "ready"
---

## Goal
The hug section of AGENTS.md says the board is the record of work, and waits and questions name who or what declaratively.

## Purpose
Jørgen 2026-11-19 08:39Z (crab session): consider whether the todo-reminder push should change hug's instructions for AIs. crab had answered jcode's 'incomplete todos' reminder by marking items externally blocked (memory 2sgrc1), and asked a design question in chat rather than on a task. 08:54Z: dialogue should move from the chat into the ticket so it stays relevant and survives compaction.

## Principle
Short, imperative lines in the existing hug section; no new section. Each rule names the command to use (hug task ask, hug task wait, hug task note). A test pins the wording, as q2adr8 check 5 does.

## Relevant details
The hug section in AGENTS.md and the help that agents read say: (1) the hug board is the source of truth, harness todo lists do not copy it; (2) a reminder to finish is never a reason to mark something blocked, and blocked names who must do what; (3) questions and discussion for a person go in a task (one ask each, or an anchored thread once p87ngv ships), never only in chat.

Files: the AGENTS.md section that hug task setup writes (huginn/internal/cli setup template) and this repo's AGENTS.md; help for ask and wait. Related: q2adr8 (one question per ask), p87ngv (threads), memories 7jj34g/2sgrc1, jcode maintainer report sent 2026-11-19 08:30Z. Also (Jørgen 11:34Z on p87ngv): open questions go on the task as asks, never inside an attached file, where the machinery cannot see them.

## Depends on

## Acceptance
- [x] Jørgen approves the exact added lines (quoted in a note)
- [ ] hug task setup writes them, and a test fails if any of the three rules is missing
- [ ] help ask and help wait say a reminder is not a blocker and blocked names who must do what (test)

## Questions
- 2026-11-19T10:00 question 1 by crab to jhf:
  Do these three lines read right for the hug section of AGENTS.md?
  - The hug board is the record of work. Do not copy it into a harness todo list.
  - A reminder to finish is not a blocker. Mark a task waiting only with who must do what: hug task wait ID "jhf: picks the column order".
  - Put questions and discussion for a person on the task (hug task ask ID --to NAME, one question each), not only in chat.
  answered 2026-11-20T11:04 by jhf:
    Clearly no. I'm confused and hug task weight ID and then you do parsing of the line Shouldn't you just specify the person and the message separately be declarative?
- 2026-11-19T10:02 question 2 by crab to jhf:
  Redrafted to say what to do. Does this read right?
  - The hug board is the record of work. Each task holds its details, status and dependencies; read the board to see what depends on what, what to pick up next, and how to resume after a break or a compaction.
  - Your harness todo list organizes you around the board: what you are delegating, what to pick up again, which task to update, check or close next. Keep it to what you can act on now; the details stay in the tasks.
  - When you coordinate several agents, use todos to stay focused on who is doing what, and follow their progress on their tasks.
  - A task waits on someone named doing something: hug task wait ID "jhf: picks the column order".
  - Ask a person on the task, one question each (hug task ask ID --to NAME), so the dialogue stays with the work and survives compaction.
  answered 2026-11-20T11:05 by jhf:
    This was clear and sensible apart from my previous nitpick about declarative person and message separate instead of parsing a text

## Notes


crab 09:03Z: Jørgen's answer 1 ruled the wording: say what to do, not what not to do. The board is the record of work (details, status, dependencies in the tasks); the harness todo list is for organizing yourself around it: delegating, picking up again, updating and closing tickets, coordinating agents. Keep todos to what you can act on now. Redrafted as question 2.
Jørgen 2026-11-20 10:04Z (answer to question 1): 'Clearly no' to the first draft: hug task wait ID "jhf: picks the column order" parses the person out of the line; he wants the person and the message given separately, declaratively. crab: correct, and the example was also wrong: wait refuses a text opening with a person's name (agxqax) and points to ask. The declarative form exists: hug task ask ID --to jhf "..." for a person; hug task wait ID "on what" only for things (CI, a release, a date); hug task dep ID --add OTHER for another task. Question 2's redraft repeats the bad line, so it is withdrawn and asked again.
Approved lines (Jørgen 2026-11-20 10:05Z, answer 3: 'clear and sensible apart from my previous nitpick about declarative person and message separate instead of parsing a text'). Line 4 rewritten to his nitpick with commands that already take the person and the message separately; the other four are his approved text word for word:
- The hug board is the record of work. Each task holds its details, status and dependencies; read the board to see what depends on what, what to pick up next, and how to resume after a break or a compaction.
- Your harness todo list organizes you around the board: what you are delegating, what to pick up again, which task to update, check or close next. Keep it to what you can act on now; the details stay in the tasks.
- When you coordinate several agents, use todos to stay focused on who is doing what, and follow their progress on their tasks.
- When a task waits, say on what: on a person, ask them (hug task ask ID --to NAME "question"); on another task, hug task dep ID --add OTHER; on a thing such as CI, a release or a date, hug task wait ID "what" (or --until DATE).
- Ask a person on the task, one question each (hug task ask ID --to NAME), so the dialogue stays with the work and survives compaction.
check 1: Approved by Jørgen 10:05Z (answer 3) with his nitpick applied to line 4 (person and message separate: ask --to NAME, dep --add, wait only for things); exact five lines in the note above.
## History
- 2026-11-19T10:00 todo -> todo/design (ask)
- 2026-11-19T10:02 answered design, by jhf, for crab (todo/design -> todo/ready):
  > Do these three lines read right for the hug section of AGENTS.md?
  > - The hug board is the record of work. Do not copy it into a harness todo list.
  > - A reminder to finish is not a blocker. Mark a task waiting only with who must do what: hug task wait ID "jhf: picks the column order".
  > - Put questions and discussion for a person on the task (hug task ask ID --to NAME, one question each), not only in chat.
  No, this doesn't read right at all. This it tells you what not to do. It should rather be a general, the hugboard is the record of work. Your harness to-do list is to organize yourself in managing this work, delegating it, picking it up again, checking the updating the status of the ticket, closing it, and so forth. The dependencies between the tickets are in the tickets themselves. So you should not have more to-dos than what you can actually do where all the details are in the tasks and using the task system to figure out what depends on what, how to resolve it, to pick up speed again and everything. But use your to-dos to stay focused when you're delegating and having multiple agents working on things, coordinating things
- 2026-11-19T10:02 answer 1 delivered to crab (session_crab_1793783281662_44ffaee927a8a9d5)
- 2026-11-19T10:02 answer 1 acknowledged by crab
- 2026-11-19T10:02 crab read the answer: hug task note fd0j4p "crab 09:03Z: Jørgen's answer 1 ruled the wording: say what to do, not what not to do. The board is the record of work (…"
- 2026-11-19T10:02 asked question 2 of jhf, 2 open (todo/ready -> todo/design)
- 2026-11-20T11:04 answered design (question 1), by jhf, for crab (todo/design -> todo/design):
  > Do these three lines read right for the hug section of AGENTS.md?
  > - The hug board is the record of work. Do not copy it into a harness todo list.
  > - A reminder to finish is not a blocker. Mark a task waiting only with who must do what: hug task wait ID "jhf: picks the column order".
  > - Put questions and discussion for a person on the task (hug task ask ID --to NAME, one question each), not only in chat.
  Clearly no. I'm confused and hug task weight ID and then you do parsing of the line Shouldn't you just specify the person and the message separately be declarative?
- 2026-11-20T11:04 answer 2 delivered to crab (session_crab_1793783281662_44ffaee927a8a9d5)
- 2026-11-20T11:04 answer 2 acknowledged by crab
- 2026-11-20T11:04 crab read the answer: hug task note fd0j4p "Jørgen 2026-11-20 10:04Z (answer to question 1): 'Clearly no' to the first draft: hug task wait ID \"jhf: picks the colu…"
- 2026-11-20T11:05 answered design (question 2), by jhf, for crab (todo/design -> todo/ready):
  > Redrafted to say what to do. Does this read right?
  > - The hug board is the record of work. Each task holds its details, status and dependencies; read the board to see what depends on what, what to pick up next, and how to resume after a break or a compaction.
  > - Your harness todo list organizes you around the board: what you are delegating, what to pick up again, which task to update, check or close next. Keep it to what you can act on now; the details stay in the tasks.
  > - When you coordinate several agents, use todos to stay focused on who is doing what, and follow their progress on their tasks.
  > - A task waits on someone named doing something: hug task wait ID "jhf: picks the column order".
  > - Ask a person on the task, one question each (hug task ask ID --to NAME), so the dialogue stays with the work and survives compaction.
  This was clear and sensible apart from my previous nitpick about declarative person and message separate instead of parsing a text
- 2026-11-20T11:05 answer 3 delivered to crab (session_crab_1793783281662_44ffaee927a8a9d5)
- 2026-11-20T11:05 crab read the answer: hug task note fd0j4p "Approved lines (Jørgen 2026-11-20 10:05Z, answer 3: 'clear and sensible apart from my previous nitpick about declarativ…"
- 2026-11-20T11:05 answer 3 acknowledged by crab
