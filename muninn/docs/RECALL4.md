# Recall4: BM25 keyword scoring (2026-11-14)

**Shipped:** `bm25_scale: 2.0` (config). Keyword matches are scored with SQLite
FTS5 BM25 (title weight 2) instead of flat points per word. `bm25_scale: 0`
restores the old scoring.

## Real memories (the shipping test)

277 frozen real messages from recall3 (200 development, 77 held-out; 124 short
typed, 153 long or machine-written), replayed through the production recall
engine against each split's frozen memory snapshot. Every injected memory was
judged blind with recall3's unchanged rubric (DeepSeek Flash, no tools,
credentials redacted). Deltas are per message against flat points, with 95%
paired session-cluster bootstrap intervals (2000 replicates).

| messages | relevant per message | irrelevant per message |
|---|---|---|
| long, all 153 | 0.52 -> 0.92, +0.39 [+0.25, +0.55] | 1.08 -> 1.02, -0.07 [-0.22, +0.10] |
| long, held-out 50 | 0.48 -> 0.74, +0.26 [+0.03, +0.48] | 1.54 -> 1.62, +0.08 [-0.19, +0.42] |
| short, all 124 | 0.49 -> 0.50, +0.01 [-0.04, +0.06] | 0.23 -> 0.21, -0.02 [-0.06, +0.02] |

Scale sweep (long, all): 0.25 +0.13, 0.5 +0.19, 1 +0.27, 1.5 +0.36, **2 +0.39**,
3 +0.41, 4 +0.41; short messages fall below baseline from 3 (-0.02, 4: -0.04).
Not shipped: keeping only the 12 rarest query words (+0.14, worse than all
words); dropping the all-words bonus (no change); lowering the relative floor to
0.50 (+0.10 short, but +0.17 irrelevant per long message); removing all gates
(+0.74 relevant but +1.49 irrelevant per short message).

The gain comes from rare names in long messages: ticket ids, branch and lane
names, extension names. Flat points count those the same as "test" or "review".

Latency, 277 messages: p50 356 -> 322 ms, p95 3381 -> 3308 ms.

## LongMemEval_S (context, not the shipping test)

`scripts/recall4.py`: plain FTS5 arms per question, round level, all evidence
rounds in the top 5. Held-out 379 questions: weighting the user's words 5x
+0.084 [+0.053, +0.119]; best-passage mix +0.045 [+0.016, +0.077]; stemming,
coverage-diversity and recency below +0.02 and not significant; spreading across
sessions -0.053. User weighting needs captured `source` on memories, which the
frozen snapshot lacks (0 of 11,885 memories), so it is not testable on real
memories yet.

## Reproduce

```sh
go build -o $JCODE_SCRATCH_DIR/recall4/recall4 ./muninn/cmd/recall4
python3 muninn/scripts/run_real.py baseline default   # needs recall3's frozen snapshots
python3 muninn/scripts/judge_real.py                  # blind judge + report.md
```
