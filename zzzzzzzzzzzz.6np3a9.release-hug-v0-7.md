---
id: 6np3a9
title: "Release hug v0.7.60"
state: "doing"
type: feature
priority: ""
assignee: ["crab"]
labels: []
created: 2026-11-18
updated: 2026-11-18
doing_since: "2026-11-18T09:45"
stage: "active"
model: "openai"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
---

## Goal
Publish and install v0.7.60 from the exact all-jobs-green recovery commit after the failed v0.7.59 publication.

## Purpose
Deliver the reviewed false-idle handoff fix and faster forget without changing the failed tag or mixing layout 3 into the release.

## Principle
Use exact green source, preserve tags, verify release workflow before targeted installation, and never change machine configuration.

## Relevant details
v0.7.59 tag 6d994e71 remains unchanged. Recovery candidate dcb80a5d passed all six CI jobs in 44268461420. Layout 3 stays separate. Historical release failure cause remains unproven.

## Depends on

## Acceptance
- [ ] Exact green source, CHANGELOG grouping and conflict-marker checks verified
- [ ] Release workflow succeeds before installation
- [ ] Targeted brew upgrade installs hug v0.7.60

## Notes


## History
- 2026-11-18T09:45 todo -> doing/active (take)
