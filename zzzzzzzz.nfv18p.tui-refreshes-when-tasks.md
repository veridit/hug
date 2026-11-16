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

2026-11-16 Jørgen approves: 'auto-refresh, of course, yes'. Raised to high. Queued after rx74bp (the worker is editing the same TUI files).

probe: does a note move the ref file

2026-11-16 Jørgen asks how to coordinate without polling. Design: watch, do not poll. Every hug task command commits to branch hug/tasks, and that rewrites one small file, <git-common-dir>/refs/heads/hug/tasks (measured: its mtime moved from 1794827894 to 1794827909 on one 'hug task note'). The TUI asks the OS to notify it about that file's folder (fsnotify v1.10.1 is already in go.mod) and reloads when the notification arrives, so idle cost is zero. Details: watch the folder, not the file, because git replaces the file by rename; also watch packed-refs in the git dir (git gc can move the ref there) and the tasks folder itself for hand edits that are not committed; wait about 200 ms after an event so a burst of commits gives one reload; reload off the UI thread, keep selection and scroll, never interrupt a prompt or editor. If a watch cannot be set up (network drive, some containers), fall back to checking the ref's mtime every few seconds. Later the same notification can drive hug mcp and other open views.
