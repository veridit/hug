---
id: ynh84t
title: "hug mem install writes 'muninn hook'; task install names the MCP entry huginn"
state: todo
type: feature
priority: ""
assignee: []
labels: []
group: ""
rank: "zzzzzzzzzzzt"
created: 2026-11-18
updated: 2026-11-18
parent: "gem59b"
---

## Goal
Hooks and MCP entries that hug writes into users' harness configs run and name hug: hug mem install writes 'hug mem hook claude|codex|jcode' (not '<path>/muninn hook ...'), hug task install names its jcode MCP entry hug (not huginn), and hug upgrade --apply --machine moves existing configs over with a preview.

## Purpose
p277bp check 2 (crab 22:19): users type only hug, but a user who opens ~/.claude/settings.json, ~/.codex/hooks.json, a jcode config or mcp.json sees muninn and huginn, written by hug itself. Found in the p277bp check 2 sweep.

## Principle
Behaviour that edits users' configs needs an upgrade path: preview first, --apply --machine moves them, backups kept, old commands keep working as compatibility links. No alias added; nothing is changed without the upgrade preview.

## Relevant details
muninn/run/surfaces.go:105 selfPath: 'writing .../hug into a hook would run hug hook jcode, which hug does not know' (so hook commands are <dir>/muninn hook X). muninn/internal/harness/install.go:149-185 InstallClaude/Codex/Jcode use bin+' hook claude|codex|jcode'. huginn/internal/cli/install.go:67 'MCP config runs huginn mcp (task tools only), not hug mcp'; hug task help install: 'Adds huginn -> hug task mcp to Jcode mcp.json'. huginn/internal/cli/upgrade_env.go ours()/renameJSON/renameTOML already recognise hug|muninn|huginn commands, a place to hang the move. Evidence: /Users/jhf/.jcode/scratch/p277bp-check2.md.

## Depends on

## Acceptance
- [ ] hug mem install claude|codex|jcode writes a hook command that runs hug (hug mem hook ...), test
- [ ] hug task install writes a jcode MCP entry named hug running hug, test
- [ ] hug upgrade (preview) lists each existing muninn/huginn hook and MCP entry it would move; --apply --machine moves them with backups, and the hooks still recall (test with a fake HOME)
- [ ] On Jørgen's machine after the release, no harness config names muninn or huginn (grep in a note)

## Notes
