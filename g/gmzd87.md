---
id: gmzd87
title: "Requests reach the board coordinator by design"
state: todo
type: feature
priority: ""
assignee: []
labels: []
group: ""
rank: "zzzzzzzzzzzz"
created: 2026-11-18
updated: 2026-11-18
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
