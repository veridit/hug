---
id: c3hszg
title: "Recall: user-weighted BM25F and best-passage scoring in muninn search"
status: todo
type: spike
priority: ""
assignee: []
labels: ["recall"]
created: 2026-11-13
updated: 2026-11-13
---

## Goal
Find out whether the two ideas that survived recall4 on LongMemEval_S also help muninn on real memories, and ship whichever does

## Depends on

## Acceptance
- [x] muninn search can use FTS5 bm25() (title/body weights) instead of flat +2/+1 word points, behind an option
- [ ] captured memories with source=user weigh more than source=assistant; long memories score by best passage mixed with whole-memory score
- [x] measured on the recall3 real-memory replay (development + heldout, blind judgments) against current production; ship only with a CI-clear gain
- [x] recall4 report linked: $JCODE_SCRATCH_DIR/recall4/report-{dev,heldout}.md

## Notes

recall4 (LongMemEval_S, round level, all@5, paired bootstrap vs plain BM25): heldout n=379: user-weight x5 +0.084 [+0.053,+0.119]; best-passage 0.65 +0.045 [+0.016,+0.077]; stemming +0.008 (n.s.); coverage +0.008..0.011 (n.s.); recency +0.008..0.016 (n.s.); session-spread -0.053 [-0.090,-0.016]. dev n=91: user x10 +0.066 [0.000,+0.132]; dense alone 0.527 vs bm25 0.692. Script: $JCODE_SCRATCH_DIR/recall4/recall4.py
