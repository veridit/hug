---
id: tzc9hb
title: "Agents use todos for their own next steps, tickets for everything else"
state: "done"
type: feature
priority: "high"
assignee: ["hatchling"]
labels: []
group: ""
rank: "0i"
created: 2026-11-16
updated: 2026-11-16
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
---

## Goal
The instructions hug installs (AGENTS.md section from hug setup, MCP tool descriptions) and the output of hug task take teach one loop: todos hold only your own next actions for the ticket you have taken; all details, dependencies and waits live in tickets; when your todos are exhausted you run hug task next (or next --mine), take a ticket, and take prints a short reminder to write todos for it, record evidence on the ticket, and finish with hug task done.

## Purpose
Jørgen 2026-11-16, after seeing two coordinators (hug's and seid's) fill their todo lists with 20+ 'Blocked: after X lands' items that copy the board: 'we're using the todos wrong, and that's maybe on us'. The harness pokes on open todos and suggests marking them blocked, so a todo list that mirrors the board becomes a wall of blocked items. With todos as own next actions only, an empty todo list means 'pick the next ticket', never 'blocked'.

## Principle
Todos are the agent's working memory for the current ticket; tickets are the record. Never put another agent's work, a wait, or a dependency in a todo: waits go in Depends on (task, group or condition, see 16dxsr). A coordinator's todos are its own actions only (review, release, ask). Keep the instruction text short and harness-neutral (no jcode-specific tool names). take's reminder is two lines at most and only shown on success.

## Relevant details
Today: agentSection in huginn/internal/cli/setup.go says Start/While working/Finish but nothing about todos. hug task take prints only 'next: hug task show ID'. Related: qawmnb board nudges (shipped v0.7.35) tell an agent when board and work disagree; 16dxsr adds group/condition dependencies. Reported the harness side to the jcode maintainer 2026-11-16.

## Depends on

## Acceptance
- [x] the AGENTS.md section from hug setup states the loop: todos for your own next steps on the taken ticket, details and waits in tickets, empty todos -> hug task next (test on the text)
- [x] hug task take prints a reminder to write todos for the ticket and to note evidence on it, and next: hug task show ID (test)
- [x] MCP task_take's result carries the same reminder (test)
- [x] after release, hug setup in this repo and in seid-lang updates their AGENTS.md section (shown with a diff)

## Notes

guppy 2026-11-17 10:08: ci/tzc9hb d355085. Check 1: agentSection (huginn/internal/cli/setup.go) adds the loop: 'Todos are your own next steps on the task you took, nothing else: details, evidence, waits and dependencies go in tasks (hug task note, hug task dep), never in todos, and another agent's work is never your todo. When your todos run out, finish the task and run hug task next: an empty todo list means take the next task, not blocked.' Test TestAgentSectionStatesTodoLoop (also asserts no harness words). Check 2: take's guide is two now: lines (todos = own next steps on ID; a wait/dependency/another agent's work is not a todo: hug task note / hug task dep; check, note, done, then hug task next) and next: hug task show ID; failed take prints no reminder. Test TestTakeRemindsTodosAndShowsTask, TestGuideAtStateChanges. Check 3: MCP task_take returns the same guide (2 lines) and next (TestTakeRecordsAgentAndModel). Check 4 prep: hug setup now marks 'agent instructions' open when a '## Tasks: hug' section differs from this hug's text and gives the replacement (TestSetupFlagsStaleTaskSection, CRLF-safe). huginn tests all ok niced. Check 4 itself waits for a release with this commit.

guppy 2026-11-17 10:40: on main 71d7ff7 (fast-forward, rebased over v0.7.39; only CHANGELOG conflicted). CI 44052782107 green on ubuntu, macos, windows (windows on rerun: first attempt failed the known flake TestAliasInvalidationCanonicalEdges in huginn/internal/cache) and secrets. Check 4 needs a release (in the next tag); not mine to tag.

crab 2026-11-17: check 4 is a release-time step; crab ticks it after the next release, once hug setup has run here and in seid-lang with the diff shown.

check 4: after v0.7.40 (tag c6e0bff, release run 44068178004), hug setup in hug and seid-lang listed '[ ] agent instructions' with the new section; applied it. seid-lang e10b80734 (AGENTS.md + CLAUDE.md: old '## Tasks: huginn' with huginn commands -> v0.7.40 section). hug 0be2c34 on ci/agents-v0740 (CI 44069237799), adds the todos paragraph and the alias line. hug setup now shows '[x] agent instructions' in both.
