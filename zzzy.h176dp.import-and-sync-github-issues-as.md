---
id: h176dp
title: "Import and sync GitHub issues as tasks"
status: todo
type: feature
priority: ""
assignee: []
labels: ["github", "import"]
created: 2026-11-13
updated: 2026-11-13
---

## Goal
A repository's GitHub issues appear as huginn tasks and stay in step, so agents work from one task list while people keep using GitHub

## Depends on

## Acceptance
- [ ] hug task import github [owner/repo] imports open and closed issues idempotently with --dry-run; alias GH-<n>; labels, assignees, milestone->group; closed->done, not_planned->dropped
- [ ] re-import uses the importer's three-way merge: local edits survive, GitHub-side changes come in; newest updated wins per field
- [ ] issue body becomes goal plus a Description section; task-list checkboxes become acceptance items
- [ ] phase 2: status, checks and notes flow back (close/reopen, comment) behind an explicit --push, never automatically
- [ ] auth via gh CLI token or GITHUB_TOKEN; no token stored by hug

## Notes
