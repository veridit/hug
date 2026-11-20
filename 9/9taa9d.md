---
id: 9taa9d
title: "Merge muninn and huginn into one hug binary"
state: todo
type: feature
priority: "high"
assignee: []
labels: []
group: ""
rank: "3"
created: 2026-11-11
updated: 2026-11-11
aliases: ["merge-muninn-and-huginn-into-one"]
---

## Goal
One hug binary carries tasks, memories and messages, with muninn and huginn kept only as compatibility aliases.

## Purpose
Jørgen 2026-11-18 (request 1 on this task): the goal is a unified, easy-to-navigate, non-confusing CLI and system. The decision behind it is his of 2026-11-11 (memory j9jdkh, docs/DESIGN.md): one tool named hug for memory, tasks and messages, files on branches as the truth, and memory file-oriented too (one Markdown file per memory) so humans can edit it and git merges stay clean.

## Principle
Consistency through a clean sweep: once we decide hug is the thing, we change everything at once (commands, help, messages, docs, branches, folders, tests, every project's setup), never a mix of old and new names left behind.

## Relevant details
A unified, easy-to-navigate, non-confusing CLI and system: one hug binary with tasks, memories and messages; .hug/<kind>/ on branches hug/<kind>; muninn and huginn kept only as compatibility aliases.

Done: v0.7.0 (2026-11-13) one hug binary, v0.7.1 one hug mcp server, v0.7.3 tasks on branch hug/tasks at .hug/tasks (this repo and seid-lang migrated). Open: memories still live in the SQLite store. On main the sync branch is still muninn-notes (muninn/internal/project/project.go:39); commit c16b7fd6 (2026-11-18, on lane branches, not yet on main) moves project memory to hug/memory through an explicit layout upgrade, and this repo already has a hug/memory branch. Jørgen names it 'memory', not 'memories' (note 2026-11-16). Item 3 (one file per memory, retire = move to _retired/) needs a design talk with him before building. Children: 34r1ft (per-repo memory setting), gav9nw (personal vs shared).

## Depends on

## Acceptance
- [x] hug setup, hug task ..., hug memory ... work; one MCP server
- [x] branches hug/tasks and hug/memories; huginn branch migrated in muninn, huginn, seid-lang
- [ ] done when task:b8mk9s is done (memories stored one Markdown file per memory, retire = move to _retired/)

## Requests
- 2026-11-18T12:00 request 1 by jhf:
  I think the principle here is that we always are consistent. So when we decide that, okay, we're going to go into make hug the thing, we have to run a clean cone chip and do everything, and so that is the principle. And the goal is a unified, easy-to-navigate, non-confusing CLI and system
  resolved 2026-11-18T12:20 by crab:
    Principle set to: consistency through a clean sweep, everything changes at once, no mix of old and new names. Goal now leads with 'a unified, easy-to-navigate, non-confusing CLI and system', then the concrete one-binary layout. Read 'clean cone chip' as 'clean sweep'; correct me if you meant something else.
- 2026-11-20T13:08 request 2 by jhf:
  This one could probably go pretty soon. It's just cleanup work, doesn't require anything from me, right?

## Notes

v0.7.0 released 2026-11-13: one hug binary (hug task / hug mem / hug setup / hug tui), muninn and huginn as links. CI green on ubuntu, macos, windows (run 43437457236); release run 43439376684 published 6 archives + SHA256SUMS. brew upgrade huginn installed hug 0.7.0 with muninn/huginn symlinks; 'hug v0.7.0 (memory: muninn v0.7.0, tasks: huginn v0.7.0)'. Claude and jcode hooks exit 0, both MCP servers answer initialize. Steps 2 and 3 (hug/<kind> branches, file-per-memory) remain.

Correction: item 1 was ticked too early. 'One MCP server' is NOT done: muninn mcp and huginn mcp are still two servers. Follow-up: a combined 'hug mcp'.

v0.7.1 (2026-11-13): hug mcp serves memory_* and task_* from one process. Switched 6 configs to it (~/.jcode, Brovs, GenderedGames, truth_tribe_and_treasure jcode mcp.json; ~/.claude.json; Claude app), backups *.bak-<stamp>. Verified: Brovs binding client:brovs, 17 tools. Item 1 now complete.

Status 2026-11-16: item 2 half done: tasks moved to branch hug/tasks at .hug/tasks (hug upgrade, v0.7.3; this repo and seid-lang migrated). Memory is still in the muninn SQLite DB; branch hug/memory (user's name: 'memory', not 'memories') and item 3 (file per memory, retire = move to _retired/) need a design discussion with the user before building.


crab 17:29 local, from bonehound: the move to hug/memory is already on main (c16b7fd6, v0.7.61; hug, statbus, seid-lang moved). But gitbranch.Upgrade (gitbranch.go:1139) pushes hug/memory and deletes muninn-notes on the remote with no share check: a likely route for veridit/hug's public 868 memories (wxsyc5). Fix goes with 34r1ft: the upgrade renames locally only unless the repo opts in.
check 2: crab 2026-11-20 12:08Z: git ls-remote origin on veridit/hug and seid-lang: only hug/tasks and hug/memory; no huginn, muninn-notes or muninn branch. Code: internal/layout/memory.go MemoryBranch = "hug/memory"; muninn-notes is kept only as OldMemoryBranch for the upgrade path and the commit-msg exemption. Branch name is 'memory' (Jørgen 11-16), not 'memories' as the check text says.
## History
- 2026-11-18T12:00 request 1 by jhf
- 2026-11-18T12:20 resolved request 1 by crab
- 2026-11-20T13:08 request 2 by jhf
- 2026-11-20T13:08 request 2 delivered to crab (session_crab_1793783281662_44ffaee927a8a9d5)
- 2026-11-20T13:08 request 2 acknowledged by crab
- 2026-11-20T13:08 request 2 in progress by crab: hug task check 9taa9d 2 "crab 2026-11-20 12:08Z: git ls-remote origin on veridit/hug and seid-lang: only hug/tasks and hug/memory; no huginn, mu…"
