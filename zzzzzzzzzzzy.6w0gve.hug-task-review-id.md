---
id: 6w0gve
title: "hug task review ID --by NAME --show FILE: a move into review, no hand edit"
status: "doing"
type: feature
priority: ""
assignee: ["nautilus"]
labels: []
created: 2026-11-17
updated: 2026-11-17
doing_since: "2026-11-17T19:25"
stage: "active"
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
---

## Goal
A doing task asks for review with one command: hug task review ID --by NAME [--show FILE] (MCP task_update action review, by, show) moves doing/active to doing/review; the owner keeps the task. It writes review_by, review_since and, with --show, review_show. The TUI detail shows the file under the review question, as for wait and design. hug task answer clears review_by, review_since and review_show, and the owner is active again.

## Purpose
Today only the TUI t on a todo/review task writes review_by (moves.go:359). No CLI or MCP move puts a task into review, so asking for one takes a hand edit, and fixes must never need one. A review is often 'look at this', so it carries a picture like wait and design (5z4zt4).

## Principle
Same form as wait/design: --by takes a person's or agent's short name and is required (no implied reviewer). --show must name a file the task carries, or it is refused with a fix line. Tests fail first. Lands after the 6ephqk step 4 rename.

## Relevant details
Plumbing to copy: task.WaitShowing/DesignShowing (moves.go), states.go node fields (add review_show to todo/review and doing/review), answer.go clear lists, tui/picture.go questionPicture map (stage review -> review_show), cli.go allowed flags + case, help.go, mcp server.go action list. Crab agreed 2026-11-17 17:43.

## Depends on

## Acceptance
- [x] hug task review ID --by jhf moves a doing task to todo/review with review_by jhf and review_since, one commit, shown by hug task show
- [x] With a show file, review writes review_show; a file the task does not carry is refused with a fix: line; the TUI detail draws it under the review question
- [x] hug task answer on a review clears review_by, review_since and review_show
- [x] MCP task_update action review with by and show does the same; help and hug task states list the move and the field

## Notes


crab 18:30: doing -> doing/review (owner keeps it), not todo/review: answer on doing/review returns the owner to doing/active, a todo/review answer would leave it unassigned in todo/ready; a review does not count against one-doing-per-agent. Check 1's 'moves a doing task to todo/review' reads doing/review.
landed 77806c9 on main (317f5c2..77806c9). Red first: TestReviewMove failed 'unknown command review'; TestQuestionShowsItsPicture with the old picture.go failed 'the review picture is not under the review question'. check 1: TestReviewMove: doing -> doing/review, review_by jhf, review_since, one commit, show prints 'stage: in review by jhf'. check 2: review_show written; --show nope.ans refused with 'hug task attach ID FILE --name nope.ans'; TUI draws it under '? review: review it'. check 3: answer clears review_by, review_since, review_show; stage active. check 4: MCP bridge row review {by, show} -> review ID --by jhf --show rows.ans (TestToolsSchemasAndRoundTrip); help 'review'; states text 'doing/active   -> doing/review   hug task review ID --by NAME' and review_show (TestStatesTree). -race -count=1: cli 156.7s, task 60.8s, mcp, cmd/hug, tui(-short) 134.3s ok. Binary demo: review -> 'stage: in review by jhf' -> answer -> 'stage: active', 0 review_ fields left.
## History
- 2026-11-17T19:25 todo -> doing/active (take)
