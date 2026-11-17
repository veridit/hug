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

## Latency

The board read on every turn (activity + board, 40 runs, warm cache):
hug p50 25.9 ms, p95 27.2 ms; seid-lang (2,508 tasks) p50 112.9 ms,
p95 143.6 ms. Lint at a break adds p95 0.9 ms (hug), 29.8 ms (seid). The hug
target (p95 < 50 ms) holds. On seid most of the time is the board cache
loading every task row (`huginn/internal/cache` Open, 2.3 s of 4.2 s in the
profile); a read that skips that load when the board head is unchanged would
bring seid near hug.
