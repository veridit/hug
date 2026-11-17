# Landing queue in hug: design note (47sxn2 open question)

Status: proposal only. Jørgen has not approved it; no code. Author: kestrel,
2026-11-17 14:05, at crab's request.

## The problem, with today's evidence

Several agents fast-forward `main` from their own worktrees. Each landing is:
rebase onto `origin/main`, push `ci/<id>`, wait for CI on all three OSes
(10-15 min; Windows is the long pole), check
`git merge-base --is-ancestor origin/main HEAD`, then `git push origin HEAD:main`.

- When two agents do this at once, the second one's CI was run on a main that is no
  longer main. crab, 12:27: "3 rebases raced in the last hour".
- The fix today is a queue crab keeps by hand in DMs ("Order: 1. hatchling: v3k6cp
  ... 2. kestrel: sfa8w4 ..."). The holder DMs crab when its push is on main or its CI
  is red, and crab passes the turn on.
- It works, but it lives in crab's context. Jørgen can't see it on the board, it
  breaks if crab is reaped (30 min idle), and every hand-off is a DM round trip.
- The same day, the queue order also decided which shared code lands first:
  `internal/sharefile` (kbc99j) is needed by zd3x24. Order is part of the plan,
  not just a mutex.

## What `hug land` would be

One command that holds a turn on main the way `take` holds a ticket: by a commit
on `hug/tasks`, pushed with take's lock. `UpdateMessage` (gitbranch.go:655) pulls,
writes, commits and pushes; a lost push race is `ErrRaced`. No new transport, no
server, no daemon (47sxn2 principle).

The turn is data on the board, so every clone sees it at its next command (at most
about 7 s later, per 4vcwxk), the TUI can show it, and nobody's context has to
hold it.

### The turn file

One file on `hug/tasks`: `.hug/tasks/land.md` (not a task; lint knows it).

```
---
holder: kestrel
model: claude-opus-5-5
tickets: [kbc99j, 4vcwxk]
branch: kestrel/land-kbc99j-4vcwxk
base: 5b80e87          # origin/main the holder rebased onto
ci: 44098887933        # the run on that rebase, once pushed
since: 2026-11-17T14:05+01:00
---
## Queue
- hatchling: zd3x24 (after: kbc99j)    # waits on what it needs, by ticket
- kestrel: n2hq8p
## History
- 13:29 hatchling landed v3k6cp at 206038a (CI 44083...)
```

### Commands

| command | does | refused when |
|---|---|---|
| `hug land queue ID... [--after ID]` | appends you to the queue for those tickets | a ticket is not yours, or not doing |
| `hug land` | takes the turn when you are first and the turn is free; prints the steps (rebase, push ci/, wait) | someone holds it: names the holder, since when, and their CI run |
| `hug land ci RUN` | records the CI run id on the turn (the holder's proof of progress) | not the holder |
| `hug land push` | checks `base == origin/main`, runs the is-ancestor check, pushes `HEAD:main`, writes History, frees the turn, and marks each ticket landed (the deduction g9dg32 would make) | main moved since `base` (fix: rebase, new CI), CI not green |
| `hug land release "why"` | gives the turn back without landing | not the holder (others: see stale below) |

`hug land push` reads CI status with `gh run view RUN --json conclusion`. Without
`gh`, it asks the holder to say `--ci-green RUN` and records that as a claim, never
a deduction ("Never guess", 47sxn2).

## Who holds the turn

- The turn belongs to a name, as a ticket does: `--as` / HUG_USER / harness
  session, else the model (`whoTakes`, cli.go:283). Same identity rule as take and
  n2hq8p, so "one ticket in doing" and "one turn on main" talk about the same agent.
- First in the queue whose `after:` tickets are landed. Order is the queue order
  unless a coordinator reorders it (`hug land queue --move NAME --to N`). A
  coordinator is just whoever runs that. No roles in hug; the commit says who did
  it.
- One turn per repository's main. A second protected branch (release/*) would get
  its own file; not needed now.
- A person can hold it (`--model human`), as with take.

## Release when CI goes red

Red CI is the common case of "the turn must move on", so it should not need a
DM:

1. `hug land push` with a red run is refused, and the refusal itself offers the
   two ways out as fix: lines (the n2hq8p shape):
   - `fix: hug land release "CI 44098887933 red: TestX on windows"` gives the turn back,
     and you stay in the queue at the front of the remaining order
   - `fix: hug land ci NEW_RUN` after a fix commit and a fresh run (keep the turn)
2. Keep-the-turn on red is limited: one retry by default. A second red releases
   on the next `hug land` by anyone else, with a note. A turn must not sit on a
   flake for an hour while the queue waits.
3. When the holder's CI is red and they have gone (no `hug` command from their name
   for N minutes, and the 30-minute reap makes this real), the next in the queue
   may `hug land --take-over`. It is refused unless the run is red or the turn is
   stale. The History line names who took over from whom and why.

Stale without red: no `hug land ci` within 30 min of taking the turn, or the
recorded run finished more than 15 min ago without a push. lint reports
`land-stale` with the fix (`hug land release` for the holder, `--take-over` for
the next one).

## What it does not do

- It does not run CI, merge, or rebase. The holder does. hug holds the turn,
  checks what it can check (base, is-ancestor, the run's conclusion), and writes
  down what happened.
- No auto-advance by a daemon. The next agent learns the turn is free at its next
  hug command (the board sync) or from the TUI. A `wakes NAME` on a stated
  condition (16dxsr) could nudge it: `when land free wakes hatchling`.
- No change to how main is protected on GitHub. A push that bypasses `hug land` still
  works; `hug land push` notices main moved and refuses with a rebase fix.
- One check it can make cheaply (observed 14:05 today, twice: hatchling's
  rebase and mine): a release commit renames `## Unreleased` to `## vX.Y.Z`, so
  a rebase over it silently files the lander's CHANGELOG lines under the
  release. `hug land push` can refuse when the diff against `origin/main` adds
  CHANGELOG lines below the first `## v` heading, with the fix of moving them up.

## Open questions for Jørgen

1. Is a file on `hug/tasks` the right home, or should the turn be a stage on the
   tickets (`doing/landing`, beside waiting/review in 6ephqk)? A stage needs no new
   file, but the queue order across agents has no home there.
2. Should `hug land push` push to main itself, or only check and record, leaving the
   `git push` to the agent? Pushing itself is one fewer step to forget. Checking
   only keeps hug out of the code branch, as today.
3. Is one retry on red the right limit, and who may reorder the queue: anyone, or
   only a name listed as coordinator in the repo's hug config?
4. Release tags (crab's turn "after the release" today): is a release a queue
   entry like any landing (`hug land queue --release v0.7.42`)?

## Smallest first step, if approved

`hug land` / `hug land release` / `hug land push` with the turn file and the
red-CI refusal, nothing else. The queue order stays crab's until the turn itself
has been used for a day.
