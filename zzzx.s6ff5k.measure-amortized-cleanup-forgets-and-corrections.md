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
doing_since: "2026-11-17T16:08"
stage: "waiting"
waiting_since: "2026-11-17T17:38"
waiting_asked_by: "hatchling"
waiting_on: "crab: real replay failed bar part 2 (wrong calls 3 -\u003e 37); park or one named arm?"
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

hatchling 12:20: confirm run (bg 948146oo3l, header4/confirm.json) paused by its load guard at 32 of 140 since 10:42; machine load average 66-84 (uptime 12:19). It resumes on its own when load falls below 16; no result yet, so no header change.

hatchling 13:23: confirm run restarted in resume mode (header4.py RESUME=1 keeps the 32 done jobs in confirm.json, backup confirm.part32.json). Load guard now LOADMAX=40 on the 5-min average (was 16, which held the one serial niced call for 2.5 h at load 26-113 on this 10-core machine). Progress 34/140 at 13:22; bg 7471093wrm.

hatchling 13:43, check 3 confirmation (header4/confirm.json, confirm-report.txt; fresh scenarios stale2, stale-quiet2, offtopic2 + the 4 original; 5 models x 2 reps; rule-scored): H0-main 53/70, H2-server 62/70. H2 vs H0: stale2+stale-quiet2 +10 of 20 [+6, +14] (stale-quiet2 0/10 -> 8/10); offtopic2 +0 of 10 (10/10 both, no wrongful forget); original 4 cases -1 of 40 [-5, +2]; all +9 of 70 [+2, +16]. The -1: deepseek-flash 'newer' rep 1 under H2 answered with a tool call instead of JSON (harness non-answer). Round 1 had the same shape (stale +5 of 20 [+2, +9], originals -1 of 40). H2 = H0 + 'Fix with the muninn MCP tools:'. Not shipped yet: the eval is synthetic and rule-scored, not replayed real messages judged blind, so whether it meets the recall rule is the coordinator's ruling (asked).

hatchling 14:06, crab ruling B: the header is part of what recall shows, so the recall rule applies (real replayed messages, blind judge, fresh confirm). The synthetic result stays as supporting evidence: H2 vs H0 +9 of 70 [+2, +16]; stale-quiet2 0/10 -> 8/10; offtopic2 10/10 both; originals -1 of 40 [-5, +2]. Plan (after tpnd3n): filter recall2's 200-message dev set and the held-out 100 (sessions after 2026-11-05 23:00) for turns where recall showed a memory that was stale or later contradicted; run H0 and H2 on those turns; judge blind as RECALL4 (DeepSeek-Flash, 20-pair spot check); confirm on a fresh slice from sessions after 2026-11-13.

hatchling 15:07, real-replay set (crab ruling B), size reported before running: 40 turns from recall3's 277 frozen messages (25 dev, 15 held-out) where a shown memory was stale. Selection, made before any arm ran: (a) the blind recall3 judge's reason calls the memory stale/superseded/older/contradicted (34 turns, regex in scratch/header4/stale-candidates.json), or (b) the memory is inactive in today's store (forgotten since: 5 turns, sids y6mjp3, c13rat, 06ppp9). File: scratch/header4/real-set.json. Arms H0-main vs H2-server, same 5 models as header4, 1 rep each = 400 runs. Prompt as header4 stale-quiet (the real message, then the block as jcode delivers it, tools: read_file, edit_file, run, your configured MCP tools; list calls as JSON). Scoring: DeepSeek Flash judge, blind to arm, rates each memory_* call 'right' (fixes a stale/wrong note) or 'wrong' (forgets/rewrites a note that still holds), plus a 20-pair spot check by me. PASS BAR (stated before running): H2 minus H0 in turns with a right memory fix > 0 with the 95% paired bootstrap interval above 0, and wrong memory calls H2 <= H0. Then confirm on sessions after 2026-11-13.


hatchling 16:05, scoring fixed before any judge verdict was seen (replay at 232/400): paired (turn, model) cells; a cell scores 1 when the run made at least one memory call the blind judge rated right; d = H2 - H0 per cell; bootstrap of sum(d) as header4 report.py (Random(813), 2000 resamples, 2.5/97.5 pct); wrong = count of judge-'wrong' memory calls per arm. Reported for all 40 turns (the bar) and split development 25 / heldout 15. Script: header4/real_report.py.
file:real-report.txt (634 bytes, sha256:1ccef44f) real replay real-report.txt
file:spot.txt (36333 bytes, sha256:dc7c17b7) real replay spot.txt
file:real-report.py.txt (2651 bytes, sha256:599141ac) real replay real-report.py.txt
file:real-judge.json (25261 bytes, sha256:91e024ea) blind judge verdicts, keyed by run+call hash
hatchling 16:38, real replay RESULT: FAILS the bar stated at 15:07 (H2-server header not shipped). 400/400 runs (5 models x 40 turns x 2 arms; actual models 80 each), 105 memory_* calls, all judged blind by DeepSeek Flash. Scoring fixed at 16:05 before any verdict was read. Part 1 (right-fix turns) PASSES: H2 minus H0 +32 of 200 paired cells, 95% bootstrap [+22, +43]; development +21 of 125 [+13, +30]; heldout +11 of 75 [+5, +17]. Part 2 (wrong memory calls H2 <= H0) FAILS: H0 3 wrong, H2 37 wrong (+34; development +22, heldout +12). Calls: H0 right 7 / wrong 3 / unclear 1 in 9 runs; H2 right 51 / wrong 37 / unclear 6 in 49 runs. Where H2's wrong calls come from (descriptive, not a new arm): forgets 28 of 37 (16 of the turn's selected-stale sid, 12 of another shown note), remembers 9. Forgets overall in H2: 15 right, 28 wrong. So the header makes agents act on stale notes far more (37 vs 5 runs with a right call) but forgets notes that still hold about twice as often as it forgets correctly. Spot check (20 pairs with a memory call, Random(1117), file:spot.txt): I agree with the judge on 18 of 20 pairs' verdicts as I read them; doubtful 2: #17 forget 0n7exf rated right (the message's 5051 passed supersedes 5048, defensible), #3 forget cn6r6f right (rc.04 hold, superseded by rc.11 release: agree). Examples of wrong forgets: turn 51 (2 models) forgot zgf3kp/x27rm4 that the DM confirms; turn 144 (2 models) forgot fhgceg, a still-true root cause; turn 152 forgot a workspace rule on an unrelated message. Files: file:real-report.txt, file:spot.txt, file:real-judge.json, file:real-report.py.txt. Not run: fresh-data confirmation (only for a pass). Per crab: no variant chained; next step is crab's call.
hatchling 16:40, CORRECTION to the 16:38 spot-check line (I miscounted): I agree with the judge on 17 of 20 pairs. Doubtful 3, all H2 calls rated right: #2 (turn 268, sol) remember of a watcher SUCCESS, a new fact rather than a fix of a stale note; #9 (turn 22, flash) remember of a stalled watcher, ephemeral; #17 (turn 258, luna) forget 0n7exf, the newer post-fix baseline, rated right on 5051 vs 5048, which I would call unclear. #3 (forget cn6r6f, rc.04 hold superseded by rc.11) I agree with. So if the judge leans, it leans toward H2 on part 1; the part-2 failure (wrong calls 3 -> 37) stands either way.
## History
- 2026-11-17T16:08 todo -> doing/active (take)
- 2026-11-17T17:33 doing/active -> doing/waiting (wait)
- 2026-11-17T17:34 doing/waiting -> doing/active (resume); was waiting_asked_by: hatchling; waiting_on: replay: ~370/400, judged blind after; waiting_since: 2026-11-17T17:33
- 2026-11-17T17:38 doing/active -> doing/waiting (wait)
