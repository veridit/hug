---
id: wxsyc5
title: "Stop pushing hug's own memories to the public repo"
state: "doing"
type: bug
priority: "high"
assignee: ["crab"]
labels: []
group: ""
rank: "zzzzzzzzzzzy"
created: 2026-11-18
updated: 2026-11-18
doing_since: "2026-11-18T16:36"
stage: "active"
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
---

## Goal
No real memories go to a public remote: the installed hug stops adding to veridit/hug's hug/memory branch now, and Jørgen decides what happens to the 868 memories already there (46 commits since 2026-11-17 11:02, head 1abcc497).

## Purpose
iwazaru on 34r1ft (2026-11-18 15:25Z): veridit/hug is PUBLIC and origin/hug/memory holds memories.jsonl with 868 lines (earlier branch name muninn-notes). DESIGN.md says project memories reach a branch only when the repo opts in; today sync pushes by default. A credential-pattern scan (crab, counts only) found no password, private key, API key, ghp_/sk- token or email; 15 lines mention 'secret' and 5 'token' as words.

## Principle
Stop the leak first, by design (34r1ft: sync refuses unless the repo opts in, default local). Deleting a public branch and its history is Jørgen's call, never an agent's. No rewrite of main's history.

## Relevant details
Fix path: 34r1ft check 2 (iwazaru, guard in muninn/internal/project SyncBranch, commits 186f9b6c/be1fcd24) ships in the next release once Jørgen picks where the setting lives (34r1ft check 1). Until then every hug mem remember with sync on pushes. Options for the existing branch: delete refs/heads/hug/memory on origin (git push origin :hug/memory) after a local backup; GitHub keeps unreachable objects until GC; or keep it if he judges the content harmless.

## Depends on

## Acceptance
- [ ] Jørgen decides on the existing hug/memory branch (keep, or delete with a local backup), recorded in a note
- [ ] After the 34r1ft release, a hug mem remember in hug's repo does not move origin/hug/memory (git ls-remote before/after in a note)

## Notes


## History
- 2026-11-18T16:36 todo -> doing/active (take)
