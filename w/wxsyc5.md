---
id: wxsyc5
title: "Stop pushing hug's own memories to the public repo"
state: "done"
type: bug
priority: "high"
assignee: ["crab"]
labels: []
group: ""
rank: "zzzzzzzzzzq"
created: 2026-11-18
updated: 2026-11-18
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
done_at: "2026-11-20T13:04"
---

## Goal
No real memories go to a public remote, and Jørgen decides what happens to those already on veridit/hug.

## Purpose
iwazaru on 34r1ft (2026-11-18 15:25Z): veridit/hug is PUBLIC and origin/hug/memory holds memories.jsonl with 868 lines (earlier branch name muninn-notes). DESIGN.md says project memories reach a branch only when the repo opts in; today sync pushes by default. A credential-pattern scan (crab, counts only) found no password, private key, API key, ghp_/sk- token or email; 15 lines mention 'secret' and 5 'token' as words.

## Principle
Stop the leak first, by design (34r1ft: sync refuses unless the repo opts in, default local). Deleting a public branch and its history is Jørgen's call, never an agent's. No rewrite of main's history.

## Relevant details
No real memories go to a public remote: the installed hug stops adding to veridit/hug's hug/memory branch now, and Jørgen decides what happens to the 868 memories already there (46 commits since 2026-11-17 11:02, head 1abcc497).

Fix path: 34r1ft check 2 (iwazaru, guard in muninn/internal/project SyncBranch, commits 186f9b6c/be1fcd24) ships in the next release once Jørgen picks where the setting lives (34r1ft check 1). Until then every hug mem remember with sync on pushes. Options for the existing branch: delete refs/heads/hug/memory on origin (git push origin :hug/memory) after a local backup; GitHub keeps unreachable objects until GC; or keep it if he judges the content harmless.

## Depends on

## Acceptance
- [x] Jørgen decides on the existing hug/memory branch (keep, or delete with a local backup), recorded in a note
- [x] After the 34r1ft release, a hug mem remember in hug's repo does not move origin/hug/memory (git ls-remote before/after in a note)

## Notes


bonehound: local backup written, nothing on the remote touched (no delete, no push). /Users/jhf/.jcode/scratch/wxsyc5/hug-memory-1abcc497.bundle: git bundle create of refs/remotes/origin/hug/memory after git fetch origin hug/memory; head 1abcc4979512cf17c030abfe87c3ca9ba78d5673, git rev-list --count = 46; 114821 bytes, sha256 fbde393e2c5a0c0acf1fae16f9dced4c541f7bf660c3b54330d675971d261a27. Verify (in a repo): 'The bundle records a complete history.' / 'The bundle uses this hash algorithm: sha1'. Test clone /Users/jhf/.jcode/scratch/wxsyc5/test-clone (remote removed): HEAD 1abcc497, 46 commits, wc -l memories.jsonl = 868. Restore: git fetch BUNDLE refs/remotes/origin/hug/memory:refs/heads/hug/memory.
crab 18:25 local, on Jørgen's answer (keep, if the agent places memories right): measured on the branch head 1abcc497, 868 memories. No memory carries a personal/project/elsewhere field today (keys: id, content, category, active, created, updated); everything captured in hug's checkout goes to hug's branch, so no judgment is made now. Pattern counts (lower bounds): 32 are about other projects (e.g. 'Seid lives at /Users/jhf/Veridit/src/seid-lang and is observe-only', Statbus ticket steps), 6 contain /Users/jhf paths, 4 are Jørgen preferences, 39 are category preference. None holds a secret. So: keep the branch (not deleted); sharing resumes only once each memory is placed at write time (gav9nw), and hug opts in (34r1ft A).
check 1: Jørgen 18:19: keep the branch, provided memories are placed right (personal vs repo). Kept; nothing deleted. Placement is gav9nw; opt-in waits for it.
landed cee68ee on main 2026-11-18: upgrade: the memory layout rename stays local unless the repo shares memories…
shipped in v0.7.70 (cee68ee)
check 2: crab 2026-11-20 12:04Z: hug mem remember ran in hug's repo at 10:28Z and 10:44Z today (v0.7.89/90 era, after the 34r1ft release); git ls-remote origin refs/heads/hug/memory = a9ba43773dd7 before and after, the 2026-11-18 20:30 commit; origin/hug/memory did not move.
Reflection (dismissed, 2026-11-20, crab): Its last check waited two days on a check I could run myself any time.
## History
- 2026-11-18T16:36 todo -> doing/active (take)
- 2026-11-18T16:36 doing/active -> doing/asked (ask)
- 2026-11-18T18:19 answered asked (check 1), by jhf, for crab (doing/asked -> doing/active):
  > hug's public repo has 868 of our memories on the hug/memory branch, pushed since yesterday 11:02 (no passwords, keys or tokens found; 15 mention 'secret' as a word). The push stops when 34r1ft ships (your answer there picks where the opt-in lives). For the branch already there: keep it, or delete it from GitHub after I save a local copy?
  I think we have to discuss this because truly other repositories are also public, and as long as there are no secrets and the system is working, it's sensible to have it with the repository, isn't it? As long as the classification between personal things and repository things stay, like if like the discussion is really, are we able to properly place things? And does the agent that puts the memory there do the proper judgment call. If it does, then it's okay
- 2026-11-18T18:24 crab read the answer: hug task note wxsyc5 "crab 18:25 local, on Jørgen's answer (keep, if the agent places memories right): measured on the branch head 1abcc497, …"
- 2026-11-18T18:48 doing/active -> doing/waiting (wait)
- 2026-11-20T13:04 doing/waiting -> done (done); was doing_since: 2026-11-18T16:36; waiting_asked_by: crab; waiting_on: check 2: after the 34r1ft release, ls-remote before/after a hug mem remember; waiting_since: 2026-11-18T18:48
