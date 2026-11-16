---
id: ez50rb
title: "Memories hold while a stated condition holds, and name no agent"
status: todo
type: feature
priority: "high"
assignee: []
labels: []
created: 2026-11-16
updated: 2026-11-16
parent: "d634kr"
---

## Goal
A memory that is only true for a while says so when it is written: 'holds until <date>', 'holds while task:ID is open', 'holds while commit/branch X is unmerged'. When the condition ends, the memory stops being recalled and goes to a dream item for an agent to retire or keep. Memories never name an ephemeral agent; they name the ticket, commit, file or model instead.

## Purpose
Jørgen 2026-11-16: memories naming agents are bad memories; 'if you know it at the time, write the condition or when it expires ... if that's no longer valid, it will be gone'. Evidence the same day: orangutan scanned 10,107 seid memories, found 239 naming 97 stopped agents (including words like tiger and swan that needed hand checks) and bulk-deleted them; 233 of the first 239 forgets failed under concurrent writes. bx26kc (2026-11-12) already said capture should not store short-lived agent status.

## Principle
State the condition at write time, the same way jrj1vq states when/then: the storer knows it then, nobody knows it later. Conditions are checkable by hug without a model call: a date, a task's status, a git ref. When one ends, the memory is hidden from recall and offered once as a dream item (agent-judged, never deleted automatically, per 156x77). Capture rewrites agent names to what they refer to (the ticket the agent was on), and drops pure agent status. No condition means the memory holds until judged otherwise, as today.

## Relevant details
Builds on the when-then format (jrj1vq, same epic d634kr): this is one more field, holds_while/until. Capture prompt: muninn/internal/extract/extract.go:239. Recall filter: muninn/internal/recall. Old memories naming agents: dream items, one at a time, not a regex purge. Ship recall changes only with a measured, CI-clear gain on replayed messages (AGENTS.md).

## Depends on

## Acceptance
- [ ] memory_remember and hug mem remember accept a condition (until DATE, while task:ID open, while REF unmerged) and store it (test)
- [ ] a memory whose condition has ended is not recalled and appears once as a dream item; nothing is deleted automatically (test)
- [ ] capture on a transcript that names a running agent writes the ticket or file instead and never stores agent status (replayed sample, counted)
- [ ] on the seid store, memories naming stopped agents are offered as dream items with the ticket they referred to (count reported)

## Notes
