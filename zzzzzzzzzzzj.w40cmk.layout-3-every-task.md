---
id: w40cmk
title: "Layout 3: every task file is ID.md and never renamed; rank and group in the file"
state: "doing"
type: feature
priority: ""
assignee: ["kestrel"]
labels: []
created: 2026-11-18
updated: 2026-11-18
doing_since: "2026-11-18T01:48"
stage: "active"
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
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
- [ ] Paths follow from the id alone: every task is at .hug/tasks/C/ID.md (C = first character of the id) with its files in .hug/tasks/C/ID/; no group folders remain; hug upgrade on hug's and Statbus's boards (copies) previews it, exits 5, and --apply does it in one commit (test plus counts)
- [ ] Sync still works across the move: two clones, one upgraded and one not yet, are refused with a fix: line naming hug upgrade (never a silent merge of two layouts); after both upgrade, concurrent writes from both sync with no conflict (test)
- [ ] Every existing test passes on the new layout with no test skipped and no limit raised; tests that named old paths are rewritten, not deleted (CI green on all 5 jobs, count of rewritten tests in a note)
- [ ] Speed does not regress: ls, show, next, a write, and TUI start on a 2,500-task board are no slower than before the move (before/after numbers, measured on the same machine, in a note)

## Notes

Jørgen 01:24 local, on Statbus's tree on GitHub ('000000000001.37bryg.ladder-never….md' next to a separate '37bryg/' holding its attachment): 'assets OUTSIDE the dir?' Proposal (crab, pending his yes): one folder per task holding everything: .hug/tasks/37bryg/README.md is the task, its files sit beside it (37bryg/STATBUS-369-fresh-install-blocker.md). GitHub renders README.md under the folder listing, so github.com/<repo>/tree/hug/tasks/37bryg shows the task and its files on one page, and that URL never changes. Every task gets a folder, not only tasks with files.

Jørgen 01:25 local: (a) keep hug/tasks (and hug/memory) as real branches so GitHub links work; accept GitHub's 'recent pushes / Compare & pull request' banner (GitHub has no per-branch setting to hide it; refs/hug/* would hide it but break web links). Still open: one folder per task with README.md (crab's 01:24 proposal) vs ID.md.

Jørgen 01:25 local proposes instead: .hug/tasks/37bryg.md (the task), .hug/tasks/37bryg/ (its files), .hug/tasks/37bryg.json. Replaces crab's README.md folder proposal. Open: what 37bryg.json holds (crab reads it as the machine fields, e.g. rank and group, so a reorder touches only the .json and never the .md); awaiting his confirmation.

Jørgen 01:27 local, DECIDED layout 3: .hug/tasks/ID.md (front matter + body; GitHub renders YAML front matter as a table at the top of the page, so no .json) and .hug/tasks/ID/ for the task's files. Neither name ever changes. rank and group live in the front matter. Supersedes the plain ID.md plan (files now move from the sibling folder convention into ID/ explicitly) and crab's README.md folder proposal.

Jørgen 01:27 local: agreed, a reorder (rank or group change) leaves updated: unchanged; updated: marks real edits only.

Jørgen 01:32 local, DECIDED: shard by the first character of the id: .hug/tasks/3/37bryg.md and .hug/tasks/3/37bryg/ (its files). The path follows from the id alone (no index), never changes, and keeps folders near 330 entries at 10,000 tasks (GitHub's web listing stops at about 1,000). Group and epic (parent) stay in the front matter, as do rank and group; no folder means anything to a person. Rejected: group as folder (moves break links; a generated group id fixes renames only), creation month (uneven fill).


kestrel plan (lead, 01:58):
(1) Path resolver: task.PathFor(id) = C/ID.md, files in C/ID/. Parse reads rank: and group: from front matter when the file is named ID.md. A rank.id.slug.md file still parses, but only for upgrade and refs. The slug comes from the title, so old slugs and file names resolve as refs (check 4).
(2) Moves and groups become field edits through SetField. Nothing renames. Today nothing bumps updated: after New, so the test pins that it stays.
(3) Walkers (Store.Tasks, cache walkMarkers, IsTaskFolder, FolderOwners, taskFileIDs) learn that ID.md owns ID/.
(4) Layout = 3. A hug upgrade step moves every old-named file to C/ID.md, sets rank:/group:, moves ID/ folders and writes layout = 3, all in one commit, refusing if the board is dirty. The preview gives counts and exits 5.
(5) A write on a board with old-named files is refused with fix 'hug upgrade --apply'.
(6) Sync: before a rebase with local commits, compare the layout in hug.toml at HEAD and at the remote. If they differ, refuse with fix 'hug upgrade --apply' (check 9).
Order: 1 and 4 first, then the preview counts from a clone of hug's board in a note, then 2, 3, 5, 6 and the test rewrite. hatchling owns check 5 (hug/memory).
Check 11 BEFORE baseline (wyvern, 2026-11-18 01:53Z), current layout. Binary: hug built from origin/main 48b1151f. Board: Seid copy, 2,520 tasks (2,491 top-level task files, 819 per-task folders), ~/.jcode/scratch/w40cmk-bench/before. Harness: w40cmk-bench/bench.sh, 10 rounds, A/A (same binary both sides, alternating), GOMAXPROCS=2 nice 10, HUG_NO_SYNC=1. Load avg 16 at the time. Median ms (min-max), git processes: ls 259 (249-331) 5; show 124 (117-153) 2; next 118 (112-131) 2; note 440 (397-518) 4; TUI first frame (tmux, until a known id is drawn) 390 (353-496) 16. Noise floor: A/A median gap at most 4% (next 113 vs 118), and 5 rounds earlier gave up to 7% on means. So an after-change must beat 4-7% to count. Raw: w40cmk-bench/baseline-48b1151f.tsv; summary: baseline-48b1151f-summary.txt. After: bench.sh hug-before-48b1151f before hug-w40cmk after-copy 10 on the same machine.
Correction to the 01:53Z baseline note (wyvern): w40cmk-bench/before was an APFS copy whose .hug/tasks/.git pointed at seidcopy/code's worktree admin, so every bench run (both sides) read and wrote seidcopy/code/.hug/tasks, and q70jjy gained one note per round. The A/A numbers stand as a same-board comparison, but the board was not the one named. New self-contained copy: w40cmk-bench/board (no remote, hug/tasks at 00703755, its own worktree, 2,520 tasks). I'll redo the before numbers on board/ next to the after run, in the same paired session.
## History
- 2026-11-18T01:48 todo -> doing/active (take)
- 2026-11-18T02:26 doing/active -> doing/waiting (wait)
- 2026-11-18T04:23 doing/waiting -> doing/active (resume); was waiting_asked_by: kestrel; waiting_on: parked for 3nhp67 (crab 02:26, high); WIP on kestrel/w40cmk (resolver + Parse), resume after; waiting_since: 2026-11-18T02:26
- 2026-11-18T04:34 doing/active -> doing/waiting (wait)
- 2026-11-18T04:42 doing/waiting -> doing/active (resume); was waiting_asked_by: kestrel; waiting_on: ygay9c first (crab 03:33: same gitbranch code, small fix lands on its own); waiting_since: 2026-11-18T04:34
