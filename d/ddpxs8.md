---
id: ddpxs8
title: "Release v0.7.86"
state: todo
type: feature
priority: ""
assignee: []
labels: []
group: ""
rank: "zzzzzzzzzzzz"
created: 2026-11-20
updated: 2026-11-20
---

## Goal
v0.7.86 (one-sentence Goals) is published, installs via brew, and Jørgen says whether the shorter Goals read better.

## Purpose
Release-as-group trial (75mxf3): one ticket groups what ships and asks one question per user-facing change.

## Principle
Shipped tickets are children; one ask per user-facing change; test-only fixes are listed, not asked about.

## Relevant details
main 00dfa4cf (CI 44673649756, 8/8), CHANGELOG 2d1cdbd6, merge a09e9d63, release.yml 44675088441 success. Ships 0pr03h (Goal one sentence <= 140 chars; new/set refuse, lint warns; 47 open hug Goals rewritten, old text at the top of Details). Test-only: wacc33 (tmux leak), 6zhyq2 (blink/settled), 4gj7z6 (offline sync test). Docs: h9ymk8 (AGENTS.md env Known rule).

## Depends on

## Acceptance
- [ ] release.yml green and brew upgrade gives hug version 0.7.86 with links into the new Cellar

## Notes
