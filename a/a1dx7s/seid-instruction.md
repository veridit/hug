# Instruction for the seid-lang coordinator: restore epic links (a1dx7s)

Copy everything below the line to the seid-lang agent.

---

Task from hug's coordinator (crab), for seid-lang's board. Work in
/Users/jhf/Veridit/src/seid-lang. Run every command niced:
`GOMAXPROCS=2 nice -n 19 ...`. Use hug 0.7.90 or later (`hug version`).

Why: the 2026-11-15 move from backlog lost every parent link. Today 0 of
2,555 tasks have a parent, so epics and `hug task ls --epic` show nothing.
The links are still in `.backlog` (276 files carry `parent_task_id:`), and
re-running the import puts them back. Edits made in hug since then win;
the import only fills in what is missing.

Steps:

1. Restore the parents:
   `hug task import backlog .backlog`
   Check: `hug task ls --json` shows about 282 tasks with a parent
   (282 on crab's test copy, 2026-11-20).

2. Fix the parents that point at a task which has a parent itself (13 on
   the test copy). Each finding carries its own fix line:
   `hug task lint --code parent-too-deep`
   Read each one, then run its `fix:` line (it moves the task up to the
   top epic), or set a better parent if you know one. Repeat until the
   command prints no findings.

3. Import the evidence files as task files:
   `hug task import files .backlog/evidence --cite .backlog/evidence`
   It took 30 seconds on the copy. Run `hug task lint --code missing-file`
   before and after: on the copy there were 8 after, from notes that cite
   files never attached (e.g. p39w4a's store-context-contamination.md).
   Attach those files or mark the citations lost, as each fix line says.

4. Report back on hug's task a1dx7s (in /Users/jhf/Projects/hug), one
   note with the numbers:
   `hug task note a1dx7s "seid: N tasks with a parent; parent-too-deep 0; evidence imported"`

Do not:
- run `hug task import backlog .backlog --cleanup` (that removes .backlog;
  it is a separate decision for Jørgen);
- bulk-rewrite titles or Goals. The 1,247 goal-too-long findings are
  advice, handled later one by one.

If a step fails, stop and write the exact error in a note on a1dx7s.

Also (Jørgen asked, 2026-11-20):

A. The board line you saw about am8zx0 is real: Jørgen wrote request 1 on
   am8zx0 ('Why don't you just run it on the other machine? ... cron on the
   remote machine itself ... pick up the files in the morning'). hug shows a
   person's requests in command output on purpose; that is how they reach
   you. Act on it: turn it into changes to am8zx0, then
   `hug task resolve am8zx0 1 "what changed"`. `hug task requests` lists
   every open one (8629gy has one too).

B. Make yourself the board's coordinator, so requests reach you directly:
   `hug task coordinator NAME` (your session name).

C. Bring this repo's agent instructions up to date: run `hug task setup` and
   do its 'agent instructions' step (it replaces the '## Tasks: hug' section
   of AGENTS.md with today's text, which explains requests and how they are
   delivered). Show Jørgen the diff before committing.
