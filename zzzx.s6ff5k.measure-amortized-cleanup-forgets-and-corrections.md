---
id: s6ff5k
title: "Measure amortized cleanup: forgets and corrections after recall"
status: "doing"
type: feature
priority: ""
assignee: ["hatchling"]
labels: []
created: 2026-11-12
updated: 2026-11-12
parent: "d634kr"
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
---

## Goal
Know whether agents act on the recall header's Fix line, and add an off-topic signal that never deletes

## Depends on

## Acceptance
- [x] stats: share of recall blocks followed by memory_forget/remember of a shown id within the session
- [ ] memory_feedback <id> off-topic lowers recall of that memory for similar messages, no deletion
- [ ] header wording change only if the 5-agent eval does not regress

## Notes

Item 1 measured 2026-11-16 (local data only): of 454 jcode sessions that received recall injections (injections table, keys jcode:<session>), 0 called memory_forget as a tool (counted from tool-use records '"name":"mcp__muninn__memory_forget"', not text mentions; the recall header itself contains 'memory_forget'), and 13 memory_remember calls occurred across them. Agents essentially never act on the header's Fix line. Implication for item 2: cleanup must not depend on agents volunteering it; an off-topic feedback signal needs to be cheap and prompted, or inferred.

2026-11-16 feedback from another agent on the recall header (via Jørgen): it did not act on 'Fix: memory_forget <id>, memory_remember' because (1) it read the header as a caution, not a task; (2) 'not the user' read as 'not mine to edit'; (3) it assumed the tools were missing and did not check; (4) it was in wait-for-owner mode; (5) the .0 .1 .2 table hides the one instruction. Suggested: plain imperative ('fix it now without asking'), 'part of your job', name the tool server, one note per line. Coordinator's reading: consistent with item 1 (0 of 454 sessions called memory_forget). The 2026-11-10 5-agent test (b56828e: F1 38/40 vs old 28/40, old 0/10 on fixing a wrong note) measured whether agents know HOW to fix a note once it is plainly wrong; it did not measure whether they act unprompted mid-task. Risk in the suggestion: 'fix it now' without separating wrong from merely off-topic would delete good memories; off-topic must never delete (item 2). Proposal for item 3: rerun the 5-agent test with two added scenarios, (a) a stale note appears while the agent works on something else (measures unprompted fixing), (b) an off-topic but true note appears (must NOT be forgotten); compare F1 against the suggested wording and a variant that only names the server. Ship only a wording that wins (a) without losing (b) or the original scenarios.

2026-11-16 Jørgen: queue the header test. He tested the memory format a lot and it scored very well; the odd '.0 .1 .2' table is deliberate, it looks otherworldly so it stands apart from the normal text flow. TSV is one of his favourite formats, but jcode shows tabs with no visible space between columns. So the test keeps the current format as the baseline and varies only the header line, unless a format variant is added as its own arm.

guppy 2026-11-17 11:05, check 2 measured offline before building ($JCODE_SCRATCH_DIR/header4/offtopic_sim.py; recall4's frozen replay, default arm, blind verdicts in recall4/real/judgments.json, cached nomic query vectors). Oracle feedback: every development hit the judge called irrelevant (94 hits, 77 memories) counts as 'memory_feedback ID off-topic', applied to the same memory on later messages with cosine >= T to a feedback message. Held-out (77 msgs): T=0.7 drops 16 irrelevant and 10 relevant hits, irr/msg -0.208 [-0.419, -0.097], rel/msg -0.130; T=0.9 drops 5 irrelevant, 6 relevant. Fresh (147 msgs): T=0.6-0.8 drops 3 irrelevant, 0 relevant, irr/msg -0.020 [-0.052, +0.000] (not CI-clear). Reading: a memory off-topic for one message is often on-topic for a similar one (of 8 memories irrelevant twice on dev, 4 were also judged relevant there), so per-memory similar-message suppression costs relevant recall at every threshold, even with perfect feedback. Not built; by the recall rule (AGENTS.md) it cannot ship. A different signal (e.g. per-session, or demoting only below a score margin) would need its own eval. Check 3 eval running: $JCODE_SCRATCH_DIR/header4/ (3 headers x 7 cases x 5 models x 2).

guppy 2026-11-17 11:39, check 3 round 1 ($JCODE_SCRATCH_DIR/header4/: header4.py, results.json, report.txt). The b56828e harness (tool-less jcode run, block as [transcription], rule-scored JSON), only the header line varies, 5 models (flash, luna, haiku 5.5, sonnet 5.5, sol 6.1) x 2 reps x 7 cases = 70 per arm. New cases: stale (a stale note surfaces mid-task, user's message contradicts it, memory tools listed), stale-quiet (same, only 'your configured MCP tools'), offtopic (true unrelated note: must not be forgotten). H0 main 59/70 (stale 8/10, stale-quiet 3/10, offtopic 10/10, original 4 cases 38/40). H1 imperative ('A wrong note is yours to fix, now, without asking ... (muninn MCP tools). An off-topic note is not wrong: leave it.') 63/70: stale+quiet +7 of 20 [+3, +11], offtopic 10/10, original 4 -3 of 40 [-7, +0] (flash answered by calling tools twice, luna obey 2x, haiku obey 1x). H2 server ('Fix with the muninn MCP tools: memory_forget <id>, memory_remember') 63/70: stale+quiet +5 of 20 [+2, +9], offtopic 10/10, original -1 of 40 [-3, +0] (sol source 1x). Confirmation on fresh scenarios (stale2, stale-quiet2, offtopic2: other facts and ids) H0 vs H2 running as confirm.json.

guppy (hatchling) 12:41, back to todo (not being worked on now): no code branch yet. Check 2 measured offline (header4/offtopic_sim.py), not built: fresh only 3 irrelevant dropped, not CI-clear. Check 3 header eval in $JCODE_SCRATCH_DIR/header4 (H0 59/70, H2 63/70); confirmation run H0 vs H2 on fresh scenarios is bg task 948146oo3l writing header4/confirm.json; decide when it finishes.
