---
id: typq1e
title: "Capture stores no agent status at all (follow-up to ez50rb)"
state: "done"
type: bug
priority: ""
assignee: ["hatchling"]
labels: []
group: ""
rank: "zzzzzzzzzz3"
created: 2026-11-17
updated: 2026-11-17
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
done_at: "2026-11-17T21:13"
done_release: "v0.7.47"
---

## Goal
Capture stores fewer memories that name a swarm agent, with durable memories kept. Measured (blind judge, main prompt, per agent transcript, judge agent-name memories left): shipped Find 0.087 / 0.163 / 0.175 / 0.237 on sample 1 / fresh / fresh2 (held out) / fresh3 -> FindIn 0.025 / 0.050 / 0.050 / 0.100. Durable lost: 0 on 3 sets; on fresh3 1 of 80 run-items (0.013/transcript), item 1011.0 run 2, a work order naming a gone agent, which the judge split on across runs. Not 0: role words ('the foreman') and names the chunk shows only outside agent context ('via fox', goat, daisy) remain (muninn/docs/CAPTURE4.md).

## Purpose
ez50rb check 3 says 'never stores agent status'; the shipped post-check (4cf6414) cut it from 0.31/0.42 to 0.19/0.25 per transcript in the blind eval (muninn/docs/CAPTURE4.md), not to 0. The prompt rule that went further lost durable memories and was not shipped. ez50rb was closed with this remainder; this ticket owns it.

## Principle
Recall changes ship only with a measured, blind-judged gain on replayed data (AGENTS.md). Durable memories must not drop. Prefer a deterministic check (agent names from the harness session list) over a prompt rule.

## Relevant details
Eval sets and method: muninn/docs/CAPTURE4.md. Post-check: commit 4cf6414. Agent names: the harness session list, as in ez50rb's agents matcher.

## Depends on

## Acceptance
- [ ] ~~on the CAPTURE4 sets, memories naming an agent are 0 per transcript, durable memories unchanged (blind-judged)~~ (not relevant: 0 per transcript not reached and not pursued (crab 20:12): measured 0.025 / 0.050 on the CAPTURE4 sets with FindIn, 0 durable lost; replaced by check 3, hatchling, 2026-11-17)
- [ ] ~~the same holds on a fresh set fixed before tuning~~ (not relevant: replaced by check 4: same goal stated as measured, not 0, hatchling, 2026-11-17)
- [x] on the CAPTURE4 sets (sample 1, fresh), judge agent-name memories left per agent transcript fall below the shipped check's, durable agent-free memories unchanged (blind-judged)
- [x] on a held-out set fixed before tuning (fresh2), the same holds, measured once against a bar written before the run

## Notes


hatchling 19:44, fresh set fixed before any tuning (check 2): muninn/scripts/capture4_sample.py SEED=1794 EXCLUDE=(CAPTURE4 sample 1 + fresh, 120 sessions) CONTROLS=20 -> 53 items: 40 agent, 13 control (only 13 control sessions left), capture runs 2026-11-06T11:21 .. 2026-11-17T17:18, 0 sessions shared with CAPTURE4. sha256 ccb8f3f5699bcec14c583970cdf802d44b8dc13e6f0dc705fecca0ffbfd4e13a. No output of it read yet.
hatchling 19:52, design and tuning (CAPTURE4 sample 1 + fresh outputs, already judged; no new model calls): the 16 'known name outside agent context' misses (CAPTURE4.md) are memories naming an agent the TRANSCRIPT itself shows in agent context ('coordinator spider' in the chunk, 'spider pushes' in the memory). New agents.Matcher.FindIn(text, transcript): Find(text) plus every name Find(transcript) returns that text uses, not before a tag (Spider-2, dragon-task) and, for common nouns, not after a determiner (the spider). Capture passes the chunk. capture4 gains -rematch (re-runs the matcher on earlier outputs). Tuning numbers (main prompt, 2 runs, per agent transcript, judge agent name or status left): sample 1 shipped 0.188 -> FindIn 0.125, agent-name left 0.087 -> 0.025; fresh 0.250 -> 0.138, name 0.163 -> 0.050; durable agent-free +0.000 [+0.000, +0.000] on both; 14 new drops, all judged agent_name; controls drop 0. What it cannot reach: role words with no agent name ('the foreman', 'root owns', 'mechanic'), judged agent status not name; the name-only goal (0 per transcript) is about names. BAR for fresh2 (stated before its first model call, set fixed 19:44, sha ccb8f3f5...): two runs of the main-prompt extractor (capture4, cap 1024), blind judge capture4_judge.py, agent items: (1) judge agent_name left with FindIn <= 0.05 per transcript and below shipped Find, (2) durable agent-free memories dropped by FindIn but not by Find = 0, (3) controls: 0 drops. Fail on any: park, no variant.
hatchling 19:57, fresh2 measured once against the 19:51 bar: PASSES all three. 2 runs of the main prompt on 53 items (0 errors, 270 + 251 candidates), 468 memories judged blind (capture4_judge.py). Agent transcripts (n=40), judge agent_name left: shipped Find 0.175 -> FindIn 0.050 (bar <= 0.05 and below Find: yes); FindIn vs Find agent name or status left -0.125 [-0.250, -0.025], durable agent-free kept +0.000 [+0.000, +0.000]; 10 newly rejected, all judged agent_name, 0 durable agent-free among them (bar 0: yes); controls (n=13) 0 rejections (bar 0: yes). Check 1 as written says 0 per transcript; reached is 0.025 / 0.050 / 0.050 on the 3 samples. Residual 10 memories over 6 runs: 4 role words without a name ('the foreman verifies every land', judge counts as agent name), 6 'via fox' / 'operator squid' / 'jaguar (comment ...)' where the chunk names that agent only outside the matcher's context words. Not ticking checks 1-2: '0' is not met; asking crab whether the residual is acceptable or a further deterministic step is wanted (which would widen ctx, the store-noun risk ez50rb measured).
file:fresh2-bar.txt (2985 bytes, sha256:c202a163)
file:bar.py.txt (3648 bytes, sha256:cdc282b1)
file:names-view.txt (9164 bytes, sha256:2b4c3e1d)
hatchling 20:04, step 1 landed: 2f5f206 (FindIn), main CI 44164856243. Step 2 per crab 20:01 (session-list names count wherever they stand; no wider ctx; role words left alone). Fresh3 fixed before any step-2 code: capture4_sample.py SEED=2058 EXCLUDE=(sample 1 + fresh + fresh2) -> 40 agent items, 0 controls (no unused control sessions remain), sha256 60e2d1196773f43e543ff566ef1a9f93d44ca4fe2b3d064ce5d87a50962242bc. Rule (decided before fresh3 output): a name from the harness session list with a session written in the last 30 days counts as an agent wherever it stands (not before a tag), EXCEPT agents.go's common nouns (bug, cat, fox, spider, dragon...: keep the context rules) and ordinary words that are names here (ram, cow, mouse, fish, penguin, monkey, turkey, herb, maple, clover, daisy, rose, seedling, mushroom, sunflower, evergreen, palmtree, blossom, horse, dog, owl, eagle, dove, ox, rat, chick, calf, hog, boar, lizard: these keep the context rules too). Tuning check before fresh3: hand-read store memories newly matched (sample) for non-agent hits. BAR on fresh3 (2 main-prompt runs, blind judge, agent items): (1) judge agent_name left below FindIn's on the same outputs; (2) durable agent-free memories rejected by step 2 but not FindIn = 0; plus on sample 1/fresh/fresh2 outputs: 0 durable agent-free newly rejected, 0 control rejections. Fail on any: close typq1e with the residual, no variant.
hatchling 20:06, correction to 20:04: step 1 was NOT in 2f5f206 (that is a v0.7.46 merge by someone else; my first push was rejected non-fast-forward and I misread the output). Rebased and landed now: 7ebea22 (2f5f206..7ebea22), agents+app -race green locally after the rebase.
landed 7ebea22 on main 2026-11-17: hug mem capture: a memory naming an agent its transcript uses as one is rejected…
hatchling 20:12, step 2 (live session names count wherever they stand) measured once on fresh3 against the 20:04 bar: FAILS (1). Not landed, no variant; code kept as attachment step2-live.diff.txt. Fresh3: 2 main-prompt runs on 40 agent items (0 errors), 430 memories judged blind. Per agent transcript, mean of 2 runs: Find (before step 1) rejected 0.163, judge agent name left 0.237, durable agent-free rejected 0.000; FindIn (step 1, on main 7ebea22) rejected 0.312, name left 0.100, durable agent-free rejected 0.013; Live (step 2) identical to FindIn on every fresh3 output (cmp), so bar (1) 'below FindIn' fails (0.100 = 0.100). Why: the names left on fresh3 are goat, fox (common nouns) and daisy (ordinary word), which step 2 excludes by design; 2 more are 'the foreman' (role word, left alone per crab). On the tuning sets step 2 added 1 + 1 rejections over FindIn (s1 run 2, fresh2 run 2), 0 durable lost. Store hand-read: 511 store memories newly matched; read 90 (two random samples, 50 + 40): all name an agent after the joined-identifier exclusion (claude-oauth@claude-otter, cry-wolf, .zebra). NEW FINDING on step 1: on fresh3 FindIn rejected 1 memory the judge rated durable and agent-free (item 1011.0 run 2: a TASK-2034.02 work order that cites '14 EMITDIFF files from tiger's norename probe'; the transcript has 'You take over from tiger (session_tiger_..., gone)'); the near-identical sentence in run 1 was judged agent_name=True durable=True. So step 1's '0 durable lost' holds on 3 samples and is 1 of 80 run-items (0.013 per transcript) on the 4th.
file:step2-live.diff.txt (7256 bytes, sha256:213c8116)
file:three-arms.txt (543 bytes, sha256:d81e1068)
hatchling 20:13, closing per crab (a) 20:12. Check 3: sample 1 judge agent-name left 0.087 -> 0.025, fresh 0.163 -> 0.050 (FindIn vs shipped Find on the same judged outputs), durable agent-free +0.000 [+0.000, +0.000] on both; tests TestFindIn, TestCaptureRejectsAgentNamedInTranscript (fail before). Check 4: fresh2 (seed 1794, fixed 19:44, bar 19:51) 0.175 -> 0.050, -0.125 [-0.250, -0.025], 0 durable lost, 0 control rejections (attached fresh2-bar.txt). Fresh3 caveat as in CAPTURE4.md: 1 of 80 run-items durable-rated lost (item 1011.0 run 2, gone agent tiger, judge split across runs); kept by crab's ruling, which crab reports to Jørgen. Landed 7ebea22 (code), b20d191 (CAPTURE4.md + CHANGELOG). Step 2 (live names) not shipped: no gain on fresh3.
Reflection (dismissed, 2026-11-17, hatchling): One gap: the check text I was given said '0 per transcript' while the principle forbids losing durable memories; the two pull against each other once the easy cases are gone, and nothing in the ticket said which wins. Writing the bar before each run and asking crab at that point worked; a ticket line like 'if 0 costs durable memories, stop at the measured value' would have saved a round.
shipped in v0.7.47 (7ebea22)
## History
- 2026-11-17T20:41 todo -> doing/active (take)
- 2026-11-17T21:13 doing/active -> done (done); was doing_since: 2026-11-17T20:41
