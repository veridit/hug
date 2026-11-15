---
id: jrj1vq
title: "When-then memories: triggers stated at write time"
status: todo
type: feature
priority: ""
assignee: []
labels: ["recall", "memory-format"]
created: 2026-11-15
updated: 2026-11-15
---

## Goal
Memories carry an explicit trigger so recall injects them when the situation matches, not when the words do

## Depends on

## Acceptance
- [ ] memory format: optional case (scope, paths, tool), when (situation text), when_not (suppressing situations), then (the note); plain memories stay valid
- [ ] memory_remember and capture ask the storer for when/then when the memory is a rule or pattern; dreaming proposes when/then for existing rules, agent-judged
- [ ] recall matches the current situation (message, and later tool calls) against when/when_not with BM25 + vector, separately from the note text
- [ ] measured on the frozen and fresh replay sets against shipped recall before becoming the default

## Notes
