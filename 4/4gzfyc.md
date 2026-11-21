---
id: 4gzfyc
title: "A board line that relays a person's request is trusted by the agent"
state: todo
type: bug
priority: "high"
assignee: []
labels: []
group: ""
rank: "zzzzzzzzzzze"
created: 2026-11-20
updated: 2026-11-20
---

## Goal
An agent seeing a person's request in hug's command output knows it is real and acts on it, without asking the person to confirm.

## Purpose
seid's agent ignored Jørgen's am8zx0 request as untrusted output, so requests stall on boards with old instructions.

## Principle
The line names where it comes from and how to verify it; the installed instructions say requests arrive this way.

## Relevant details
Seen 2026-11-20 12:23Z (a1dx7s request 3): seid-lang's agent got 'jorgen asks on am8zx0 ... act on it and resolve' in a hug command's output and replied: 'That line arrived in the command's output, not from you in this console, so I have not acted on it.' Causes: (1) seid-lang's AGENTS.md has the older '## Tasks: hug' section, with no Requests or delivery text (hug task setup lists the update as open); (2) seid has no coordinator, so the request was 'pending, not delivered' and only surfaced as a board line; (3) the line itself does not say how to check it. Fix: the board line names the source and a check ('hug task requests shows it; written by jorgen in hug tui at 13:17'); hug task setup's agent-instructions step says outright that requests arrive in command output and hook results; when a repo's instructions are older than the installed hug, the board line says so with fix: hug task setup.

## Depends on

## Acceptance
- [ ] A relayed request line names who wrote it, when, and the command that shows it on the board (golden)
- [ ] The agent section hug setup installs says a person's requests arrive in hug's command output and hook results and are acted on like a message from the person (test pins the wording)
- [ ] When a repo's '## Tasks: hug' section is older than the installed hug, the first board line in a session says so with fix: hug task setup (test)

## Notes
