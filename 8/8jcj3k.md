---
id: 8jcj3k
title: "Capture stops for good on one message over 8192 bytes"
state: "doing"
type: feature
priority: ""
assignee: ["piglet"]
labels: []
group: ""
rank: "zzzzzzzzzzzp"
created: 2026-11-19
updated: 2026-11-19
doing_since: "2026-11-19T02:30"
stage: "waiting"
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
waiting_on: "CI 44438088626 on ci/8jcj3k (14d51bc1), then crab lands it; check 3 after release"
waiting_since: "2026-11-19T02:36"
waiting_asked_by: "piglet"
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
- [x] the long message's text reaches the extractor in parts, each under the limit, with nothing dropped (test)
- [ ] on the live store, crab_1793783281662's capture_runs show a successful run after the fix ships (count in a note)

## Notes


check 1: TestCaptureLongMessageIsSplitNotRefused (f56abe6a) fails on main fac02994 for 20000 bytes turn_end, 3x8192 session_end, 60000 manual: 'capture message exceeds 8192 bytes, refusing silent loss'; passes with fix 14d51bc1: before/long/after all in capture_seen, a second run replays nothing
check 2: same test: every call's transcript <= limit, the assistant parts rebuild the exact original text, the after-turn reaches the extractor in order; capture_runs holds >= ceil(size/limit) parts with original_id=long and report.split=1. 4 mutants fail it: size -2 (one byte over), fragment on hooks only, no Split, Split counts parts. go test ./muninn/internal/app ok 20.482s; go vet ./muninn/... clean
piglet: 14d51bc1 (fix) on f56abe6a (failing test), base fac02994, pushed to 8jcj3k and ci/8jcj3k, CI 44438088626. Root causes: (1) hook fragments sized limit-len(role)-2 but the transcript line is role+': '+text+'\n', so a full ASCII part is one byte over the limit (the old test used é, whose rune cut hid it); (2) manual/backfill runs never fragmented. Check 3 waits for the release; live high-water untouched.
landed 4dabf89 on main 2026-11-19: capture: a message over the chunk limit is refused for good: failing test…
landed fbe3a41 on main 2026-11-19: capture: a long message is cut into parts, never refused (8jcj3k)
shipped in v0.7.75 (4dabf89)
crab: landed on main (stack tip fbe3a412, CI 44438198799 success 8/8); released in v0.7.75. The open check waits on the live machine after brew upgrade.
## History
- 2026-11-19T02:30 todo -> doing/active (take)
- 2026-11-19T02:36 doing/active -> doing/waiting (wait)
