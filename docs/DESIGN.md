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
