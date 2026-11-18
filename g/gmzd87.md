---
id: gmzd87
title: "Requests reach the board coordinator by design"
state: "doing"
type: feature
priority: ""
assignee: ["mizaru"]
labels: []
group: ""
rank: "zzzzzzzzzzxh"
created: 2026-11-18
updated: 2026-11-18
doing_since: "2026-11-18T12:34"
stage: "active"
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
---

## Goal
A request a person writes on any task reaches the board coordinator (hug.toml [board] coordinator) through hug itself, even when that session is idle, and its receipt is visible: pending, acknowledged, in progress, answered.

## Purpose
Jørgen's requests sat unseen until the coordinator happened to run a hug command; the coordinator then improvised a poller. Delivery must be designed in, not improvised by an agent.

## Principle
By design, not by doctrine: hug delivers, and the instructions hug writes (setup's AGENTS.md block, MCP tool descriptions, hook messages) describe exactly what the code does, kept in step by a test. Saved is not received: acknowledged only on the coordinator's actual receipt. No per-agent pollers.

## Relevant details
Split from ec6z6j (multi-task requests, which still waits on row marking rxr4z8); this covers delivery of single requests and the shared inbox it builds on. Today: requests surface only as a board cue on the next hug command by any agent (muninn/internal/recall/boardcue.go). Coordinator wake path to investigate: jcode hooks, harness install code (muninn/internal/harness), swarm/DM delivery.

## Depends on

## Acceptance
- [ ] A request written while the coordinator session is idle reaches it without any agent-side poller, shown by an end-to-end test or recorded live run (session ids, timestamps).
- [ ] Each request shows its receipt state (pending, acknowledged, in progress, answered) in hug task requests and the TUI, and acknowledged is set only by the coordinator's actual receipt (test).
- [ ] The instructions hug installs (AGENTS.md block from hug setup, MCP tool descriptions, hook text) describe this delivery flow, and a test fails when the instruction text and the behaviour disagree.
- [ ] Jørgen writes a request in hug tui and the coordinator reports receiving it, unprompted.

## Notes


-
mizaru 11:58 design (before code; implementing unless crab objects by 12:18Z).
1. WAKE PATH (investigated in ~/Projects/jcode 28ac5e603). jcode hooks are all per-session observers/feedback: none can start a turn in another, idle session. The swarm DM path needs a caller session. What can: the daemon socket request {"type":"notify_session","session_id","message"} (crates/jcode-protocol/src/wire.rs:269), jcode's own schedule-tool delivery. handle_notify_session (jcode-app-core/src/server/client_actions.rs:51): idle live session -> run_live_system_turn_if_idle starts a turn now; busy -> queued soft interrupt injected at the next safe point; not live -> error "Session 'X' is not currently live". No desktop input, no debug socket (that is disabled here). Socket: <JCODE_HOME>/runtime/jcode.sock (the zshrc wrapper sets JCODE_RUNTIME_DIR=$root/.jcode/runtime; global ~/.jcode/runtime/jcode.sock, matches JCODE_SOCKET here).
   So: when hug writes a request (hug task request, MCP task_update request, TUI r box), right after the commit hug itself sends notify_session to the coordinator's newest jcode session (session_<coordinator>_*.json, found in every jcode home harness.DiscoverJcode lists). Push on write, no poller. Target: hug.toml [board] coordinator (ec6z6j: the coordinator inbox, not an arbitrary worker); the assignee keeps today's recall cue. No coordinator set: no push, setup says so.
   Message: 'hug request ID#N from WHO, to you as board coordinator: "TEXT" ... then: hug task resolve ID N "what changed"'.
2. RECEIPT STATE. Requests block format unchanged (old hugs on other clones keep parsing it); state is read from History lines, like Woken:
   pending      = 'request N by WHO' (written by request, today)
   (delivery)   = 'request N delivered to crab (session_crab_...)' or 'request N not delivered: crab has no live jcode session' (written by the hug process that wrote the request, from the daemon's reply). Delivered is not acknowledged.
   acknowledged = 'request N acknowledged by crab' written by muninn's jcode hook IN crab's session (turn_start for an idle wake, post_tool_feedback/turn_end for a queued soft interrupt) when the session's own transcript holds the 'hug request ID#N' message. The ack waits in a small inbox file per session (one stat on the hot path, like park). That is actual receipt: the text is in the coordinator's context.
   in progress  = 'request N in progress by crab: <write>' written by Store on the acknowledger's first write on that task after the ack (like markRead).
   answered     = 'resolved request N by X' (resolve, today).
   Shown in hug task requests (state per row), show --json requests[].state, TUI REQUESTS label 'open' -> the state (one label, no highlighting).
3. INSTRUCTION TEXTS that change: setup's AGENTS.md block 'Requests:' paragraph; MCP task_requests and task_update request descriptions; setup coordinator step detail; the delivered message itself. One Go constant describes the flow and is used by all of them; a test drives a fake daemon end to end and fails if the text names a state, command or route the code does not produce. boardcue.go (duckling) untouched; toolcalls.go untouched.
Open: crab session must be live in a daemon; an offline coordinator gets 'not delivered' plus the existing next/cue paths, and no redelivery loop (would be a poller).
## History
- 2026-11-18T12:34 todo -> doing/active (take)
