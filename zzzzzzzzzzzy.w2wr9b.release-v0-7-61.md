---
id: w2wr9b
title: "Release v0.7.61: standalone layout 3"
state: todo
type: feature
priority: ""
assignee: []
labels: []
created: 2026-11-18
updated: 2026-11-18
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
- [ ] Release changelog includes only layout commits since v0.7.60, released history byte-identical and no conflict markers
- [ ] Exact tag release workflow succeeds before targeted brew upgrade and hug version verifies v0.7.61
- [ ] Hug own-board preview is reviewed, board-only apply verified, machine configs/native bytes not migrated, Seid and Statbus agents notified

## Notes
