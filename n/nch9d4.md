---
id: nch9d4
title: "Sort picker offers default board order explicitly"
state: "doing"
type: feature
priority: ""
assignee: ["macaque"]
labels: []
group: ""
rank: "zzzzzzzzzzzx"
created: 2026-11-18
updated: 2026-11-18
stage: "active"
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
doing_since: "2026-11-18T12:18"
---

## Goal
Make the default ordering visible and directly selectable in the sort picker, clearing every explicit sort field.

## Purpose
Jorgen testing v0.7.61 could not find how to return from priority sorting to the default order.

## Principle
Default means board rank, the manually arranged order. Offer a clearly named default option rather than implying tasks are unordered. Preserve existing sort choices.

## Relevant details
User screenshot 2026-11-18: filter line says priority descending while fresh picker draft says rank, with no default row. Current implementation openSort clears draft, so Enter on untouched picker restores rank but this is undiscoverable. Explain the distinction between current order and draft.

## Depends on

## Acceptance
- [ ] Sort picker shows an explicit selectable default board order (rank) option.
- [ ] Choosing default from a multi-field sort clears every sort key and restores board rank order, with a regression test.
- [ ] Picker text does not misleadingly present an unapplied rank draft as the current priority ordering.

## Notes


Coordinator crab hands this to a swarm engineer at 2026-11-18 11:18Z; no code changes made by crab.
Letter r for rank: free (fields are p t s u d g a n, c is column), and rank is the word the filter line, ': sort rank' and the move refusal already use. Picker row: '  r  rank      board order, the default: moves work only in it'. r applies at once (as c does), clears draft and sort. Branch nch9d4, worktree ~/.jcode/scratch/nch9d4.
## History
- 2026-11-18T12:15 todo -> doing/active (take)
- 2026-11-18T12:17 doing/active -> todo/ready (give back; was assignee: crab); was doing_since: 2026-11-18T12:15
- 2026-11-18T12:18 todo/ready -> doing/active (take)
