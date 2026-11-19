# hug

One tool for an AI-assisted project's **memory**, **tasks** and (soon) **messages**,
shared by every agent and harness you use: jcode, Claude Code, Codex, the Claude
app, and plain shells.

You type one command, `hug`. The name is Old Norse *hugr*: mind, thought and intent
(Norwegian still says *komme i hug*, to come to mind). Odin's two ravens are named
from it and are hug's mascots: Huginn, thought, keeps the tasks, and Muninn, memory,
keeps the memories. And in English, a hug holds things together.

## Install

| Platform | Install | Update |
|---|---|---|
| macOS, Linux (Homebrew) | `brew trust --formula veridit/tap/hug`<br>`brew install veridit/tap/hug` | `brew upgrade hug` |
| Windows (Scoop) | `scoop bucket add veridit https://github.com/veridit/homebrew-tap`<br>`scoop install hug` | `scoop update hug` |
| macOS, Linux (no package manager) | `curl -fsSL https://raw.githubusercontent.com/veridit/hug/main/install.sh \| sh` | run it again |
| Go (from a clone) | `git clone https://github.com/veridit/hug && cd hug && go install ./cmd/hug` | pull, run again |

Requires Git 2.42 or newer on PATH. Machines set up before hug was one tool keep
working: their hooks, MCP configs and Homebrew formulas named after the ravens now
run or install hug.

Then, inside a repository:

```sh
hug setup                     # read-only checklist of what is wired and what is not
hug mem install all           # memory hooks for Claude Code, Codex and jcode; keeps backups
hug task init                 # task branch for this repository
brew services start hug       # optional: memory watcher + web UI on http://127.0.0.1:7431
```

Tab completion for hug, hug task (commands, flags, open task ids) and hug mem:

```sh
source <(hug completion zsh)  # in ~/.zshrc; bash 4.4+: source <(hug completion bash) in ~/.bashrc
hug completion fish > ~/.config/fish/completions/hug.fish
```

## Commands

| Command | What it does |
|---|---|
| `hug mem ...` | memory: recall hooks, capture, search, remember, tidy, dream |
| `hug task ...` | tasks: Markdown files on a git branch, claims, goals, checks, questions to people |
| `hug task search "words"` | tasks that best match, ranked (BM25 over id, aliases, title, goal, notes, ...), with the field and a snippet; MCP `task_search` |
| `hug tui` | the tasks TUI (`hug mem tui` is the memory TUI) |
| `hug mcp` | one MCP server: `memory_*` and `task_*` tools |
| `hug apps [install]` | Claude Desktop, Claude Code, ChatGPT/Codex: start `hug mcp` |
| `hug setup` | read-only setup checklist |
| `hug upgrade [--apply]` | preview, then apply, every migration to this hug |
| `hug task install commit-msg` | refuse commits without one `Model:` trailer |
| `hug version` | the version of hug |

## Sharing between clones

Tasks (`hug/tasks`) and shared project memories (`hug/memory`, see below) reach
the other clones of a repository by themselves; nobody runs sync. A write commits
locally and starts one detached syncer, which waits until the burst is quiet
for 1.5 s and then fetches, rebases and pushes. Another clone fetches when a
hug command finds its last fetch older than 5 s: another clone sees a change
at its next command, at most about 7 s later. `hug task sync` and
`hug mem sync` do it at once. Logs are under `.git/hug/sync/`.

Project memories stay on this machine unless the repository opts in:
`hug task memory share` commits `[memory] share = true` to `hug.toml` on
`hug/tasks`, and every clone follows it. Without it, `hug mem sync` says
`project memories stay on this machine` and pushes nothing, so a public
repository never publishes memories. `hug task memory local` goes back.

## Who made a commit

Every commit names what made it in one trailer line: `Model: claude-opus-5-5`
(the model id), or `Model: human` when the git author wrote it themselves.
`hug task install commit-msg` refuses a commit without exactly one, with a
`fix:` line to finish it; `hug setup` lists it until it is in place. With
`core.hooksPath` (a project's own `.githooks`) hug never edits the tracked hook
and prints the two lines to add instead. hug's own commits on `hug/tasks` and
`hug/memory` add `Model:` from `--model` or `HUG_MODEL` (the TUI: `human`).
Emergency exit: `git commit --no-verify`.

```sh
git log --format='%h %(trailers:key=Model,valueonly,separator=%x2C) %s'   # every commit and its model
git log --grep='^Model: gpt-6.1-sol$' --oneline            # one model's commits
git -C .hug/tasks log --format='%(trailers:key=Model,valueonly,separator=%x2C) %s'   # task changes
```

## Layout

The source keeps the ravens' names for its two parts.

| Folder | Part | Details |
|---|---|---|
| [`muninn/`](muninn/) | memory (`hug mem`) | [muninn/README.md](muninn/README.md) |
| [`huginn/`](huginn/) | tasks (`hug task`) | [huginn/README.md](huginn/README.md) |
| [`cmd/hug/`](cmd/hug/) | the `hug` command | dispatches to the parts |

Tasks and memories are files on git branches (`hug/tasks`, `hug/memory`), with a
local SQLite cache for speed. Coming next: messages, on `hug/messages`.

## Releases

One tag, `vX.Y.Z`, releases everything: archives `hug_<version>_<os>_<arch>`,
the Homebrew formula and the Scoop manifest. Before v0.7.0 the parts were released
separately as `muninn-v*` and `huginn-v*`.

## License

[MPL-2.0](LICENSE). Contributions need the contributor agreement in
[CONTRIBUTING.md](CONTRIBUTING.md).
