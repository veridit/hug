---
id: khwzpt
title: "hug task answer panics when the stage's question is not numbered"
state: "done"
type: feature
priority: ""
assignee: ["dove"]
labels: []
group: ""
rank: "zzzzzzzzzzzx"
created: 2026-11-20
updated: 2026-11-20
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
parent: "dswkgj"
done_at: "2026-11-20T20:23"
done_release: "v0.7.92"
---

## Goal
A bare hug task answer on a design question outside ## Questions answers it or refuses with a fix: line, never panics.

## Purpose
A Go stack trace is the worst answer a person or agent can get.

## Principle
Every refusal names what is wrong and ends with a fix: line; whether the design stage's own question is answered (q=0) or refused is crab's call.

## Relevant details
Found on branch 5vehza (2026-11-20 19:52 local): new task, hug task ask ID --to jhf 'first?' --as piglet; hug task answer ID yes --as jhf; hug task design ID 'second design?' --for jhf; hug task answer ID ok --as jhf -> panic: runtime error: index out of range [0] with length 0 at huginn/internal/task/asks.go:309 (pickQuestion, openNs[0] with no open numbered question). AnswerQuestion (answer.go:245) calls pickQuestion whenever len(Questions) > 0, though the design question lives in design_* fields, not ## Questions.

## Depends on

## Acceptance
- [x] A test runs the repro and gets either the design question answered or exit 2 with a fix: line, no panic (fails first on main with the panic)

## Notes

crab 2026-11-20 18:53Z design call: answer the design question (q=0). A bare answer answers what the stage shows as waiting; with no open numbered question and a pending design question, that is the design question. Refuse only when nothing is open: exit 2, 'no open question on ID', fix: hug task show ID. Several open numbered questions: today's rule stays. Built by dove on branch 5vehza as a second commit.


check 1: TestAnswerDesignAfterNumberedQuestion (huginn/internal/cli/askwithdrawn_test.go:72). On ea218b3d (before the fix) it failed with 'panic: runtime error: index out of range [0] with length 0' at huginn/internal/task/asks.go:309 (pickQuestion). Fix 13ce8145 (crab's ruling, q=0): a bare answer with no open numbered question answers the design question ('answered design, by jhf'); with nothing open it exits 2, 'no open question on ID', fix: hug task show ID. Mutants (answer.go restored, shasum 062c3fad before and after): bare := n == 0 && false -> fails at :80 exit 2; refusal without 'no open question on' -> fails at :88. CI run 44799862741 on 13ce8145da04e1eeeb9fc002145e26741335bc7c: all 8 jobs success.
Correction to check 1 evidence: TestAnswerDesignAfterNumberedQuestion is at huginn/internal/cli/askwithdrawn_test.go:71, not :72.
landed 13ce814 on main 2026-11-20: hug task answer answers a design question after numbered ones (khwzpt)
shipped in v0.7.92 (13ce814)
## History
- 2026-11-20T19:58 todo -> doing/active (take with 5vehza)
- 2026-11-20T20:23 doing/active -> done (done, by release v0.7.92: every check holds); was doing_since: 2026-11-20T19:58; doing_with: 5vehza
