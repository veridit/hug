---
id: kf00pd
title: "Release v0.7.62: default sort row"
state: todo
type: chore
priority: ""
assignee: []
labels: []
group: ""
rank: "zzzzzzzzzzzy"
created: 2026-11-18
updated: 2026-11-18
---

## Goal
v0.7.62 ships nch9d4 (sort picker r rank row, now/new lines) from exact all-green 6aee8e30, published before targeted install.

## Purpose
Jørgen is testing releases live this morning; his default-sort feedback is on main with all 6 CI jobs green.

## Principle
Release only new content with exact-SHA all-job CI green; release.yml success before brew upgrade; no --machine migration.

## Depends on

## Acceptance
- [ ] CHANGELOG v0.7.62 heading covers only nch9d4, no conflict markers, released history unchanged
- [ ] Tag release workflow succeeds before targeted brew upgrade, hug version prints v0.7.62

## Notes
