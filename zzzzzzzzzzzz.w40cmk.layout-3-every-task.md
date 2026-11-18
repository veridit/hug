---
id: w40cmk
title: "Layout 3: every task file is ID.md and never renamed; rank and group in the file"
state: todo
type: feature
priority: ""
assignee: []
labels: []
created: 2026-11-18
updated: 2026-11-18
---

## Goal
Each task lives at .hug/tasks/ID.md for its whole life, so a path or URL to it never breaks (e.g. github.com/veridit/hug/blob/hug/tasks/6ephqk.md) and git log -- ID.md is its full history. The rank prefix, the slug and the group folder leave the file name: rank and group become front-matter fields. A move edits the moved task's rank: line; it never renames. The task's own folder stays .hug/tasks/ID/. Memories move to the matching branch name: muninn-notes becomes hug/memory (9taa9d check 2). hug upgrade migrates both, preview first.

## Purpose
Jørgen 2026-11-18 01:16-01:21 local: 'we're encoding multiple things into the file name, so if you change priority or something you get a new file name and you can't link to a persistent ticket'; 'a fixed link for each ticket you can link to from anywhere'; another agent was surprised a file name could change. Measured on hug's board, last 7 days: 172 board commits renamed files, 467 renames in all, every one a rank change (rank re-spacing renames whole windows). 01:19, seeing branches hug/tasks and muninn-notes: 'we didn't go hug/ all the way?!'

## Principle
A task's name is its id, nothing else. Task files are the truth: no index file, no marker files (an order file would conflict between concurrent movers). A reorder is not an update: it changes rank: only, and leaves updated: as it was (Jørgen 01:21). No backward compatibility: hug upgrade moves every board in one commit per step, previews by default, and an older hug refuses to write (dx9vw2's layout guard, layout = 3).

## Relevant details
Front matter, grouped: identity (id, title, type, priority, labels, group, parent, rank), state (state, stage, and the stage's object e.g. asked: {what, for, check, since, by} per 41hhgn), who and how (assignee, model, branch, worktree), time (created, updated, done_at, done_release). rank keeps today's string keys and re-spacing (ordering.go rankPlan), now as field edits; rebalance edits many files' rank: lines, never names. Order on read comes from the cache (one parse per file already). Group folders (later/, design/) become group: values. Slugs and old file names stay resolvable as refs (ids, prefixes, aliases already work), so old links in notes still find the task. Memory branch: hug/memory with .hug/memory/; muninn-notes is moved by hug upgrade and deleted after. Owner: wyvern (store, cache, sync) with kestrel (upgrade, import); hatchling for the memory branch. Seid and Statbus run their own hug upgrade.

## Depends on

## Acceptance
- [ ] On a scratch clone of hug's board, hug upgrade previews the move of every task to ID.md with rank: and group: set and exits 5; --apply does it in one commit; afterwards ls, next and the TUI show the same order and groups as before (test plus the clone's numbers)
- [ ] A move (J/K, move --before, rebalance) changes only rank: lines, renames nothing, and leaves updated: unchanged; git log --follow is never needed (test)
- [ ] Group moves (mv ID GROUP) edit group: and keep the file and its ID/ folder where they are (test)
- [ ] An old file name or slug still resolves as a ref: hug task show 00i.6ephqk.task-state-stage-hard finds 6ephqk (test)
- [ ] Memories live on branch hug/memory at .hug/memory/; hug upgrade moves muninn-notes there, and hug mem search finds the same memories before and after (test, plus counts on Jørgen's store)
- [ ] Seid's board upgrades on a copy with the same order before and after (2,500 tasks, numbers in a note), before Seid's agent runs it for real
- [ ] Jørgen links to a task by its path from outside hug and the link survives a reorder

## Notes

Jørgen 01:24 local, on Statbus's tree on GitHub ('000000000001.37bryg.ladder-never….md' next to a separate '37bryg/' holding its attachment): 'assets OUTSIDE the dir?' Proposal (crab, pending his yes): one folder per task holding everything: .hug/tasks/37bryg/README.md is the task, its files sit beside it (37bryg/STATBUS-369-fresh-install-blocker.md). GitHub renders README.md under the folder listing, so github.com/<repo>/tree/hug/tasks/37bryg shows the task and its files on one page, and that URL never changes. Every task gets a folder, not only tasks with files.

Jørgen 01:25 local: (a) keep hug/tasks (and hug/memory) as real branches so GitHub links work; accept GitHub's 'recent pushes / Compare & pull request' banner (GitHub has no per-branch setting to hide it; refs/hug/* would hide it but break web links). Still open: one folder per task with README.md (crab's 01:24 proposal) vs ID.md.

Jørgen 01:25 local proposes instead: .hug/tasks/37bryg.md (the task), .hug/tasks/37bryg/ (its files), .hug/tasks/37bryg.json. Replaces crab's README.md folder proposal. Open: what 37bryg.json holds (crab reads it as the machine fields, e.g. rank and group, so a reorder touches only the .json and never the .md); awaiting his confirmation.

Jørgen 01:27 local, DECIDED layout 3: .hug/tasks/ID.md (front matter + body; GitHub renders YAML front matter as a table at the top of the page, so no .json) and .hug/tasks/ID/ for the task's files. Neither name ever changes. rank and group live in the front matter. Supersedes the plain ID.md plan (files now move from the sibling folder convention into ID/ explicitly) and crab's README.md folder proposal.
