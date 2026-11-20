---
id: bxqtkt
title: "Contributions: agent feedback and GitHub intake with minimal bureaucracy"
state: "doing"
type: spike
priority: ""
assignee: ["crab"]
labels: []
group: ""
rank: "zzzzzzzzzz2"
created: 2026-11-16
updated: 2026-11-16
doing_since: "2026-11-17T15:42"
stage: "asked"
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
asked_by: "calf"
asked_for: "jhf"
asked_since: "2026-11-17T15:58"
asked_what: "approve the design (attached: bxqtkt.md), check 1, and answer Q1-Q6 at its end"
---

## Goal
Outside agents and people contribute to hug through one triage step, and we can always see who contributed what.

## Purpose
Jørgen 2026-11-16: jcode makes contributing easy through its maintainer_feedback tool; hug should too. He governs hug himself (high churn), takes issues gladly and reads PRs as proposals. He wants minimal bureaucracy and has not settled the design yet.

## Principle
No home page or server unless public issues prove a problem. Nothing is posted without the user seeing the exact text; paths, repo names and code are stripped. No two-way GitHub sync: triage takes an issue in, creates our ticket, and closes the issue with a link to it. The board stays the single truth. Credit the author on everything we take.

## Relevant details
Settle how outside agents and people contribute to hug with the least process: agent feedback, tickets written on forks, and GitHub issues all reach our board through one triage step, and we can always see who contributed what and what to pull.

Candidate design. Step 1: hug feedback "..." plus an MCP feedback tool opens a GitHub issue through the user's gh login, labelled agent-feedback, after a shown and confirmed preview. Step 2: hug task upstream ID on a fork opens a short issue linking that fork's hug/tasks ticket file. Triage, run by us: hug task inbox lists open issues (with linked fork tickets shown in full); take creates our ticket (author from issue or git author plus Model: line, link back), comments the board link and closes the issue; dismiss closes it with a reason. Two-way sync was considered and rejected: GitHub lacks our states, and untriaged items would pollute the board. Existing: huginn/internal/importer/github.go (gh issue list import), CONTRIBUTING.md contributor agreement, public repo with issues on. Option kept in reserve: a private channel (email or endpoint) if public issues leak or attract noise.

## Depends on

## Acceptance
- [ ] a design note in Notes chooses: feedback path, fork-ticket path, triage commands, and what is out; Jørgen approves it
- [ ] the chosen design is split into build tickets, each with its own acceptance
- [ ] CONTRIBUTING.md wording for 'PRs are proposals, we rebuild and credit' is drafted for Jørgen's review

## Notes

Stable links (Jørgen 2026-11-16): ticket ids never change, so the close comment on GitHub links by id. File paths on hug/tasks do change (order prefix and slug, e.g. 0i.tzc9hb.agents-use-todos-their.md), so a blob URL to the file is not stable. The design must give a link keyed only on the id, e.g. a GitHub code-search URL for the id on branch hug/tasks, or an id-only index file on that branch that maps id to the current path.


kestrel 14:50: design note in tmp/lanes/bxqtkt.md. It applies Jørgen's one-way intake ruling and has no code. (1) 'hug task import github' becomes intake over open issues only: an inbox list, a preview showing the exact comment, then --yes runs 4 steps that can each be repeated: create the ticket (origin github:OWNER/REPO#N, alias GH-N, a credit note with @login and any Model: line), push hug/tasks, comment with the stable link plus a hidden marker, close. --dismiss N --reason closes as not_planned. Tests use a net/http client with HUG_GITHUB_API pointed at an httptest fake. (2) On a fork, 'hug task upstream ID' opens a short issue upstream: Goal, stable link, and the line 'hug-ticket: FORK ID @SHA'. Intake reads that ticket at SHA. (3) Links: GitHub code search only indexes the default branch, so the code-search idea fails. Proposal: stub files .id/ID.md on hug/tasks, only for ids linked from outside, rewritten in the same commit as a rename. A dot-folder because subfolders are groups and store.go:79 skips dot-folders. The note also covers 5 build tickets and a CONTRIBUTING draft. Evidence: 0 GH- aliases in 635 Statbus tasks, 0 issues on veridit/hug, and bxqtkt has had 6 file names.
QUESTIONS FOR JØRGEN (tmp/lanes/bxqtkt.md, end). Q1: replace the mirror (closed-issue import, re-merge) with intake, given nobody uses it? Q2: preview then --yes also for our own intake comment? Q3: close a taken-in issue as completed (proposed) or not_planned? Q4: is a stub at .id/ID.md on hug/tasks the stable link you want? Q5: does a fork ticket keep its id when free, or do we always mint our own? Q6: approve or edit the CONTRIBUTING wording in section 7 (that is check 3's draft).
file:bxqtkt.md (11535 bytes, sha256:d778a58f) design, moved here from tmp/lanes (git-ignored, so other clones never saw it)
kestrel 14:58: the design is attached as bxqtkt.md (crab). tmp/lanes/bxqtkt.md in my 14:50 note is a local copy only.
## History
- 2026-11-17T15:42 todo -> doing/active (take)
- 2026-11-17T15:48 doing/active -> doing/waiting (wait)
- 2026-11-18T02:05 doing/waiting -> doing/asked (hug upgrade, 41hhgn: a question to a person is its own stage)
