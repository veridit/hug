# Changelog

Releases are tagged `vX.Y.Z` and ship one `hug` binary. Earlier part releases
are in [muninn/CHANGELOG.md](muninn/CHANGELOG.md) and the `muninn-v*` and
`huginn-v*` tags.

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
