---
id: z9q42h
title: "Recall reaches jcode through a tool hook, never the person's input box"
state: "doing"
type: bug
priority: "high"
assignee: ["hatchling"]
labels: []
created: 2026-11-18
updated: 2026-11-18
doing_since: "2026-11-18T02:11"
stage: "active"
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
---

## Goal
On jcode, the recall block reaches the agent without touching the TUI's input box: turn_start computes it and parks it for the session, and the first tool call of that turn delivers it as post_tool_feedback (or pre_tool additionalContext). jcode transcript is no longer used.

## Purpose
Jørgen 2026-11-18 01:53 local: 'sometimes the transcripts inject while I'm typing or having a pause typing'. Cause: muninn/run/main.go:749 delivers recall with 'jcode transcript --mode send', which injects text into the active TUI's composer, the same box he is typing his next message in. The turn_start hook is a detached observer, so the block lands seconds later, often mid-draft. It also shows as '[transcription]' user messages that are not his words.

## Principle
Nothing hug writes ever enters the person's input. Recall is context for the agent, so it rides on a tool result. A turn with no tool call gets no block (the agent can still hug mem search); a block is delivered at most once and never into a later turn. No change to recall content or ranking.

## Relevant details
jcode docs/HOOKS.md: post_tool_feedback stdout is appended to the tool result under [post_tool_feedback] (8 KB cap, runs synchronously after every successful tool call, default 10 s timeout); pre_tool additionalContext appears under [pre_tool context]. ~/.jcode/config.toml already has post_tool_feedback = scratch_alarm.sh; hooks accept arrays, so hug setup adds its command beside it. The parked block lives under the hug home keyed by session id and turn, removed on delivery or at turn_end. JcodeTurnOpen and the transcript code go. The feedback hook must be cheap on calls with nothing parked (one stat).

## Depends on

## Acceptance
- [ ] With a draft typed in the jcode input box, a turn's recall never appears in that box or as a user message (test drives the hook path; and Jørgen confirms after a day of use)
- [ ] The block appears once, on the first tool result of the turn it was computed for; a turn without a tool call drops it at turn_end; it never shows in a later turn (tests, fail first on main)
- [ ] hug setup writes the post_tool_feedback entry next to any existing one (array), previewed first, and hug upgrade migrates current installs off jcode transcript (test)
- [ ] The feedback hook on a call with nothing parked does one file check and no store load (test counts work)

## Notes


## History
- 2026-11-18T02:11 todo -> doing/active (take)
