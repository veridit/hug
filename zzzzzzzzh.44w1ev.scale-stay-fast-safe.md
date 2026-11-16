---
id: 44w1ev
title: "Scale: stay fast and safe at 10,000 tasks"
status: "doing"
type: chore
priority: "medium"
assignee: ["humpback"]
labels: []
created: 2026-11-16
updated: 2026-11-16
model: "claude-opus-5-5"
---

## Goal
hug task commands and the TUI stay responsive and never fail on a board of 10,000 tasks, and a release cannot quietly make that worse

## Purpose
Jørgen 2026-11-16: if we launch and people use it for 10,000 tasks, it must not crash. Measured on v0.7.16 with 10,002 synthetic tasks (39 MB of Markdown, machine under load, niced): nothing failed. ls cold (builds the cache) 5.4 s / 187 MB; ls warm 1.4 to 1.6 s / 151 MB; ls --jsonl 1.5 s; next 0.6 s; show 0.7 s; board 0.65 s; lint 3.0 s; note and new (write plus commit) 1.0 s each; ls --epic 1.0 s; cache.db 14.7 MB; TUI first screen 2.4 s, 213 MB, list view and jump to the last row 0.2 s

## Principle
Measure, then fix the largest cost; no speculative rewrites. Costs that grow with the board (reading every task to answer about one) are the ones to remove. Guard with a test so a regression fails CI, as ChildIndex did for ls (19 s to 0.8 s on 2,498 tasks)

## Relevant details
Where the time goes, from the code: every command runs cache.Sync, which walks and stats every .md file (10,002 stats), then List loads every task as JSON from SQLite into memory, even for show/next/note on one task. That is why show costs 0.7 s and 80 MB. Ideas in order of value: (1) skip the walk when the hug/tasks head and the worktree are unchanged since the last sync (one git call instead of 10,002 stats); (2) let single-task commands read one row; (3) ls prints from columns instead of decoding full tasks; (4) lint at 3.0 s re-checks everything: cache findings per file (already stored) and recompute only cross-task checks; (5) TUI: 213 MB is fine, but check the detail view and the palette's value counts (valueCount re-filters per row). Not measured yet: 50,000 tasks, Windows, a cold disk, many small commits in the tasks branch (10,000 commits of history), and import of 10,000. The generator used for this run is in the task notes

## Depends on

## Acceptance
- [ ] a benchmark or test builds a 10,000-task board and fails if ls, next or show exceed a stated budget
- [ ] show, next and note on one task do not load every task (measured time and memory drop, numbers in the notes)
- [ ] an unchanged board skips the per-file walk (measured)
- [ ] the TUI opens a 10,000-task board and filters without a visible stall; numbers in the notes
- [ ] 50,000 tasks measured once and the result recorded, even if not optimised

## Notes

file:gen-scale-board.py (2949 bytes, sha256:654365fa)

2026-11-16 Jørgen sets the target: every interaction under 400 ms, and no command does all the work every time. Baseline on v0.7.16 (niced, busy machine), time / peak memory. 53 tasks: process start (hug version) 0.02 s; ls 0.18 s; next 0.17 s; show 0.23 s; board 0.14 s; lint 0.18 s. 2,499 tasks (seid copy): ls 0.48 s / 70 MB; next 0.30 s; show 0.35 s; board 0.30 s; lint 1.84 s. 10,002 tasks: ls 1.4-1.6 s / 151 MB; next 0.56 s; show 0.74 s; board 0.65 s; lint 3.0 s; note and new 1.0 s; TUI first screen 2.4 s / 213 MB. Reading: there is a fixed cost of about 0.15 s even on 53 tasks (process start is only 0.02 s, so it is the git calls and the cache open: measure and trim), plus a cost that grows with the board. Over 400 ms today: ls and lint from about 2,500 tasks, and everything at 10,000. The scratch board is deleted; the generator is attached to this task (gen-scale-board.py, deterministic, 'python3 gen-scale-board.py 10000' inside a .hug/tasks worktree, then commit).

2026-11-16 plan agreed with Jørgen. The cache already stores per file mtime_ns, size, sha256 and per-task findings (huginn/internal/cache/cache.go), and Sync already skips unchanged files. What still costs time at 10,000 tasks: (1) every command walks and stats every file (WalkDir) even when nothing changed; (2) every command loads every task row and decodes its JSON payload into memory, then filters in Go; (3) cross-task findings (missing dependency, cycle, parent depth, epic state) are recomputed over the whole board; (4) process start and SQLite open. Plan: (a) fast path: compare the hug/tasks ref (and the worktree's index mtime for uncommitted edits) with the value stored in meta; if unchanged, skip the walk entirely; if changed, use git diff --name-only between the stored and current head, falling back to the stat walk only for dirty files; (b) answer ls/next/show/lint --summary with SQL queries over indexed columns instead of loading every payload; (c) store all findings, per task and cross-task, in a findings table; recompute cross-task findings only for tasks whose dependencies, parent or children changed; lint and the TUI read the table; (d) measure each step with the attached generator (10,002 tasks) and seid (2,499): target every command and every TUI interaction under 400 ms, and an unchanged board under 100 ms. Ship step by step, each measured.

step 1 (read the board once per command, d4ae7e7 on ci/44w1ev-1): python3 bench44.py BIN board10k 5 (10,000 generated tasks, every run nice -n 19, interleaved base/after x2, load 51-86), medians before (main e5cb389) -> after: ls 1.621s/1.635s -> 1.405s/1.407s; next 0.584s/0.594s -> 0.336s/0.342s; show 0.736s/0.679s -> 0.331s/0.334s; board 0.648s/0.577s -> 0.419s/0.425s; lint 3.334s/3.023s -> 2.581s/2.563s; note 1.010s/0.973s -> 0.603s/0.593s. Cause: Store.Tasks synced (walk+stat 10k files) on every call, six per ls. Left: ls renders notes by reading every file (52% of ls CPU), lint's LintFiles lstats every path component (56% of lint CPU), Open decodes 10k JSON payloads, 13 git spawns per command (0.20 s on a 1-task board).

step 2 (index dependency lookups, LintFiles lists group folders once; e3d448e on ci/44w1ev-2): python3 bench44.py BIN board10k 5 ls,next,show,board,lint (10,000 tasks, nice -n 19, interleaved x2), medians step1 -> step2: lint 2.603s/2.699s -> 2.181s/2.172s; board 0.420s/0.425s -> 0.354s/0.347s; ls 1.350s/1.482s -> 1.196s/1.142s; next 0.331s/0.343s -> 0.349s/0.335s and show 0.336s/0.340s -> 0.346s/0.332s unchanged. Ruling from crab 19:41: always walk, never skip the walk on cheap signals; racy-clean hashing within 2 s of the last sync.

step 2 landed on main e3d448e, CI https://github.com/veridit/hug/actions/runs/43942324812 green ubuntu/macos/windows. step 3 (cache opens from one gob blob; racy-clean: a file modified within 2 s of the last sync is hashed; 3cb724e on ci/44w1ev-3): python3 bench44.py BIN board10k 5 (nice -n 19, interleaved x2), medians step2 -> step3: next 0.331s/0.334s -> 0.230s/0.255s; show 0.339s/0.333s -> 0.216s/0.242s; board 0.337s/0.331s -> 0.217s/0.242s; ls 1.070s/1.074s -> 1.056s/1.162s; lint 2.105s/2.182s -> 1.970s/2.131s; note 0.737s/0.773s -> 0.782s/0.802s (the blob rewrite on each write). seid copy (2,508 tasks, rsync of ~/Veridit/src/seid-lang/.hug/tasks into a scratch repo, read-only source), main e5cb389 -> step 3: ls 0.644s/0.670s -> 0.467s/0.468s; next 0.325s/0.310s -> 0.160s/0.161s; show 0.368s/0.350s -> 0.158s/0.159s; board 0.304s/0.294s -> 0.161s/0.162s; lint 1.747s/1.842s -> 1.507s/1.513s.

step 3 revised per crab (writes touch only their own tasks: board_delta; ls takes notes/description texts from the cached parse, SchemaVersion 9; racy-clean 2 s window): 670af8e on ci/44w1ev-3, run 43945497018. python3 bench44.py BIN board10k 5 ls,next,show,board,lint,note (10,000 tasks, nice -n 19, interleaved x2), medians step2 -> step3: ls 1.063s/1.103s -> 0.451s/0.447s; next 0.336s/0.331s -> 0.259s/0.239s; show 0.330s/0.331s -> 0.243s/0.239s; board 0.334s/0.332s -> 0.254s/0.243s; lint 2.272s/2.125s -> 2.037s/2.025s; note 0.601s/0.608s -> 0.512s/0.506s. Tests: TestHandEditedNotesSeenByLsAndShow and TestRacySameSizeEditSameTick fail with the racy rule disabled, pass with it.

step 4 (lint takes file references from the cached parse; reads files only when forbid_cite is set): 28d398a, CI https://github.com/veridit/hug/actions/runs/43947020962 success. python3 bench44.py BIN board10k 5 (nice -n 19, interleaved x2), medians step3 -> step4: lint 2.027s/1.982s -> 0.356s/0.354s; lint --summary 2.017s/2.007s -> 0.354s/0.356s; others unchanged. seid copy lint 1.501s -> 0.219s. Test: TestLintSeesHandAddedFileRef.
