---
id: p6pyk5
title: "Memory capture runs on the agent's own cheapest model"
state: todo
type: feature
priority: "high"
assignee: []
labels: []
group: ""
rank: "zzzzzzzzzzzw"
created: 2026-11-25
updated: 2026-11-25
---

## Goal
Each agent turns its captured turns into memories on its own cheapest included model; DeepSeek or a local Ollama is the fallback.

## Purpose
Jørgen 2026-11-25: capture spends his DeepSeek account (1.21 M tokens since 11-23), while each harness has a cheap included model across many instances and subscriptions.

## Principle
Hug never picks the account: it hands the batch to the agent, which knows its subscription; nothing is lost, and the fallback runs when no agent picks it up.

## Details
Agreed with Jørgen 2026-11-25 10:14-10:19Z: 'just do it right away', integrated, as hug already returns his task answers to agents.

TODAY: the capture hook calls DeepSeek Flash itself every 12 turns (muninn/internal/extract/extract.go:440, provider openai-compatible or ollama; ~/Library/Application Support/muninn/config.json). Since 2026-11-23: 1.21 M tokens; heaviest sessions statbus hibiscus 162k, rabbit 137k, sabertooth 125k, brovs sheep 93k.

DESIGN:
1. The hook queues the batch (messages, scope, session) instead of calling a model; it holds no lock while it waits.
2. The agent gets one line through the hook output that carries task answers today (post_tool_feedback, recall/boardcue.go), e.g. 'Memory capture (return): 44 messages wait. Give batch cb_123 to your cheapest model; it calls memory_capture.' At a pause, never mid-task, as memory_dream asks.
3. A new MCP tool pair: memory_capture_batch (batch id -> the messages plus the extraction prompt, so the cheap model needs nothing else) and memory_capture (batch id + proposals). Proposals go through today's apply path: dedupe, secret check, scope, so quality rules stay one.
4. The main agent passes only the batch id; its own context gets one line, not the messages.
5. Fallback: a batch not picked up within 30 min (setting), or when the session ends, goes to the configured fallback (DeepSeek, as now) so nothing is lost.
6. hug setup asks for the fallback: a DeepSeek API key, or a local Ollama instance (model name), and names which is in use.

PROBES 2026-11-25 (crab): jcode run --json -p claude -m claude-haiku-5-5 'Reply with exactly: OK' 3.6 s, 32k cache tokens; with --disable-base-tools 1.1 s, 2.8k; -p openai -m gpt-6-luna 5.5 s, 20k. Claude Code has -p --model, Codex has exec -m. Open: whether a jcode agent can spawn a subagent on a chosen model (swarm notes per-spawn model ignored as of 2026-09-02, may be fixed); whether subscription terms allow automated background calls (Jørgen to judge).

MEASURE: pickup rate (batches done by the agent vs fallback), tokens on DeepSeek before and after, and memory quality (see the comparison task).

## Depends on

## Acceptance
- [ ] The capture hook queues a batch and the agent sees one return line naming it; a tmux or harness test shows the line once per batch
- [ ] memory_capture_batch and memory_capture work end to end: a batch fetched, proposals applied through the same dedupe, secret and scope checks as today (test)
- [ ] A batch nobody picks up goes to the fallback after the set time or at session end, and none is lost (test with a killed session)
- [ ] hug setup asks for a DeepSeek key or a local Ollama model as the fallback and shows which is in use
- [ ] On Jørgen's Mac for 2 days: share of batches done by agents, and DeepSeek tokens per day before and after, noted on this task
- [ ] Jørgen confirms after release that capture still makes good memories and his DeepSeek spend fell

## Notes
