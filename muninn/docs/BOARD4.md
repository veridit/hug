# Board4: board nudges in the recall hook (2026-11-16, qawmnb)

**Shipped:** the recall hook adds at most one line about the task board, only
when the session's work and the board disagree, and the line carries the
command that fixes it. `"nudge_board": false` in muninn config.json turns it
off. Nothing paid runs in the hook: it reads the board cache and the
session's own transcript.

| kind | when | line ends with |
|---|---|---|
| work | the session ticks a check on a todo task, or notes one while editing code | `hug task take ID --model M` |
| take | code is changing, no task is the session's, for a 2nd turn | `hug task take ID ...` for a task the work named or whose title clearly matches, else `hug task next` / `hug task new` |
| done | a task that is the session's has every check ticked | `hug task done ID` |
| lint | at a break (start, return, wrap-up) | the finding's own `hug task ... ID` fix |

A task is the session's when it took it, its name is the assignee, or the
task's recorded worktree (linked worktree) or branch (off main) is where the
session works. Never on model alone. Rates: take and work share one line per
hour per session, done one per hour, lint one per 6 h; the same line about the
same task waits 24 h; a lint finding handed to one session rests 7 days
before anyone gets it again. Board-wide fixes (`hug upgrade`) are not handed
out.

## Replay (muninn/cmd/board4)

Every turn of real jcode sessions is replayed through the production
`BoardCue` with the board as it was at that minute, rebuilt from the
`hug/tasks` history, and all sessions of a set interleaved in time. A fired
line is **right** when the session ran its command within 30 minutes, the
task was closed within 24 h, or a blind judge (DeepSeek Flash, no tools,
credentials redacted, offline only: `muninn/scripts/judge_board.py`) calls it
correct from what the agent did, the user's message and the board.
**False** = the judge calls it wrong and the session did not act on it.

Sets were fixed before any output was seen. dev: crab, dragon, hare,
humpback, poodle in hug. held-out: dromedary, hog, kitten, vole in hug, and
seid-lang sessions from 2026-11-12. fresh: seid-lang sessions created
2026-11-08 to 2026-11-12, fixed after the last tuning round and run once.

| set | sessions | board turns | session-hours | fired (take/work/done/lint) | acted on | right | false | unclear | precision | false per 10 h |
|---|---|---|---|---|---|---|---|---|---|---|
| overnight (crab 15-16 Nov) | 1 | 77 | 4.8 | 3 (1/2/0/0) | 0 | 2 | 0 | 1 | 1.00 | 0.00 |
| dev | 5 | 426 | 151.4 | 19 (6/7/4/2) | 4 | 18 | 1 | 0 | 0.95 | 0.07 |
| held-out hug | 4 | 34 | 3.7 | 4 (0/0/3/1) | 3 | 4 | 0 | 0 | 1.00 | 0.00 |
| held-out seid | 24 | 971 | 199.0 | 50 (6/2/8/34) | 4 | 42 | 6 | 2 | 0.88 | 0.30 |
| fresh seid | 13 | 92 | 23.9 | 7 (1/0/0/6) | 0 | 7 | 0 | 0 | 1.00 | 0.00 |

The bar set before measuring: overnight fires >= 1 (3), precision >= 0.8
(lowest 0.88), <= 1 false fire per 10 session-hours on held-out (0.30 seid,
0.00 hug). Overnight the hook would have said, verbatim:

- 22:08 `Task board: you are working on jrj1vq but it is todo. hug task take jrj1vq --model claude-opus-5-5`
- 00:31 `Task board: you are working on p277bp but it is todo. hug task take p277bp --model claude-opus-5-5`
- 01:46 `Task board: you are changing code and no task is doing for you. Take the task this is (hug task next), or hug task new "title" --goal "..." if none fits, then hug task take ...`

The 7 false lines: 2 `take` naming a title match that was not the work
(crab 34r1ft, blossom jhtydq); 5 `lint` handing an unrelated todo's finding
to an agent busy with other work, one of them told to stop. Lint lines are
housekeeping by design (one finding per agent per 6 h, at a break); the judge
counted 27 of 34 seid lint lines right.

Not blind, recorded as it happened (tmp/lanes/qawmnb-design.md): three rules
were added after looking at held-out output: the 24 h repeat wait (56 of 103
held-out lines were one `hug task done mtzydx` to one session), the 7-day
lint rotation (17 of 34 lint lines were one finding), and per-task fixes only
(18 lint lines said `hug upgrade`). The fresh set came after all three.

## Owner: the assignee is the agent doing the work (v3k6cp, 2026-11-17)

A fifth kind, `owner`: the session works a doing task (took it, works in
its recorded worktree or branch, ticks its checks, or notes it while
editing code) but the assignee is another name. One line, once per task
per day, under the take rate:

`Task board: you are working on 44w1ev but the board says humpback. If it is yours now: hug task set 44w1ev --assignee wyvern`

The session name comes from the harness (`session_<name>_...`), never the
model. `hug task take --as NAME` with a name other than the session's
still takes, and says so with the same fix. `hug task lint` reports
`stale-owner`: a doing task whose every assignee is a jcode session name
with no live session for over 24 h (read from `JCODE_HOME/sessions`;
names it does not know, a person or a model, are never judged).

Replay, same six sets and judge, baseline main: wyvern's session of
2026-11-17 fires once, `owner` on 44w1ev at 09:46:55Z (the morning the
board still said humpback). Across the sets 14 owner lines, 13 judged
right; the false one was poodle told rx74bp was crab's now. 9 of the 14
are tasks still assigned to `Jørgen H. Fjeld` (git user.name, before
et85h1), 5 to another agent (hibiscus 2, vole, daisy, piglet); the
overnight set repeats one dev line (crab, 5g86n6). All kinds: main 97
lines, 87 right, 7 false (0.926); with owner 109 lines, 99 right, 7 false
(0.934). An owner line takes the turn's one slot, so 5 other lines move:
a work line (daisy 88s3yc, judged right) and 2 lint lines (sunflower
qaxx25 judged false, herb bbhkwq right) do not fire; sunflower's done line
on 23vyy1 fires 54 minutes later (closed by the agent, so counted right,
though the judge called the later moment wrong); orangutan gets a new
work line on xwjb13 (judged right).
## Stop orders, and lint between tasks (x325pp, 2026-11-17)

Shipped: a turn that tells the agent to stop (`stop now`, `stop immediately`,
`do not run any`, `do not edit any`, `end your session`, `stand down`, ...)
gets no board line. A scope rule ("do not start any other task") is not a
stop order.

Tried and not kept: an unrelated finding (a task the session did not touch)
only for a session with no doing task of its own, or one that ran
`hug task done` since its last turn ("between tasks"), alone and with
editing or noting a todo also counted as busy.

Same sets and judge as above (DeepSeek Flash, blind to arm), plus fresh2:
seid-lang sessions written after 2026-11-16 20:00, turns from then on
(20 sessions, 21.5 session-hours), run after the rules were fixed. Lint
lines, right / false (unclear):

| set | main | stop order | stop order + between tasks | + busy when editing |
|---|---|---|---|---|
| dev | 2: 2 / 0 | 2: 2 / 0 | 2: 2 / 0 | 2: 2 / 0 |
| held-out hug | 1: 1 / 0 | 1: 1 / 0 | 1: 1 / 0 | 1: 1 / 0 |
| held-out seid | 34: 27 / 5 (2) | 33: 30 / 1 (2) | 24: 22 / 0 (2) | 20: 17 / 1 (2) |
| fresh seid | 6: 6 / 0 | 5: 5 / 0 | 5: 5 / 0 | 2: 2 / 0 |
| fresh2 seid | 8: 8 / 0 | 8: 8 / 0 | 3: 0 / 3 | 2: 0 / 2 |
| all lines, six sets | 97: 87 / 7 | 95: 89 / 3 | 81: 73 / 5 | 73: 65 / 5 |

All lines precision: main 0.926 (0.17 false per 10 h), stop order 0.967
(0.07), between tasks 0.936 (0.12), busy when editing 0.929 (0.12). Take,
work and done lines are the same in every arm; overnight has no lint lines.

- Stop order removes 2 lines: hog p39w4a ("STOP NOW ... do not run any
  command", judged false) and skunk w96jpg on fresh ("Please stop now",
  judged right: the finding was real, the moment was not). Only hog's line
  is the rule's own effect. Freeing hog's finding reshuffles the lint
  rotation across sessions (one finding per 7 days), so 14 other held-out
  seid breaks get a different finding: all 14 judged right, replacing 11
  right and 3 false (orangutan zjedy0, pawprint 2n92r7, sunflower qaxx25).
  That gain is rotation luck, not a rule; do not count on it.
- The one false lint line left (piglet n0fzz1) names another agent's
  doing task.
- Between tasks clears all 5 held-out false lines but costs fresh lint lines
  (14 -> 8), and the 3 left on fresh2 went to sessions editing code with no
  doing task (evergreen, herb, palmtree): between tasks by the rule, busy by
  the judge. Re-judged twice, those 3 came out 3 and 0 right; main's 8 came
  out 8 and 6 right: the judge is not stable on this question.
- No lint line was acted on in any set, in any arm.

## Missing sections go to the task's own session (tpnd3n, 2026-11-17)

Shipped: a missing Purpose or Principle is handed only to a session that
touched the task (its own, named in the message, or worked this turn); a
passer-by gets other findings (long title, no acceptance). And an unrelated
finding waits while the session's own doing task has a finding of its own.

Why: fkpyg3 holds every open task to the description pattern, so the board
now has missing-section findings on most old tasks, and the hook handed them
to whoever was at a break. Same six sets as above, today's boards, same
blind judge; arms: before fkpyg3 (206038a), main (38ff464), own task first
alone, and both rules. Lint lines, right / false (unclear):

| set | before fkpyg3 | main | own task first | both (shipped) |
|---|---|---|---|---|
| dev | 2: 2 / 0 | 16: 3 / 8 (5) | 16: 3 / 8 (5) | 6: 5 / 1 |
| overnight | 0 | 1: 0 / 1 | 1: 0 / 1 | 0 |
| held-out hug | 1: 1 / 0 | 4: 1 / 3 | 4: 1 / 3 | 3: 0 / 3 |
| held-out seid | 33: 27 / 4 (2) | 33: 28 / 5 | 32: 27 / 5 | 32: 28 / 4 |
| fresh seid | 6: 6 / 0 | 5: 5 / 0 | 5: 5 / 0 | 5: 5 / 0 |
| fresh2 seid | 7: 7 / 0 | 9: 6 / 3 | 9: 6 / 3 | 9: 7 / 1 (1) |
| all lines, six sets | 109: 99 / 7 | 128: 99 / 23 | 127: 98 / 23 | 115: 101 / 12 |

All lines precision: before fkpyg3 0.934, main 0.811 (0.56 false per 10 h),
both 0.894 (0.29). All 20 false lint lines on main are missing-section
findings, 18 on tasks the session had not touched (s6ff5k to hog and
dromedary, told "do not start any other task"; vk9td0 to kitten and poodle).

- The 42 lines only main gives and the 29 only the shipped rule gives were
  judged twice more from an empty cache: main's 25 and 26 right, 17 and 15
  false; the rule's 26 and 26 right, 0 and 0 false.
- Own task first alone moves 4 held-out seid lines to another finding and
  drops 1, all judged right in both arms: on these sets it reorders, it does
  not remove a false line.
- The 3 false held-out hug lines left name s6ff5k and vk9td0 because the
  user's message quotes them ("Confirm edit s6ff5k Title?"; vk9td0 as an
  example of an unrelated `next:`), which makes them touched.
- Fresh hug, chosen after the rule was written and run once: hug sessions
  written after 2026-11-16 20:00, turns to 2026-11-17 14:00 (38.8
  session-hours): main 4 lint lines, 3 right, 1 unclear (crab, 9taa9d's
  Principle); the rule 3 lint lines, 3 right.
- A finding withheld on a stop turn is not spent: the next break of that
  session or another session gets it (test). In the replay skunk's turn
  ("Please stop now", 2026-11-11 18:13) was its last board turn, and w96jpg's
  title line went to another session later in every arm (main: hibiscus
  2026-11-16; the rule: zebra 2026-11-13).
- 12 false lines against 7 before fkpyg3: 9 are new and 4 of the old 7 are
  gone (2 acceptance and 1 title finding moved by the rotation, piglet's
  n0fzz1). The 9 new are all missing-section lint. 5 come at a session's
  start on a task its message only mentions: s6ff5k (hog, dromedary: a
  quoted prompt "Confirm edit s6ff5k Title?" in a brief that says "Do not
  start ... any other task"), vk9td0 (kitten, poodle: quoted as an unrelated
  `next:`), n8zmjw (evergreen: named as the dependency of the task it works);
  that is 4vxqp4. 4 name the session's own or noted task when the judge
  calls the moment wrong (zwgp5s, jy00xt, mtzydx with every check ticked,
  c9h01e). So the 5 extra are not real findings a session missed: they
  are the same section findings reaching a session by a mention.

## A task the message only mentions is not the session's (4vxqp4, 2026-11-18)

Shipped: a missing Purpose or Principle reaches a session for a task it
owns or acted on (took, checked, noted, set, edited; or mine()), or one its
message names as the work. A task the message names only as context does
not count: inside quoted text (backticks, double quotes, a fenced block;
a quote that is only an id or `hug task VERB ID` still counts), right after
"depends on" / "builds on" / "blocked by" / "waits on" / "parent", or in a
"do not start / take / touch ..." clause. Reading another session's doing
task with `hug task show` is reading, not acting; reading an open todo still
counts. Every other finding keeps the old rule (any mention counts).

Same six sets and windows as tpnd3n (session ids pinned from its arm),
same judge (judge_board.py, DeepSeek Flash, blind to arm), verdicts reused
from tpnd3n's cache and new lines judged once. Baseline is origin/main
4e776695, which reproduces tpnd3n's arm line for line (only `--model` in 7
crab take/work lines differs: the session file changed model since).

| arm | lines | right | false | lint right / false |
|---|---|---|---|---|
| main 4e776695 | 115 | 101 | 12 | 45 / 9 |
| context mentions | 114 | 102 | 9 | 46 / 6 |
| + show of another's doing task is reading (shipped) | 115 | 104 | 8 | 48 / 5 |
| + quoted hug commands are not calls (not shipped, 6fqhg9) | 114 | 105 | 7 | 49 / 4 |

- Gone (5, all judged false): s6ff5k to hog and dromedary, vk9td0 to
  kitten and poodle, n8zmjw to evergreen. Their breaks get another finding
  instead: qawmnb's "not a dependency" (kitten, poodle: right), p39w4a to
  evergreen (right). Rotation moves n8zmjw's Purpose to herb, whose brief
  makes it the work (right), not palmtree, who only read it.
- One false line is new: crab 2026-11-16 12:27 gets vk9td0's Principle
  instead of its Purpose. vk9td0 counts as crab's because the hook reads
  `hug task take vk9td0`, quoted in crab's own `hug task new --goal`, as a
  take. The parser fix is 6fqhg9; measured here it gains 1 line on the six
  sets but costs fresh hug (lint 3 -> 2, false 3 -> 4), so it is not shipped.
- Fresh sessions keep their right lint lines: fresh 5 / 0, fresh2 7 / 1 ->
  9 / 0, fresh hug (367 sessions after 2026-11-16 20:00) 3 / 0 -> 3 / 0 (the
  same 3 lines).
- The judge is not stable on these lines. The 12 lines that differ between
  main and the shipped arm, rejudged twice from empty caches: main-only 1
  right / 5 false, then 6 / 0; shipped-only 4 / 2, then 5 / 1. hog's s6ff5k
  was "not this worker's task" once and "board lint is independent" the
  other time. Either way the shipped arm has no fewer right lines.

## The judge states the lint ownership rule (6fqhg9, 2026-11-18)

The judge prompt used to say a lint reminder "does not need to relate to the
current work", which contradicts the ownerOnly rule (tpnd3n, 4vxqp4) for a
missing section or an owner's question. It now names the two kinds: an
owner-only finding is correct only on the agent's own task (doing for it,
taken, checked, noted, set or edited, or given as the work by the message);
any other lint finding may go to any agent at a break.

- The 15 lines that differ between 4vxqp4's base and quote arms, judged
  twice from empty caches: old prompt 7 of 15 verdicts the same, new
  prompt 15 of 15.
- Every line of main 4e776695, 4vxqp4 (8d26d050) and 8d26d050 + the
  quote-aware parser, judged twice: 134 of 145 the same (0.924). The 11
  that differ are 5 work, 3 lint, 2 owner, 1 done lines. Between 4vxqp4
  and the parser arm none of them is in only one arm, so that comparison
  is the same under both judgings; 3 sit only in main (main's lint / take
  split is 5 / 7 under one judging and 7 / 5 under the other).
- Under the new prompt the six sets read: main 115 lines, 103 right, 12
  false (both judgings); 4vxqp4 115, 106, 9; + quote-aware parser 114, 107,
  7. Fresh hug: 4 false in all three arms; the parser moves one false line
  from lint to take (guppy 2026-11-16 20:16, `take jrj1vq`, a task it only
  read with `hug task show`), so it is not shipped (6fqhg9).
- Under the new prompt nautilus's e4w9ya and hatchling's s6ff5k Purpose
  (fresh hug) are judged false: the old prompt counted them right because
  any lint was right. 4vxqp4's fresh-hug "3 right lint lines" were judged
  under the old prompt.

## Latency

The board read on every turn (activity + board, 40 runs, warm cache):
hug p50 25.9 ms, p95 27.2 ms; seid-lang (2,508 tasks) p50 112.9 ms,
p95 143.6 ms. Lint at a break adds p95 0.9 ms (hug), 29.8 ms (seid). The hug
target (p95 < 50 ms) holds. On seid most of the time is the board cache
loading every task row (`huginn/internal/cache` Open, 2.3 s of 4.2 s in the
profile); a read that skips that load when the board head is unchanged would
bring seid near hug.
