---
id: c3hszg
title: "Recall: user-weighted BM25F and best-passage scoring in muninn search"
state: "doing"
type: spike
priority: ""
assignee: ["crab"]
labels: ["recall"]
created: 2026-11-13
updated: 2026-11-13
parent: "d634kr"
group: ""
rank: "zzzz"
aliases: ["recall-user-weighted-bm25f-and-best"]
doing_since: "2026-11-18T17:19"
stage: "waiting"
model: "claude-opus-5-5"
branch: "c3hszg"
worktree: "/Users/jhf/.jcode/scratch/c3hszg"
waiting_on: "memories with a captured source (user weighting cannot be measured until then; best-passage ruled out 11-18 18:35 local, doc landed v0.7.68)"
waiting_since: "2026-11-20T04:25"
waiting_asked_by: "crab"
---

## Goal
Find out whether the two ideas that survived recall4 on LongMemEval_S also help muninn on real memories, and ship whichever does

## Purpose
recall4 (2026-11-13) tested candidate scoring ideas on LongMemEval_S, the outside benchmark, and two survived on held-out data: weighting the user's words 5x (+0.084 [+0.053,+0.119]) and best-passage scoring (+0.045 [+0.016,+0.077]). LongMemEval is only an outside check: muninn changes are judged on Jørgen's real traffic (memory q7vhnf), so each idea must prove itself on real memories before it ships. BM25 scoring came out of the same work and shipped in v0.7.2 (2026-11-14).

## Principle
The recall rule (AGENTS.md, epic d634kr): ship only a measured, CI-clear gain on real replayed messages, confirmed on fresh data and judged blind (muninn/docs/RECALL4.md). One idea at a time against shipped recall, each behind an option. Recall stays local and free. User weighting waits until memories carry a captured source, since it cannot be measured before.

## Relevant details
Open is item 2 only. User weighting is untestable today: 0 of 11,885 memories in the frozen snapshot have a captured source (RECALL4.md). Best-passage is untested on real memories. Shipped baseline: v0.7.2 bm25_scale 2.0 (long +0.39 rel/msg [+0.25,+0.55], short +0.01 n.s.). Fresh check 2026-11-14 (147 messages): long irrelevant +0.41 [+0.16,+0.64], because 59 of 64 long messages are one templated statbus review job where a ticket-id match is not task relevance. Sets: 277 frozen (200 dev, 77 held-out), 147 fresh. Scripts: muninn/scripts/run_real.py, judge_real.py; LongMemEval: $JCODE_SCRATCH_DIR/recall4/recall4.py.

## Depends on

## Acceptance
- [x] muninn search can use FTS5 bm25() (title/body weights) instead of flat +2/+1 word points, behind an option
- [ ] captured memories with source=user weigh more than source=assistant; long memories score by best passage mixed with whole-memory score
- [x] measured on the recall3 real-memory replay (development + heldout, blind judgments) against current production; ship only with a CI-clear gain
- [x] recall4 report linked: $JCODE_SCRATCH_DIR/recall4/report-{dev,heldout}.md

## Notes

recall4 (LongMemEval_S, round level, all@5, paired bootstrap vs plain BM25): heldout n=379: user-weight x5 +0.084 [+0.053,+0.119]; best-passage 0.65 +0.045 [+0.016,+0.077]; stemming +0.008 (n.s.); coverage +0.008..0.011 (n.s.); recency +0.008..0.016 (n.s.); session-spread -0.053 [-0.090,-0.016]. dev n=91: user x10 +0.066 [0.000,+0.132]; dense alone 0.527 vs bm25 0.692. Script: $JCODE_SCRATCH_DIR/recall4/recall4.py

Shipped v0.7.2 (2026-11-14): BM25 keyword scoring, bm25_scale 2.0. Real memories, 277 frozen messages, blind judge: long +0.39 rel/msg [+0.25,+0.55], irr -0.07 [-0.22,+0.10]; held-out long +0.26 [+0.03,+0.48]; short +0.01 [-0.04,+0.06]. Default reproduces measured arm on 277/277 messages. Rejected: rarest-12 terms, no all-words bonus, relative floor 0.50, no gates. Item 2 open: user-weighting needs captured source (0/11,885 memories have it) and best-passage is untested on real memories. Evidence: muninn/docs/RECALL4.md.

Fresh check 2026-11-14 (147 post-freeze messages, sessions started after 6 Nov, frozen 5 Nov snapshot, blind judge). Short typed n=83 (6 sessions, below the 8-session bar): BM25 rel +0.02 [0.00,+0.05], irr -0.07 [-0.12,-0.02]: confirmed, fewer irrelevant. Long n=64: rel +0.05 [-0.05,+0.16], irr +0.41 [+0.16,+0.64]: worse. 59 of 64 long are one templated statbus job ('You review one Backlog ticket...'); BM25 matches the ticket id to memories about that ticket's history, which the judge rates irrelevant to a description-style review. Failure mode: rare-id match is not task relevance. Artifacts: $JCODE_SCRATCH_DIR/recall4/real/report-fresh.md


piglet 18:34, best-passage half of check 2: measured and NOT shipped (no gain). Branch c3hszg d84ae1aa on c84d3273, pushed with no ci/. passage_mix option, default off, recall4 variant passageNN. Tests: TestBestPassageLiftsAFocusedLongMemory fails with the mix not applied ('the long memory ranks 7, want 1'), TestPassageMixOffIsUnchanged, TestPassageScoreIsFTS5BM25 (= bm25() within 1e-9). Arms vs shipped default (bm25_scale 2.0; default+at1118 reproduces the 14 Nov run 200/200), blind judge, 4 paid calls of the 300 cap. Long, rel / irr delta [95% session-cluster CI], mix 0.35 then mix 0.65: dev n=103 +0.00 [-0.03,+0.03] / +0.02 [+0.00,+0.07]; -0.01 [-0.06,+0.03] / +0.01 [-0.04,+0.07]. Held-out n=50 -0.02 [-0.08,+0.00] / +0.02 [-0.05,+0.13]; +0.00 [-0.06,+0.06] / -0.02 [-0.15,+0.13]. Fresh n=64 +0.00 [+0.00,+0.00] / +0.05 [-0.03,+0.13]; +0.00 [+0.00,+0.00] / +0.08 [-0.02,+0.18]. Short: +0.00 on every split. Changed injections: dev 4 and 6 of 200, held-out 8 and 16 of 77, fresh 31 and 43 of 147. Report: scratch/recall4/real/c3hszg-report.txt; RECALL4.md section on the branch. User weighting stays untestable (0 of 11,885 memories have a source).
crab 18:35 local, ruling: null result accepted (no gain on dev/held-out/fresh, 4 paid calls). Code not landed: branch c3hszg d84ae1aa stays as the record; only the RECALL4.md section lands (piglet, ci/c3hszg-doc).
landed on main 8c26ec51 (CI 44363087479 6/6, windows-cli 549.357s)
landed 8c26ec5 on main 2026-11-18: RECALL4.md: best-passage scoring measured on real memories, not shipped (c3hszg)
shipped in v0.7.68 (8c26ec5)
## History
- 2026-11-18T17:19 todo -> doing/active (take)
- 2026-11-18T18:34 doing/active -> doing/waiting (wait)
- 2026-11-20T02:45 handed to crab: piglet's session went silent (last seen 2026-11-19 02:41)
