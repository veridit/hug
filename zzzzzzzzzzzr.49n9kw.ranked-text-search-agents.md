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

Design (wyvern): an FTS5 table in the task cache, written in the same transaction and at the same places as table texts (insert/replace on a changed file, update path on rename, delete on remove), so the index follows every sync and ListAfterWrite with no rebuild. Columns, BM25 weights: id 10, aliases 10, title 8, goal 3, description (purpose, principle, details) 2, acceptance 2, notes 1. tokenize 'unicode61 tokenchars -' so STATBUS-402 and 44w1ev stay single tokens. Query: each word quoted as an fts5 prefix-free phrase, ANDed; field filters (state: type: epic: label:) parsed off the query and applied to the tasks the cache already lists. Result per task: id, status, title, matched field (the highest-weighted column whose text contains a query word), snippet (fts5 snippet() on that column, 12 tokens). Schema 13. CLI: hug task search WORDS [--json] [--limit N] (default 20); MCP task_search {query, limit}. TUI / search: ranks by the same call when the query has a word of 3+ characters; the filter rule (substring) stays for shorter input so typing is instant.

Fix before landing (wyvern): the first version made a cold cache build 18-25x slower. Measured on board10k (10,000 tasks), niced, interleaved, cache.db removed: cold ls 91.2 s and 94.3 s on 87307f4, against 3.7 s and 5.1 s on main. Cause: DELETE FROM search WHERE path=? with path an UNINDEXED FTS5 column, whose plan is 'SCAN search VIRTUAL TABLE INDEX 0:', run once per file (10,000 full scans). Fix e629a24 on ci/49n9kw: search_paths(rowid, path UNIQUE), and the FTS row shares that rowid. After: cold ls 4.1 s and 4.2 s (main 3.3 s, 4.7 s). Warm 'hug task search memory limit' at 10k: 0.197, 0.197 and 0.205 s, process included. TestSearchWritesNeverScan fails on a full-scan plan; it failed on the old statement.

Check 5 numbers (crab: note both, tick on the query). 4fd4777 on ci/49n9kw, CI 44095283897 bench job (ubuntu, HUG_BENCH=1, 10,000 tasks, median of 5). Query: 'search query warm 27ms (show 106ms, ratio 0.25, limit 0.50; target 100ms)'. Whole command: 'search warm 121ms, uncached 3.318s, ratio 0.04 (limit 0.50; target 400ms warm)'. The query was slow before 4fd4777 (it read all 7 text columns of every hit): 150-205 ms in-process locally. The bench fails if that comes back: '230ms is 1.27 of warm show 181ms, over 0.50'. Locally: query 32 ms, show 161 ms (ratio 0.20). Ticked when it lands.
