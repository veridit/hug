# Tool-call recall (66 calls, 7 sessions), strict judge

| arm | useful /100 calls | noise /100 calls | calls with any injection | useful vs default [CI] | noise vs default [CI] |
|---|---:|---:|---:|---:|---:|
| default | 30.3 | 48.5 | 44% | - | - |
| norecords1 | 18.2 | 27.3 | 30% | -12.1 [-18.4, -4.4] | -21.2 [-37.8, -5.0] |

## Reading (2026-11-16)

66 real tool calls from 7 jcode sessions, replayed against the frozen typed
snapshot, every injected memory judged blind with a strict rubric (would the
agent act differently on THIS call, and is it not already shown).

- Message recall reused as-is (default) injects on 44% of calls; 38% of
  those memories are useful (30 useful vs 49 noise per 100 calls).
- Leaving records out (norecords1) cuts noise by 21 per 100 [CI -38, -5]
  but also useful by 12 [-18, -4]; precision stays near 40%.
- In the wide candidate pool, rule memories are the most precise (28%),
  records the least (8%).

Decision: no tool-call hook ships yet. A tool-call trigger needs a signal
message recall does not have (the memory names this file, command or
service), not a stricter version of the same gate.

## Anchor trigger, dev replay (2026-11-18, n7jbmx)

Set: 300 tool calls, 77 sessions, 2026-11-16T00:00Z .. 2026-11-18T04:34:59Z (store frozen at 04:34:59Z).
Stratified: bash 150, edit/write/apply_patch 90, read 60. Instances: seid 116, brovs 111, global 57, statbus 16.
Each call sees only memories last written before it (recall4 -asof). Judge: judge_tool.py's rubric, unchanged
(strict: would the agent act differently on THIS call), DeepSeek Flash, blind to the arm. 0 pairs unjudged.

| arm | useful /100 [CI] | noise /100 [CI] | precision | calls with any | noise vs default [CI] | replay p95 ms |
|---|---:|---:|---:|---:|---:|---:|
| default (message recall on the call text) | 57.3 [47.0, 68.4] | 135.3 [121.0, 150.0] | 30% | 83% | - | 909 |
| anchor | 28.0 [20.6, 35.7] | 29.3 [23.2, 36.5] | 49% | 35% | -106.0 [-121.5, -91.1] | 8 |
| anchor+cos60 | 24.7 [17.3, 32.5] | 23.3 [18.1, 29.0] | 51% | 31% | -112.0 [-126.2, -98.0] | 54 |
| anchor+cos65 | 14.3 [9.3, 20.2] | 15.7 [11.1, 20.8] | 48% | 21% | -119.7 [-134.8, -105.6] | 14 |
| anchor+cos70 | 6.0 [3.0, 9.8] | 4.7 [2.7, 6.8] | 56% | 8% | -130.7 [-145.4, -116.5] | 11 |
| anchor+cos75 | 2.0 [0.3, 4.5] | 1.0 [0.0, 2.1] | 67% | 2% | -134.3 [-149.1, -119.9] | 7 |
| anchor+lim1 | 16.7 [12.3, 21.1] | 20.3 [16.6, 24.4] | 45% | 37% | -115.0 [-129.5, -100.6] | 7 |
| anchor+cos65+lim1 | 10.0 [6.0, 14.7] | 12.0 [8.5, 15.5] | 45% | 22% | -123.3 [-137.9, -109.2] | 6 |
| anchor+max10 | 19.7 [13.6, 26.1] | 23.3 [17.9, 29.5] | 46% | 29% | -112.0 [-126.8, -98.0] | 5 |
| anchor+near | 25.3 [18.4, 32.9] | 27.0 [20.5, 34.1] | 48% | 31% | -108.3 [-124.4, -92.9] | 6 |
| anchor+needpath | 24.0 [17.0, 31.4] | 20.0 [14.0, 26.8] | 55% | 27% | -115.3 [-130.3, -100.8] | 5 |
| anchor+near+needpath | 21.0 [14.4, 28.1] | 17.3 [11.2, 23.8] | 55% | 22% | -118.0 [-133.0, -103.5] | 5 |
| anchor+near+needpath+cos60 | 19.3 [13.0, 26.3] | 13.0 [8.5, 18.0] | 60% | 19% | -122.3 [-136.7, -108.4] | 5 |
| **anchor+near+needpath+cos65 (= "shipped")** | **10.7 [6.4, 15.7]** | **9.0 [5.2, 13.3]** | **54%** | 14% | -126.3 [-140.9, -112.1] | 5 |
| anchor+near+needpath+lim1 | 12.7 [8.4, 17.0] | 10.7 [7.1, 14.4] | 54% | 23% | -124.7 [-139.4, -110.4] | 5 |

Arm parts: near = a path anchor must end at the call's path or its folder (a memory about a far ancestor such as
Projects/Brovs/.jcode/scratch no longer matches every file under it: 13 noise / 2 useful on the plain anchor arm).
needpath = a command head alone never triggers (cmd-only matches were 15 useful / 28 noise). cosNN = the call's text
must also embed within NN/100 of the memory (nomic-embed-text). limN = at most N memories per call (default 2).

Frozen for the fresh set, before any 11-19/20 call exists: recall.ToolRequest.Shipped() = near + needpath + cos 0.65,
limit 2, an anchor shared by more than 40 memories never triggers, types rule/fact/preference, once per session.
recall4 -variant shipped reproduces the dev arm on 300 of 300 calls. Only this arm (and default) is judged on the
fresh set; no tuning on it.

Dev selection is optimistic: 15 arms were tried on these 300 calls, and the chosen one sits on the bar edge (noise
upper CI 13.3, useful lower CI 6.4). The fresh set may well fail the bar; then the hook stays off (tool_recall
defaults to false).

Hook timing, the real path (one `hug mem hook jcode` process per call, post_tool_feedback, tool_recall on),
on a copy of the live store taken 05:11Z, 300 dev calls, not niced, load average 14.9/11.6/13.0 (44 users):
calls 300, p50 59.1 ms, p95 150.3 ms, max 339.6 ms, printed 63, nonzero exit 0.
The embedder (Ollama) is reached only when an anchor already matched.
Anchor backfill on an existing store: once, 2.2 s for 29,770 active units (32,292 anchors on 8,661 units).

How it works: at index time `muninn/internal/anchor` pulls the paths, file
names and backticked command heads out of every active memory into
`unit_anchors` (no model). On a tool call the hook takes the call's touched
paths, the paths its command names and its command heads, and looks them up
exactly; ranks by rarity (log(1 + N/df)); keeps rule, fact and preference
memories in the session's binding, not yet shown this session; and only then
asks the embedder for the cosine gate.

Reproduce: `muninn/scripts/build_tool_set.py` (set), `go run ./muninn/cmd/recall4
-asof -variant shipped` (replay), `muninn/scripts/judge_tool_set.py` (judge and
report), `muninn/scripts/time_tool_hook.py` (hook latency).

Decision pending: `tool_recall` stays off by default until the fresh set
(calls of 2026-11-19 and 11-20) passes the bar above.
