---
id: p87ngv
title: "Threads anchored to quoted text in a task: comments, questions, requests"
state: todo
type: feature
priority: ""
assignee: []
labels: []
group: ""
rank: "zzzzzzzzzzzy"
created: 2026-11-19
updated: 2026-11-19
stage: "ready"
---

## Goal
A person or an agent can attach a comment, question or request to an exact piece of a task's text (a sentence in the goal, check 3, a note). Each becomes a thread in the task file, answered and replied to in place from hug tui (a mark mode that selects text), the CLI and MCP. Agents can send several questions at once, each its own thread. The dialogue lives in the ticket, survives compaction and is specific to the text it is about.

## Purpose
Jørgen 2026-11-19 08:55Z (crab session): questions today look like acceptance criteria, attached to the task or a check. He wants to switch to a mark mode, select a piece of text and attach a comment or request to it, so the agent knows exactly what it is about; and to move multi-question dialogue out of chat into the ticket, where it stays relevant and survives compaction (08:54Z). Builds on q2adr8 (several questions, v0.7.70), hbgybn/ec6z6j (requests), gmzd87 (delivery to the coordinator), 3jtfw9 (answer in place).

## Principle
One primitive, the thread: an anchor, a kind (comment, question, request), entries with who and when, and open or resolved. Anchors quote text (field plus exact quote plus a little context), never line numbers, so edits do not break them; a quote that no longer matches shows the thread as outdated instead of losing it. Everything in the task file, verbatim. Calm TUI rules (qh9ep1 check 9): one cue per meaning, no background fills. Design first, mock before build; Jørgen rules on the open questions in this task.

## Relevant details
Sketch (crab 09:00Z). File: a ## Threads section, each thread headed with its number, kind, field and quote, entries below. CLI: hug task comment|ask|request ID --on FIELD --quote TEXT 'text'; ask --file reads several questions (one per block), each its own thread. MCP: task_update action thread. TUI: in the detail view, a mark key selects a span; c comments, r requests; a marker in the margin shows where threads sit; a key jumps between them; answering opens the input under the thread (3jtfw9). Delivery: as gmzd87, to the assignee or the coordinator. Open design questions are asked on this task one by one.

## Depends on

## Acceptance
- [ ] Jørgen approves a mock (CLI transcript plus TUI frames at 100x30) of a thread on a quoted sentence, a reply and a resolve
- [ ] An outdated anchor (quoted text edited away) keeps its thread and shows it as outdated (test)
- [ ] Several questions in one call become separate threads, each answered on its own (test)
- [ ] Jørgen marks a sentence in hug tui and the assignee answers in that thread without a DM

## Questions
- 2026-11-19T09:56 question 1 by crab to jhf:
  Should questions and requests both become threads, so the ## Questions and ## Requests sections are replaced (old ones migrated), or do threads sit beside them?
- 2026-11-19T09:56 question 2 by crab to jhf:
  Who may resolve a thread: only the person who opened it, or also the agent once it has acted on it?
- 2026-11-19T09:56 question 3 by crab to jhf:
  Besides task text, should threads anchor to an attached file such as a mock sheet (a region or a line in it)?

## Notes


## History
- 2026-11-19T09:56 todo -> todo/design (ask)
- 2026-11-19T09:56 asked question 2 of jhf, 2 open (todo/design -> todo/design)
- 2026-11-19T09:56 asked question 3 of jhf, 3 open (todo/design -> todo/design)
- 2026-11-19T10:05 answered design, by jhf, for crab (todo/design -> todo/ready):
  > Should questions and requests both become threads, so the ## Questions and ## Requests sections are replaced (old ones migrated), or do threads sit beside them?
  I think it's a real risk that the conversation becomes the thing and I think it should be moved out to maybe a separate document that will link in the files section that is historic, almost like provenance or just chat history. Otherwise, the ticket will become bloated. The ticket has to stay crisp and clear. So I think the questions and requests might need replacement, but they might appear in the text so you can actually see them, just like text. We have to choose the sigils or how to recognize them and act on them. And then, as they are resolved, the ticket is changed, and any history or chat history is put into another document outside of the ticket. If we try to scram everything into the ticket, I think it will be like if you make threads inside the ticket, it will become bloated. And especially when things are complex, you will have longer and more intense chats, which will bloat them even more. When what you need is clarity. And clarity is often to find the succinctness in things and boil it down to its essence, not adding more text. So the text is the tool to which we reach short clarity very often. That's why the messy part has to go out into a different file. But I think the working document is the ticket itself. That's where the replies come, it's picked up, and then the AI gets an instruction to lead with clarity in the ticket, answer requests, do them, post any questions, and then put all of the prehistory provenance into a suitably named asset that can grow with a penned
