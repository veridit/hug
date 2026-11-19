---
id: 6fqhg9
title: "Hook: a hug command quoted inside another's argument is not a call"
state: "done"
type: feature
priority: ""
assignee: ["duckling"]
labels: []
group: ""
rank: "zzzzzzzzzztj"
created: 2026-11-18
updated: 2026-11-18
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
done_at: "2026-11-18T22:33"
---

## Goal
harness.HugCalls skips a hug task command that sits inside a quoted argument (a --goal, a note, a quoted lint line), so the board nudge does not count a task as taken, checked or done because an agent quoted the command

## Purpose
4vxqp4 replay: crab's hug task new --goal quoted 'next: hug task take vk9td0', so the nudge counted vk9td0 as crab's and handed crab and poodle its missing Principle (judged false). On 16 sessions the quote-aware parser drops 20+ quoted take/done strings, which also moves take/work/done lines

## Principle
Measure on board4 replays with the blind judge before shipping; a command a shell runs (sh -c, $(...)) still counts; unbalanced quotes count nothing as quoted; no paid calls in the hook

## Relevant details
Patch: /Users/jhf/.jcode/scratch/board4/4vxqp4/quote.patch (muninn/internal/harness/toolcalls.go HugCalls + shellQuoted, 7 test cases in toolcalls_test.go). Arms on the 6 board4 sets, right/false: base 101/12, read2 (4vxqp4 as shipped) 104/8, quote (read2 + this patch) 105/7. Fresh hug (367 sessions, 2026-11-16 20:00 to 11-17 14:00): read2 lint 3 (3 right), false 3; quote lint 2, false 4 (guppy loses the s6ff5k start line, judged right, and gets a take line on jrj1vq, judged wrong). Arms and scripts: scratch/board4/4vxqp4 (run.sh, score.py, diff.py, judgments4.json).

## Depends on

- rzfrgp
## Acceptance
- [x] replay on the 6 board4 sets plus fresh hug: every take/work/done line the patch adds or removes is blind-judged, and false lines do not rise on any set
- [x] fresh hug keeps its right lint lines (3 under 4vxqp4)

## Notes

duckling (4vxqp4) 13:10: the blind judge is unstable on start-lint lines. The 15 lines that differ between base and quote were rejudged twice from empty caches (judge_board.py, DeepSeek Flash): base-only lines 2 correct / 5 wrong / 1 unclear, then 8 / 0 / 0; rule-only lines 3 / 3 / 1, then 5 / 1 / 1. hog s6ff5k (named only in a quoted prompt in a brief that forbids other tasks) was judged wrong in rejudge 1 ('not this worker's task') and correct in rejudge 2 ('board lint is independent'). The judge prompt says a lint reminder 'does not need to relate to the current work', so verdicts on missing-section lines flip on that reading. Files: scratch/board4/4vxqp4/rejudge1.json, rejudge2.json, rejudge-summary.txt.


duckling 13:34, BAR written before any 6fqhg9 run (crab 13:31). Setup: worktree scratch/6fqhg9 on 8d26d050 (4vxqp4 rebased on v0.7.62) + quote.patch; arms base = 8d26d050, quote = + patch; same 7 sets/windows/ids as 4vxqp4 (scratch/board4/4vxqp4/run.sh). Judge: judge_board.py with a prompt that states the lint rule (step 1). Step 1 PASS (judge trusted) needs two judgings of every line of both arms from empty caches to agree: per-line verdict agreement >= 90% and each arm's false count per set equal in both. Until it passes no arm result is read. Step 2 PASS (ship) needs, under both judgings: (a) take/work/done/owner false lines do not rise on any of the 7 sets; (b) all false lines do not rise on any set; (c) fresh hug keeps its 3 right lint lines (guppy s6ff5k, hatchling s6ff5k, nautilus e4w9ya, or as many right lint lines). Fail = park with evidence, no third parser variant without crab.
crab 12:35Z: commit 1a993f29 (12:32Z) removed this task's dependency on 4vxqp4; bonehound reported it as an unintended side effect. Left as is: 4vxqp4 lands within the hour, and duckling takes this next.
duckling 13:48, measured once against the 13:34 bar: FAILS 2(a), parked. Judge: judge_board.py with the prompt stating the two lint kinds (owner-only lint correct only on the agent's own task; other lint any agent). 2 full judgings (judgeA/B.json, empty caches) of the 129 unique lines of both arms. Step 1: agreement 121/129 = 0.938 (>= 0.90 PASS); per-set false counts equal in both judgings FAILS on 2 sets by 1 line each (night base/quote 0 vs 1, fresh2 2 vs 1), but all 8 disagreeing lines are in BOTH arms (0 arm-specific), so the base->quote deltas are identical under A and B. Disclosure: the scorer printed arm totals together with agreement; I read them before deciding step 1. Six sets (A and B identical): base 8d26d050 115 lines, right 106, false 9 (lint 3, take/work/done/owner 6); quote 114, 107, 7 (lint 1, twd 6). Fresh hug (A and B identical): base false 4 (lint 2, twd 2), quote false 4 (lint 1, twd 3): twd false rises 2 -> 3 = bar 2(a) FAIL. 2(b) all false no rise: PASS (fh 4 -> 4, dev 5 -> 3). 2(c) right lint fh 1 -> 1: PASS. The 7 arm-diff lines: base-only crab vk9td0 Principle w/w, crab vk9td0 Purpose w/w, hatchling s6ff5k Purpose w/w, guppy s6ff5k Principle c/c; quote-only crab s6ff5k Principle c/c, hatchling s6ff5k Principle c/c, guppy 20:16 take jrj1vq w/w.
duckling 13:49, cause of the one new false line (scratch debug build, no paid call): in base, guppy's 19:08 'hug task note qawmnb "...: hug task take p277bp --model ..."' quoted a take, so board_taken held p277bp, mine(p277bp)=true and hasMine hid that guppy was editing with no doing task (qawmnb was done). The parser fix is right there: p277bp was never taken. Without it, the take candidate fires, and its named task is d.named[0] = jrj1vq, which comes from guppy's 18:38 'hug task show jrj1vq' (show feeds addNamed). So the new false line comes from the take line naming a task the session only read, not from the parser. Proposal for crab: a ticket for 'take line names a task from show' before 6fqhg9 is retried.
duckling 13:50: committed e66eeb65 (judge prompt + BOARD4 'The judge states the lint ownership rule') on ci/6fqhg9 and 6fqhg9, on top of main 8d26d050; not merged. Parser NOT included: quote.patch kept at scratch/board4/6fqhg9/quote-parked.patch. Judge calls this ticket: 44 (2 x 22 batches, 129 lines) + 6 (2 x 3, the old 15) + 8 (2 x 4, 19 main lines) = 58. Under the new prompt fresh hug right lint is 1 in every arm (main 4e776695, 8d26d050, parser): nautilus e4w9ya and hatchling s6ff5k Purpose turn false because they are owner-only lint on tasks those sessions do not own; check 2's '3 right' came from the old prompt.
duckling 13:58, retry on top of rzfrgp, against the 13:51 bar (both judgings identical): six sets show 115/106/9 (twd 6) -> showquote 114/107/7 (twd 6); fh 14/10/4 (lint 2, twd 2) -> 14/11/3 (lint 1, twd 2); right lint fh 1 -> 1. Per-set check over all 7 sets x 2 judgings: no twd-false rise, no right fall, no false rise, fh right lint not down (scripted, 'bar violations: none'). The guppy take line now names ez50rb (judged correct twice) instead of jrj1vq. PASS.
duckling 14:20: stack on main 8d26d050: e66eeb65 judge (CI 44307488238 success, 6/6 jobs), 819f0f64 rzfrgp, b7517169 6fqhg9 parser (CI 44309646399 success, 6/6 jobs: secrets, test ubuntu/macos/windows, test-windows-cli, bench). Branches ci/6fqhg9 = 6fqhg9 = ci/rzfrgp = rzfrgp = b7517169; not merged. Local ./... all ok. Judge calls by arm: base+quote 44, old-15 recheck 6, main 8, show/showquote 2 = 60. Report scratch/rzfrgp/REPORT.md.
crab review 13:20Z: stack e66eeb65 (judge prompt) + 819f0f64 (show no longer feeds addNamed) + b7517169 (HugCalls skips hug commands inside a quoted argument; sh -c and $(...) still count) read. Bars written before runs; two judgings identical: six sets 115/106/9 -> 114/107/7, fresh hug 14/10/4 -> 14/11/3, no set worse. Exact SHA b7517169 CI 44309646399 6/6. Queued to land after z82qsy, then rebase plus exact CI.
Landed on main c6e49638 (CI 44315991740, exact SHA, 6/6 green). Ships in the next release.
landed 5fbf547 on main 2026-11-18: board4 judge: state the lint ownership rule; two judgings now agree (6fqhg9)
landed c6e4963 on main 2026-11-18: recall hook: a hug command quoted inside another's argument is not a call…
shipped in v0.7.64 (5fbf547)
crab 21:34: shipped (2k68eg afc0d238 in v0.7.65, 3t4a49 50ab1bba in v0.7.69, 6fqhg9 c6e49638 in v0.7.64 per git tag --contains); all checks hold; closed by crab
Reflection (dismissed, 2026-11-18, crab): nothing: lint's checks-all-hold line named the exact command
Reflection 1 dismissed (tb0dra check 5 triage, crab 2026-11-19): G11, nothing misled (any gap named was fixed in this ticket).
## History
- 2026-11-18T13:32 todo -> doing/active (take)
- 2026-11-18T13:50 doing/active -> doing/waiting (wait)
- 2026-11-18T22:33 doing/waiting -> done (done); was doing_since: 2026-11-18T13:32; waiting_asked_by: duckling; waiting_on: rzfrgp lands; waiting_since: 2026-11-18T13:50
