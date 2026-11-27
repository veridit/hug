---
id: xja4ss
title: "Questions from the Seid agent's report on tasks and memory"
state: todo
type: feature
priority: ""
assignee: []
labels: []
group: ""
rank: "zzzzzzzzzzzz"
created: 2026-11-27
updated: 2026-11-27
---

## Goal
Jørgen decides which of three changes the Seid agent's report leads to, each from its context below.

## Purpose
Jørgen 2026-11-27: write the questions up in a ticket with context, not in chat.

## Principle
One question per check, each with what we have today and what would change. Nothing is built before the answer.

## Details
The report (Seid agent, 2026-11-27): its todos had become a second activity log; it cut them to the next decisions and kept evidence on tickets. Tickets with checks and logs let it verify '12 failures disappeared' instead of trusting 'passed'. Weak spots: handing a task back, and recall showing an old September staffing memory that conflicts with the current policy. Its rule: memory says where to look, the ticket what we decided, the run what happened. Already settled, not asked: retired memories are never shown (search.go:310 active=1) and will be deleted after 30 days (1htae7). Q1 recall: when a newer memory replaces an older one, the older one should be retired at once, and recall rows could say what a memory is (standing rule or a note from one run) and where it came from (ticket or session). Each memory already stores type (rule, preference, fact, record), scope and what it supersedes (store.go:33-37); recall rows show only date, id and text (recall.go:282-283). Any recall change needs a measured, blind-judged gain on replayed messages first (AGENTS.md). Q2 show: hug task show ends with next: after all the notes; the step hug names (v0.7.117) could come first, with owner and what it waits on. Q3 give back: parking is one command (hug task wait ID 'on what'); giving a task back to todo is two (hug task set ID --state todo, then a note). One command would be: hug task give ID 'where it stands'.

## Depends on

## Acceptance
- [ ] Q1: measure recall that retires a replaced memory at once and labels each row with kind and source (yes/no)
- [ ] Q2: hug task show starts with owner, next step and what it waits on (yes/no)
- [ ] Q3: hug task give ID 'where it stands' gives a task back in one command (yes/no)

## Notes
