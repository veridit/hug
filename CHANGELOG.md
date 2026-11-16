# Changelog

Releases are tagged `vX.Y.Z` and ship one `hug` binary. Earlier part releases
are in [muninn/CHANGELOG.md](muninn/CHANGELOG.md) and the `muninn-v*` and
`huginn-v*` tags.

## Unreleased

## v0.7.38 - 2026-11-17

- `hug tui` IN PROGRESS rows name the branch a task was taken on after the owner, e.g. `alice · feat/x`, unless it is the main branch; on a wide terminal the owner column grows to fit it, up to 30 cells, and on a narrow one the branch is cut, never the owner (x1t1t1).
- Task and project-memory changes reach the other clones by themselves, within seconds: every `hug task` write commits locally and starts one detached syncer per board, which waits until a burst of writes is quiet for 1.5 s, then fetches, rebases and pushes (20 notes in 2 s are one or two pushes). Reads (`ls`, `show`, `next`, board) start a fetch when the last one is over 5 s old. `hug mem remember`/`forget` of project memories, `hug mem search` and the recall hook do the same on the `muninn-notes` branch. Nothing waits on the network: offline, a note costs what it costs online, and `hug task ls` says `sync: N unpushed: ...` with `fix: hug task sync`; they go out at the next write once the remote is back. A rebase conflict keeps every local commit and says so the same way. `take` keeps its synchronous push (the race lock); `HUG_SYNC=off` turns background sync off; passes log to `.git/hug/sync/` (0hkaa5).
- Every git network call (fetch, push, ls-remote) is non-interactive and has a deadline: no password or passphrase prompt (ssh BatchMode, no askpass, Git Credential Manager non-interactive; a user's own ssh command is kept), ssh gives up connecting after 10 s, a stalled transfer after 15 s, the whole call after 60 s, with a `fix:` line. An unreachable remote is named as such, not as a key problem (0hkaa5).
- Writes to one board take turns (a board lock from reading a task to committing it), so a background rebase never moves files under a command, and 20 parallel `hug task note` commands all land (0hkaa5).
- The board turn holds on a board without a remote too: before, 5 of 20 parallel `hug task note` commands there failed with `fatal: cannot lock ref 'HEAD'` or `index already contains staged changes`; now 20 of 20 land. `hug mem capture` takes the write turn for its saves and checkpoints (never across the model call), so it no longer interleaves with a remember, forget or index; its audit record too (m8shdj).
- The task cache opens in one read instead of decoding 10,000 rows, a write stores only the tasks it changed, and `ls` takes notes and description texts from the cache instead of reading every file: at 10,000 tasks `ls` drops from 1.08 s to 0.45 s, `next`, `show` and `board` from 0.33 s to 0.24-0.26 s, `note` from 0.60 s to 0.51 s. An edit that keeps a file's size and lands in the same clock tick as the last read is now always seen: files modified within 2 s of the last sync are hashed even when size and time match (44w1ev, step 3).
- `hug task lint` takes file references from the cached parse instead of reading every task file (it reads them only when `forbid_cite` is set): at 10,000 tasks lint drops from 2.0 s to 0.36 s, on seid's 2,508 tasks from 1.50 s to 0.22 s (44w1ev, step 4).

## v0.7.37 - 2026-11-16

- A memory can say how long it is true: `hug mem remember --holds "until 2026-12-01"` (or `"while task:ID open"`, `"while branch NAME unmerged"`; MCP `holds`). When the condition ends, checked without a model call, recall stops showing it and dream offers it once to retire or keep; nothing is deleted. A memory with only a recorded branch is never hidden: a merged or deleted branch raises one dream question. Recall output is byte-identical with no conditions stored (277 frozen + 147 fresh replayed messages, 0 differing). Dream also offers memories that name a stopped swarm agent, 10 per pass, with the tickets that agent's session worked on: on the local store 291 of 29,118 memories name an agent (precision 50/50 on a hand-checked sample), 159 name one that has stopped, 119 of those with a ticket. Capture is unchanged (ez50rb).

## v0.7.36 - 2026-11-16

- `hug tui` offers to reload when a newer hug is installed where it was started from (`brew upgrade` moves the `hug` symlink to a new Cellar folder; `go install` replaces the file): only the version corner changes, to `hug v0.7.35 → v0.7.36 · R reloads`, and nothing else happens until R. R becomes the new hug in the same terminal, on the same view, task, filters, sort, focus and scroll, and says `reloaded hug vX.Y.Z`; inside a prompt, the palette or help R is just a key there. A new binary that does not start, or an older one, leaves the running TUI as it is, with the reason and a `fix:` line. Not on Windows, which has no exec (j2vwk0).

## v0.7.35 - 2026-11-16

- The recall hook keeps the task board current: when the work and the board disagree it adds one line with the command that fixes it, e.g. `Task board: you are working on p277bp but it is todo. hug task take p277bp --model claude-opus-5-5`. It says so when code changes with no task doing (`hug task take ID`, or `hug task next` / `hug task new`), when the session's task has every check ticked (`hug task done ID`), and at a break hands out one lint finding with its `hug task ... ID` fix (one per 6 h; each finding goes to one agent, then rests 7 days). At most one line per turn, one per kind per hour. A task is the session's when it took it, its name is the assignee, or the task's recorded worktree or branch is where it works, never on model alone. `"nudge_board": false` in muninn config.json turns it off. Replayed on 46 real sessions (378 session-hours): precision 0.88 to 1.00 per set, 0.30 false lines per 10 session-hours at most; 3 lines on the 15-16 Nov overnight run where no task was taken; board read p95 27 ms on hug's board, 144 ms on seid-lang's 2,508 tasks (muninn/docs/BOARD4.md) (qawmnb).
- `hug task lint` and `board` at 10,000 tasks: a dependency or parent is looked up in an index, not by scanning the board, and lint lists each group folder once instead of checking every task path part by part: lint 2.65 s -> 2.18 s, board 0.42 s -> 0.35 s, ls 1.42 s -> 1.17 s (44w1ev, step 2).

## v0.7.34 - 2026-11-16

- `hug task` reads the board once per command instead of up to six times: at 10,000 tasks `next` and `show` drop from 0.59 s / 0.71 s to 0.34 s / 0.33 s, `note` from 0.99 s to 0.60 s, `ls` from 1.63 s to 1.41 s (44w1ev, step 1).

## v0.7.33 - 2026-11-16

- `hug tui` rows show acceptance progress before the id, a pie that fills by quarters and ticked/total, e.g. `◑ 2/4 8bsx4y`: green when all are ticked, yellow when some are, dim when none are; a task without checks leaves the cell blank. `hug task ls` prints ticked/total after the type, e.g. `1/2`, so the id stays the first field.
- A task reads top down: Goal, Purpose, Principle, Relevant details, Depends on, Acceptance, any other section, then Notes and History. `hug tui` shows Purpose, Principle and Relevant details under the goal, above the checks (they used to come after History). `hug task show` prints them after `goal:` and adds `purpose`, `principle` and `details` to `--json`. A new note on a task with History goes above it. `hug task lint` flags a file out of order (`section-order`, fix: `hug upgrade`), and `hug upgrade` puts every task file in order without changing any section's text, then commits (jnaspy).

## v0.7.32 - 2026-11-16

- `hug tui` ends the key bar with the version, `hug vX.Y.Z`, right-aligned and dim, the same string `hug version` prints; when the line is too narrow the version goes first and no key hint is pushed off (8bsx4y, from Jørgen).
- `hug tui` overview: rows the focused section does not need are no longer left blank; they go to IN PROGRESS until it shows all its tasks, then to NEXT UP, then the rest. With IN PROGRESS focused holding 2 tasks and NEXT UP 32, a 140x40 terminal now shows 21 of NEXT UP instead of 3 (z6kz20).
- `hug tui` shows a task's attached files: `⎘2` before the title on its row in overview, list and board, and a FILES section inside the task. Enter on a file opens it in the editor, and `E` opens the task's whole folder; the key shows only for a task with files. When the editor closes, the change is committed (`hug task edit ID file NAME`, `hug task edit ID files`) (zgr01t).

## v0.7.31 - 2026-11-16

- `hug task take` records the branch and worktree it runs in (`branch`, `worktree` on the task, `--branch`/`--worktree` override). `hug task show` prints `taken by: dragon · claude-opus-5-5 · feat/x`; `hug task ls` rows add the branch when it is not the main one; `ls --json` carries both (x1t1t1).
- MCP `task_take` requires `cwd`, the folder the agent works in: without it, with a relative path, or outside git it fails with a `fix:` line; the branch and worktree are read from that folder, not the server's (x1t1t1).
- `hug mem` records the branch on memories remembered or captured in a linked worktree or on a branch other than the main one (x1t1t1).
- `hug task lint` flags a doing task whose branch is merged into main or deleted (`[branch-gone]`, `fix: hug task done ID`); nothing is closed automatically. `hug tui` names the branch in the detail header (x1t1t1).

## v0.7.30 - 2026-11-16

- `hug tui` sorts by fields: `: sort` opens a builder where a letter adds a field (p priority, t type, s status, u updated, c created, g group, a assignee, n title), the same letter again flips its direction, ⌫ removes the last, Enter applies and Esc keeps the order there was. The header shows the order, e.g. `sort: priority ↓ · type`, and `: sort rank` goes back to rank order. While sorted, J/K, m, y and P/p move nothing and say `moving needs rank order: : sort rank` (mkzcqr).

## v0.7.29 - 2026-11-16

- `hug mem` writes wait their turn instead of failing when many agents write at once. Every write (remember, forget, rescope, correct, tidy, index) queues for one write turn, a lock file waited on in the kernel; a waiter fails only when one holder keeps it 20 minutes, and then names the holder (pid, command, since) with a `fix:` line. A waiter prints `hug mem: waiting 12s for the write turn, held by pid N (hug mem forget ...)` on stderr after 2 s and every 15 s after. Before, waiters shared SQLite's 30 s busy_timeout as one budget for the whole queue: on a copy of the real store 19 of 20 parallel forgets failed at 31 s with `database is locked (5) (SQLITE_BUSY)` behind one 32 s index pass; now 20 of 20 succeed (me51r2).
- `hug mem forget ID ID ...` and `hug mem forget --stdin` (one id per line; `hug mem list` lines work) forget many memories in one turn, with one save and one index pass; one unknown id fails the call and forgets nothing. `--json` prints `{"forgotten": [...]}` (me51r2).
- `hug mem list --json` and `hug mem search --json` print one object, `{"memories": [...]}`, each with id, kind, scope, category, type, tags, text and updated (search adds score) (me51r2).
- A recall hook's skipped index pass no longer leaves its connection's busy_timeout at 5 s instead of 30 s (me51r2).
- `hug tui` shows inline code and code blocks in a quiet blue-grey on a faint background (`#c9d6e3` on `#1a2630` dark, `#2b3a48` on `#e8eef3` light), without padding and without red; red is left for blocked and lint (8bsx4y).

## v0.7.28 - 2026-11-16

- `hug tui` overview: the section you are in takes every spare row and the others keep 3; RECENTLY DONE lists every closed task newest first and scrolls through them when focused; each section and board column keeps its scroll and row when you Tab away and back; an empty section or column is one line, `None` on its title (h9zghj).

## v0.7.27 - 2026-11-16

- CI runs the race detector once, on ubuntu; macOS and Windows run plain
  tests, so the release gate takes about 8 minutes instead of 14.
- `hug tui` shows task ids bare in their colour, without brackets, in rows, the detail header and the trail; an open task lists every `task:ID` its text names under Links, and Enter follows one (Esc comes back) (xxvm6t).

## v0.7.26 - 2026-11-16

- `hug tui` shows the model next to the assignee in a task's detail:
  `assigned:poodle · claude-opus-5-5` (et85h1).
- Every command and MCP tool that takes a task ref accepts `task:ID`, the form notes and goals use, as well as the bare id; printed commands keep the bare id (xxvm6t).
- CI is green on Windows again: the lint, auto-refresh and memory worktree tests no longer assume a shell, a fast `git` or forward-slash paths (4emme8).

## v0.7.25 - 2026-11-16

### Fixed

- `hug task promote ID N` gives the new task the source's Purpose,
  Principle and Relevant details, so it starts complete and `hug task lint`
  reports it clean (it used to get `missing section Purpose, Principle`).
  A source without them gives a child without them. (heype5)
- `hug task tui` shows text in angle brackets as written in the task
  detail: a note like `git worktree add --detach <dir> <rev>` no longer
  loses `<dir>` and `<rev>`, which the markdown renderer read as HTML.
  Code spans and fenced blocks are unchanged. (heype5)

## v0.7.24 - 2026-11-16

### Changed

- `hug tui` moves a task far in one go, one commit per move (3xcesq).
  `m` grabs the selected task: `j`/`k` (with a count such as `5k`), `gg`
  and `G` move it on screen, nothing is written until `Enter` drops it
  with one commit, and `Esc` puts it back. The footer says `Moving ID · row
  N of M in SECTION (was K) · nothing written until Enter` and the key bar
  shows only the keys that work. `y` yanks the selected task, then `P`
  pastes it above the selected row and `p` below. In lists `P` and `p` only
  paste: with nothing yanked they are not shown and say "nothing yanked: y
  on a task first"; `k` moves up. `J` and `K` move one row with no confirm, since the other key
  undoes it. Moves stay within the rows shown in one section; RECENTLY
  DONE, ordered by close time, refuses them and says where to move. In
  lists, the type palette moved from `y` to `:type` and `f t` (`y` inside
  a task still opens it).

## v0.7.23 - 2026-11-16

### Changed

- `hug task take` needs the model doing the work: `--model claude-opus-5-5`
  (or `HUG_MODEL`; a person gives `--model human`). Without it, take changes
  nothing and ends with `fix: hug task take ID --model MODEL`. The agent or
  session name is optional: `--as poodle`, otherwise `HUG_USER`, otherwise
  the jcode session name (from `JCODE_SESSION_ID`); with no name the
  assignee is the model. hug never uses `git config user.name` and no
  longer reads `HUGINN_USER`. Both are recorded on the task (`assignee`,
  and a new `model` field), and `hug task show` (`taken by: poodle ·
  claude-opus-5-5`), `ls` (doing rows), `board --wide` and JSON show them.
  The MCP tool `task_take` takes `model` and `agent`. `--mine` means the
  same name take records. `hug setup` and the AGENTS.md section tell
  agents to name their model on take. (et85h1)

### Fixed

- `hug task tui` puts every task in one place. A task in progress that waits
  on a dependency stays in IN PROGRESS, drawn blocked, and no longer shows a
  second time in BLOCKED; BLOCKED holds only tasks nobody started that
  cannot start. The header counts no longer overlap and add up to the tasks
  shown, and doing says how many of its tasks are blocked: `todo 8  ·  doing
  15 (1 blocked)  ·  blocked 7  ·  done 15  ·  dropped 15` (it used to read
  `todo 15 · doing 15 · blocked 8` on the same 60 tasks). The board columns
  and the status filter (`blocked`, `todo`, keys 1 to 4) follow the same
  rule. (zaec08)

### Added

- The overview has a RECENTLY DONE section, between BLOCKED and LINT: the 3
  tasks closed last, newest first, by the commit that closed them (`hug
  task done` or `drop`, or a `set` or edit that made the status done or
  dropped), with how many are closed in all. On a short terminal it gives
  way first and keeps one row. (zaec08)

## v0.7.22 - 2026-11-16

### Fixed

- `hug mem sync` in a linked Git worktree (`git worktree add`) writes and
  commits project memories in the main worktree's one `.muninn-notes/`
  checkout, the same rule as tasks. It used to check `muninn-notes` out
  under the linked worktree, which git refuses ("already used by worktree")
  when the main worktree has it, or else made a second, separate checkout
  there. `hug mem project` shows the main worktree's file. (065wwn)

### Added

- `hug task tui` refreshes by itself when tasks change on disk: a
  `hug task note`, `set` or `take` in another terminal or by an agent shows
  within a fraction of a second, with no key pressed. The selected task, the
  list's scroll and the open task's scroll stay put. An open search, palette,
  confirm question or editor is never interrupted; the change shows once it
  closes. The TUI asks the OS to tell it when the task branch's ref, the
  `packed-refs` file or the tasks folder changes, so it does nothing while
  nothing changes; where the OS cannot do that, it checks the ref's
  modification time every 2 seconds. `r` still refreshes at once. (nfv18p)

## v0.7.21 - 2026-11-16

### Fixed

- `hug task` works in a linked Git worktree (`git worktree add`). It used
  to try a second checkout of `hug/tasks`, which git refuses ("already used
  by worktree"), and its `fix:` line repeated the same failing command. Now
  every linked worktree uses the one checkout at the main worktree's
  `.hug/tasks/`, so `hug task ls`, `new`, `take`, the MCP tools and the TUI
  read and write the same board from any worktree. `hug task init` in a
  linked worktree of a fresh repository creates the board in the main
  worktree. A checkout folder that was deleted or that git no longer knows
  now gets a `fix:` that works (`git worktree prune`, or move it aside and
  check the branch out again). (065wwn)

## v0.7.20 - 2026-11-16

### Fixed

- In `hug task tui`, J and K move a task past the row shown below or
  above it, in the section on screen (NEXT UP, a board column, the list)
  and after filters. They used to move it past the next task in the full
  order, often an epic, a closed task or a filtered-out one, so a press
  made a commit and nothing on screen changed. The confirm line names both
  tasks (`Move a1dx7s below x1t1t1 in NEXT UP: ...?`), the result says
  where it went (`moved a1dx7s below x1t1t1, row 2 of 3 in NEXT UP`), and
  at the first or last row shown the message says so and nothing is
  committed.

## v0.7.19 - 2026-11-16

v0.7.18 (tagged at 1fe040f) carried everything below except the hiding of
findings on closed tasks.

### Fixed

- `hug task lint` is never silent: a clean board prints `no findings` and a
  `next:` line. On a board with more than 30 findings, plain lint prints a
  summary by code, the first 10 findings and how to see the rest
  (`--all`), instead of every one. On a 2,478-finding board the old
  318 KB output came back as nothing in an agent's tool panel.
- `hug task lint help` and `hug task lint help --code` print the lint help
  (they said "expected 0 positional arguments" and "missing value for
  --code"). The same goes for other commands without arguments (`ls help`,
  `next help`, ...). `new help` still makes a task titled "help".
- The `not-a-dependency` finding starts with what to do when a task has
  no dependencies: "(for no dependencies, leave the section empty or
  remove it; ...". It used to suggest `none` was missing a form.

### Added

- `hug task lint --summary`: one line per code with its count, its meaning
  and the command that lists it, aligned with spaces. JSON lint replies
  carry `summary` (`total`, `codes`) and still list every finding.
- `hug task lint --code` with no value (or `--code help`) lists every code
  with its count and meaning. An unknown code says so, names the closest,
  lists the valid ones and ends in `fix: hug task lint --code`.
- `hug task lint --all` prints every finding, including those on done
  and dropped tasks.

### Changed

- `hug task lint` (CLI, MCP `task_lint` and the TUI) leaves out findings
  on done and dropped tasks and ends with how many it left out:
  `986 findings on closed tasks hidden (432 done-unchecked, ...): hug task lint --all`.
  JSON replies carry `hidden` and `summary.hidden_codes`. Opening a closed
  task in the TUI still shows its own findings.
- A mistyped flag's `fix:` is that command's help (`hug task lint --help`),
  not the general index.

## v0.7.17 - 2026-11-16

### Fixed

- A line in `## Depends on` that is not a dependency (prose, a question) no
  longer becomes one. It used to take the first word as a task id: a
  question showed the task as blocked on "Is". A dependency line is
  `- <task id>` (or an alias like `TASK-123`, optionally `task:ID`, then a
  note); anything else is reported by `hug task lint` with the new code
  `not-a-dependency`, quoting the line and showing the form to write. It
  creates no dependency and never blocks the task.
- `hug task lint`'s `next:` repairs the first finding (its fix command)
  instead of naming an unrelated task to take. Same in `--json`. A clean
  board keeps the usual next.

### Changed

- TUI shortcut letters are picked by one rule shared by the header, palette
  and help: each choice gets its first letter no earlier choice took (todo,
  doing, blocked, done, dropped become t, d, b, o, r), and a label that
  starts with a sigil, like `!lint`, gets that sigil.
- Every lint finding carries a fix: text output prints `fix: COMMAND` under
  each finding, JSON findings have a `fix` field. A dependency on no task
  now reads "depends on ID, but no task has that id or alias; remove the
  line, or write an existing task's id" with `fix: hug task dep TASK
  --remove ID` (code `dangling-dependency` unchanged).
- TUI: after an edit that leaves the task with a new finding, the footer
  shows it (task, line, problem) and a bar with its fix and `e edit again ·
  Esc dismiss`; `e` reopens the editor on that task at once. The key shows
  only while the finding is there. The task detail header shows the
  finding's line and, on its own line, the fix.
- `hug task edit`'s `next:` is the fix for the edited task's first finding.
- The task cache schema is 4, so lint stops serving findings cached under
  the old dependency reading.

## v0.7.16 - 2026-11-16

### Fixed

- TUI palette: after `f`, a field can be shortened, so `: f g ` lists the
  groups and `: f t spike` filters by type. It said "No completions".

### Changed

- TUI task detail has three focus states: page, acceptance and dependencies.
  Tab and Shift-Tab cycle them and skip empty sections; a task opens on its
  checks when it has any. Only the focused heading is highlighted (with a
  `▸`); with page focus no check shows the `>` pointer, j/k scroll the page
  and the task id in the tab bar carries the focus.
- Inside a task the tab bar marks no view; it shows the trail instead, for
  example `list › 000001`, on the same line.
- Key hints follow focus and show only keys that work there. Keys are never
  written as `[ / ]` (read as the `/` key): headings say `j/k or [ ] step`.
  Space ticks only with the checks focused.
- Every confirm prompt (take, done, drop, move, edit, new, tick) is one
  full-width bold warning-coloured bar: `▶ …? Enter yes · Esc no`, the
  question and its keys surviving truncation. The tick wording stays
  `Tick: ☐ 2. text`; the edit prompt names the editor it will run
  (`Edit qawmnb in subl --wait?`), and help says how it is chosen ($VISUAL,
  else $EDITOR, git core.editor, vi).
- Board legend names its columns: `Type: F feature … · Priority: C critical
  …` with T and P emphasised; narrow boards show `T: … · P: …`.
- The task header shows no empty `[]` badge for an unset type or priority.
- In a task, `n` and `p` open the next and previous task (Ctrl-N and Ctrl-P
  still work); `{` and `}` are gone, being hard to reach on a Norwegian
  keyboard. In lists `n` and `p` move like `j` and `k`. New task is `N` in
  every view.

## v0.7.15 - 2026-11-16

### Added

- State-change reminders: `hug task take`, `check`, `done`, `drop` and
  `promote` print `now:` lines saying what to do at that moment (MCP: a
  `guide` field). Take says to keep one todo named with the task id, not one
  per check, and that the task stays doing through detours. Done on an epic's
  last child says to close the epic.
- TUI detail: `{` and `}` (also Ctrl-P and Ctrl-N) open the previous and next
  task of the list the detail was opened from. The footer shows "Task 3 of 22".
- TUI palette is a tree and uses the full height: `filter`, then the field,
  then its values with task counts. `: f done` finds a value without naming
  its field. The short forms (`status todo`, `done`) still run when typed.

### Changed

- TUI rows: the lint mark `!` sits in the gutter left of the id instead of
  its own column. Board rows show type and priority as one coloured letter,
  with a legend under the board, so titles get the width.
- TUI: Space in a list no longer asks to tick a check you cannot see. It
  points to Enter. Inside a task the prompt reads `Tick: ☐ 1. text`, the
  check drawn as the ticket shows it.
- TUI detail: open acceptance checks use the plain text colour, so they no
  longer look like the blue headings. Ticked checks stay green.

### Fixed

- `hug mem --help` and `hug mem` errors say `hug mem` (was `muninn`), and an
  unknown command ends with a `fix:` line.

## v0.7.14 - 2026-11-16

v0.7.13 was tagged but not published: its own test fixtures tripped the
release's no-secrets gate. v0.7.14 carries the same changes plus:

- TUI: the help bar is limited to the terminal width.

## v0.7.13 - 2026-11-16 (not published)

### Fixed

- The secret check (memory writes and task attach) catches keys stored as
  `name=value` under a secret-like name, such as `aws_secret_access_key=...`
  or `DB_PASSWORD: ...`, when the value looks like a key (12+ characters
  mixing letters and digits). On the 28,476 stored memories here it flags
  nothing new; sudoers `NOPASSWD:` lines and variable names stay unflagged.

## v0.7.12 - 2026-11-16

### Fixed

- `hug task show ID --file NAME` prints the file. Since v0.7.4 it printed
  only the next hint unless `--json` was given.
- `hug task take` commits as `hug task take ID` (was "Update hug/tasks"),
  and its check for unpushed tasks looks at the current task branch
  (it still named the old `huginn` branch).
- TUI help lists the filter keys s/t/p/g/l/a/e and the `epic` command.

## v0.7.11 - 2026-11-16

### Fixed

- On a machine without a git identity, task commands say so and give the
  `git config --global user.name/user.email` fix (found in a clean Linux
  container), instead of suggesting to re-run the failed commit.
- TUI commits read `hug task check ID` (not `huginn: check ID`), and every
  version line says `hug vX.Y.Z`.

## v0.7.10 - 2026-11-16

### Fixed

- Speed with epics: parents are resolved once per command. On 2,498
  seid-lang tasks `hug task ls` went from 19.1 s to 0.8 s and the TUI starts
  in 1.4 s instead of 18.2 s.
- TUI: NEXT UP shows what `hug task next` would pick (unassigned, not an
  epic); the header says HUG TASKS; the Space prompt no longer ends in "??".

## v0.7.9 - 2026-11-16

### Changed

- `hug mem` commands end with a `next:` hint, like `hug task` (on stderr
  for commands that print JSON, so stdout stays parseable; hooks, mcp,
  serve and tui stay silent).

## v0.7.8 - 2026-11-16

### Changed

- Memory TUI: places read as folder names ("hug" instead of
  "project:rc-0ca4bf6be5d989c3") and plain words for shared scopes
  ("all agents", "jcode, all projects"). The names come from the folders
  jcode sessions ran in, scanned in the background and cached for a day.

## v0.7.7 - 2026-11-16

### Added

- Task description pattern: `hug task new --purpose --principle --details`
  (and the same `task_new` fields) write Purpose, Principle and Relevant
  details after Goal. Lint (`missing-section`) asks for Purpose and
  Principle on open tasks created from today; older tasks are not flagged.
  The names come from a blind test: named sections scored +0.49 [+0.28,
  +0.71] over none (docs/task-vocabulary-test.md).

## v0.7.6 - 2026-11-16

### Added

- Lint findings carry a code, shown as `[code]` and as `code` in JSON
  (long-title, no-acceptance, parent-too-deep, ...). `hug task lint --code
  CODE` keeps one kind, so problems can be fixed one by one.
- `hug task board --wide`: one section per column with full titles.
- This repository has an AGENTS.md; setup's agent section now covers the
  task description pattern, epics and promote.

### Changed

- `hug task init` makes the first commit and says what it set up: branch,
  worktree, head and remote.

## v0.7.5 - 2026-11-16

### Added

- Epics. `hug task set ID --parent EPIC` puts a task under an epic (epic >
  task > acceptance item; an epic has no parent, `--parent none` clears).
  `show` lists the epic and its children; `done` on an epic refuses while a
  child is open (exit 3); lint reports a missing parent or one that is too
  deep, with the exact fix.
- `hug task import backlog` maps `parent_task_id` to the parent epic. On a
  copy of seid-lang 282 of 294 links came back; rerun to restore them.
- `hug task promote ID N`: acceptance item N becomes its own task, ranked
  right after ID under the same epic. The item then reads "done when
  task:NEW is done" and is ticked when NEW is done. MCP: `task_update`
  action `promote`.
- `--epic EPIC` on `ls`, `next` and `board` (and MCP list/next); `next`
  never picks an epic itself. TUI: filter field `e` and `:epic ID`.
- `hug apps [install] [--dry-run]`: Claude Desktop, Claude Code and
  ChatGPT/Codex get the hug MCP server (backup kept, other servers kept).
  `hug setup` lists it.
- Lint: open tasks with titles over 80 characters, with the fix.

### Changed

- When the remote refuses SSH (apps start MCP servers without your SSH
  agent), the error says so and gives a fix that works; reading tasks never
  needs the remote.
- `hug mcp` notes in its replies when Homebrew upgraded hug underneath it,
  so the app gets restarted.
- TUI: the selected row is tinted across its full width.

## v0.7.4 - 2026-11-16

### Added

- Task files: `hug task attach ID FILE`, `hug task show ID --file NAME`,
  `hug task detach ID NAME`. A task's files live in a folder named by its id
  next to the task file. Text only, 256 KiB each, secrets refused. Notes cite
  them as `file:NAME` (own task) or `task:REF/NAME`; lint flags missing ones.
- `hug task import files DIR [--cite PREFIX]` moves an existing evidence
  folder into task files and rewrites citations (seid-lang:
  `hug task import files .backlog/evidence --cite .backlog/evidence`).
- `hug task ls --jsonl`: one JSON task per line.

### Changed

- Settings file is `hug.toml` (`huginn.toml` still read; `hug upgrade`
  renames it).
- Every message, fix and help line names `hug task ...` / `hug mem ...`.
  `hug version` prints one version.
- `hug task ls` columns: id, status, type, slug, title.
- New slugs: the first four meaningful words of the title, at most 32
  characters. Existing file names are unchanged.
- `hug task rebalance` prints one summary line, and its help explains it
  only renames filenames (order and contents unchanged).
- Recall: `skip_records_above` defaults to 600. On 147 fresh real messages,
  judged blind, long messages got 2.56 fewer irrelevant memories each
  [CI -3.00, -2.08] with relevant unchanged (-0.08 [-0.26, +0.08]).
- TUI: acceptance items wrap with a hanging indent, the selected item is
  highlighted, j/k step through items; Space asks "Tick item N of ID: ...?";
  o/i/b/! work in detail; overview sizes sections by need with ↑/↓ marks
  where rows are hidden; "owner" is now "assigned".

## v0.7.3 - 2026-11-15

### Fixed

- The jcode recall hook no longer delivers a memory after the turn it was
  recalled for has ended. Before, jcode started a new turn with it, which
  looked like an empty "[transcription]" message from the user.
- `hug task import backlog .` explains that the repository root holds the task
  folder and suggests `huginn import backlog .backlog --dry-run`.
- `hug setup` lists the `task_*` MCP tool names that `hug mcp` serves.

### Added

- Memory type: rule, preference, fact or record. `memory_remember` and
  `muninn remember --type` take it, capture's existing model call sets it, and
  `muninn types` fills it in for older memories (resumable, `--dry-run`).
- `skip_records_above` (off by default): automatic recall can leave "record"
  memories out for long messages; `memory_search` still finds them.
- `hug task import github [OWNER/REPO] [--dry-run]`: issues become tasks
  through the gh CLI, with alias GH-<number>; re-running syncs, huginn edits win.
- `hug upgrade`: tasks move from branch `huginn` at `.huginn/` to branch
  `hug/tasks` at `.hug/tasks/` (branch rename and worktree move, history
  unchanged, remote branch renamed). Task commands on an old repository stop
  with `fix: hug upgrade`, and `hug setup` lists it first. Safe to run again.

## v0.7.2 - 2026-11-14

### Changed

- Recall scores keyword matches with BM25 (`bm25_scale: 2.0`): rare words such
  as ticket ids and branch names count more, common words less. On 277 replayed
  real messages, judged blind, long messages got 0.39 more relevant memories
  each [95% CI +0.25, +0.55] with no more irrelevant ones; short messages are
  unchanged. `bm25_scale: 0` restores the old scoring. Details:
  muninn/docs/RECALL4.md.

## v0.7.1 - 2026-11-13

### Added

- `hug mcp`: one MCP server with the memory tools (`memory_*`) and the task
  tools (`task_list`, `task_next`, `task_take`, ...). If memory cannot open, the
  task tools are still served. `muninn mcp` and `huginn mcp` are unchanged.
- The installers leave a configured `hug` server alone instead of adding
  separate muninn/huginn servers; `hug setup` suggests `hug mcp`.

### Fixed

- Hooks record the path the program was started from (e.g.
  `/opt/homebrew/bin/muninn`), not the versioned Cellar path, which broke on
  every Homebrew upgrade.

## v0.7.0 - 2026-11-13

### Added

- One binary, `hug`: `hug task ...` (huginn), `hug mem ...` (muninn), `hug setup`,
  `hug tui`, `hug version`. Called as `muninn` or `huginn` it acts as that part,
  so existing hooks and MCP configs keep working.
- One release per tag: `hug_<version>_<os>_<arch>` archives with hug, muninn and
  huginn; Homebrew `veridit/tap/hug` (the `muninn` and `huginn` formulas now
  install hug); Scoop `hug`; one root `install.sh`.
- `muninn dream --summary`, recall counts in `muninn show`, and dream items that
  ask the driving agent to vet memories unrecalled for 60 days (never archived
  automatically).

### Fixed

- Hooks and MCP configs installed from hug name the `muninn`/`huginn` link, not
  `hug`, which does not know `hook` or `mcp` at the top level.
