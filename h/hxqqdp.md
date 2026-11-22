---
id: hxqqdp
title: "An old hug writing set --details adds a second Relevant details section"
state: "done"
type: bug
priority: ""
assignee: ["rooster"]
labels: []
group: ""
rank: "zzzzzzzzzzzt"
created: 2026-11-22
updated: 2026-11-22
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
done_at: "2026-11-22T13:44"
done_release: "v0.7.106"
parent: "jffwyv"
---

## Goal
A file with both '## Details' and '## Relevant details' shows both texts, lints, and hug upgrade merges them.

## Purpose
ram 2026-11-22 (6q1d1t): v0.7.101 still running (hug tui/mcp from 11-17..11-19) writing set --details to a '## Details' file adds '## Relevant details'; v0.7.104 show then prints only the Details text, hiding what the old hug wrote.

## Principle
Never hide text: if a file holds both names, show and lint it; upgrade merges them.

## Details
Old binaries cannot be fixed; the new side can: Parse sees both headings, lint reports 'two Details sections' with fix: hug upgrade, and hug upgrade merges them (old text after new). Restarting the old processes ends the risk.

## Depends on

## Acceptance
- [x] A file with both '## Details' and '## Relevant details' shows both texts and lints with a fix: line (test)
- [x] hug upgrade merges them into one '## Details', no text lost (test)

## Notes


rooster 2026-11-22: branch hxqqdp = ci/hxqqdp at 668e6984 on origin/main 58849209 (tests 18c35765, 03c0d2c4 first; fix 668e6984). Order per crab 12:17Z: the ## Details text, a blank line, then the ## Relevant details text, no marker, in show/--json/MCP/TUI and upgrade. Failing on main code: task hxqqdp_test.go:29 'Details text "New hug wrote this.", want "New hug wrote this.\n\nOld hug wrote this."'; hxqqdp_test.go:64 'MergeDetails changed nothing'; cli details_test.go:88 'show hides a Details text'; tui hxqqdp_test.go:42 'Details text at 672, Relevant details text at 474: want both, Details first'. Fix: Document.sectionText joins both (promote.go), Parse adds old-layout 'two Details sections' at the Relevant details heading (fix: hug upgrade), Document.MergeDetails + Store.MergeDetailsAll, hug upgrade step commits 'hug upgrade: two Details sections merged in N tasks', TUI sectionText Details first. Cache SchemaVersion 31 -> 32; corpus x/x0both.md added (fingerprint on main code with it: 4169124b..., with fix: 5830158d...). Mutants 11/11 killed. Local go test ./... 42 ok. CI run 45022930847 pending. Live board: no file has both headings today.
check 1: CI run 45022930847 attempt 1 on 668e6984, 8/8 jobs success. TestBothDetailsHeadingsReadAndLinted (task), TestBothDetailsHeadingsShownLintedMerged (cli: show 'details: New hug wrote this\n\nOld hug wrote this', --json the same, lint 'broken [old-layout] old layout: two Details sections ...' with 'next: hug upgrade'), TestPageShowsBothDetailsTexts (tui, Details text first); all failed on main code first
check 2: CI run 45022930847 attempt 1 on 668e6984, 8/8 jobs success. TestMergeDetailsKeepsBothTexts (one ## Details: Details text, blank line, Relevant details text; every other line as it was; second run unchanged; old block last ends the file as before) and TestBothDetailsHeadingsShownLintedMerged (hug upgrade --apply: '1 tasks with two Details sections merged', commit 'hug upgrade: two Details sections merged in 1 tasks', lint clean, second upgrade unchanged); failed on main code first
landed 18c3576 on main 2026-11-22: tests first: a file with both ## Details and ## Relevant details shows both…
landed 03c0d2c on main 2026-11-22: tests first: the old hug's Relevant details sits before ## Details; the page…
landed 668e698 on main 2026-11-22: A file with both ## Details and ## Relevant details shows both texts (Details…
shipped in v0.7.106 (18c3576)
## History
- 2026-11-22T13:10 todo -> doing/active (take)
- 2026-11-22T13:44 doing/active -> done (done, by release v0.7.106: every check holds); was doing_since: 2026-11-22T13:10
