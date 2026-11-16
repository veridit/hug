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
- [x] setup lists each app config found with done/open and the exact install command
- [x] hug install apps adds hug mcp to each, keeps a timestamped backup, idempotent, --dry-run
- [ ] a server whose binary predates the installed hug is reported with 'restart the app'

## Notes

Built: 'hug apps [install] [--dry-run] [--json]' (cmd/hug/apps.go, TestAppsInstall): Claude Desktop, ~/.claude.json, ~/.codex/config.toml; reports ok/missing/stale; install adds hug mcp, drops old muninn/huginn entries, keeps other servers, timestamped backup. hug setup has an 'app MCP' step. hug mcp adds a note to tool results when the Homebrew hug was upgraded under a running server ('restart the app'). On this machine all three show ok.
