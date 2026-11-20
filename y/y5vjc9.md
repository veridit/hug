---
id: y5vjc9
title: "Rename what users see from muninn and huginn to hug"
state: todo
type: chore
priority: "medium"
assignee: []
labels: []
group: ""
rank: "zzzzzzzzzzzx"
created: 2026-11-20
updated: 2026-11-20
---

## Goal
Every name a user sees says hug: the data folder, the database file, the config, log lines and messages, with old ones moved once.

## Purpose
Jørgen still meets 'muninn' in paths and messages, which makes it look like two tools.

## Principle
Rename what users see in one sweep with a one-time move and a backup; package folders in the code may keep their names.

## Relevant details
Seen 2026-11-20 (Jørgen, rnpdnm request 5: 'why do we have something called muninn still? Shouldn't we rename everything consistently to be hug memory?'). Today on macOS: data and config in ~/Library/Application Support/muninn/ (muninn.db 805 MB, memories.json 18.6 MB, config.json, hook.log); Linux ~/.local/share/muninn and ~/.config/muninn (muninn/internal/config/config.go:155-179). Code mentions: 460 'muninn' and 256 'huginn' in non-test Go (many are package paths, which stay). Proposal: ~/Library/Application Support/hug/ with memory/ inside (hug.db, memories.json, config.json); hug upgrade --apply moves the folder once (backup first), a symlink from the old path for a release, and every message, help text and log line says hug. The muninn/ and huginn/ code folders and Go package names are internal and may stay (users never see them); the README folder table explains them.

## Depends on

## Acceptance
- [ ] A list of every user-visible 'muninn'/'huginn' (paths, file names, messages, help, logs) is in a note, from a grep with the command
- [ ] hug upgrade --apply moves the data folder to the hug name with a backup, and hug mem config shows the new paths (test on a temp HOME)
- [ ] No message, help or log line a user sees says muninn or huginn, except the mascot text and the compatibility links (test greps the strings)

## Notes
