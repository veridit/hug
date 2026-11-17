---
id: 3mavpt
title: "hug setup takes 34 s on Seid: it dry-runs the whole backlog import"
state: "doing"
type: bug
priority: "high"
assignee: ["kestrel"]
labels: []
created: 2026-11-17
updated: 2026-11-17
doing_since: "2026-11-17T20:17"
stage: "active"
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
---

## Goal
hug setup answers in under 1 s on Seid (7,658 .backlog files, 2,520 tasks): the backlog step decides 'imported or not' from what is cheap and certain (the import recorded in hug.toml, origins already on the board) and never converts every file to say so. Any slow step shows that it is working.

## Purpose
Jørgen 2026-11-17 19:46 local: 'hug setup ... it runs very slow' (he pressed ^C first). crab measured 33.7 s on v0.7.44 at load 18. CPU profile (main 9f3e38f+): runSetup 32.3 s, of which importer.Run 28.3 s (convert 17.8 s, os.ReadFile 7.0 s, task.Parse 2.8 s; Parse -> FileRefs regexp 19.2 s overall). git: 18 calls, under 1 s. Source: huginn/internal/cli/setup.go:131 runs importer.Run(..., DryRun: true) on every setup.

## Principle
setup only reads, and reads only what it needs. Fast, never at the cost of correctness: if a cheap answer is not certain, say 'run the dry run to see' instead of computing it. A test counts files read by setup on a fixture with a backlog folder and fails on main.

## Relevant details
Cheap signals: hug.toml's recorded import (ggydj2), and the board's origin: fields (imported tasks carry their source path), compared with the backlog's task file names and mtimes. Also check FileRefs' regexp: 19 s for ~10,000 parses suggests backtracking; one pass with a cheaper scanner would help import too. Seid's output also showed 'Model: trailer gate: the commit-msg gate needs a Git checkout as its working directory' while run in seid-lang's own checkout: check that step too.

## Depends on

## Acceptance
- [x] a test runs setup on a fixture with a 500-file backlog already imported and counts backlog files read: at most the directory listing, no file contents; fails on main
- [ ] on Seid, hug setup takes under 1 s warm (median of 3), numbers in the notes
- [x] the commit-msg step in seid-lang's checkout reports the real state, not 'needs a Git checkout' (test or note with evidence)

## Notes


Measured (kestrel, 2026-11-17 20:20 CET, v0.7.45, load ~12-30), read-only on /Users/jhf/Veridit/src/seid-lang.
- hug setup: real 35,57 user 33,31. Profile of runSetup (main a64fcc3): 27.38 s cum, of which importer.Run 26.57 s: task.Parse 18.96 s, FileRefs 17.29 s (regexp backtrack + NFA), convert 16.33 s, Document.SetField->replace 15.42 s (each SetField reparses the whole file, so FileRefs runs once per field), os.ReadFile 7.18 s. ImportFiles (evidence step) 0.77 s.
- Dry run: added 0, updated 282, unchanged 2212, skipped 183. Of those 282, 0 have a source file newer than the last import commit on hug/tasks touching .backlog-import (e8e16413b, ct 1794783932). They are converter drift, not backlog edits: baseline 812nm2.json has status: and import_type_inference:, while the local file has state: (the 6ephqk rename).
- File names: 2,677 backlog .md files. Name prefix before ' - ' == frontmatter id in 2,677 of 2,677 (1 file has no id). 2,490 distinct name ids, 2,483 board origins, 13 name ids without an origin (TASK-1095, TASK-1946.05, TASK-1958 ...). The dry run adds 0, so these are linked by alias, baseline or a qualified id@folder.
- Check 3 reproduced: hug setup, then SIGINT after 4 s. It printed the whole checklist with '[ ] import backlog-md: .backlog/ exists. Import it...' and '[ ] Model: trailer gate: the commit-msg gate needs a Git checkout as its working directory', exit 0. Cause: the cancelled ctx kills exec.CommandContext in findHooks, and the error reads as 'not a checkout'.

Design:
1. Setup's backlog step reads the directory listing and lstat only, never file contents. 'Imported' means: every listed id (file-name prefix) has an origin, alias or baseline on the board (the cached task list gives aliases), and no listed file is newer than the last import commit (one git log -1). If a name gives no id, an id is not on the board, or a file is newer, setup says so with counts and the first ids, plus 'hug task import backlog DIR --dry-run' to see the exact diff. Never a guessed number. retire is offered only when the cheap answer is complete, and --cleanup keeps its full dry-run gate (cleanup.go:154), so retiring stays exactly as safe as before.
2. Converter drift (re-rendering already imported tasks) is not 'the import is incomplete'. setup does not report it. The dry run and import still do.
3. Interrupt: runSetup returns the ctx error when ctx is cancelled (no checklist from half-run steps). findHooks reports ctx.Err() as an interruption, not as 'needs a Git checkout'.
4. FileRefs: a literal prefilter (bytes.Contains 'file:', 'task:', '.hug/tasks/') before each regexp, and SetField chains in convert stop reparsing per field if cheap. Measured before and after on Seid's 2,677 sources.
Test (check 1): a 500-file backlog, imported, then every source file chmod 000 (skipped on Windows and as root). setup must report the import complete. On main the dry run hits EACCES and the step stays open.
Implementation (branch kestrel/3mavpt, not landed yet). The design changed from the 19:35 note after measuring:
- 'imported' is not decidable from names and origins alone. Of Seid's 282 'to update', all 282 diffs add 'parent:' (a converter change since the import), and 2 of the 13 ids without an origin are files the dry run skips (broken YAML: TASK-1095 'mapping key labels already defined', TASK-1958 'did not find expected key').
- So setup remembers the dry run's own answer instead, keyed on the hug binary and the backlog listing (name, type, size, mtime: lstat only), plus the board tasks it matched still being on the board. Both import and dry run write it, to <git-common-dir>/huginn/setup-backlog.json.
- The evidence step is remembered per folder, keyed on its listing and the board commit (porcelain v2, clean board only).
- FileRefs runs a regexp only when its literal is present: 2519923 -> 26059 ns/op on a 22 KB note with no citations (BenchmarkFileRefsNoRefs). TestFileRefsPrefilterSameAnswer compares it against the old function.
Seid copy (scratch, Seid board tree c528765d0, Seid .backlog), hug setup 'real':
- main: 34.92 / 33.26 / 33.22 / 33.31
- branch: cold 9.55; warm 0.54 / 0.53 / 0.51 (median 0.53).
Failing first on main (3 of 3):
- TestSetupInterruptedSaysSo: 'cancelled before start: hug task setup needs a Git checkout as its working directory'
- TestSetupReadsNoBacklogFileAfterImport: 'setup read the backlog: {Step:import backlog-md Done:false Detail:.backlog/ exists. Import it; ...}'
- TestSetupComparesAgainWhenBacklogOrBoardChanges: 'edited source: ... 0 to add, 1 to update, 499 in sync ... stderr: ' (no comparing line)
A -race run caught one bug of mine: tasks the import had added were not in the matched list, so deleting one left setup answering from memory. Fixed: importedReport marks them matched.
Landed on main 9c9d4c4..e971d24. -race clean on cli, task and importer (cli 163.572s). Check 1: TestSetupReadsNoBacklogFileAfterImport (500 files, chmod 000 after import; setup reports '0 to add, 0 to update, 500 in sync' Remembered, with no comparing line). Check 3: TestSetupInterruptedSaysSo; setup returns 'hug task setup was interrupted (context canceled); it changed nothing' with fix 'hug task setup', both before start and in the middle of the dry run. Check 2 so far: warm 0.54/0.53/0.51 s on a copy of Seid (board tree c528765d0 plus Seid's .backlog). Ticking it needs the real seid-lang checkout with a hug that has e971d24: the first setup there is cold (about 9.5 s), then a warm median of 3.
## History
- 2026-11-17T20:17 todo -> doing/active (take)
