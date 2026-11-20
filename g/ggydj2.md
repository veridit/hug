---
id: ggydj2
title: "Import is a one-time move: --cleanup retires the source, re-import is refused"
state: "doing"
type: feature
priority: "high"
assignee: ["crab"]
labels: []
group: ""
rank: "zzzzzzzzzz98"
created: 2026-11-17
updated: 2026-11-17
doing_since: "2026-11-17T15:58"
stage: "waiting"
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
waiting_on: "check 4: orangutan moves Seid's pre-commit hook and tools/evidence.py off .backlog (cvbpdy); then the Seid cleanup diff (ggydj2-seid-dry-run.md) goes to Jørgen"
waiting_since: "2026-11-18T00:45"
waiting_asked_by: "kestrel"
---

## Goal
An import is a one-time move: --cleanup retires the source directory and a second import of it is refused.

## Purpose
Jørgen 2026-11-17 14:45: after an import the backlog stays, and running the import again once people work in hug causes trouble. Seen today: hug setup in seid-lang still lists 'import backlog-md: 282 to update' and offers the import again although Seid has worked in hug for days; Statbus removed its .backlog by hand 21 hours ago. 14:46 ruling: the flag is --cleanup, and it removes the folder with git rm, because that is what cleanup implies.

## Principle
One source of truth: once tasks live in hug, the old system is history, not a second copy. Nothing is lost: git rm keeps it in history, and the refusal names the commit. Cleanup is an explicit step after the import, never automatic.

## Relevant details
After hug task import backlog DIR, hug task import backlog DIR --cleanup ends the move: it removes DIR with git rm in one commit (git history keeps it), records the import in hug.toml (source, commit, date), and from then on a new import of that source is refused with a fix: line naming the commit where the old copy lives. hug setup stops offering the import once it is recorded, and flags a leftover DIR with the --cleanup fix line.

Today: huginn/internal/cli/setup.go:82-101 offers 'hug task import backlog .backlog --dry-run && hug task import backlog .backlog' whenever .backlog/ or backlog/ exists. Seid: ~/Veridit/src/seid-lang/.backlog (last changed 67753633a, 2 days ago). Statbus: .backlog removed in cbd4834e2. import github does the same through bxqtkt's intake (close the issue with a link). Open questions for Jørgen are noted on the ticket.

## Depends on

## Acceptance
- [x] hug task import backlog DIR --cleanup removes DIR with git rm in one commit and records source, commit and date in hug.toml (test)
- [x] a later hug task import backlog DIR is refused, naming the commit that holds the old copy, with a fix: line (test)
- [x] hug setup no longer offers the import once it is recorded, and flags a leftover DIR with the --cleanup fix line (test)
- [ ] on seid-lang, hug setup shows the --cleanup step, and after Jørgen approves the diff it runs and setup is clean (shown before and after)

## Notes

Jørgen 2026-11-17 14:47: --cleanup also removes backlog's other traces, in the same commit after the diff is shown: backlog's MCP server in .mcp.json (and other MCP configs hug knows), and backlog's sections in AGENTS.md / CLAUDE.md. Where hug cannot edit something itself (a file it does not own, a tracked hook), the command prints what to remove, addressed to the agent: 'agent: remove ... from FILE', with the exact lines.


file:ggydj2-seid-dry-run.md (9631 bytes, sha256:bdff9e31) Seid cleanup dry run (scratch clone)
Check 4, part 1 (scratch clone of seid-lang, Seid itself untouched): the attached ggydj2-seid-dry-run.md has the full --dry-run. It shows git rm of 7658 files under .backlog, 61 documents cite .backlog, and about 64 agent: lines. Before the cleanup can run, Seid has to import .backlog/evidence (4738 files, 2766 citations) and move its own tooling off .backlog (.githooks/pre-commit:40 runs tools/backlog_integrity.py; tools/evidence.py reads .backlog/tasks). A real run on the clone was refused by Seid's pre-commit (FileNotFoundError .backlog/tasks) and the clone was left unchanged.
Checks 1-3 landed in main 55fd4f9 (e96d6f8..55fd4f9). Each test fails on base dde0b41 and passes on 55fd4f9. 1: TestImportCleanupRetiresSourceInOneCommit: git rm of DIR in one commit with a Model: trailer, plus [[import]] source/commit/date in hug.toml. TestImportCleanupRefusedByHookChangesNothing: a refusing hook leaves the tree unchanged. 2: TestImportRefusedAfterCleanup: re-import refused naming the commit, with a fix: line. 3: TestSetupStopsOfferingRecordedImport: no offer once recorded, and a leftover DIR is flagged with the --cleanup fix. -race on cli/task/importer/mcp: 'ok' for all 4 after rebasing onto e96d6f8.
landed 55fd4f9 on main 2026-11-17: import is a one-time move: hug task import backlog DIR --cleanup retires the…
landed 06297ea on main 2026-11-17: Merge v0.7.43 into main (ggydj2 stays under Unreleased)
shipped in v0.7.44 (55fd4f9)
## History
- 2026-11-17T15:58 todo -> doing/active (take)
- 2026-11-17T17:46 doing/active -> doing/waiting (wait)
- 2026-11-18T00:45 doing/waiting -> doing/waiting (wait); was waiting_for: jhf
