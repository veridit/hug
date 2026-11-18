---
id: w2wr9b
title: "Release v0.7.61: standalone layout 3"
state: "doing"
type: feature
priority: ""
assignee: ["crab"]
labels: []
group: ""
rank: "zzzzzzzzzzzy"
created: 2026-11-18
updated: 2026-11-18
doing_since: "2026-11-18T10:46"
stage: "active"
model: "openai"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
---

## Goal
Publish and install the exact all-jobs-green layout-3 release after v0.7.60, then safely migrate only hug own board.

## Purpose
Deliver fixed task paths and safe project-memory migration without mixing unrelated changes or changing machine settings.

## Principle
Release exact green source only, preserve released tags and changelog history, wait for successful publication before targeted install, preview before board-only apply, never --machine.

## Relevant details
Source ead1be4ca37a1042e64c4878d0afcdba7c921b3f, CI44277147694 all6 success including WindowsCLI302.891s. w40cmk all implementation checks ticked, owner7 remains. Isolated release worktree rel-7d600ff, root checkout dirty and untouched. After successful release/install, preview/apply hug own board and notify actual Seid/Statbus agents.

## Depends on

## Acceptance
- [x] Release changelog includes only layout commits since v0.7.60, released history byte-identical and no conflict markers
- [ ] Exact tag release workflow succeeds before targeted brew upgrade and hug version verifies v0.7.61
- [ ] Hug own-board preview is reviewed, board-only apply verified, machine configs/native bytes not migrated, Seid and Statbus agents notified

## Notes


Standalone v0.7.61 tag7da883ae245e2fb9097060a9493b7c930679701d, exact source ead1be4c CI44277147694 all6 green. CHANGELOG adds only version heading over exactly2 w40cmk bullets; all .60-and-earlier bytes identical, repocheck0.491s and diff/marker guards pass. Trailer Model: claude-opus-5-5/high. Atomic push main+tag completed; release44279353151 running. Publish-gated targeted brew install then preview-only background3606026c6i owns follow-through. No --machine or apply yet.
## History
- 2026-11-18T10:46 todo -> doing/active (take)
