---
id: typq1e
title: "Capture stores no agent status at all (follow-up to ez50rb)"
state: "doing"
type: bug
priority: ""
assignee: ["hatchling"]
labels: []
created: 2026-11-17
updated: 2026-11-17
doing_since: "2026-11-17T20:41"
stage: "active"
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
---

## Goal
Capture never stores a memory that names a swarm agent or its status: from 0.19 and 0.25 such memories per agent transcript (after ez50rb's post-check) to 0, with durable memories unchanged.

## Purpose
ez50rb check 3 says 'never stores agent status'; the shipped post-check (4cf6414) cut it from 0.31/0.42 to 0.19/0.25 per transcript in the blind eval (muninn/docs/CAPTURE4.md), not to 0. The prompt rule that went further lost durable memories and was not shipped. ez50rb was closed with this remainder; this ticket owns it.

## Principle
Recall changes ship only with a measured, blind-judged gain on replayed data (AGENTS.md). Durable memories must not drop. Prefer a deterministic check (agent names from the harness session list) over a prompt rule.

## Relevant details
Eval sets and method: muninn/docs/CAPTURE4.md. Post-check: commit 4cf6414. Agent names: the harness session list, as in ez50rb's agents matcher.

## Depends on

## Acceptance
- [ ] on the CAPTURE4 sets, memories naming an agent are 0 per transcript, durable memories unchanged (blind-judged)
- [ ] the same holds on a fresh set fixed before tuning

## Notes


hatchling 19:44, fresh set fixed before any tuning (check 2): muninn/scripts/capture4_sample.py SEED=1794 EXCLUDE=(CAPTURE4 sample 1 + fresh, 120 sessions) CONTROLS=20 -> 53 items: 40 agent, 13 control (only 13 control sessions left), capture runs 2026-11-06T11:21 .. 2026-11-17T17:18, 0 sessions shared with CAPTURE4. sha256 ccb8f3f5699bcec14c583970cdf802d44b8dc13e6f0dc705fecca0ffbfd4e13a. No output of it read yet.
hatchling 19:52, design and tuning (CAPTURE4 sample 1 + fresh outputs, already judged; no new model calls): the 16 'known name outside agent context' misses (CAPTURE4.md) are memories naming an agent the TRANSCRIPT itself shows in agent context ('coordinator spider' in the chunk, 'spider pushes' in the memory). New agents.Matcher.FindIn(text, transcript): Find(text) plus every name Find(transcript) returns that text uses, not before a tag (Spider-2, dragon-task) and, for common nouns, not after a determiner (the spider). Capture passes the chunk. capture4 gains -rematch (re-runs the matcher on earlier outputs). Tuning numbers (main prompt, 2 runs, per agent transcript, judge agent name or status left): sample 1 shipped 0.188 -> FindIn 0.125, agent-name left 0.087 -> 0.025; fresh 0.250 -> 0.138, name 0.163 -> 0.050; durable agent-free +0.000 [+0.000, +0.000] on both; 14 new drops, all judged agent_name; controls drop 0. What it cannot reach: role words with no agent name ('the foreman', 'root owns', 'mechanic'), judged agent status not name; the name-only goal (0 per transcript) is about names. BAR for fresh2 (stated before its first model call, set fixed 19:44, sha ccb8f3f5...): two runs of the main-prompt extractor (capture4, cap 1024), blind judge capture4_judge.py, agent items: (1) judge agent_name left with FindIn <= 0.05 per transcript and below shipped Find, (2) durable agent-free memories dropped by FindIn but not by Find = 0, (3) controls: 0 drops. Fail on any: park, no variant.
hatchling 19:57, fresh2 measured once against the 19:51 bar: PASSES all three. 2 runs of the main prompt on 53 items (0 errors, 270 + 251 candidates), 468 memories judged blind (capture4_judge.py). Agent transcripts (n=40), judge agent_name left: shipped Find 0.175 -> FindIn 0.050 (bar <= 0.05 and below Find: yes); FindIn vs Find agent name or status left -0.125 [-0.250, -0.025], durable agent-free kept +0.000 [+0.000, +0.000]; 10 newly rejected, all judged agent_name, 0 durable agent-free among them (bar 0: yes); controls (n=13) 0 rejections (bar 0: yes). Check 1 as written says 0 per transcript; reached is 0.025 / 0.050 / 0.050 on the 3 samples. Residual 10 memories over 6 runs: 4 role words without a name ('the foreman verifies every land', judge counts as agent name), 6 'via fox' / 'operator squid' / 'jaguar (comment ...)' where the chunk names that agent only outside the matcher's context words. Not ticking checks 1-2: '0' is not met; asking crab whether the residual is acceptable or a further deterministic step is wanted (which would widen ctx, the store-noun risk ez50rb measured).
file:fresh2-bar.txt (2985 bytes, sha256:c202a163)
file:bar.py.txt (3648 bytes, sha256:cdc282b1)
file:names-view.txt (9164 bytes, sha256:2b4c3e1d)
## History
- 2026-11-17T20:41 todo -> doing/active (take)
