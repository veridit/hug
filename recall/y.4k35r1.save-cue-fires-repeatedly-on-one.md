---
id: 4k35r1
title: "Save cue fires repeatedly on one message"
status: todo
type: bug
priority: "medium"
assignee: []
labels: ["recall"]
created: 2026-11-10
updated: 2026-11-10
---

## Goal
The 'Sounds like a lasting preference' cue appears at most once per user message

## Depends on

## Acceptance
- [x] regression test: one message, one cue
- [x] hook log shows no repeated cue injections for one message

## Notes

Fixed in hug 819fde8 (2026-11-12) with regression tests.
