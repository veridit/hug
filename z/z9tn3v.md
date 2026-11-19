---
id: z9tn3v
title: "Release v0.7.75"
state: "done"
type: feature
priority: ""
assignee: ["crab"]
labels: []
group: ""
rank: "zzzzzzzzzzzy"
created: 2026-11-19
updated: 2026-11-19
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
done_at: "2026-11-19T03:09"
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
- [x] release.yml for v0.7.75 succeeds (run id)
- [x] hug version prints v0.7.75 after brew upgrade

## Notes


check 1: release.yml 44441121338 completed success
check 2: brew upgrade veridit/tap/hug run on its own; hug version prints hug v0.7.75
Reflection (open, 2026-11-19, crab): A rebase conflict had cut 5mmqjk's CHANGELOG entry to its first line, and only reading Unreleased before the cut caught it. rebase-land.sh's CODE-DIFF check excludes CHANGELOG, so a truncated entry passes it.
## History
- 2026-11-19T03:01 todo -> doing/active (take)
- 2026-11-19T03:09 doing/active -> done (done); was doing_since: 2026-11-19T03:01
