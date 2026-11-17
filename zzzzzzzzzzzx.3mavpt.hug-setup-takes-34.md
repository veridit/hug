---
id: 3mavpt
title: "hug setup takes 34 s on Seid: it dry-runs the whole backlog import"
status: todo
type: bug
priority: "high"
assignee: []
labels: []
created: 2026-11-17
updated: 2026-11-17
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
- [ ] a test runs setup on a fixture with a 500-file backlog already imported and counts backlog files read: at most the directory listing, no file contents; fails on main
- [ ] on Seid, hug setup takes under 1 s warm (median of 3), numbers in the notes
- [ ] the commit-msg step in seid-lang's checkout reports the real state, not 'needs a Git checkout' (test or note with evidence)

## Notes
