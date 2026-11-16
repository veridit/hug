---
id: 48hx44
title: "hug setup wires desktop apps"
status: todo
type: feature
priority: ""
assignee: []
labels: []
created: 2026-11-16
updated: 2026-11-16
---

## Goal
hug setup finds Claude Desktop, Claude Code (~/.claude.json), Codex/ChatGPT (~/.codex/config.toml) and jcode configs, reports which lack the hug MCP server, and adds it with a backup on request

## Depends on

## Acceptance
- [ ] setup lists each app config found with done/open and the exact install command
- [ ] hug install apps adds hug mcp to each, keeps a timestamped backup, idempotent, --dry-run
- [ ] a server whose binary predates the installed hug is reported with 'restart the app'

## Notes
