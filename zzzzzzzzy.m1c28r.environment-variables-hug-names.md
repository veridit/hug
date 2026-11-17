---
id: m1c28r
title: "Environment variables are HUG_*; hug upgrade migrates, no old names"
status: "doing"
type: chore
priority: "medium"
assignee: ["wyvern"]
labels: []
created: 2026-11-16
updated: 2026-11-16
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
---

## Goal
Every environment variable is HUG_* (HUG_USER, HUG_HOME, HUG_NO_EMBED, ...). The old HUGINN_*/MUNINN_* names are gone from the code. hug upgrade rewrites any place hug itself wrote them (installed hooks, harness configs, its own config).

## Purpose
Users type hug; help that says HUGINN_USER or MUNINN_HOME teaches the old names (Jørgen 2026-11-16: 'HUGINN -> HUG?')

## Principle
No backwards compatibility (Jørgen 2026-11-16: 'You upgrade, you run the latest and greatest'). The code reads only HUG_*; hug upgrade is the one migration step and moves everything hug owns. Test-only variables are renamed too. One shared lookup helper, not scattered os.Getenv calls. If an old name is set in the environment, hug says once which new name to use, with a fix: line.

## Relevant details
Counted 2026-11-16 in the repo: MUNINN_HOME 40 uses, HUGINN_USER 18, MUNINN_NO_EMBED 9, HUGINN_UPDATE_GOLDEN 7, MUNINN_FULL_SPEED 5, plus 11 rarer ones. AGENTS.md and the golden instructions name HUGINN_UPDATE_GOLDEN. et85h1 (identity) should read HUG_USER first.

## Depends on

## Acceptance
- [x] hug task take with HUG_USER=x assigns x (test); grep finds no HUGINN_/MUNINN_ in Go code
- [x] hug upgrade rewrites old names in hooks and configs hug installed (test on a fixture home)
- [x] with MUNINN_HOME or HUGINN_USER set, hug prints the new name to use and a fix: line (test)
- [x] help, README, AGENTS.md and errors name only HUG_* (grep shows no HUGINN_/MUNINN_ in user-facing text)

## Notes

dragon 3caceb2 on dragon/m1c28r (stacked on t709he 3e8d6d3, local only). internal/env (HUG_* constants, Get/Lookup, Stale/Warning); every read goes through env.Get; cmd/hug warnStale names an old name still set with a fix: line. hug upgrade (upgrade_env.go) renames old names only in hug's own hook commands and MCP server entries in ~/.jcode/{config.toml,mcp.json}, ~/.claude/settings.json, ~/.claude.json, ~/.codex/{hooks.json,config.toml} and the Claude Desktop config, with a .bak-hug-upgrade backup. TestUpgradeRenamesEnvInHugConfigs PASS (renamed: the hug hook command's HOME variable and the hug MCP env USER variable; kept: other-tool hook, 'theirs' server; a second upgrade renames nothing). Grep of tracked files for old names: only CHANGELOG.md (1 historical line). Memory TUI goldens regenerated (muninn/scripts/tui-pty.py --update): 2 files, 1 line each (Source: native path). Niced tests: internal/env, cmd/hug, huginn cli/tui, muninn app/throttle/project/harness/mcpserver/store/tui/web/run all ok. CI after 0hkaa5, x1t1t1 and t709he land.

2026-11-17 09:00 crab: NOW waiting on 44w1ev's SQLITE_BUSY fix; stacked on it as ece85d9 (ci/m1c28r); wyvern lands it after 44w1ev.

Check 2: TestUpgradeRenamesEnvInHugConfigs (huginn/internal/cli/upgrade_env_test.go:14) on a fixture home renames the hug hook command and hug MCP env entries, keeps other tools' entries, a second upgrade renames nothing. Check 3: TestStaleNamesWarning (internal/env/env_test.go:10) and TestWarnStale (cmd/hug/env_test.go:11): an old name set prints 'X is now HUG_Y ... the old name is ignored' and a fix: export/unset line. Both in a08bb34 (v0.7.40), CI 44065886978 green on ubuntu, macOS, Windows. Checks 1 and 4 wait: main has the old golden-update variable again in huginn/internal/tui/rows_test.go:234,244 (from e0af424, after the rename); fix plus a repository guard test on ci/m1c28r-golden.

Landed on main d74af5d (fast-forward f583cac..d74af5d, CI 44087396396 green: ubuntu, macos, windows, bench). Check 1: TestTakeAsAndModelRecordsBoth/'HUG_USER and HUG_MODEL' (take_test.go) assigns the HUG_USER name; TestTakeModelOnlyAssigneeIsModel sets the old HUGINN_USER and asserts it is not read. 'git grep -nE "HUGINN_|MUNINN_" -- "*.go"' on d74af5d finds only internal/env (the old-name table and its test). Check 4: the same grep over README.md, AGENTS.md, */README.md and huginn/internal/cli/help.go finds nothing; 'git grep -l' lists only CHANGELOG.md and internal/env/oldnames_test.go. TestNoOldNamesInRepository (2c27e42, internal/env/oldnames_test.go) walks the repository and fails on any old name outside the changelogs: ok on d74af5d, and it failed before the fix (tui/rows_test.go:234,244).
