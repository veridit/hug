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
- [ ] muninn search can use FTS5 bm25() (title/body weights) instead of flat +2/+1 word points, behind an option
- [ ] captured memories with source=user weigh more than source=assistant; long memories score by best passage mixed with whole-memory score
- [ ] measured on the recall3 real-memory replay (development + heldout, blind judgments) against current production; ship only with a CI-clear gain
- [ ] recall4 report linked: $JCODE_SCRATCH_DIR/recall4/report-{dev,heldout}.md

## Notes
