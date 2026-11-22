---
id: btk63z
title: "Design: local messages between agents across repos and harnesses"
state: todo
type: spike
priority: "low"
assignee: []
labels: []
group: ""
rank: "zzzzzzzzzzyl"
created: 2026-11-20
updated: 2026-11-20
---

## Goal
Decide whether hug gets local channels where any agent, in any repo or harness, can post to another agent or a channel and be answered.

## Purpose
Coordinating seid-lang and hug today goes through Jørgen copying files between consoles.

## Principle
Local first: one machine-wide store, never pushed; a remote push only for a use case we have, and none exists yet.

## Relevant details
Jørgen 2026-11-20 16:02Z (crab console): 'sometimes we need to coordinate between different agents and different repositories ... messages that could allow global communication, meaning that you could find and insert a message to that agent and he can find you back ... across agents, across LLMs ... not swarm, but channel communication, chat ... local only and a local branch that wasn't pushed remotely ... for remote pushing ... I don't see a use case for that, at least not yet.'
Today's case: crab (hug) and orangutan (seid-lang) coordinated on a1dx7s and mmrd22 only through Jørgen copying seid-instruction.md and the agents' answers between consoles; seid's agent also distrusted a hug board line (4gzfyc).
What exists: hug already hands requests and answers to a board's coordinator session through jcode's daemon (gmzd87, notify_session), and docs/DESIGN.md plans hug/messages (one file per message, never edited) but nothing is built. 5dxzxe asks the larger question (own harness, persistent workers).
Questions to settle: where the store lives (a machine-wide folder beside the memory data, not a git branch, since it is never pushed); addressing (agent name, board coordinator, channel name like #seid-hug); delivery (reuse notify_session for jcode, a hook line for others, read on demand otherwise); how an agent finds who exists (live sessions with their boards); trust (a message names its sender and board, so 4gzfyc's distrust does not repeat); what never goes in (secrets, client data across client boundaries).

## Depends on

## Acceptance
- [ ] A design note with the options for store, addressing, delivery and trust, and what it costs, attached here
- [ ] Jørgen decides: build it, fold it into 5dxzxe, or not now

## Notes
