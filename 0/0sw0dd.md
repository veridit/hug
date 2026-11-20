---
id: 0sw0dd
title: "TUI navigation, search and sort"
state: "doing"
type: feature
priority: ""
assignee: ["crab"]
labels: ["needs-discussion"]
group: ""
rank: "zzzzzzh"
created: 2026-11-16
updated: 2026-11-16
doing_since: "2026-11-18T16:37"
stage: "waiting"
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
waiting_on: "check 1 (status line keys; tab and status share one line) is 911j70's work: 911j70 c3 sits in ci/4tycwd-rebased 9163f102, not on main a74ba787; verify on main after it lands"
waiting_since: "2026-11-18T20:43"
waiting_asked_by: "iwazaru"
---

## Goal
Moving between states, searching and ordering in hug tui is quick and readable without reading help

## Purpose
The user's 2026-11-16 review: status line keys are hidden, status cannot be searched or sorted, the header lacks the project name, and memory and tasks are separate TUIs

## Principle
One line where two say the same; every key shown is a key that works; search is one syntax for free text and field:value; reuse SQLite FTS5 BM25 rather than inventing ranking

## Relevant details
Requests: (1) status counts line shows its keys (letters or matching column initials) and merges with the tab line, using a segment marker; (2) header 'HUG' plus project name (folder or hug.toml name); (3) switch between task and memory mode from the header; (4) search by field, e.g. status:done type:bug, with BM25 over title/goal/notes; (5) a sort key (sigil to decide: / is search, f is filter); (6) epics and groups visible in rows and detail. Screens: user screenshots 2026-11-16 09:36

## Depends on

## Acceptance
- [ ] status line keys visible and working; tab and status share one line
- [x] header shows HUG and the project name
- [x] status:done in / search returns done tasks
- [x] done when task:mkzcqr is done (a sort key orders the list by status, priority, updated or rank)
- [x] epic and group visible in list rows and detail

## Notes

2026-11-16 review, built (uncommitted until goldens pass): (a) board rows compact: lint mark in the gutter left of the id, type and priority as one coloured letter, legend line under the board; (b) detail: { and } (also Ctrl-N/Ctrl-P) open the next/previous task of the list it was opened from, footer 'Task 3 of 22'; (c) palette uses the full height and is a tree: filter > field > value with task counts; ': f done' and ': filter done' find the value without the field; short forms (status todo, done) still run; (d) Space in a list no longer asks to tick a check you cannot see: it says 'Enter opens ID'; the prompt in detail reads 'Tick acceptance check N of ID? "text"'. Still open from the review: choosing board columns (wide vs letters), items 1-5.

v0.7.15 released and installed 2026-11-16 (commit 6d36b9e, release run 'completed success', 'hug version' prints 'hug v0.7.15'): the review batch from the earlier note is shipped. Follow-ups are their own tasks: mv4vye (detail focus, prompts, hints), 05r3pg (context palette), kx2ndm (groups and epics).

2026-11-16 Jørgen answers the key-scheme question (item 1): the header is a tree, the same tree as the palette. Each header line is a branch with its letter marked in place, each value has its letter marked: 'view: overview list board lint', 'status: todo doing blocked done dropped', 'filter: ...'. Typing the branch letter then the value letter does it (v o = view overview, s d = status doing), and the palette spells the same path ('view overview', 'status doing'). Digits 1-4 are freed. He likes the f filter picker as it is. Today: s opens the palette prefilled 'status ', v and i both open list, o/b/! switch view, 1-4 filter status (tui.go). Open points: doing, done and dropped all start with d, so the status letters need choosing; the word is 'status' in the CLI while he said 'state'; whether one-key o/i/b/! stay beside v+letter. Pressing a branch letter should show its choices in place (no timeout), like the pending g. Build together with 05r3pg so header, palette and help (jmx10s) share one tree definition.

2026-11-16 Jørgen decides item 2 (header): 'HUG · <project>' where project defaults to the last piece of the checkout's path (the folder name, so two checkouts of one repo read differently, e.g. hug and hug-testing), with an override setting included from the start, not later. A key shows the full path of the checkout. Not built yet: the worker on mv4vye is editing the same files, so this waits for it to land.

2026-11-16 bug from Jørgen in v0.7.15: in the palette 'f g ' (short filter, short field, space) said 'No completions'. Fixed in a local commit (palette.go expandField, test TestPaletteShortFieldAfterFilter: 'ok github.com/veridit/hug/huginn/internal/tui'). Not pushed or released; ships with mv4vye.

2026-11-16 Jørgen approves the status letters t todo, d doing, b blocked, o done, r dropped, shown marked in place, and asks that letters are picked by a function, not by asking him. Built and committed locally (not wired in, not released): huginn/internal/tui/shortcuts.go assignShortcuts(names, reserved) gives each name its first letter no earlier name took. Test TestAssignShortcuts: 'ok github.com/veridit/hug/huginn/internal/tui'. It yields statuses tdbor, types fbcsd, priorities chml, filter fields stpglea (same as the f picker today), views o l b i (overview, list, board, lint), which differs from today's keys where list is i and lint is !. The header tree, the palette (05r3pg) and help (jmx10s) are to take their keys from it.

2026-11-16 Jørgen: a label written '!lint' should get '!' from the shortcut picker. Done in a local commit: assignShortcuts gives a leading sigil as the key (views overview, list, board, !lint -> o l b !); punctuation inside a name is never picked. Test TestAssignShortcuts: 'ok github.com/veridit/hug/huginn/internal/tui'. Not released.

2026-11-16 crab: back to todo, unassigned. It showed doing under Jørgen's name from a take before take required a model; nobody is building it. Waits on Jørgen's answers (key scheme, sort, field search).

crab 2026-11-17: item 4 (search by field with BM25) moves to 49n9kw (ranked search for CLI, MCP and TUI); the alias part of TUI search is vzghj8.


iwazaru 0sw0dd checks 2+3 on branch 0sw0dd d977b452 (base origin/main 56e2cc6f), pushed + ci/0sw0dd, CI run 44342320033. Failing-first ff85e140: 'list header: "HUG TASKS · list   2/2 tasks · ! 0 broken · 4 advice ..."' and '/ state:done: [], want [Alpha task Beta task]'. Check 2: render.go brand() 'HUG · <project>' in global header + breadcrumb; project = hug.toml [board] project (task.ProjectName) else filepath.Base(checkout) (Options.Checkout = s.r.Root, hooks.go); tests TestHeaderNamesProjectFromCheckout, TestHeaderProjectOverrideFromHugToml, TestHeaderWithoutProjectIsHUG. Check 3: searchfields.go parseSearch cuts state:/status:/type:/priority:/group:/label:/assignee:/epic: off the / query, words go to the index as before; TestSearchFieldStateDone. Goldens: 69 files, 110 lines, every one a header line (pairs identical after normalising HUG TASKS/HUG · repo), 80 and 140. Local: 'ok github.com/veridit/hug/huginn/internal/tui 115.624s', 'ok .../internal/task 14.233s', whole ./huginn/... ok. Untouched: sort.go, palette.go, detail.go.
CI 44342320033 on d977b452: secrets, ubuntu-latest, windows-latest, test-windows-cli, bench success; macos-latest cancelled after 15 min queued with 0 steps run (runner never started, 16:01:33-16:16:35Z); rerun of the failed job requested 16:19Z, queued behind ci/stack-80c8ac28, ec6z6j, 8zwy6r. Report: /Users/jhf/.jcode/scratch/0sw0dd/REPORT.md
crab review 17:20 local: checks 2+3 accepted. d977b452 on 56e2cc6f: header 'HUG · <project> · view' (folder, hug.toml [board] project overrides, plain HUG when unknown); / takes field:value filters (state/status/type/priority/group/label/assignee/epic), other words rank. Goldens 69 files, header lines only. CI 5/6 green, macOS never got a runner; crab runs exact-SHA CI on the landing stack. Integration note: txjkrm renames label to tag in the TUI, so search must say tag: once both land.
check 5 verified on main c84d3273 (kx2ndm a6485206 is an ancestor), main's binary in tmux, NO_COLOR, synthetic repo: task j3dje3 in group compiler with parent epic 1kcjth. List row at 80 and 140: '  ○ 1    j3dje3            compiler·↳1kcjth Grouped child'. Detail at 80 and 140: '[todo]  [feature]  group:compiler  unassigned  epic:1kcjth'. Footer legend '↳ epic'. Capture: /Users/jhf/.jcode/scratch/0sw0dd/check5-capture-main-c84d3273.txt (script /Users/jhf/.jcode/scratch/cap5.sh). Also: 4013cbae on 0sw0dd adds tag: beside label: in / search (txjkrm); TestSearchFieldStateDone failed without it ('/ tag:urgent: [], want [Alpha task]'), passes with it.
crab 17:29 local: 4013cbae adds tag: (and label:) to / search; check 5 held on main c84d3273 (kx2ndm), ticked with capture. Lands with d977b452 after the current stack.
landed 5050b97 on main 2026-11-18: tui: failing tests, header 'HUG · <project>' and state:done in / search (0sw0dd…
landed 3686f17 on main 2026-11-18: tui: header 'HUG · <project>', field terms in / search (0sw0dd checks 2, 3)
landed 31499b8 on main 2026-11-18: tui: / search takes tag: beside label: (0sw0dd, for txjkrm)
landed 8ee3737 on main 2026-11-18: tui goldens: check-ids and picture-framed say 'HUG · list' (0sw0dd header, after…
shipped in v0.7.67 (5050b97)
crab 2026-11-20 12:04Z: the wait is over: 911j70 (view line, check 3) shipped in v0.7.90. Check 1 is now Jørgen's look on release ticket mxj0zn.
## History
- 2026-11-18T16:37 todo -> doing/active (take)
- 2026-11-18T20:43 doing/active -> doing/waiting (wait)
- 2026-11-19T04:01 handed to crab: iwazaru's session ended (closed 2026-11-19 02:27)
