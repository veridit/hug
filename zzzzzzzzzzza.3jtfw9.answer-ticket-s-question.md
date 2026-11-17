---
id: 3jtfw9
title: "Answer a ticket's question in place; the asking agent is told"
status: "doing"
type: feature
priority: "high"
assignee: ["nautilus"]
labels: []
created: 2026-11-17
updated: 2026-11-17
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
---

## Goal
A question to a person lives in the task as a field of its stage (design_question, waiting_on with a question, review_by with what to look at). In hug tui the detail shows it at the top with its context. Pressing a opens a one-line or multi-line input right under the question (not $EDITOR), and Enter writes the answer into the task beside the question: the answer field of that node, the answering person's name and the time. The stage then re-derives (design -> ready, waiting -> active, review -> done or back to active). The agent that asked sees it at its next turn: the recall hook's board cue says 'jhf answered your question on ID: <first line>', hug task next and show put it first, and MCP task_show returns it. CLI: hug task answer ID "text".

## Purpose
Jørgen 2026-11-17 11:45: AI drives most work and sometimes needs him. He wants to find the ticket, see the question, and write the answer straight where the agent needs it, without opening an editor on the whole file, and the agent should notice that the answer landed.

## Principle
The question and the answer sit together in the task, never in a chat. The answer is typed where the question is. Answering is one key, and the stage follows from the fields (6ephqk rule 2). The agent is told by what it already reads (recall cue, next, show, MCP); no new daemon. An answer is never lost: History keeps question and answer after the stage moves on.

## Relevant details
Builds on 6ephqk (stages, node fields, ? marker, '? N for you' header) and needs-you mock tmp/lanes/needs-you.md. Board cue: muninn/internal/recall/boardcue.go (hatchling's area). TUI input: reuse the note prompt. The agent that asked = the one that set the field (record asked_by). Persistent wake-up of an idle agent is 5dxzxe (our own harness spike); until then the coordinator watches the board.

## Depends on

## Acceptance
- [ ] in the detail of a task whose stage waits on you, a opens an input under the question; Enter stores the answer next to the question with your name and time, and the stage re-derives (golden + test)
- [ ] hug task answer ID "text" does the same from the CLI, and MCP task_update action answer (test)
- [ ] the agent that asked sees 'NAME answered your question on ID: ...' in its recall board cue on its next prompt, and first in hug task next/show (test)
- [ ] after the stage moves on, History holds the question, the answer, who and when (test)
- [ ] Jørgen answers a real question from a worker in the TUI and the worker picks it up without a DM

## Notes

nautilus 13:27 (crab 13:26): dependency on 6ephqk removed. 3jtfw9 needs only 6ephqk checks 3 (moves, design_asked_by/waiting_asked_by) and 8 (for you, design_for, waiting_on NAME), both on main at d74af5d..d198e01. Design ruling (crab 13:26): the answer is a History line 'answered design: <q> -> <a>, by NAME, DATE TIME, for ASKER'; show and next --mine put the newest answer first until the asker writes anything on the task after it (that write counts as seen); the board cue reads the same. No new front-matter fields, nothing new for lint. answer re-derives the stage; d stays separate.
