---
id: dt02yw
title: "Measure qwen3-embedding and embeddinggemma-2 against nomic for recall"
state: todo
type: feature
priority: "medium"
assignee: []
labels: []
group: ""
rank: "zzzzzzzzzzzv"
created: 2026-11-25
updated: 2026-11-25
---

## Goal
A replayed, blind-judged measurement says whether qwen3-embedding:0.6b or embeddinggemma-2:270m beats nomic-embed-text for recall.

## Purpose
Jørgen 2026-11-25 installed both in Ollama and recommends qwen3-embedding:0.6b as the text default; recall changes ship only on a measured gain (AGENTS.md).

## Principle
Recall stays local and free; a new embedder ships only with a CI-clear gain on replayed messages, confirmed on fresh data, judged blind (muninn/docs/RECALL4.md).

## Details
TODAY (2026-11-25): hug mem stats says embedder 'nomic-embed-text @ http://127.0.0.1:11434'; muninn.db holds 38,564 embeddings, all 768-dim. Default in muninn/internal/config/config.go:125 (EmbedModel: nomic-embed-text). Jørgen's notes: qwen3-embedding:0.6b ~640 MB, 32K context, 1024-dim (Matryoshka), multilingual, better retrieval; embeddinggemma-2:270m ~380 MB, 8K context, 768-dim (Matryoshka), text+images; nomic 274 MB, fastest. COSTS TO MEASURE: a switch re-embeds every memory (38,564 rows) on a store copy first; 1024-dim grows the index; qwen3 may want a query/document instruction prefix (pipeline_query_prefix in meta is empty today); cosine gates tuned on nomic (e.g. dedupe_cosine 0.95, review_cosine 0.88, the tool-recall cos gate in n7jbmx) must be re-tuned per model. Also measure embed latency per call: recall runs in hooks with a 200 ms budget (n7jbmx missed it already). Inkling 2 Mini (26b-a5b-nvfp4) is a local candidate for 9pxcy4 (extraction), not here.

## Depends on

## Acceptance
- [ ] Each embedder replayed on the same frozen message set from a store copy (recall4), with its re-embed time, index size and per-call latency recorded
- [ ] Useful and noise per 100 messages judged blind for nomic, qwen3-embedding:0.6b and embeddinggemma-2:270m, with confidence intervals; a winner only on a CI-clear gain, confirmed on fresh data
- [ ] Jørgen decides from the table whether to switch the default
- [ ] The index records the embedder model and vector length it was built with, and a query with another model or length re-embeds or refuses with a fix: line (test)

## Notes

Jørgen 2026-11-25 10:39Z: all three support Matryoshka truncation (qwen3-embedding:0.6b 1024, embeddinggemma-2:270m 768, nomic 768): keep the first N dims, then L2-renormalize; same N for documents and queries; 512 is almost free, 256 a good balance, 128 a noticeable loss. His default proposal: qwen3-embedding:0.6b at 512 or 256. Arms to measure: nomic 768 (today), qwen3 at 1024, 512 and 256, gemma-2 at 768 and 256. Each arm records model and length with the index (in the pipeline hash), so a query never meets vectors of another length. Truncation of one embedding run needs no re-embed, so one qwen3 pass gives all three qwen3 arms.
