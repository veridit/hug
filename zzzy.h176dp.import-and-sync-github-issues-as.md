---
id: h176dp
title: "Import and sync GitHub issues as tasks"
status: "doing"
type: feature
priority: ""
assignee: ["kestrel"]
labels: ["github", "import"]
created: 2026-11-13
updated: 2026-11-13
doing_since: "2026-11-17T15:40"
stage: "active"
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
---

## Goal
A repository's GitHub issues appear as huginn tasks and stay in step, so agents work from one task list while people keep using GitHub

## Depends on

## Acceptance
- [x] hug task import github [owner/repo] imports open and closed issues idempotently with --dry-run; alias GH-<n>; labels, assignees, milestone->group; closed->done, not_planned->dropped
- [x] re-import uses the importer's three-way merge: local edits survive, GitHub-side changes come in; newest updated wins per field
- [x] issue body becomes goal plus a Description section; task-list checkboxes become acceptance items
- [ ] phase 2: status, checks and notes flow back (close/reopen, comment) behind an explicit --push, never automatically
- [x] auth via gh CLI token or GITHUB_TOKEN; no token stored by hug

## Notes

Built 2026-11-15 (commit above): huginn import github [OWNER/REPO] [--dry-run], via gh (no token stored). statisticsnorway/statbus: 42 issues -> 42 added; second run 42 unchanged, 0 updated. Closed issues with an open checklist show 'done with unchecked acceptance' in lint (7 tasks), mirroring GitHub. Item 4 (push status back) not built.


## History
- 2026-11-17T15:40 todo -> doing/active (take)
