---
id: z9tn3v
title: "Release v0.7.75"
state: todo
type: feature
priority: ""
assignee: []
labels: []
group: ""
rank: "zzzzzzzzzzzz"
created: 2026-11-19
updated: 2026-11-19
---

## Goal
v0.7.75 is tagged, release.yml publishes it, and brew upgrade brings hug version v0.7.75 on this machine.

## Purpose
Hourly release: 5mmqjk, z25pb2, ghycrq, j0zvee, nrp268, ynh84t, 8jcj3k.

## Principle
Release from exact-SHA green main only; brew runs on its own, never in a timed watcher.

## Relevant details
main fbe3a412 CI 44438198799 success 8/8. CHANGELOG commit 4591627d (restored the 5mmqjk entry, cut to one line by a rebase conflict), release merge 0b3e267a, tag v0.7.75.

## Depends on

## Acceptance
- [ ] release.yml for v0.7.75 succeeds (run id)
- [ ] hug version prints v0.7.75 after brew upgrade

## Notes
