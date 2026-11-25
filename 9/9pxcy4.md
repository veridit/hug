---
id: 9pxcy4
title: "Rank the models for memory capture: which to recommend, in what order"
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
A ranked list of extraction models (DeepSeek, Haiku, Luna, local Ollama, this week's new ones) that hug setup recommends from.

## Purpose
Jørgen 2026-11-25: 'another task for just comparing this so we can have an order of we recommend you use this or that'; p6pyk5 lets each agent choose, so the choice needs evidence.

## Principle
Same replayed turns, same prompt, judged blind; quality first, then cost and speed; re-run when a new model appears, not by taste.

## Details
Candidates (2026-11-25): deepseek-flash (today's default), claude-haiku-5-5, gpt-6-luna, a local Ollama model, and the new cheap model Jørgen is checking this week. Set: a fixed sample of past capture batches from muninn.db capture_runs on a store snapshot (as recall4 does), no live writes. Measures per model: proposals judged useful, noise or missed by a blind judge with one rubric, tokens, latency, failures. Output: a table and an order in muninn/docs, so hug setup can name the first choice and the fallback (p6pyk5). Re-run recipe in the doc.

## Depends on

## Acceptance
- [ ] A frozen sample of capture batches and a written rubric, with its sha noted, before any model is scored
- [ ] Each candidate scored blind on the same sample: useful, noise, missed, tokens, seconds; table attached
- [ ] Jørgen picks the recommended order from the table

## Notes

Jørgen 2026-11-25: Ollama now has inkling-2-mini:26b-a5b-nvfp4 (20 GB, coding/agentic), gemma4:26b, qwen3.8:27b, qwen3-coder:30b. Add Inkling 2 Mini as the local extraction candidate, timed under real load (the Mac often runs at load 50-150).
