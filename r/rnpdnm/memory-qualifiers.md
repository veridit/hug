# rnpdnm: should the 8 be memories at all? How hug, jcode, engram and mem0 decide

Crab, 2026-11-20, for Jørgen's request 6. Sources: hug's capture prompt
(muninn/internal/extract/extract.go `prompt`, main 5eeadcea), hug's
memory_remember tool text (mcpserver/mcp.go:172), jcode's extraction prompt
(strings in ~/.local/bin/jcode), engram v1.20.0's tool texts (strings in its
binary), mem0's documented custom-instructions prompt.

## The 8, one by one

| # | Memory (verbatim, shortened) | Should it exist? | Where | Why |
|---|---|---|---|---|
| 1 | crab runs `hug task done` himself; agents should not run it | No | (agent instructions) | A role rule for one swarm, and stale: workers run done now. If it holds, it belongs in the spawn prompt, not memory. |
| 2 | Use absolute paths when applying patches; relative ones aimed at the root checkout fail | Yes | global | A tool gotcha true in every repo. |
| 3 | Crab approves pushes for this investigation; a 4th push was approved | No | none | One-assignment coordination. hug's prompt already excludes it; the extractor missed. |
| 4 | Crab's limit for task proposals is 15 lines | No | (agent instructions) | A coordinator's working rule; belongs in instructions if it holds. |
| 5 | Background jobs here are capped at 600 s, so poll every 9 minutes | Yes | global | A harness limit on this machine, true in every repo. |
| 6 | In this TUI project, use the edit tool rather than perl one-liners | No | none | Already in the agent's system prompt ("Edit files with edit ... not sed, perl"). Redundant. |
| 7 | The hug CLI installed on this machine is v0.7.64 | No | none | "Current installed version" is excluded by hug's prompt word for word; the extractor missed. Stale within a day. |
| 8 | git stash is shared across worktrees; park work on a branch | Yes | global | A git fact true in every repo. |

So: 5 of 8 should not be memories at all, and the 3 that should are global,
not this project's. The leak is mostly an extraction problem, not a placement
problem.

## How others decide what to keep

- **hug (capture, after a session)**: "durable facts, user preferences,
  corrections, and settled decisions useful in a later session"; long
  exclusion list (task progress, installed version, one-assignment
  coordination, approvals); "when uncertain, omit". Placement asks: "Would this
  still be true and useful in a different project?"
- **hug (agent calls memory_remember)**: "a rule, preference, decision or
  correction the user stated that will matter in future sessions. One fact per
  call."
- **jcode (its own extraction)**: one quality bar: "Only extract information
  that would ACTUALLY BE USEFUL if recalled in a future session on a different
  topic... weeks from now"; excludes debugging details, commit and push
  details, line-by-line changes, and "self-evident project context already in
  the system prompt"; separate consolidation later (ambient "garden":
  consolidate, prune, verify).
- **engram**: no extractor; the agent saves "PROACTIVELY after completing
  significant work" (decisions, bugfixes, patterns, discoveries), nudged when
  it has not saved for a while; conflicts with older memories go to a judge
  step and, for decisions and policy, to the user.
- **mem0**: an extraction prompt for "crucial, new, actionable facts", made
  specific per product with custom instructions and few-shot examples of what
  yields nothing.

## What hug lacks, by the 8

1. **Already in the instructions** (6, and 1 and 4 if they hold): jcode
   excludes what the system prompt already says; hug does not. hug cannot see
   the system prompt at capture time, but it can be told: a rule about how the
   agent itself should work (tools, editing, roles in a swarm) is an
   instruction, not a memory.
2. **Names a session or role** (1, 3, 4: "crab", "this investigation"): a
   memory that only makes sense inside one swarm is coordination. Rule: a
   memory naming an agent or session name is dropped, or rewritten without it.
3. **Few-shot examples** (mem0): hug's prompt has rules but almost no
   examples of input that yields nothing. These 8 are good examples.
4. **The misses on rules hug already has** (3, 7): the model ignored them.
   A cheap check after extraction can catch some: "installed", "is v0.", "this
   investigation" yield nothing.

Proposal: tighten the capture prompt with 1-3 and a post-check for 4, then
measure on fresh captures (the 1-in-200 bar and the blind count from
rnpdnm's earlier attempts), as AGENTS.md requires for memory changes.
