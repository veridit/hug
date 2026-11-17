---
id: ggydj2
title: "Import is a one-time move: --cleanup retires the source, re-import is refused"
status: "doing"
type: feature
priority: "high"
assignee: ["kestrel"]
labels: []
created: 2026-11-17
updated: 2026-11-17
doing_since: "2026-11-17T15:58"
stage: "active"
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
---

## Goal
After hug task import backlog DIR, hug task import backlog DIR --cleanup ends the move: it removes DIR with git rm in one commit (git history keeps it), records the import in hug.toml (source, commit, date), and from then on a new import of that source is refused with a fix: line naming the commit where the old copy lives. hug setup stops offering the import once it is recorded, and flags a leftover DIR with the --cleanup fix line.

## Purpose
Jørgen 2026-11-17 14:45: after an import the backlog stays, and running the import again once people work in hug causes trouble. Seen today: hug setup in seid-lang still lists 'import backlog-md: 282 to update' and offers the import again although Seid has worked in hug for days; Statbus removed its .backlog by hand 21 hours ago. 14:46 ruling: the flag is --cleanup, and it removes the folder with git rm, because that is what cleanup implies.

## Principle
One source of truth: once tasks live in hug, the old system is history, not a second copy. Nothing is lost: git rm keeps it in history, and the refusal names the commit. Cleanup is an explicit step after the import, never automatic.

## Relevant details
Today: huginn/internal/cli/setup.go:82-101 offers 'hug task import backlog .backlog --dry-run && hug task import backlog .backlog' whenever .backlog/ or backlog/ exists. Seid: ~/Veridit/src/seid-lang/.backlog (last changed 67753633a, 2 days ago). Statbus: .backlog removed in cbd4834e2. import github does the same through bxqtkt's intake (close the issue with a link). Open questions for Jørgen are noted on the ticket.

## Depends on

## Acceptance
- [ ] hug task import backlog DIR --cleanup removes DIR with git rm in one commit and records source, commit and date in hug.toml (test)
- [ ] a later hug task import backlog DIR is refused, naming the commit that holds the old copy, with a fix: line (test)
- [ ] hug setup no longer offers the import once it is recorded, and flags a leftover DIR with the --cleanup fix line (test)
- [ ] on seid-lang, hug setup shows the --cleanup step, and after Jørgen approves the diff it runs and setup is clean (shown before and after)

## Notes

Jørgen 2026-11-17 14:47: --cleanup also removes backlog's other traces, in the same commit after the diff is shown: backlog's MCP server in .mcp.json (and other MCP configs hug knows), and backlog's sections in AGENTS.md / CLAUDE.md. Where hug cannot edit something itself (a file it does not own, a tracked hook), the command prints what to remove, addressed to the agent: 'agent: remove ... from FILE', with the exact lines.


file:ggydj2-seid-dry-run.md (9631 bytes, sha256:bdff9e31) Seid cleanup dry run (scratch clone)
## History
- 2026-11-17T15:58 todo -> doing/active (take)
