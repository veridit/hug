# AGENTS.md

hug is one Go binary for agent memory and tasks. The name is Old Norse
*hugr* (mind); the mascots are Odin's ravens Huginn (thought: tasks) and
Muninn (memory). Users type `hug`; `huginn` and `muninn` remain as
compatibility links only. Never tell a user to type them.

## Layout

- `cmd/hug`: the binary: dispatch (`hug task`, `hug mem`, `hug mcp`,
  `hug apps`, `hug setup`, `hug upgrade`, `hug version`).
- `huginn/`: tasks. `internal/task` (model, store, lint, epics, promote),
  `internal/cli` (commands, setup, help), `internal/mcp` (task_* tools),
  `internal/tui`, `internal/importer` (backlog, github, files),
  `gitbranch` (the `hug/tasks` branch worktree).
- `muninn/`: memory. `internal/recall`, `internal/search`, `internal/store`,
  `internal/extract`, `internal/harness` (hooks), `cmd/recall4` (replay).
- Tasks for this repo live on branch `hug/tasks`, checked out at
  `.hug/tasks/`. Never on code branches.

## Building and testing

- The owner's machine is shared: run builds and tests niced and bounded:
  `GOMAXPROCS=2 GOFLAGS=-p=2 nice -n 19 go test ./...`.
- TUI goldens: `HUG_UPDATE_GOLDEN=1 go test -run TestTmuxGolden ./huginn/internal/tui/`,
  then review the diff before committing.
- A test that fails when code gets slower is a benchmark, not a test: name it
  `Test...Bench...`, skip it unless `HUG_BENCH=1`, and compare the new path
  with the old one in the same process (a ratio, never an absolute time). CI
  runs them in the `bench` job (ubuntu, no -race):
  `HUG_BENCH=1 go test -run Bench ./...`. A bound that proves a timeout or
  a non-wait (a hanging remote gives up) stays a normal test.
- Messages, fixes and help must name `hug task ...` / `hug mem ...`, and every
  error ends with a `fix:` line an agent can run.
- Landing: rebase onto origin/main, run the touched packages' tests locally
  with `-race` (as CI's ubuntu job does), then push straight to main. No
  pre-push CI run. If main CI goes red from your commit, fixing it forward
  comes first; revert if it is not fixed within 30 minutes.
- Releases: hourly, from the newest main commit green on every job. Add your
  CHANGELOG.md line under `## Unreleased`; the release turns it into
  `vX.Y.Z`, tags, pushes, and CI publishes the Homebrew formula
  (`brew upgrade veridit/tap/hug`).

## Recall changes

Ship a recall change only with a measured, CI-clear gain on real replayed
messages, confirmed on fresh data and judged blind (`muninn/docs/RECALL4.md`).
Recall itself stays local and free; paid model calls happen only at write time.

## Tasks: hug
Track all work with hug tasks (on the git branch "hug/tasks", at .hug/tasks/).
Start: hug task next, then hug task take ID --model MODEL (the model you are, e.g.
claude-opus-5-5 or gpt-6.1-sol; a person: --model human; --as NAME adds your agent
or session name, optional), then hug task show ID.
While working: hug task check ID N with evidence, hug task note ID "evidence".
Finish: hug task done ID when every acceptance check is ticked.
Todos are your own next steps on the task you took, nothing else: details,
evidence, waits and dependencies go in tasks (hug task note, hug task dep), never
in todos, and another agent's work is never your todo. When your todos run out,
finish the task and run hug task next: an empty todo list means take the next
task, not blocked.
New work: hug task new "short title" --goal "..." --purpose "..." --principle "..." --details "..." --accept "observable check".
Describe a task with Goal (where we aim), Purpose (why), Principle (what we do
and do not do), Relevant details, then Acceptance. Titles stay under 80
characters; detail goes in the Goal.
Epics: hug task set ID --parent EPIC; hug task promote ID N turns item N into
its own task. hug task ls --epic EPIC shows an epic and its children.
MCP tools (hug mcp): task_list, task_show, task_new, task_next, task_take (model
and cwd required: your working folder, its branch is recorded; agent optional),
task_done, task_update (actions set, check, promote, note, move, dep, drop).
Old backlog ids (TASK-123) work as aliases.
Never edit .hug/tasks/ files by hand while another agent may be writing; use the commands.
