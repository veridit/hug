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
- [x] a test replaces the binary at the started path and the TUI re-execs into it with the same view, selection and filters
- [x] with a prompt or the editor open, the reload waits until it closes
- [ ] a broken new binary leaves the old TUI running with a message and a fix: line
- [ ] the footer says which version it reloaded into, e.g. 'reloaded hug v0.7.21'

## Notes

2026-11-16 Jørgen: no backwards compatibility anywhere; the reload hands state only to the same or newer hug and needs no old-format reader.
2026-11-16 Jørgen: Showing the version somewhere would also be nice, what version I'm currently running.
Jørgen 2026-11-16 ruling: no automatic reload. When a newer hug is installed, the version corner (lower right) shows it, e.g. 'hug v0.7.35 → v0.7.36 · R reloads', and R re-execs keeping view, selection, filters, focus and scroll. Nothing else changes until he presses R. R is shown only while a newer version exists (keys never change meaning on hidden state: if R is already bound, pick a free key and show that one).

Jørgen: the R key also makes the upgrade testable in a controlled way. Test: install a newer binary at the started path, check the TUI keeps running the old one and only the corner changes, press R, check the new version runs with the same view/selection/filters/scroll. Also do this once by hand after the release that ships it, from the previous version.

progress (vole): commit 6360498 on ci/j2vwk0 (on 3ea79d3 v0.7.35). CI run 43947993605: ubuntu success (-race), windows success, macos queued (runner shortage: the earlier run 43945189594 macos job was cancelled with 'The job was not acquired by Runner of type hosted even after multiple attempts'). Local: TestTmuxReload PASS 'v0.0.1 -> v0.0.2 by symlink swap, same pane pid, same place; broken v0.0.3 refused'; tui package ok 69.404s. Report: tmp/lanes/j2vwk0.md

check 1: 6360498 on origin/main (CI run 43947993605: ubuntu, macos, windows success). TestTmuxReload (huginn/internal/tui/reload_tmux_test.go), real binary: v0.0.1 started as 'hug tui' through bin/hug -> Cellar/hug/0.0.1/bin/hug; list, filter status:doing, sort priority, 3rd row open, Tab to checks; symlink renamed to Cellar/hug/0.0.2; same pane pid after; screens before/after (tmp/lanes/j2vwk0/reload/) differ only in line 38 '< sort: priority ↓' / '> reloaded hug v0.0.2' and line 40 version corner 'hug v0.0.1' / 'hug v0.0.2'. Model: TestReloadHandsOverPlace (view, row, open task, focus, scroll, chips, search, sort), TestReloadKeepsSectionScroll. Negative control: with syscall.Exec stubbed to fail, TestTmuxReload fails 'screen never showed "reloaded hug v0.0.2"'.

check 2: 6360498 on origin/main (CI run 43947993605: ubuntu, macos, windows success). TestReloadWaitsForPromptAndEditor: with / prompt, : palette, ? help, or the editor open a new binary leaves binPending set and no reload; closing it (Esc / editorMsg) probes and reloads to v0.7.34. TestTmuxReload: palette open for 6s after the swap, screen still 'hug v0.0.1'; after Esc 'reloaded hug v0.0.2'. Confirm and grab also block (reloadBlocked = busy() || help).
