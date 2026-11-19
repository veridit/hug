---
id: 8jcj3k
title: "Capture stops for good on one message over 8192 bytes"
state: "doing"
type: feature
priority: ""
assignee: ["piglet"]
labels: []
group: ""
rank: "zzzzzzzzzzzw"
created: 2026-11-19
updated: 2026-11-19
doing_since: "2026-11-19T02:30"
stage: "active"
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
---

## Goal
A session whose transcript holds one message longer than capture's chunk limit (8192 bytes) keeps being captured: the long message is split or summarised, and the turns before and after it reach the extractor.

## Purpose
66ex4y daily 2026-11-19: 5 live sessions have captured nothing since one long message, and every hook retries and fails again. capture_runs: crab_1793783281662 841 refusals 2026-11-16 00:09 to 2026-11-19 01:21 (last capture 2026-11-16T00:09), hibiscus 562 since 11-07, rabbit 466 since 11-10, spider 262 since 11-12, sabertooth 46 since 11-07. hook.log: 515 'capture failed class=input' lines from 2026-11-18T02:31Z to 2026-11-19T00:59Z, 577 on 2026-11-18 alone.

## Principle
No silent loss stays the rule: nothing is dropped unseen. A message that cannot be read whole is cut into parts, never skipped without a record. No paid call is added for it.

## Relevant details
muninn/internal/app/capture.go:285: if len(line) > chunkLimit, return 'capture message exceeds %d bytes, refusing silent loss' (since ab68a9bc, 2026-11-05). The high-water mark never moves past that message, so every later turn retries from it. Coordinator sessions (crab) write the longest messages, so the sessions that make most rulings are the ones not captured.

## Depends on

## Acceptance
- [x] a session with a 20,000-byte message is captured on the next hook: turns after it produce memories and the high-water mark moves past it (test)
- [ ] the long message's text reaches the extractor in parts, each under the limit, with nothing dropped (test)
- [ ] on the live store, crab_1793783281662's capture_runs show a successful run after the fix ships (count in a note)

## Notes


check 1: TestCaptureLongMessageIsSplitNotRefused (f56abe6a) fails on main fac02994 for 20000 bytes turn_end, 3x8192 session_end, 60000 manual: 'capture message exceeds 8192 bytes, refusing silent loss'; passes with fix 14d51bc1: before/long/after all in capture_seen, a second run replays nothing
## History
- 2026-11-19T02:30 todo -> doing/active (take)
