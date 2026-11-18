---
id: q0a8rx
title: "Public hug/memory: keep or rewrite without the 58 misplaced memories"
state: "doing"
type: chore
priority: ""
assignee: ["mizaru"]
labels: []
group: ""
rank: "zzzzzzzzzzzx"
created: 2026-11-18
updated: 2026-11-18
stage: "active"
model: "claude-opus-5-5"
effort: ""
branch: "txjkrm5"
worktree: "/Users/jhf/.jcode/scratch/txjkrm5"
doing_since: "2026-11-18T20:44"
---

## Goal
The 58 memories that don't belong on hug's public hug/memory branch are first moved to their right scope in the local store (personal, the other project or client, or global for team rules), then the public branch is rewritten without them, through all its history, and force-pushed once (Jørgen 19:42Z: 'Rewrite, but maybe we should first move the memories to the right spot')

## Purpose
A pushed memory can't be taken back by forget or rescope: git history and other clones keep it. Only a rewrite of the public branch removes it, and that's Jørgen's call.

## Principle
No rewrite without Jørgen's yes. A backup bundle exists. hug's repo stays local after 34r1ft, so nothing new is pushed meanwhile.

## Relevant details
Labels: /Users/jhf/.jcode/scratch/gav9nw (measure.json corrected_counts: personal 29, other 16, team 13; hand90.json holds the hand verdicts). Move tool: hug mem rescope --from FILE (lines '<memory-id> <scope>', --dry-run first, one write, one reindex; --cross-client needs admin approval). For the 16 'other', the target project comes from hug mem tidy place's catalog, or by reading each line. Team rules go to global (OPTIONS: until a team scope exists). Backup bundle of the branch exists. Rewrite after 34r1ft lands, so nothing re-pushes them.

## Depends on

## Acceptance
- [x] Jørgen answers: rewrite or leave
- [ ] If rewrite: the branch is rewritten from the backup minus the 58, force-pushed once, and ls-remote and a fresh clone show none of them (evidence in a note)
- [x] A moves file lists all 58 with their target scope and a one-line reason; rescope --dry-run passes, then rescope writes, and hug mem search finds each one in its new scope (evidence in a note)
- [ ] The rewritten branch holds none of the 58 in any commit (git log -p | grep for each id finds nothing), and the 810 that belong are byte-identical to before in the tip's memories.jsonl
- [ ] Force-pushed once after 34r1ft is on main; ls-remote shows the new tip; a fresh clone's hug mem sync imports none of the 58

## Notes


check 1: Jørgen 19:42Z: rewrite, after moving the memories to the right place first.
check 3 (mizaru 19:51Z): moves file /Users/jhf/.jcode/scratch/q0a8rx/moves.tsv, 58 lines, each with its label, full id and a one-line reason (the58.json holds content and targets). Final split per crab's ruling B (19:49Z): personal 45 (29 personal + 16 other: 8 machine state; 8 about Seid/Statbus, which are never written into Seid's or Statbus's scope because both sync a hug/memory branch and are observe-only), global 13 (team rules). Dry-run 1 needed --cross-client for the 8 (dryrun1.txt: 'nothing moved, 8 of 58 items failed checks'); after B, dryrun2.txt: 'would rescope 45 to personal / 13 to global'. Real: rescope.txt 'rescoped 58 memories', backup ~/Library/Application Support/muninn/tidy/rescope-20261118T194951Z-4179401420 (+ my copy memories.json.before-rescope, sha256 cf0c2d97). DB: 56 active at target + 2 at target and inactive (cap_32ec68 w1x6w0, cap_6e05d2 rt2hky: inactive before the move, and inactive on the branch tip too). hug mem search --all finds 56/56 active in their new scope (search-check.txt; the 2 inactive are not searchable, and hug mem show --all shows both). hug's project scope now holds 827 memories.
check 4 prepared locally, not pushed (mizaru 19:51Z): scratch bare clone /Users/jhf/.jcode/scratch/q0a8rx/mem.git, ref hug/memory-rewritten a9ba4377, built by rewrite.py from origin hug/memory 46d7089f. Note the tip moved past the 1abcc497 backup: 4 more commits, 17 new memories, all repo-scope, none of the 58. 50 commits -> 50 commits, same authors, dates and messages, linear; map in rewrite-map.txt (removed 41 lines in older commits, 58 from 1abcc497 on). git log -p of the rewritten ref: 0 of the 58 ids and 0 of their texts (first 80 chars) appear. Tip: 885 -> 827 lines, equal to old tip minus the 58; the 810 from the labeled snapshot are byte-identical at the new tip. Bundles: hug-memory-before-46d7089f.bundle (the true before-image) and hug-memory-rewritten-a9ba4377.bundle. origin and the local repo still at 46d7089f. Push waits for crab's go after 34r1ft.
## History
- 2026-11-18T20:41 todo -> doing/active (take)
- 2026-11-18T20:41 doing/active -> doing/asked (ask)
- 2026-11-18T20:42 answered asked (check 1), by jhf, for crab (doing/asked -> doing/active):
  > hug's public hug/memory branch still holds 58 memories that don't belong there (29 personal, 16 other projects, 13 team). Forget can't remove them; only rewriting the branch can, and clones that already fetched it keep their copies. A backup exists. Rewrite the branch without them, or leave it?
  Rewrite, but maybe we should first move the memories to the right spot. We have a functionality for that, right?
- 2026-11-18T20:42 answer 1 delivered to crab (session_crab_1793783281662_44ffaee927a8a9d5)
- 2026-11-18T20:42 answer 1 acknowledged by crab
- 2026-11-18T20:43 crab read the answer: hug task set q0a8rx
- 2026-11-18T20:43 doing/active -> todo/ready (give back); was doing_since: 2026-11-18T20:41
- 2026-11-18T20:44 todo/ready -> doing/active (take)
