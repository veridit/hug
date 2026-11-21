# n7jbmx fresh held-out report, 2026-11-21 00:36Z (crab)

Protocol: hatchling-handoff-20261118.md, unchanged. Arm frozen 2026-11-18 (recall.ToolRequest.Shipped(): near + needpath + cos .65, max 2, once per session); no tuning on this set.

## Data
- 150 real jcode tool calls, 2026-11-19 00:01Z to 2026-11-20 22:41Z, 55 sessions, 75 bash / 45 edit / 30 read
  (build_tool_set.py --from 2026-11-19 --to 2026-11-21 --n 150 --seed 1121 --first-id 400000). Instances: global 105, brovs 21, seid 19, statbus 5.
- Store: snapshot-20261121.db, taken with sqlite3 `.backup` from the live muninn.db opened read-only (integrity_check ok); replayed on a copy.
- recall4 built from main f2da2991 = v0.7.93 (main-sha.txt), `-asof` per call. Both arms: exit 0, 150/150 lines.

## Result (blind judge, judge_tool.py rubric unchanged, judge_tool_set.py)
| arm | useful /100 [CI] | noise /100 [CI] | precision | calls with any | noise vs default [CI] |
|---|---:|---:|---:|---:|---:|
| shipped | 22.0 [13.6, 31.1] | 5.3 [2.2, 9.1] | 80% | 19% | -113.3 [-133.1, -96.5] |
| default | 63.3 [50.0, 78.5] | 118.7 [101.8, 138.3] | 35% | 81% | - |

Pairs: 314 hits, 296 distinct (call, memory) pairs, 296 labels (113 relevant, 183 not), 0 skipped.

## Bar, fixed before the run
| bar | result | pass |
|---|---|---|
| noise <=10/100, CI upper <=15 | 5.3, upper 9.1 | yes |
| useful >=10/100, CI lower >0 | 22.0, lower 13.6 | yes |
| precision >=50% | 80% | yes |
| noise below default, CI excluding 0 | -113.3 [-133.1, -96.5] | yes |
| hook p95 <200 ms on the LIVE store under real load >100 | not measured | PENDING |

Compared with dev (300 calls): shipped useful 10.7 -> 22.0, noise 9.0 -> 5.3, precision 54% -> 80%. The fresh set is better than dev on every quality measure.

## Hook latency (NOT the gate)
On a copy of today's snapshot (fresh/hookhome, tool_recall on, hug v0.7.93, one `hug mem hook jcode` process per call, un-niced) at load 18.6: 150 calls, p50 97.7 ms, p95 130.4 ms, max 272.4 ms, 39 printed, 0 nonzero exits (hook-timing-copy.json).
The gate needs the live store at load >100. Not done: (a) timing on the live store means the hook writes its "shown once" rows (recall/tool.go e.record) into the live muninn.db under timing session names, a live write that needs Jørgen's yes; (b) no period with load >100 has occurred since this run began (load 11-20). Load is not manufactured.

## Noise examples
noise-examples.md, 5 per arm, verbatim (shipped had 8 noise injections in all; 3 of the 5 shown are the same memory on reads of Brovs/AGENTS.md).

## Decision
tool_recall stays OFF by default. Every quality bar passes; the live-load latency gate is pending.
