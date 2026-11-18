---
id: q0a8rx
title: "Public hug/memory: keep or rewrite without the 58 misplaced memories"
state: "doing"
type: chore
priority: ""
assignee: ["crab"]
labels: []
group: ""
rank: "zzzzzzzzzzzz"
created: 2026-11-18
updated: 2026-11-18
doing_since: "2026-11-18T20:41"
stage: "active"
model: "claude-opus-5-5"
effort: "high"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
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
- [ ] A moves file lists all 58 with their target scope and a one-line reason; rescope --dry-run passes, then rescope writes, and hug mem search finds each one in its new scope (evidence in a note)
- [ ] The rewritten branch holds none of the 58 in any commit (git log -p | grep for each id finds nothing), and the 810 that belong are byte-identical to before in the tip's memories.jsonl
- [ ] Force-pushed once after 34r1ft is on main; ls-remote shows the new tip; a fresh clone's hug mem sync imports none of the 58

## Notes


check 1: Jørgen 19:42Z: rewrite, after moving the memories to the right place first.
## History
- 2026-11-18T20:41 todo -> doing/active (take)
- 2026-11-18T20:41 doing/active -> doing/asked (ask)
- 2026-11-18T20:42 answered asked (check 1), by jhf, for crab (doing/asked -> doing/active):
  > hug's public hug/memory branch still holds 58 memories that don't belong there (29 personal, 16 other projects, 13 team). Forget can't remove them; only rewriting the branch can, and clones that already fetched it keep their copies. A backup exists. Rewrite the branch without them, or leave it?
  Rewrite, but maybe we should first move the memories to the right spot. We have a functionality for that, right?
- 2026-11-18T20:42 answer 1 delivered to crab (session_crab_1793783281662_44ffaee927a8a9d5)
- 2026-11-18T20:42 answer 1 acknowledged by crab
- 2026-11-18T20:43 crab read the answer: hug task set q0a8rx
