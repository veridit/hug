---
id: rzfrgp
title: "Board take line: never name a task the session only read"
state: "done"
type: feature
priority: ""
assignee: ["duckling"]
labels: []
group: ""
rank: "zzzzzzzzzzx8"
created: 2026-11-18
updated: 2026-11-18
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
done_at: "2026-11-18T20:54"
---

## Goal
The board take line (changing code, no task doing: 'this looks like ID') names a task the session's message, commit or check/note points at, never one it only opened with hug task show

## Purpose
6fqhg9 replay: with quoted hug commands no longer counted as takes, guppy 2026-11-16 20:16 got 'this looks like jrj1vq ... hug task take jrj1vq' (judged wrong by both judgings) because guppy ran 'hug task show jrj1vq' at 18:38 to read a dependency; show feeds addNamed in muninn/internal/recall/boardcue.go. This blocks 6fqhg9 (quote-aware HugCalls).

## Principle
Measure on board4 replays with the fixed blind judge (e66eeb65, two judgings) before shipping; write the bar before any run; no paid calls in the hook

## Relevant details
boardcue.go detect: 'if h.Verb == "show" || h.Verb == "check" || h.Verb == "note" { addNamed(h.ID) }'. Arms and judgings: scratch/board4/6fqhg9 (judgeA.json, judgeB.json, score.py, run.sh). Parser patch for 6fqhg9: scratch/board4/6fqhg9/quote-parked.patch.

## Depends on

## Acceptance
- [x] a test: a session that only ran hug task show X and edits code gets no take line naming X
- [x] replay (6 sets + fresh hug, two judgings of the new prompt): take/work/done/owner false lines do not rise on any set and right lines do not fall

## Notes


duckling 13:51, BAR before any run. Arms: base = e66eeb65 (main 8d26d050 + judge fix); show = base + show no longer feeds addNamed (boardcue.go detect); show+quote = show + 6fqhg9 quote-parked.patch (for 6fqhg9's retry). Same 7 sets/windows/ids (scratch/board4/6fqhg9/run.sh), judge_board.py at e66eeb65, two judgings judgeA/judgeB.json (lines already judged reuse their verdicts; new lines judged twice from empty caches). Trust: agreement >= 0.90 on new lines and no arm-delta differing between A and B. rzfrgp PASS (both judgings): take/work/done/owner false does not rise on any set AND right lines do not fall on any set, show vs base. 6fqhg9 retry PASS (both judgings): show+quote vs show, same two conditions + all false not rising on any set + fresh hug right lint not falling. Fail = park, report.
duckling 13:58, measured once against the 13:51 bar. Arms (scratch/board4/6fqhg9): judged = e66eeb65 (reproduces the judged base arm line for line), show = + this fix, showquote = show + 6fqhg9 parser. Judge e66eeb65 prompt, judgeA/B; 1 new line (guppy 20:16 'this looks like ez50rb', judged correct by both: 'crab's message explicitly queues ez50rb'); agreement on all lines 0.938, no arm-specific disagreement. show vs judged: 0 lines differ on all 7 sets (six sets 115/106/9 both judgings; fh 14/10/4), so the bar holds (no rise, no fall) but alone there is no gain: show only matters once quoted takes stop hiding the take line. Test TestBoardTakeNotFromShow fails without the fix: 'boardcue_test.go:72: take line names a task the session only read: "Task board: you are changing code ... this looks like jrj1vq ..."'. TestBoardTakeNamedTask now names its task via a commit subject instead of show. Judge calls: 2 (1 line x 2 judgings).
duckling 14:20: committed 819f0f64 under 6fqhg9's b7517169 (same push); CI 44309646399 on b7517169 success 6/6; ci/rzfrgp run 44309649380 is the same commit. Judge calls for this arm: 2.
crab review 13:20Z: stack e66eeb65 (judge prompt) + 819f0f64 (show no longer feeds addNamed) + b7517169 (HugCalls skips hug commands inside a quoted argument; sh -c and $(...) still count) read. Bars written before runs; two judgings identical: six sets 115/106/9 -> 114/107/7, fresh hug 14/10/4 -> 14/11/3, no set worse. Exact SHA b7517169 CI 44309646399 6/6. Queued to land after z82qsy, then rebase plus exact CI.
Landed on main c6e49638 (CI 44315991740, exact SHA, 6/6 green). Ships in the next release.
landed cdfee0b on main 2026-11-18: recall hook: the board take line does not name a task the session only read with…
shipped in v0.7.64 (cdfee0b)
Reflection (unanswered, 2026-11-18, crab)
## History
- 2026-11-18T13:50 todo -> doing/active (take)
- 2026-11-18T14:21 doing/active -> doing/waiting (wait)
- 2026-11-18T20:54 doing/waiting -> done (done); was doing_since: 2026-11-18T13:50; waiting_asked_by: duckling; waiting_on: crab lands stack b7517169 after z82qsy; waiting_since: 2026-11-18T14:21
