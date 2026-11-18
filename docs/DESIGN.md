# hug: design (draft, 2026-11-11)

Decided by the user today (memory j9jdkh). Proposals are marked **proposal**.

## One tool

`hug` replaces `muninn` (memory) and `huginn` (tasks), and adds messages.
One binary, one `hug setup`, one MCP server, one config, one identity, one git
sync code path. Parts keep their raven names in docs: huginn = tasks/intent,
muninn = memory.

**proposal:** `muninn` and `huginn` stay as thin aliases for a few releases, so
installed hooks and MCP configs keep working; `hug setup` rewrites them.

## Storage

- Truth is files on namespaced git branches, one per kind:
  `hug/tasks`, `hug/memories`, `hug/messages`, checked out at `.hug/tasks/`,
  `.hug/memories/`, `.hug/messages/` (each listed in `.git/info/exclude`).
- Files form a tree; the generated id is in the filename, so moving a file
  between folders never breaks a link.
- One file per item: Markdown with short front matter. Retire = move to
  `_retired/`, never edit dated text.
- A local SQLite cache is rebuilt from files and updated incrementally from
  `git diff` (measured: 4,958 files, git status 0.05 s, list 0.45 s).
- Private data (personal, global, client memories) uses the same layout in a
  local folder, e.g. `~/.hug/`. **proposal:** that folder may itself be a private
  git repo to sync machines.

## Sync

- Commit, fetch, rebase, push per branch, at each branch's own pace.
- `take` keeps using the push as the claim lock.
- Messages: one file per message, never edited by anyone else, so no conflicts.
- Host repo hooks never run on hug branches (fixed in huginn v0.1.4).
- Shared memories only reach a branch if the repo setting allows it; default off
  for public repos (task 34r1ft).

## Also in scope

- Budgets on tasks (budget, estimate, spent), per memory 1ymfzr.
- Supervisor (who is alive, what wakes them), later.

## When dream runs (vk9td0, decided 2026-11-18)

**Decision.** The daily dream pass starts at the beginning of the person's
quiet stretch. hug learns that from an hour-of-week histogram (168 buckets)
of the person's own turns. The histogram is trusted once it holds a week of
activity. Until then the old rule stays: run once nobody has typed for 30
minutes. Dream items are still handed out at the next break turn. After a
pass in the quiet stretch, that is the person's first message on returning.

**Reason.** Jørgen (2026-11-18): his syncthing fork already does this for its
database defrag (`~/Projects/syncthing`, `internal/activityhist`, patch/43
and patch/53). It histograms activity cyclically, needs 7 days before it
predicts, and starts the work at the beginning of the longest idle run. A
week is the natural cycle, and hour-of-week shows that days differ. Weekday
nights are short and weekend nights long. "Away 30 minutes" fires at the
first lunch break. The learned stretch fires when the person is usually away
for hours.

**How.** It is local and free: it costs one row write per person's turn and
makes no model or network calls.

- `muninn/internal/activity` counts each turn the person types in
  `activity_hours` (turns per clock hour, kept 8 weeks). Swarm DMs and
  harness notices are not counted. The first read seeds the table once from
  `hook.log`, which already holds weeks of jcode turns.
- The histogram covers the last 4 weeks. An hour of the week is quiet when it
  had a turn in at most a quarter of the weeks seen. One late night in a
  month does not break a stretch, but in the only week of data it does.
  Recent weeks replace old ones, so a changed rhythm shows within a month.
  activityhist halves its counters weekly for the same reason.
- Dream is daily, so each day's longest quiet stretch is a window. The
  week's longest stretch is the one named. A waiter started in the morning
  waits for that day's window, plus at most 12 hours. Inside the window the
  pass runs at once, unless the person has typed since it began. Then the
  30-minute rule decides. The load gate (below 120) holds either way.
- `hug mem dream --week` shows the histogram, the longest stretch and each
  day's start. `hug mem dream --status` names the next stretch, or how many
  of the 7 days are still to learn.

**Not taken.** activityhist's minute-of-day histogram: turns are sparse, and
hour precision is enough for a pass that takes 10 s. A fixed clock time, such
as 03:00, would be a guess, not learned. Moving the hand-out to wrap-up only
(hatchling's open question) was not taken either. Items made in the quiet
stretch are handed out at the person's return, which is a break already.

## Migration

- huginn: branch `huginn` -> `hug/tasks`, folder `.huginn/` -> `.hug/tasks/`.
  git cannot hold `huginn` and `huginn/x` at once, so rename then push and delete
  the old remote branch (only seid-lang is pushed).
- muninn: no repo has a notes branch yet; project markers in `.muninn/` move
  into `hug/memories`; client markers in non-git folders stay.

## Open

1. Repository: rename veridit/muninn to veridit/hug and merge huginn in with
   history, or a new repo.
2. Personal vs shared memories (gav9nw).
