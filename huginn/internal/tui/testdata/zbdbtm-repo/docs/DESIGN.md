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

## Personal vs shared memories (gav9nw, decided 2026-11-18)

**Decision.** A repository that shares memories (34r1ft: `hug.toml [memory]
share = true`, absent means local) pushes a memory only when two checks both
say push. A+: the capture extractor places the memory, with personal and
other-project places added to the ones it has. B: a local rule at sync holds
back any line that names something outside the repository. Jørgen 19:40Z
chose A+ and B together, "unless someone clever can dream up something
better". Before any repository opts in, A+ is confirmed on fresh captures.

**Reason.** A pushed memory cannot be taken back. `hug mem forget` keeps the
line with its text and `active: false` (all 22 retired lines on 1abcc497
still carry it), git history keeps every earlier commit, and another clone
keeps its copy, since an import is a copy and not a mirror
(`ingest/importer.go`). So the leak rate decides. A memory held back that
belonged is the cheap mistake, released later by rescope. Of the 868
memories on hug's public `hug/memory` (1abcc497), 58 (6.7%) do not belong
there: 29 personal, 16 about another project, 13 cross-project team rules.
They got there because the extractor already places every capture (here,
client or global, since v0.4.0-31) but has no personal or other-project
place and is told "when unsure choose here". In 21 of 570 placed captures it
named what the memory was about and still chose here ("Specific local repo
path and access mode for this project").

**How.** No extra paid call: A+ uses the call capture already makes, and B
uses no model.

- A+: the extractor's places gain `personal` (the person, his machine, home
  paths) and `project:<other>` / `client:<name>`, resolved from the catalog
  `hug mem tidy place` already builds. `here` stays the repository. Team
  rules go to global until a team scope exists. `hug mem remember --place`
  takes the same places.
- B: a gate in sync beside the secret gate (`LooksSecret`, `sync.go`) holds a
  line that names a home path (`/Users/`, `~/`), another known project or
  client, or the person, or whose category is preference or correction. Its
  term list lives in local config, never in the repository: a list that
  names clients is itself private. Being a gate at sync, it also covers
  memories written before it.
- Measured on the 868 (OPTIONS.md, attached to gav9nw): a blind labeler
  (DeepSeek Flash, 30 calls) labeled each memory, corrected by hand where it
  was suspect (all 90 it kept off the branch and 63 regex suspects read; a
  random 30 agreed 30/30). A+ and B together pushed 699 and leaked 0,
  holding back 13.7% of the memories that belonged. **This is a hypothesis,
  not a measurement of the extractor.** The A+ row is the labeler's verdict
  standing in for the extractor, so its leak is a lower bound, and the 868
  are the data the options were chosen on.
- Confirmation (rnpdnm check 3): on fresh captures, not the 868, A+ and B
  leak at most 1 in 200 pushed, judged blind, with the withheld rate and its
  interval reported. No repository opts in to sharing before it holds.

**Confirmation, 2026-11-18: FAIL.** Blind hand 8/254 leaked (3.1%, 95% CI
1.6 to 6.1), judge 17/254. The 0/699 proxy overstated A+: the labeler saw
the memory alone, the extractor sees a hug transcript. 7 of 8 are team rules
said in hug work ("crab runs hug task done himself", "use absolute paths
when applying patches"); the eighth is personal. So A+ stays off
(`extract.wide_places`), B ships (it only ever holds more), and no
repository opts in. hug keeps its memories local (34r1ft), so nothing leaks
meanwhile. Next attempt: coordination terms in B's gate plus a prompt rule
that agent and coordination rules are global, measured only on captures
made after 2026-11-19 00:00Z. A team scope (873tqd) is the larger design
behind both. Evidence: rnpdnm notes (pre-registration, sealed hand labels,
50 Flash calls).

**Not taken.**

- C, share only what an agent writes on purpose (`remember`) and keep the
  rest local. The deliberate writes leaked most, 7 of 38 (18.4%, 95% CI 9.2
  to 33.4), mostly coordinator rules, and C held back 96.2% of hug's own
  memories, so clones would lose nearly all of them. It is the option
  Jørgen first leaned to (wxsyc5: keep the public branch as long as the
  agent placing a memory judges personal vs repository well); on this
  branch the agents did not.
- A+ alone: 1 leak in 778 (0.1%), but it rests on the labeler alone, and a
  model's slip is pushed with nothing behind it. B catches the slips that
  name a path, a project or the person.
- B alone: 18 leaks in 738 (2.4%, CI 1.5 to 3.8), memories with no marker
  word ("the background shell's python3 lacks PIL"), and 11.1% held back,
  such as hug decisions credited to Jørgen.
- Today's placement as it is (A0): 58 leaks in 866 pushed (6.7%).

What to do with the 58 already on the public branch is its own question
(q0a8rx). The build is rnpdnm.

## Migration

- huginn: branch `huginn` -> `hug/tasks`, folder `.huginn/` -> `.hug/tasks/`.
  git cannot hold `huginn` and `huginn/x` at once, so rename then push and delete
  the old remote branch (only seid-lang is pushed).
- muninn: no repo has a notes branch yet; project markers in `.muninn/` move
  into `hug/memories`; client markers in non-git folders stay.

## Open

1. Repository: rename veridit/muninn to veridit/hug and merge huginn in with
   history, or a new repo.
