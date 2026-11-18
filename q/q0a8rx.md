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
Jørgen decides whether hug's public hug/memory branch is rewritten without the 58 memories that don't belong there (29 personal, 16 other projects, 13 team; gav9nw OPTIONS.md), or left as is

## Purpose
A pushed memory can't be taken back by forget or rescope: git history and other clones keep it. Only a rewrite of the public branch removes it, and that's Jørgen's call.

## Principle
No rewrite without Jørgen's yes. A backup bundle exists. hug's repo stays local after 34r1ft, so nothing new is pushed meanwhile.

## Relevant details
Branch tip 1abcc497 (46 commits). The 58 are listed by the corrected labels next to /Users/jhf/.jcode/scratch/gav9nw/OPTIONS.md. Rewriting a public branch breaks clones that already fetched it; MergeImport doesn't propagate deletions, so their copies remain.

## Depends on

## Acceptance
- [ ] Jørgen answers: rewrite or leave
- [ ] If rewrite: the branch is rewritten from the backup minus the 58, force-pushed once, and ls-remote and a fresh clone show none of them (evidence in a note)

## Notes


## History
- 2026-11-18T20:41 todo -> doing/active (take)
- 2026-11-18T20:41 doing/active -> doing/asked (ask)
- 2026-11-18T20:42 answered asked (check 1), by jhf, for crab (doing/asked -> doing/active):
  > hug's public hug/memory branch still holds 58 memories that don't belong there (29 personal, 16 other projects, 13 team). Forget can't remove them; only rewriting the branch can, and clones that already fetched it keep their copies. A backup exists. Rewrite the branch without them, or leave it?
  Rewrite, but maybe we should first move the memories to the right spot. We have a functionality for that, right?
- 2026-11-18T20:42 answer 1 delivered to crab (session_crab_1793783281662_44ffaee927a8a9d5)
