---
id: nanvnt
title: "A withdrawn question reads as withdrawn, never as open"
state: "done"
type: feature
priority: ""
assignee: ["piglet"]
labels: []
group: ""
rank: "zzzzzzzzzzsv"
created: 2026-11-18
updated: 2026-11-18
model: "claude-opus-5-5"
branch: "hpqt99"
worktree: "/Users/jhf/.jcode/scratch/hpqt99"
done_at: "2026-11-19T01:15"
---

## Goal
A question withdrawn by hug task resume (or by leaving asked) shows as withdrawn wherever it is drawn: hug task show prints 'withdrawn <at>' with it, the TUI detail draws it dim with 'withdrawn <at>', and ASKED never counts or lists it.

## Purpose
Found on hpqt99 2026-11-18: question 1 was withdrawn because its wording was wrong (it called 2kb574 a leak), and question 2 replaced it. hug task show hpqt99 still printed 'question 1 to jhf (check 1): ... one leak today: 2kb574 ...' with no marker, so Jørgen, the person it was asked of, would read a withdrawn question as open. The JSON had it right: questions[0].withdrawn = {at: 2026-11-18T23:32, text: doing/asked -> doing/active}.

## Principle
Display only: the ## Questions format (iwazaru's q2adr8 core) and what withdraws a question stay as they are. A withdrawn question keeps its text, for the record, but never looks askable. Tests fail first; TUI goldens if a screen changes.

## Relevant details
huginn/internal/cli/askquestions.go questionText renders q.Answer and has no q.Withdrawn branch. huginn/internal/tui/detail.go questionRecord heads a withdrawn one '· withdrawn' (dim) but draws its text lines plain and gives no time. ASKED reads task.OpenQuestions (asks.go:138), which already skips withdrawn ones: pin it with a test. The plain CLI show has no styling, so its marker is text.

## Depends on

## Acceptance
- [x] hug task show prints a withdrawn question with 'withdrawn <at> (<why>)' and no reading as open (test, failing first)
- [x] the TUI detail draws a withdrawn question dim, head and text, with 'withdrawn <at>' (test or golden, failing first)
- [x] ASKED neither lists nor counts a withdrawn question, also when the task holds a second open one (test)

## Notes


Branch nanvnt / ci/nanvnt at 0d494b92 on main 356fb240 (tests cedc3248 failed first: TestShowMarksWithdrawnQuestion 'question 1 is withdrawn but show does not say so'; TestWithdrawnQuestionDrawnDim 'question 1 not marked withdrawn with its time'). TestAskedSkipsWithdrawnQuestion passed before the fix: ASKED already uses OpenQuestions; it pins that. 5 mutants all fail (no time on TUI head; TUI text not dim; CLI no withdrawn line; CLI no why; Open() counting withdrawn -> ASKED test). No golden changed (grep: no golden held '· withdrawn'). Local -race: task ok 49 s; tui hit the 20 m timeout at load ~270 (reload_test.go:90, in progress of being rechecked without -race). CI 44420635338 running.
landed e8eca02 on main 2026-11-19: test: a withdrawn question reads as withdrawn (nanvnt, failing)
landed f17745a on main 2026-11-19: hug task show and the TUI mark a withdrawn question (nanvnt)
shipped in v0.7.73 (e8eca02)
check 1: Shipped in v0.7.73 (f17745ad on main; tests-first e8eca029). TestShowMarksWithdrawnQuestion failed first ('question 1 is withdrawn but show does not say so'), passes after; show prints '  withdrawn <at> (<why>)'. CI 44420635338 success on all 8 jobs (0d494b92, same tree).
check 2: Shipped in v0.7.73 (f17745ad). TestWithdrawnQuestionDrawnDim failed first ('question 1 not marked withdrawn with its time'); head now '· withdrawn 2026-11-17 09:50', text lines rendered with styles.dim. Mutants: no time / not dim both fail. No golden held '· withdrawn'. CI 44420635338 success.
check 3: Shipped in v0.7.73 (f17745ad). TestAskedSkipsWithdrawnQuestion: one withdrawn + one open counts once (ASKED [3]); a task whose only question is withdrawn leaves ASKED (count 3 -> 2). Passed before the fix (ASKED uses OpenQuestions), so it pins; mutant Open() counting withdrawn fails it. CI 44420635338 success.
Reflection (open, 2026-11-19, piglet): nothing misled: 'Ask each question on its own' (q2adr8 refusal) made me rephrase the re-ask as one choice; the gap was the bug itself, a withdrawn question drawn as open.
## History
- 2026-11-18T23:34 todo -> doing/active (take)
- 2026-11-19T01:15 doing/active -> done (done); was doing_since: 2026-11-18T23:34
