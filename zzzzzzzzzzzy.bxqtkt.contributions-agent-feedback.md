---
id: bxqtkt
title: "Contributions: agent feedback and GitHub intake with minimal bureaucracy"
status: todo
type: spike
priority: ""
assignee: []
labels: []
created: 2026-11-16
updated: 2026-11-16
---

## Goal
Settle how outside agents and people contribute to hug with the least process: agent feedback, tickets written on forks, and GitHub issues all reach our board through one triage step, and we can always see who contributed what and what to pull.

## Purpose
Jørgen 2026-11-16: jcode makes contributing easy through its maintainer_feedback tool; hug should too. He governs hug himself (high churn), takes issues gladly and reads PRs as proposals. He wants minimal bureaucracy and has not settled the design yet.

## Principle
No home page or server unless public issues prove a problem. Nothing is posted without the user seeing the exact text; paths, repo names and code are stripped. No two-way GitHub sync: triage takes an issue in, creates our ticket, and closes the issue with a link to it. The board stays the single truth. Credit the author on everything we take.

## Relevant details
Candidate design. Step 1: hug feedback "..." plus an MCP feedback tool opens a GitHub issue through the user's gh login, labelled agent-feedback, after a shown and confirmed preview. Step 2: hug task upstream ID on a fork opens a short issue linking that fork's hug/tasks ticket file. Triage, run by us: hug task inbox lists open issues (with linked fork tickets shown in full); take creates our ticket (author from issue or git author plus Model: line, link back), comments the board link and closes the issue; dismiss closes it with a reason. Two-way sync was considered and rejected: GitHub lacks our states, and untriaged items would pollute the board. Existing: huginn/internal/importer/github.go (gh issue list import), CONTRIBUTING.md contributor agreement, public repo with issues on. Option kept in reserve: a private channel (email or endpoint) if public issues leak or attract noise.

## Depends on

## Acceptance
- [ ] a design note in Notes chooses: feedback path, fork-ticket path, triage commands, and what is out; Jørgen approves it
- [ ] the chosen design is split into build tickets, each with its own acceptance
- [ ] CONTRIBUTING.md wording for 'PRs are proposals, we rebuild and credit' is drafted for Jørgen's review

## Notes
