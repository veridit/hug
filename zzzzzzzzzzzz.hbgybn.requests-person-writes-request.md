---
id: hbgybn
title: "Requests: a person writes a request in a task, an agent turns it into edits"
state: todo
type: feature
priority: ""
assignee: []
labels: []
created: 2026-11-17
updated: 2026-11-17
---

## Goal
In a task's detail, R opens a multi-line box where the viewer writes a request to the agents in their own words (short, dictated, rough). hug stores it on the task as an open request: who, when, the text verbatim. The task's owner, or whoever works the board, is told at once (recall hook and the write-command footer), reads it, turns it into systematic changes to the ticket (goal, checks, details, notes, state), and closes the request with a one-line account of what changed. The person sees open requests on the row ('req 1') and the account under the request in the detail.

## Purpose
Jørgen 23:52 local: 'when I look at the tasks I sometimes would like to have you fix something in the task but if I have to describe it to you it's a lot of work... just go into the task and write like a comment... that becomes something you read and act on... I will write something shorter or use speech to text and then you will transform it into a systematic change in that ticket.' 23:52: 'it's not you asking me, it's me making an ask of you.' The reverse of 41hhgn's asked stage.

## Principle
The person writes, the agent does the systematic work. The request text is kept verbatim, never rewritten; the change it caused is shown next to it. A request is never lost or silently dropped: it stays open until an agent closes it with what it did, or asks back (41hhgn) when unclear. Nothing writes silently. Concurrent writes wait.

## Relevant details
Not the asked stage: that blocks the task on a person; a request is an open item for agents on any task in any state, and does not change its stage. Storage proposal (kestrel to confirm): a '## Requests' section, each request a block 'request by jhf 2026-11-17 23:52:' + indented text, closed by 'done by nautilus 23:58: set check 9's wording, added check 10'. CLI: hug task request ID - (stdin) | "text"; hug task requests (open ones, all tasks); hug task resolve ID N "what changed". MCP: task_update action request / resolve; task_list filter requests. Agent notice: boardcue gets kind 'request' (one line: 'Task board: jhf asks on 6ephqk: "..." Act on it, then: hug task resolve 6ephqk 1 "what changed"'), routed to the task's assignee, else to the coordinator. TUI: R in a detail (and on a list row); the overview shows a REQUESTS count in the header and the row marks 'req N'. Multi-line input shared with 41hhgn's answer box. Owner: kestrel (model, CLI, MCP, cue) then nautilus (TUI).

## Depends on

## Acceptance
- [ ] hug task request ID - stores a multi-line request verbatim with who and when, and hug task requests lists every open one across the board (test)
- [ ] hug task resolve ID N "what changed" closes request N with the account and who; resolving an unknown N refuses with a fix: line (test)
- [ ] The assignee's session gets one recall line naming the task, the request text and the resolve command, on its next turn; a task with no assignee routes to the coordinator (harness test)
- [ ] In the TUI, R in a detail opens a multi-line box (Enter sends, Alt-Enter adds a line); the row shows 'req 1' while open, and the detail shows each request with its account once resolved (golden)
- [ ] MCP task_update request and resolve behave like the CLI (test)
- [ ] Jørgen writes a real request on hug's board after release and an agent resolves it into ticket changes he agrees with

## Notes
