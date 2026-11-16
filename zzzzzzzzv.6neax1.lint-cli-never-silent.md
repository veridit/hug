---
id: 6neax1
title: "Lint CLI: never silent, summary, codes, help"
status: "doing"
type: feature
priority: "high"
assignee: ["Jørgen H. Fjeld"]
labels: []
created: 2026-11-16
updated: 2026-11-16
---

## Goal
hug task lint always says what it found, offers a summary by code, lists the codes when --code has no value, and treats 'help' as asking for help

## Purpose
Jørgen 2026-11-16 in seid-lang: plain 'hug task lint' printed nothing at all (no count, no next line); '--summary' does not exist; '--code' alone says 'missing value for --code' instead of listing the codes; 'hug task lint help' says 'expected 0 positional arguments, got 1'; 'lint help --code' says 'missing value'. On the seid copy the same command prints 2,478 lines, so a summary is what a reader needs first

## Principle
Never silent: a clean board says 'no findings' and the next step. Large output starts from a summary (count per code, with the command to see each), then one kind at a time. Anything that looks like a request for help gets help. Errors end with a fix: line that runs

## Relevant details
Flags today: --code, --json (cli.go ~576 lint; text printed near cli.go:984). Proposed: --summary prints one line per code: count, code, plain meaning, and 'hug task lint --code CODE'; when there are more than about 30 findings, plain lint prints the summary plus the first few findings and says how to see all (--all). '--code' with no value lists the codes with counts. 'help' as a positional argument, before or after flags, prints the lint help. Ties in with pbh9xq (severity: broken vs advice), whose summary should split by severity, and rx74bp (wording, next: line). First find out why lint printed nothing in Jørgen's seid-lang checkout: run there with --json and check whether .hug/tasks exists and has tasks

## Depends on

## Acceptance
- [ ] plain hug task lint on a clean board prints 'no findings' and a next: line; it is never silent (test)
- [ ] hug task lint --summary prints one line per code with its count and the command to list it
- [ ] hug task lint --code with no value lists the codes and counts
- [ ] hug task lint help and hug task lint help --code print the lint help
- [ ] the cause of the silent run in seid-lang is found and recorded in the notes

## Notes

cause of the silent run (check 5): hug never exits 0 silently. Tested with v0.7.16 and a cd587a1 build: outside a repo -> "hug task needs a Git checkout as its working directory" + fix, exit 1; repo without .hug/tasks and empty board -> only "next: hug task new ...", exit 0 (no "no findings" line); inside .hug/tasks -> worktree error + fix; subdir, NO_COLOR/TERM=dumb, stdout closed -> normal. The seid copy prints 2,478 lines / 318,382 bytes (time: real 2,30). Run through the jcode bash tool, that output came back cut ("... (output truncated)") after ~300 lines, before the next: line; such tools withhold or truncate results of that size, so to an agent (or the person reading its tool panel) lint shows nothing. Cause: unbounded output, not a code path. Fix here: bounded default output (summary + first findings + how to see all), and "no findings" when clean.

2026-11-16 Jørgen: by default lint hides findings on done and dropped tasks, in the CLI and the TUI. Built in this ticket: open tasks only by default, one count line for what is hidden ('986 findings on closed tasks hidden: hug task lint --all'), --all shows everything. Sent to the worker.
