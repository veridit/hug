---
id: 7js5hw
title: "TUI: human names for places"
status: "doing"
type: bug
priority: "medium"
assignee: ["Jørgen H. Fjeld"]
labels: ["tui"]
created: 2026-11-10
updated: 2026-11-10
---

## Goal
Places pane shows 'home folder', project names and client names instead of raw scopes like global:73d8f1c06765a0b4 or scope:global

## Depends on

## Acceptance
- [x] tmux golden shows readable place names

## Notes

Done 2026-11-16: places pane shows folder names for project scopes (project:rc-0ca4bf6be5d989c3 -> 'hug', 316 memories) using the working folders of jcode sessions (byte scan of ~1,950 session files, 5 s, built in the background and cached a day in project-labels.json); shared scopes in words: global -> 'all agents', jcode@global -> 'jcode, all projects', client:x -> 'x (client)'. Goldens updated (80x24: '  all agents 4'). Open question for the user: the binding's own 'global' place and the raw global scope still sit side by side.
