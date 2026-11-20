---
id: 3cqvx5
title: "Board coordinator by default: the live agent session, no setup"
state: "doing"
type: feature
priority: "medium"
assignee: ["crab"]
labels: []
group: ""
rank: "zzzzzzzzzzzw"
created: 2026-11-20
updated: 2026-11-20
doing_with: "p87ngv"
doing_since: "2026-11-20T11:30"
stage: "active"
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
---

## Goal
With no coordinator named, or the named one gone, requests go to the board's live agent session without any setup.

## Purpose
Naming a coordinator by hand breaks when the agent changes or the harness restarts, and the common case has one agent.

## Principle
Deduce it in the common case, say who it is, and let hug task coordinator NAME override.

## Relevant details
Proposed rule (crab, to confirm in check 1): when [board] coordinator is unset or names no live session, the coordinator is the live agent session on this board: the only one if there is one, else the one that touched the board first and is still live. It stays coordinator while it lives; when it ends, the next live one takes over. hug task coordinator (no name) shows who it is and why (named, the only live agent, or first live). An explicit name still wins while that session lives. Today: [board] coordinator = "crab" in .hug/tasks/hug.toml; with --clear every session hears an unowned request once. Code: huginn delivery of requests (gmzd87, hbgybn). Jørgen 2026-11-20 10:30Z on gmzd87: 'they have to register. I am the coordinator, but shouldn't we implicitly do that if there's only one task active or only one agent active? And can we deduce this in the common case so that the first task that interacts is set as active as long as it's active. And if you want to override, you override. Rather than you have to do the setup and you have to pick the task because the agent can change, you restart the harness, something happens. So can we do something sensible by default here?'

## Depends on

## Acceptance
- [ ] Jørgen agrees with the rule in Details (or corrects it)
- [ ] With no coordinator set and one live agent session on the board, a request on an unowned task reaches that session (test)
- [ ] With several live sessions, the first live one to touch the board receives it; when it ends, the next live one does (test)
- [ ] A named coordinator wins while live; a named one that is gone falls back to the rule, and hug task coordinator says who and why (test)
- [ ] Jørgen restarts his coordinator's harness, writes a request, and it arrives with no setup

## Questions
- 2026-11-20T11:31 question 1 by crab to jhf (check 1):
  Default coordinator when none is named, or the named one is gone: the only live agent session on the board, else the first live one to touch it, handing over when it ends; a name you set still wins. Is that the rule you want?
  answered 2026-11-20T12:55 by jhf:
    Yes, but when you when you deduce, you can just ask. We're giving this to you received this because you were perceived to be the coordinating task interacting with a human at a console. If you are not and instruction to fix the first time you see an agent who you have deduced

## Notes


## History
- 2026-11-20T11:30 todo -> doing/active (take with p87ngv)
- 2026-11-20T11:31 doing/active -> doing/asked (ask)
- 2026-11-20T12:55 answered asked (check 1), by jhf, for crab (doing/asked -> doing/active):
  > Default coordinator when none is named, or the named one is gone: the only live agent session on the board, else the first live one to touch it, handing over when it ends; a name you set still wins. Is that the rule you want?
  Yes, but when you when you deduce, you can just ask. We're giving this to you received this because you were perceived to be the coordinating task interacting with a human at a console. If you are not and instruction to fix the first time you see an agent who you have deduced
- 2026-11-20T12:55 answer 1 delivered to crab (session_crab_1793783281662_44ffaee927a8a9d5)
