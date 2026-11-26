---
id: dt02yw
title: "Measure qwen3-embedding and embeddinggemma-2 against nomic for recall"
state: "doing"
type: feature
priority: "medium"
assignee: ["dt02yw-engineer"]
labels: []
group: ""
rank: "003"
created: 2026-11-25
updated: 2026-11-25
doing_since: "2026-11-26T09:01"
stage: "active"
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
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
- [x] Each embedder replayed on the same frozen message set from a store copy (recall4), with its re-embed time, index size and per-call latency recorded
- [x] Useful and noise per 100 messages judged blind for nomic, qwen3-embedding:0.6b and embeddinggemma-2:270m, with confidence intervals; a winner only on a CI-clear gain, confirmed on fresh data
- [ ] Jørgen decides from the table whether to switch the default
- [ ] The index records the embedder model and vector length it was built with, and a query with another model or length re-embeds or refuses with a fix: line (test)

## Notes

Jørgen 2026-11-25 10:39Z: all three support Matryoshka truncation (qwen3-embedding:0.6b 1024, embeddinggemma-2:270m 768, nomic 768): keep the first N dims, then L2-renormalize; same N for documents and queries; 512 is almost free, 256 a good balance, 128 a noticeable loss. His default proposal: qwen3-embedding:0.6b at 512 or 256. Arms to measure: nomic 768 (today), qwen3 at 1024, 512 and 256, gemma-2 at 768 and 256. Each arm records model and length with the index (in the pipeline hash), so a query never meets vectors of another length. Truncation of one embedding run needs no re-embed, so one qwen3 pass gives all three qwen3 arms.


# dt02yw: qwen3-embedding and embeddinggemma-2 against nomic for recall

Harness: recall4 at a5df9478 (branch dt02yw, CI 8/8 on that SHA, run 45803351521). Frozen RECALL4 sets:
277 messages (development 200, held-out 77) and fresh 147, each replayed against the frozen snapshot
recall3/baseline-*.db (20,764 distinct texts). Judge: RECALL4's judge_real.py, unchanged (DeepSeek Flash,
no tools, redacted, pairs shuffled, arms lettered A-F per key.json, unblinded only after both reports).
The judge rated 904 new pairs (304 dev+held-out, 600 fresh) in 188 logged calls (real/judge-logs). Every
other pair came from a copy of the recall4 judgment cache.

Harness check: the nomic arm reproduces RECALL4 `default` exactly on 199/200 dev, 76/77 held-out and
147/147 fresh messages (identical hit lists).

check 1: store copy (frozen snapshot, 20,764 texts) re-embedded per model: nomic 560 s, qwen3 1329 s, gemma2 714 s active; index MB 125.9/220.0/107.5/59.4/125.9/59.4; query embed p95 ms nomic 101, qwen3 321, gemma2 180 (dt02yw/reembed, latency, real/*.size.json; recall4 a5df9478, CI run 45803351521 8/8)
check 2: blind RECALL4 judge, 904 new pairs, session-cluster CIs: qwen3@1024 useful +19.1 [+5.9,+31.5] dev+ho and +36.7 [+26.3,+55.7] fresh; fresh noise -37.4 [-119.5,+2.6] (not clear), short-message noise +44.6 [+16.3,+71.9]; no unconditional winner (dt02yw/RESULTS.md)
## Result table (dev+held-out n=277 | fresh n=147). Per 100 messages, 95% session-cluster bootstrap

| arm | useful/100 dev+ho [CI] | noise/100 dev+ho [CI] | useful/100 fresh [CI] | noise/100 fresh [CI] | p50 / p95 query embed ms | re-embed 20,764 texts (active) | index MB (20,764) |
|---|---|---|---|---|---|---|---|
| nomic@768 (today) | 72.6 [54.3, 97.4] | 65.7 [48.0, 92.4] | 32.0 [21.2, 42.5] | 165.3 [108.7, 292.0] | 26 / 101 | 560 s, 37.1/s | 125.9 |
| qwen3@1024 | 91.7 [67.8, 119.5] | 45.1 [36.0, 60.3] | 68.7 [56.0, 86.8] | 127.9 [101.9, 181.3] | 61 / 321 | 1329 s, 15.6/s | 220.0 |
| qwen3@512 | 86.6 [65.9, 113.2] | 49.1 [37.2, 66.4] | 59.9 [48.1, 72.8] | 129.9 [109.5, 176.1] | same model | same pass | 107.5 |
| qwen3@256 | 79.4 [60.0, 103.2] | 53.1 [42.0, 68.3] | 49.0 [38.0, 61.6] | 154.4 [129.8, 202.3] | same model | same pass | 59.4 |
| gemma2@768 | 87.7 [64.7, 117.0] | 62.1 [51.5, 78.9] | 49.7 [29.2, 64.7] | 202.0 [151.8, 314.8] | 32 / 180 | 714 s, 29.1/s | 125.9 |
| gemma2@256 | 79.1 [61.1, 102.5] | 61.4 [50.1, 78.8] | 48.3 [28.8, 61.4] | 196.6 [158.8, 283.7] | same model | same pass | 59.4 |

Paired deltas vs nomic (per 100, [CI]):

| arm | dev+ho useful | dev+ho noise | fresh useful | fresh noise | fresh short-msg noise | fresh long-msg noise |
|---|---|---|---|---|---|---|
| qwen3@1024 | +19.1 [+5.9, +31.5] | -20.6 [-36.2, -9.5] | +36.7 [+26.3, +55.7] | -37.4 [-119.5, +2.6] | +44.6 [+16.3, +71.9] | -143.8 [-169.4, -116.7] |
| qwen3@512 | +14.1 [+4.2, +23.6] | -16.6 [-29.9, -7.5] | +27.9 [+19.6, +40.9] | -35.4 [-117.4, +2.9] | +51.8 [+37.0, +60.9] | -148.4 [-172.6, -122.7] |
| qwen3@256 | +6.9 [-3.5, +15.8] | -12.6 [-29.2, -1.9] | +17.0 [+10.4, +29.3] | -10.9 [-101.2, +31.7] | +74.7 [+38.5, +104.2] | -121.9 [-150.0, -93.9] |
| gemma2@768 | +15.2 [+6.1, +25.3] | -3.6 [-17.9, +6.4] | +17.7 [+4.8, +28.2] | +36.7 [+14.5, +52.0] | +53.0 [+37.8, +64.3] | +15.6 [-10.0, +41.9] |
| gemma2@256 | +6.5 [-1.9, +14.0] | -4.3 [-20.3, +7.8] | +16.3 [+4.4, +23.2] | +31.3 [-15.7, +54.2] | +77.1 [+58.5, +87.1] | -28.1 [-59.7, +3.2] |

Full strata: report-development-heldout-unblinded.md, report-fresh-unblinded.md (blind versions without -unblinded).

## Verdict

qwen3-embedding:0.6b at 1024 and at 512 are the only arms with a CI-clear useful gain on both the
frozen and the fresh set, and they cut noise CI-clear on dev+held-out. On fresh, total noise is not
CI-clear (upper bound +2.6), and on short typed messages noise rises CI-clear (+44.6 per 100 at 1024).
So no unconditional winner yet: switching needs the short-message gate re-tuned for qwen3 and a re-run.
gemma2 adds noise on fresh (+36.7 [+14.5, +52.0]) and loses.

## Method notes

- Prefixes (prefixes.json): nomic none (prompts.go measured decision). qwen3 query
  "Instruct: Given a web search query, retrieve relevant passages that answer the query\nQuery:",
  documents none (HF config_sentence_transformers.json). gemma2 query "task: search result | query: ",
  document "title: none | text: " (HF model card).
- Matryoshka: one full pass per model, truncated to N and then L2-renormalised for documents and queries alike.
- Gates: the recall gates (semantic floor 0.5, min 0.64 / 0.58, weight 10) are kept and each arm's cosine is mapped
  onto nomic's scale by quantile matching. The arm's cosine that admits as many (message, memory) pairs as nomic
  does at L, for L in 0.45..0.75, is fitted with nomic ~= a*arm + b (max residual 0.016). It is applied exactly
  through one extra coordinate (recall4 -affine). Calibration used only the 208 non-notice dev+held-out messages,
  never fresh. calib/calib.md.
- Write-time gates, re-derived the same way (nearest-neighbour cosine of 3000 sampled memories, same count flagged):
  dedupe 0.95 maps to qwen3 0.929/0.935/0.937 (1024/512/256) and gemma2 0.950/0.955; review 0.88 maps to qwen3
  0.854/0.863/0.871 and gemma2 0.907/0.916. These were not exercised by the replay.
- Re-embed scope (coordinator 10:39): only the 20,764 texts the replay can reach. The full store has 39,218 distinct
  texts. Extrapolated (not measured): nomic about 18 min, qwen3 about 42 min, gemma2 about 22 min of active time.
  Comparability: the gate starts each 640-text chunk only at 1-min load <= 20. The load during a chunk was not
  controlled. Load samples <= 20 while each model ran had medians 17.7 (nomic), 17.4 (qwen3) and 16.6 (gemma2),
  and the median over all samples (pauses included) was 88, 25 and 19 (load.log). So the active times are roughly
  comparable, not controlled. The gate paused nomic 6009 s, qwen3 2671 s and gemma2 270 s.
- Latency: 277 single uncached query embeds per model, warmed, run back to back at load 13-18. Over 200 ms:
  nomic 1/277, qwen3 47/277, gemma2 8/277. qwen3 p95 321 ms is over the 200 ms hook budget.
- Index MB: dbstat bytes of embeddings + qvecs + sketches + pk index in the 20,764-text arm DB, including the one
  calibration coordinate. Scale by about 1.9 for the full store. Of qwen3@1024's 220 MB, the qvecs table is 97 MB
  (21 MB at 768 dims). The cause was not investigated.
## History
- 2026-11-26T09:01 todo -> doing/active (take)
