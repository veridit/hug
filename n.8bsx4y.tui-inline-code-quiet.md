---
id: 8bsx4y
title: "TUI: inline code in a quiet tint, not red"
status: "doing"
type: bug
priority: "medium"
assignee: ["vole"]
labels: []
created: 2026-11-16
updated: 2026-11-16
model: "claude-opus-5-5"
---

## Goal
Inline code and code blocks in the task detail read as code without shouting: light blue-grey text on a faint background, no padding, no red; red stays for blocked and lint problems

## Purpose
Jørgen 2026-11-16 on fpp2ns: the red-on-grey inline code everywhere in the description 'isn't it'. Red is reserved for diagnostic meaning. A real syntax colouring tool comes later, not now.

## Principle
Override glamour's Code and CodeBlock styles in huginn/internal/tui/detail.go (next to the heading colours). Dark: text #c9d6e3, background #1a2630, no prefix/suffix padding. Light: #2b3a48 on #e8eef3. Code blocks: same background, no syntax colours. NO_COLOR: plain text.

## Relevant details
Agreed palette from crab's proposal, approved by Jørgen 'Yes'.

## Depends on

## Acceptance
- [ ] the detail of a task with inline code shows no red; inline code uses #c9d6e3 on #1a2630 on a dark terminal (golden)
- [ ] light terminal uses #2b3a48 on #e8eef3; NO_COLOR shows plain text (test)

## Notes
