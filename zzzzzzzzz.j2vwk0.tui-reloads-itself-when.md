---
id: j2vwk0
title: "TUI reloads itself when a new hug version is installed"
status: "doing"
type: feature
priority: "medium"
assignee: ["vole"]
labels: []
created: 2026-11-16
updated: 2026-11-16
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
---

## Goal
After brew upgrade (or any new hug binary at the same path), an open hug tui switches to the new version without losing its place: same view, selected task, filters, focus and scroll

## Purpose
Jørgen 2026-11-16: 'Can the tui reload a new version, a bit like how jcode does?' Today every release needs a manual restart of each open TUI, and with a release per fix that is several times a day

## Principle
Detect by watching the executable path the TUI was started from (brew swaps the /opt/homebrew/bin/hug symlink to a new Cellar dir): compare the resolved target and mtime. Reload by exec of the same path with the state handed over in a small file, never by forking a second TUI. Never reload while a prompt, the palette or the editor is open; wait until it closes. If the new binary fails to start, the old one keeps running and says why. Open: reload by itself at a quiet moment, or show a footer note and reload on a key (ask Jørgen).

## Relevant details
Same fsnotify watcher style as nfv18p (auto-refresh), so build after it, same files (huginn/internal/tui). State to carry: view, selected id, active filters, focus (page/checks/deps), scroll offset, open task.

## Depends on

## Acceptance
- [ ] a test replaces the binary at the started path and the TUI re-execs into it with the same view, selection and filters
- [ ] with a prompt or the editor open, the reload waits until it closes
- [ ] a broken new binary leaves the old TUI running with a message and a fix: line
- [ ] the footer says which version it reloaded into, e.g. 'reloaded hug v0.7.21'

## Notes

2026-11-16 Jørgen: no backwards compatibility anywhere; the reload hands state only to the same or newer hug and needs no old-format reader.
2026-11-16 Jørgen: Showing the version somewhere would also be nice, what version I'm currently running.
Jørgen 2026-11-16 ruling: no automatic reload. When a newer hug is installed, the version corner (lower right) shows it, e.g. 'hug v0.7.35 → v0.7.36 · R reloads', and R re-execs keeping view, selection, filters, focus and scroll. Nothing else changes until he presses R. R is shown only while a newer version exists (keys never change meaning on hidden state: if R is already bound, pick a free key and show that one).
