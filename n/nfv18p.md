---
id: nfv18p
title: "TUI refreshes when tasks change on disk"
state: "done"
type: feature
priority: "high"
assignee: ["poodle"]
labels: []
group: ""
rank: "zzzzzzzz"
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
- [x] a task changed by another process appears in an open TUI within 3 seconds, selection and scroll kept
- [x] an open prompt, palette or editor is never interrupted by a refresh
- [x] no measurable idle load: no reload while nothing changed (test counts loads)
- [x] tests cover a change arriving in overview, list and detail

## Notes

2026-11-16 Jørgen approves: 'auto-refresh, of course, yes'. Raised to high. Queued after rx74bp (the worker is editing the same TUI files).

probe: does a note move the ref file

2026-11-16 Jørgen asks how to coordinate without polling. Design: watch, do not poll. Every hug task command commits to branch hug/tasks, and that rewrites one small file, <git-common-dir>/refs/heads/hug/tasks (measured: its mtime moved from 1794827894 to 1794827909 on one 'hug task note'). The TUI asks the OS to notify it about that file's folder (fsnotify v1.10.1 is already in go.mod) and reloads when the notification arrives, so idle cost is zero. Details: watch the folder, not the file, because git replaces the file by rename; also watch packed-refs in the git dir (git gc can move the ref there) and the tasks folder itself for hand edits that are not committed; wait about 200 ms after an event so a burst of commits gives one reload; reload off the UI thread, keep selection and scroll, never interrupt a prompt or editor. If a watch cannot be set up (network drive, some containers), fall back to checking the ref's mtime every few seconds. Later the same notification can drive hug mcp and other open views.

2026-11-16 Jørgen asks how the SQLite copy is kept in step. As read from the code: the Markdown files on branch hug/tasks are the truth; .git/huginn/cache.db is a disposable copy (cache.go: 'disposable, incrementally synchronized SQLite mirror'). Nothing syncs it in the background. Every read goes through Store.Refresh -> cache.Sync (cli.go 146-150): it lists the .md files with their size and modification time, asks git which files changed since the last seen head, and if nothing differs returns at once; otherwise it hashes and re-parses only the changed files and updates their rows. A SchemaVersion change rebuilds everything once. The TUI uses the same path (tui.go:84 m.store.Tasks()). So auto-refresh needs no separate sync: on a watch notification the TUI calls the same refresh, which re-parses only the changed files.

check 1 (198572b): real binary in tmux, TestTmuxAutoRefresh: another process ran hug task set and hug task note; the open TUI showed them with no key pressed: "list: shown 8.87175ms after hug task set returned, no key pressed", "detail: shown 144.479459ms after hug task note returned, no key pressed", and the open detail stayed on Beta. In-process (TestChangeOnDiskAppears): "change on screen 292.901416ms after the commit" (overview), 294.8ms (list), 295.7ms (detail); selection kept by task id (a move elsewhere renames the file), detail scroll kept at a nonzero offset (ctrl+d first, test fails if it did not scroll), TestChangeOnDiskKeepsListScroll keeps the list window after G k k. Design as decided: huginn/internal/tui/watch.go watches the folder of <git-common-dir>/refs/heads/<branch>, the git common dir (packed-refs only) and the tasks folder and its group folders (.md files, dot files ignored, chmod ignored); 200 ms debounce, 1 s cap under a steady stream; load runs off the UI thread (tea.Cmd loadSnapshot; measured 0.45-0.54 s per load on the 2,498-task seid board); a generation counter drops a load older than an action refresh (TestOlderLoadNeverOverwritesNewer).

check 2 (198572b): busy() = prompt (search, palette, jump, new title, filter picker) or confirm open or the editor running (m.editing set in runEditor, cleared by editorMsg); a change then only marks stale and the reload runs when it closes. TestChangeOnDiskWaitsForPrompt: search prompt "/Al", palette ":status ", confirm "d": after the change signal prompt, confirm and typed text are unchanged, loads unchanged, then Esc -> the change shows; editor: no load while editing, editorMsg -> shown. tmux: typed "/ Alp", another process renamed Alpha, 1.5 s later "/ Alp" still on screen and the rename not shown; after Esc "Alpha renamed while typing" shown.

check 3 (198572b): TestNoReloadWhileIdleOneForABurst: loads counted (loadCounter in loadSnapshot); after j k Enter Esc, no change signal and 0 loads in 1.5 s; 5 commits in quick succession give 1 load (test allows 2 if the burst straddles the debounce) and the last one is shown. Real TUI on the seid copy (2,498 tasks) idle in tmux: ps time 0:00.73 -> 0:00.77 over 15 s (0.04 s CPU), no reload. Fallback: TestChangeOnDiskPollsWithoutWatcher (newWatcher fails -> mode "poll", stat of the ref + packed-refs every pollEvery=2 s, no signal while unchanged, change shown after a commit).

check 4 (198572b): TestChangeOnDiskAppears/overview, /list, /detail all PASS, plus TestTmuxAutoRefresh (list and detail, real binary, real hug task set / hug task note in another process). go test -v: "--- PASS: TestChangeOnDiskAppears (1.47s)", "--- PASS: TestChangeOnDiskKeepsListScroll (1.00s)", "--- PASS: TestChangeOnDiskWaitsForPrompt (1.97s)", "--- PASS: TestNoReloadWhileIdleOneForABurst (2.85s)", "--- PASS: TestChangeOnDiskPollsWithoutWatcher (0.93s)", "--- PASS: TestOlderLoadNeverOverwritesNewer (0.15s)", "--- PASS: TestTmuxAutoRefresh (8.97s)". Full suite GOMAXPROCS=2 GOFLAGS=-p=2 nice -n 19 go test -count=1 ./huginn/... ./cmd/...: 11 ok, 0 FAIL, last "ok  	github.com/veridit/hug/cmd/hug	0.394s". Help hint r now "refresh now (changes on disk show by themselves)"; only that golden line changed.
