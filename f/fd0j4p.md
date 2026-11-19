---
id: fd0j4p
title: "Agent instructions: questions in tickets, no blocked to quiet a reminder"
state: todo
type: feature
priority: ""
assignee: []
labels: []
group: ""
rank: "zzzzzzzzzzzz"
created: 2026-11-19
updated: 2026-11-19
stage: "ready"
---

## Goal
The hug section in AGENTS.md and the help that agents read say: (1) the hug board is the source of truth, harness todo lists do not copy it; (2) a reminder to finish is never a reason to mark something blocked, and blocked names who must do what; (3) questions and discussion for a person go in a task (one ask each, or an anchored thread once p87ngv ships), never only in chat.

## Purpose
Jørgen 2026-11-19 08:39Z (crab session): consider whether the todo-reminder push should change hug's instructions for AIs. crab had answered jcode's 'incomplete todos' reminder by marking items externally blocked (memory 2sgrc1), and asked a design question in chat rather than on a task. 08:54Z: dialogue should move from the chat into the ticket so it stays relevant and survives compaction.

## Principle
Short, imperative lines in the existing hug section; no new section. Each rule names the command to use (hug task ask, hug task wait, hug task note). A test pins the wording, as q2adr8 check 5 does.

## Relevant details
Files: the AGENTS.md section that hug task setup writes (huginn/internal/cli setup template) and this repo's AGENTS.md; help for ask and wait. Related: q2adr8 (one question per ask), p87ngv (threads), memories 7jj34g/2sgrc1, jcode maintainer report sent 2026-11-19 08:30Z.

## Depends on

## Acceptance
- [ ] Jørgen approves the exact added lines (quoted in a note)
- [ ] hug task setup writes them, and a test fails if any of the three rules is missing
- [ ] help ask and help wait say a reminder is not a blocker and blocked names who must do what (test)

## Questions
- 2026-11-19T10:00 question 1 by crab to jhf:
  Do these three lines read right for the hug section of AGENTS.md?
  - The hug board is the record of work. Do not copy it into a harness todo list.
  - A reminder to finish is not a blocker. Mark a task waiting only with who must do what: hug task wait ID "jhf: picks the column order".
  - Put questions and discussion for a person on the task (hug task ask ID --to NAME, one question each), not only in chat.

## Notes


## History
- 2026-11-19T10:00 todo -> todo/design (ask)
- 2026-11-19T10:02 answered design, by jhf, for crab (todo/design -> todo/ready):
  > Do these three lines read right for the hug section of AGENTS.md?
  > - The hug board is the record of work. Do not copy it into a harness todo list.
  > - A reminder to finish is not a blocker. Mark a task waiting only with who must do what: hug task wait ID "jhf: picks the column order".
  > - Put questions and discussion for a person on the task (hug task ask ID --to NAME, one question each), not only in chat.
  No, this doesn't read right at all. This it tells you what not to do. It should rather be a general, the hugboard is the record of work. Your harness to-do list is to organize yourself in managing this work, delegating it, picking it up again, checking the updating the status of the ticket, closing it, and so forth. The dependencies between the tickets are in the tickets themselves. So you should not have more to-dos than what you can actually do where all the details are in the tasks and using the task system to figure out what depends on what, how to resolve it, to pick up speed again and everything. But use your to-dos to stay focused when you're delegating and having multiple agents working on things, coordinating things
