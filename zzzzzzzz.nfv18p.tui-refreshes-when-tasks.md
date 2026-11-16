---
id: nfv18p
title: "TUI refreshes when tasks change on disk"
status: todo
type: feature
priority: "high"
assignee: []
labels: []
created: 2026-11-16
updated: 2026-11-16
---

## Goal
An open hug tui shows changes made by agents and other terminals within a second or two, without pressing r

## Purpose
Jørgen's overview on 2026-11-16 at 11:15 showed 46 tasks and mv4vye as doing, while the board had 50 tasks and mv4vye had been done since 10:42. The TUI reads the files at start and after its own actions only. With agents working the board all day, a stale screen misleads

## Principle
Cheap and quiet: watch or poll the tasks folder, reload only when something changed, keep the selection and scroll position, never steal focus or interrupt an open prompt or editor. Say when the view was last refreshed only if it could be stale

## Relevant details
No watcher or timer exists in huginn/internal/tui today (no fsnotify, no tea.Tick). r calls m.refresh(path). Options: poll the hug/tasks branch head (git rev-parse on .hug/tasks) every 2 s, since every hug command commits; or fsnotify on the folder. Polling the head is simplest and has no new dependency. The load uses the cache (huginn/internal/cache), so an unchanged board costs little; on 2,498 tasks a reload took about 1.4 s at start, so reload off the UI thread

## Depends on

## Acceptance
- [ ] a task changed by another process appears in an open TUI within 3 seconds, selection and scroll kept
- [ ] an open prompt, palette or editor is never interrupted by a refresh
- [ ] no measurable idle load: no reload while nothing changed (test counts loads)
- [ ] tests cover a change arriving in overview, list and detail

## Notes
