# hug

One tool for an AI-assisted project's **memory**, **tasks** and (soon) **messages**,
shared by every agent and harness you use: jcode, Claude Code, Codex, the Claude
app, and plain shells.

The name: Old Norse *hugr* is the mind, thought and intent, the root of Huginn,
Odin's raven of thought. Odin's other raven, Muninn, is memory. And in English, a hug
holds things together.

## Status

This repository joins two tools that were developed separately, with their full
histories:

| Folder | Part | What it does | Install today |
|---|---|---|---|
| [`muninn/`](muninn/) | memory | Local-first memory recall for AI coding agents: automatic recall hooks, MCP server, capture, tidy, TUI | `brew install veridit/tap/muninn` |
| [`huginn/`](huginn/) | tasks | Tasks as Markdown files on a git branch, with goals, acceptance checks, ranks and a push-based claim lock; CLI, MCP and TUI | `brew install veridit/tap/huginn` |

They are being merged into a single `hug` binary. Design: files on namespaced
git branches (`hug/tasks`, `hug/memories`, `hug/messages`) are the truth, a local
SQLite cache makes them fast, and every harness talks to the same MCP server.

## Releases

Until the merge is done, each part is released on its own with a prefixed tag:
`muninn-v0.5.3`, `huginn-v0.1.5`.

## License

[MPL-2.0](LICENSE). Contributions need the contributor agreement in
[CONTRIBUTING.md](CONTRIBUTING.md).
