# hug

One tool for an AI-assisted project's **memory**, **tasks** and (soon) **messages**,
shared by every agent and harness you use: jcode, Claude Code, Codex, the Claude
app, and plain shells.

The name: Old Norse *hugr* is the mind, thought and intent, the root of Huginn,
Odin's raven of thought. Odin's other raven, Muninn, is memory. And in English, a hug
holds things together.

## Install

| Platform | Install | Update |
|---|---|---|
| macOS, Linux (Homebrew) | `brew trust --formula veridit/tap/hug`<br>`brew install veridit/tap/hug` | `brew upgrade hug` |
| Windows (Scoop) | `scoop bucket add veridit https://github.com/veridit/homebrew-tap`<br>`scoop install hug` | `scoop update hug` |
| macOS, Linux (no package manager) | `curl -fsSL https://raw.githubusercontent.com/veridit/hug/main/install.sh \| sh` | run it again |
| Go (from a clone) | `git clone https://github.com/veridit/hug && cd hug && go install ./cmd/hug` | pull, run again |

Requires Git 2.42 or newer on PATH. Every install also provides `muninn` and
`huginn`, which are the same program under the part's name, so existing hooks and
MCP configs keep working. `brew install veridit/tap/muninn` (or `huginn`) still
works and installs hug.

Then, inside a repository:

```sh
hug setup                     # read-only checklist of what is wired and what is not
hug mem install all           # memory hooks for Claude Code, Codex and jcode; keeps backups
hug task init                 # task branch for this repository
brew services start hug       # optional: memory watcher + web UI on http://127.0.0.1:7431
```

## Commands

| Command | Same as | What it does |
|---|---|---|
| `hug mem ...` | `muninn ...` | memory: recall hooks, MCP server, capture, tidy, dream |
| `hug task ...` | `huginn ...` | tasks: Markdown files on a git branch, claims, goals, checks |
| `hug task search "words"` | | tasks that best match, ranked (BM25 over id, aliases, title, goal, notes, ...), with the field and a snippet; MCP `task_search` |
| `hug tui` | | the tasks TUI (`hug mem tui` is the memory TUI) |
| `hug mcp` | | one MCP server: `memory_*` and `task_*` tools |
| `hug setup` | `huginn setup` | read-only setup checklist |
| `hug task install commit-msg` | | refuse commits without one `Model:` trailer |
| `hug version` | | versions of hug and both parts |

## Who made a commit

Every commit names what made it in one trailer line: `Model: claude-opus-5-5`
(the model id), or `Model: human` when the git author wrote it themselves.
`hug task install commit-msg` refuses a commit without exactly one, with a
`fix:` line to finish it; `hug setup` lists it until it is in place. With
`core.hooksPath` (a project's own `.githooks`) hug never edits the tracked hook
and prints the two lines to add instead. hug's own commits on `hug/tasks` and
`muninn-notes` add `Model:` from `--model` or `HUG_MODEL` (the TUI: `human`).
Emergency exit: `git commit --no-verify`.

```sh
git log --format='%h %(trailers:key=Model,valueonly,separator=%x2C) %s'   # every commit and its model
git log --grep='^Model: gpt-6.1-sol$' --oneline            # one model's commits
git -C .hug/tasks log --format='%(trailers:key=Model,valueonly,separator=%x2C) %s'   # task changes
```

## Layout

| Folder | Part | Details |
|---|---|---|
| [`muninn/`](muninn/) | memory | [muninn/README.md](muninn/README.md) |
| [`huginn/`](huginn/) | tasks | [huginn/README.md](huginn/README.md) |
| [`cmd/hug/`](cmd/hug/) | the `hug` command | dispatches to the parts |

Coming next: files on namespaced git branches (`hug/tasks`, `hug/memories`,
`hug/messages`) as the truth, a local SQLite cache for speed, and one MCP server
for every harness.

## Releases

One tag, `vX.Y.Z`, releases everything: archives `hug_<version>_<os>_<arch>`,
the Homebrew formula and the Scoop manifest. Before v0.7.0 the parts were released
separately as `muninn-v*` and `huginn-v*`.

## License

[MPL-2.0](LICENSE). Contributions need the contributor agreement in
[CONTRIBUTING.md](CONTRIBUTING.md).
