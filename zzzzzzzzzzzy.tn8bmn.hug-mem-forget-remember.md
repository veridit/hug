---
id: tn8bmn
title: "hug mem: forget and remember report what the store really holds"
state: "doing"
type: bug
priority: "high"
assignee: ["hatchling"]
labels: []
created: 2026-11-18
updated: 2026-11-18
doing_since: "2026-11-18T03:38"
stage: "active"
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
---

## Goal
Every hug mem write reports the store's state after the write: forget exits non-zero unless the memory is inactive afterwards, and remember prints the id the store keeps (after any twin fold), so an agent never acts on a write that did not happen.

## Purpose
hatchling 2026-11-18 daily check (66ex4y, file:lost-forget-repro.txt) on Jørgen's live store: (A) 'hug mem forget xr634s' printed 'forgot' with exit 0, yet xr634s stayed active until a second forget; (B) 'hug mem remember' printed yx7vxv while the store holds erf4tk, and 'show yx7vxv' resolves to erf4tk. Neither reproduces on a HUG_HOME copy, which points at the live store's concurrency (several hooks, capture and MCP servers write it at once) or at the twin fold (5sc5rw). Jørgen's rules: nothing writes silently; concurrent writes wait, never fail.

## Principle
Find the cause first, then fix the cause, not the message. A write's reply is read back from the store under the same lock that wrote it. If a concurrent writer undoes it, the write waits and redoes it; it never reports success it does not have.

## Relevant details
Live store writers today: capture (turn_end, session_end hooks), recall hook, hug mem CLI, 8 hug mcp servers on old binaries (mbdfzs fixes those after one restart). Look at ingest.ForgetMany's lock and the read-modify-write of memories.json; and at remember's id when the fold merges a new memory into an existing twin. wyvern is changing forget's reindex in 4vd1mz: coordinate, don't collide.

## Depends on

## Acceptance
- [ ] A test with two concurrent writers (forget and a capture or remember) shows the forget survives: the memory is inactive after both finish (fails first if the race is the cause)
- [ ] remember prints the id the store keeps; when it folds into a twin it says so: 'kept as erf4tk (twin of your text)' (test)
- [ ] forget and remember read their result back under the write lock and exit non-zero with a fix line if it is not there (test)
- [ ] The cause of (A) and (B) is named in a note, with the evidence

## Notes


file:tn8bmn-notes-history.txt (716 bytes, sha256:1618809b)
## History
- 2026-11-18T03:38 todo -> doing/active (take)
