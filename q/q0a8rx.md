---
id: q0a8rx
title: "Public hug/memory: keep or rewrite without the 58 misplaced memories"
state: "done"
type: chore
priority: ""
assignee: ["mizaru"]
labels: []
group: ""
rank: "zzzzzzzzzzr3"
created: 2026-11-18
updated: 2026-11-18
model: "claude-opus-5-5"
effort: ""
branch: "txjkrm5"
worktree: "/Users/jhf/.jcode/scratch/txjkrm5"
done_at: "2026-11-18T21:13"
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
- [x] If rewrite: the branch is rewritten from the backup minus the 58, force-pushed once, and ls-remote and a fresh clone show none of them (evidence in a note)
- [x] A moves file lists all 58 with their target scope and a one-line reason; rescope --dry-run passes, then rescope writes, and hug mem search finds each one in its new scope (evidence in a note)
- [x] The rewritten branch holds none of the 58 in any commit (git log -p | grep for each id finds nothing), and the 810 that belong are byte-identical to before in the tip's memories.jsonl
- [x] Force-pushed once after 34r1ft is on main; ls-remote shows the new tip; a fresh clone's hug mem sync imports none of the 58

## Notes


check 1: Jørgen 19:42Z: rewrite, after moving the memories to the right place first.
check 3 (mizaru 19:51Z): moves file /Users/jhf/.jcode/scratch/q0a8rx/moves.tsv, 58 lines, each with its label, full id and a one-line reason (the58.json holds content and targets). Final split per crab's ruling B (19:49Z): personal 45 (29 personal + 16 other: 8 machine state; 8 about Seid/Statbus, which are never written into Seid's or Statbus's scope because both sync a hug/memory branch and are observe-only), global 13 (team rules). Dry-run 1 needed --cross-client for the 8 (dryrun1.txt: 'nothing moved, 8 of 58 items failed checks'); after B, dryrun2.txt: 'would rescope 45 to personal / 13 to global'. Real: rescope.txt 'rescoped 58 memories', backup ~/Library/Application Support/muninn/tidy/rescope-20261118T194951Z-4179401420 (+ my copy memories.json.before-rescope, sha256 cf0c2d97). DB: 56 active at target + 2 at target and inactive (cap_32ec68 w1x6w0, cap_6e05d2 rt2hky: inactive before the move, and inactive on the branch tip too). hug mem search --all finds 56/56 active in their new scope (search-check.txt; the 2 inactive are not searchable, and hug mem show --all shows both). hug's project scope now holds 827 memories.
check 4 prepared locally, not pushed (mizaru 19:51Z): scratch bare clone /Users/jhf/.jcode/scratch/q0a8rx/mem.git, ref hug/memory-rewritten a9ba4377, built by rewrite.py from origin hug/memory 46d7089f. Note the tip moved past the 1abcc497 backup: 4 more commits, 17 new memories, all repo-scope, none of the 58. 50 commits -> 50 commits, same authors, dates and messages, linear; map in rewrite-map.txt (removed 41 lines in older commits, 58 from 1abcc497 on). git log -p of the rewritten ref: 0 of the 58 ids and 0 of their texts (first 80 chars) appear. Tip: 885 -> 827 lines, equal to old tip minus the 58; the 810 from the labeled snapshot are byte-identical at the new tip. Bundles: hug-memory-before-46d7089f.bundle (the true before-image) and hug-memory-rewritten-a9ba4377.bundle. origin and the local repo still at 46d7089f. Push waits for crab's go after 34r1ft.
crab 21:11 local: ~/Projects/hug/.hug/memory was still on the old 46d7089f (885 lines, all 58); a sync would have re-imported 51 and conflicted. Reset to origin a9ba4377 (worktree was clean; 46d7089f is in q0a8rx/hug-memory-before-46d7089f.bundle). Dry-run after: in added=0, unchanged=827, file changed=false. For Jørgen: any other machine with a hug clone needs, before its next hug mem sync: git -C <clone> fetch origin hug/memory && git -C <clone>/.hug/memory reset --hard origin/hug/memory. v0.7.70 (memshare) keeps hug local, closing the push side.
checks 2, 4, 5 (mizaru 20:07Z, crab's go 20:06Z after 34r1ft on main 8dc479b2): fresh fetch git@github.com:veridit/hug.git hug/memory = 46d7089f (unchanged since the rebuild), rewrite.py rerun -> identical a9ba4377, 50 -> 50 commits, git log -p: 0 of the 58 ids, 0 of their texts. Push: git push --force-with-lease=refs/heads/hug/memory:46d7089f ... a9ba4377:refs/heads/hug/memory -> '+ 46d7089f...a9ba4377 a9ba43773dd776e4963220c61c3bba18e381c0bb -> hug/memory (forced update)' (q0a8rx/push.txt). ls-remote: 'a9ba43773dd776e4963220c61c3bba18e381c0bb refs/heads/hug/memory' (ls-remote-after.txt). Tip: 885 -> 827 lines, equal to old tip minus the 58; the 810 byte-identical. Fresh clone q0a8rx/fresh/clone: origin/hug/memory a9ba4377, log -p 0 of 58. hug v0.7.69 'hug mem sync' there with an isolated store: 'in: added=827 updated=0 ... out: 827 memories, 0 withheld' (fresh/sync.txt); imported store holds 0 of the 58 by id and 0 by text; ls-remote after the sync still a9ba4377. Before-image: q0a8rx/hug-memory-before-46d7089f.bundle (verified).
Clone scan (mizaru 20:12Z, read-only, scratch + ~/Projects + ~/Veridit, script q0a8rx/scan.sh, output scan.txt). Pre-rewrite history on a local memory branch: (1) q0a8rx/mem.git 46d7089f, mine: reset to a9ba4377 (old ref kept in the verified before-bundle); (2) q0a8rx/rebasesim, mine: removed; (3) wxsyc5/test-clone 1abcc497, no remote, the labeling input (crab decides); older history holding some of the 58 ids: (4) replay/mirror.git muninn-notes 543b50df, origin=/Users/jhf/Projects/hug (local path, 126 history lines with the ids); (5-7) kestrel-w40cmk-memory-real, -v2, -final-e6681fda repo hug/memory, no remote (41 history lines each). None has a GitHub remote, so none can push to veridit/hug on its own. ~/Projects/hug: reset by crab to a9ba4377 (dry-run in: added=0). Other repos found (Brovs/DevOps f6d74871, seid-lang 9a9f2257) are their own branches, not hug's. FOR JØRGEN: any other machine with a clone of hug that has .hug/memory (or a muninn-notes branch) still holds the old history, and on v0.7.69 its next hug mem sync would import and push those memories back. On each, before syncing: git fetch origin hug/memory && git -C .hug/memory reset --hard origin/hug/memory, then hug mem sync --dry-run should say in: added=0. v0.7.70 (memshare, 8dc479b2) stops the push side, since hug.toml has no [memory] share.
Reflection (unanswered, 2026-11-18, crab)
crab 21:13 local: scratch clones with pre-rewrite history (wxsyc5/test-clone, replay/mirror.git, kestrel-w40cmk-memory-*) have no GitHub remote, so nothing public can be re-pushed from them; kept as evidence. mirror.git's origin is the local ~/Projects/hug, so never push from it.
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
- 2026-11-18T21:13 doing/active -> done (done); was doing_since: 2026-11-18T20:44
