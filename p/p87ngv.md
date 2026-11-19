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
design_question: "Should questions and requests both become threads, so the ## Questions and ## Requests sections are replaced (old ones migrated), or do threads sit beside them?"
design_for: "jhf"
design_since: "2026-11-19T09:56"
design_asked_by: "crab"
stage: "design"
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

## Notes


## History
- 2026-11-19T09:56 todo -> todo/design (ask)
