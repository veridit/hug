---
id: 49n9kw
title: "Ranked text search for agents and people: hug task search, MCP task_search"
status: "doing"
type: feature
priority: "high"
assignee: ["wyvern"]
labels: []
created: 2026-11-17
updated: 2026-11-17
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
---

## Goal
hug task search "words" and MCP task_search return the tasks that best match, ranked by BM25 over id, aliases, title, goal, purpose, principle, details, acceptance and notes (title and aliases weighted highest), each with its id, state, title, the matched field and a short snippet around the match. Field filters combine with the words as in the TUI (state:todo type:bug epic:ID). The same index serves the TUI / search, which then ranks by relevance instead of only filtering. At 10,000 tasks a search answers in under 100 ms warm.

## Purpose
Jørgen 2026-11-17 12:10: agents need an efficient way to find tasks; today there is no text search in the CLI or MCP (hug task ls filters only by fields), so agents list everything or grep files, and old ids cited in notes (STATBUS-402) are not found by meaning. 0sw0dd item 4 asked for BM25 over title, goal and notes in the TUI.

## Principle
Reuse, don't invent: SQLite FTS5 with BM25 in the existing task cache (muninn/internal/search already uses FTS5 BM25). The index updates with the cache delta, never a full rebuild per command. Local and free. Results say why they matched.

## Relevant details
Task cache: huginn/internal/task cache (schema now 12 after 16dxsr; 44w1ev keeps notes/description texts in their own table, a natural FTS source). muninn/internal/search/search.go has fts5Quote and BM25 weighting to copy. MCP: huginn/internal/mcp/server.go registers task_* tools. Bench in the HUG_BENCH job (ratio, not absolute).

## Depends on

## Acceptance
- [ ] hug task search "memory limit exit 137" on a fixture ranks the task with those words in its title first and prints the matched field and snippet (test)
- [ ] a search for STATBUS-402 finds the task with that alias first and tasks whose notes cite it after (test)
- [ ] MCP task_search returns the same results as the CLI, as JSON (test)
- [ ] the index follows writes: a note added is found by the next search without a rebuild (test)
- [ ] HUG_BENCH: search at 10,000 tasks warm costs at most 0.5 of an uncached search, and under 100 ms on ubuntu CI (bench job)
- [ ] the TUI / search ranks by the same index (test)

## Notes
