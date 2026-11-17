# Capture4: capture names no agent (2026-11-17, ez50rb check 3)

**Shipped:** capture rejects a candidate memory that names a swarm agent in
agent context ("report to coordinator fox", "spider pushes"), with the reason
`names an agent: NAME` in the capture audit. It runs after the model call,
with no extra call, using the same matcher as the dream items for old
memories (`muninn/internal/agents`, names from this machine's jcode session
files). **Not shipped:** a capture-prompt rule saying "never name an agent":
it lost durable memories (below).

## Method

- Items are real capture calls from the local audit (`capture_runs`, read
  only): the exact transcript chunk, known memories and places the hook sent.
  One item per session, sessions picked at random (fixed seeds). 40 items
  name an agent in agent context (the Go matcher agrees on 40 of 40), 20
  controls name no agent name at all.
- Two samples: **sample 1** (seed 20261117) and **fresh** (seed 31, 0 sessions
  shared with sample 1), fixed before any output was seen; fresh run once.
- Each item goes through the extractor of each tree (`muninn/cmd/capture4`,
  hook token cap 1024), twice per prompt, since DeepSeek varies at
  temperature 0. A: the prompt on main. C: main plus the agent rule. B and D:
  A and C with the post-check applied to the same outputs.
- Blind judge (`muninn/scripts/capture4_judge.py`): DeepSeek Flash via jcode,
  no tools, credentials redacted. Per transcript it sees every memory any
  arm proposed, deduped, shuffled, with hashed ids, never the arm, and
  answers agent_name, agent_status (only about what an agent is doing or
  owns) and durable (a later session would be glad to have it). 1,041 memories
  judged on sample 1, 1,110 on fresh.
- Per-item values are the mean of the two runs; deltas use a paired bootstrap
  over items (2,000 reps, 95%) (`muninn/scripts/capture4_report.py`).

## Results, 40 agent-naming transcripts, per item

| sample | arm | memories | agent name or status | durable | durable, no agent |
|---|---|---|---|---|---|
| 1 | A main | 5.10 | 0.31 | 4.46 | 4.41 |
| 1 | B main + post-check | 4.97 | 0.19 | 4.45 | 4.41 |
| 1 | C prompt rule | 4.50 | 0.14 | 3.92 | 3.90 |
| fresh | A main | 5.44 | 0.42 | 4.70 | 4.56 |
| fresh | B main + post-check | 5.26 | 0.25 | 4.66 | 4.56 |
| fresh | C prompt rule | 5.17 | 0.19 | 4.31 | 4.25 |

B vs A, agent name or status: sample 1 −0.125 [−0.225, −0.037], fresh −0.175
[−0.350, −0.050], both CI-clear. Durable memories without an agent: +0.000
[+0.000, +0.000] on both. On 20 controls per sample B changes nothing (the
matcher fires 0 times).

C vs A, agent name or status: −0.175 [−0.300, −0.062] and −0.237 [−0.425,
−0.087], CI-clear; but durable −0.537 [−1.025, −0.100] on sample 1 (CI-clear
loss) and −0.388 [−0.950, +0.175] on fresh. The rule makes the model drop
memories wholesale, not just rewrite the name. C vs B on agent memories is
not CI-clear (−0.050 [−0.150, +0.037], −0.062 [−0.163, +0.025]).

Run-to-run noise on the same prompt (A vs A): agent −0.07 to +0.35 and
durable +0.20 to +1.15 on sample 1, which is why each arm is the mean of two
runs.

## The post-check against the judge

Across both samples and both main-prompt runs, 1,270 memories: the judge
flags 74 as naming an agent or being agent status; the matcher rejects 24,
and all 24 are among the judge's 74 (precision 24/24, 0 judged durable and
agent-free). The 50 it leaves: 16 use a known agent name outside the
matcher's agent context ("spider pushes", "(via fox)", "operator squid"),
21 name a lane or role with no agent name ("lane C2 runs read-only", "root
owns the migrations"), 13 are per-assignment rules with neither. The
matcher stays strict because a looser one hit nouns on the store (ez50rb
notes: seed-23 sample); those memories go to dream instead.

Tokens: 233,521 to 241,298 per arm per sample (60 calls). 2 calls of 480
were truncated and are left out of the paired items.
