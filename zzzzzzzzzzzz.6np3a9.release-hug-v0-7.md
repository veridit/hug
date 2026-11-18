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
- [x] Exact green source, CHANGELOG grouping and conflict-marker checks verified
- [x] Release workflow succeeds before installation
- [ ] Targeted brew upgrade installs hug v0.7.60

## Notes


Recovery source dcb80a5d independently verified all six jobs success in CI44268461420. Tag4c94ede4 and merge7f9f7bd8 pushed. CHANGELOG grouping manually reviewed, old v0.7.59-and-earlier section byte-identical via cmp, only CHANGELOG differs from green source, repocheck0.275s and diff-check pass. Failed v0.7.59 tag unchanged. Publication/install pending.
Release workflow44269866555 succeeded on exact tag4c94ede4 before targeted installation.
## History
- 2026-11-18T09:45 todo -> doing/active (take)
