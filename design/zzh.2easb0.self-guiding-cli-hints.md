---
id: 2easb0
title: "Self-guiding CLI hints"
status: "done"
type: feature
priority: "low"
assignee: []
labels: ["cli"]
created: 2026-11-10
updated: 2026-11-10
---

## Goal
Commands end with a next-step hint and fix-it errors, like huginn

## Depends on

## Acceptance
- [x] every command prints next: hint

## Notes

Done 2026-11-16: hug mem commands end with 'next: ...' from one PersistentPostRun (muninn/run/main.go nextHints); JSON-printing commands (stats, config) put it on stderr so stdout stays parseable; hook, mcp, serve, tui, capture stay silent; --json suppresses it. Fix-it errors were already in place.
