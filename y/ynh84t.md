---
id: ynh84t
title: "hug mem install writes 'muninn hook'; task install names the MCP entry huginn"
state: "doing"
type: feature
priority: ""
assignee: ["crab"]
labels: []
group: ""
rank: "zzzzzzzzzzz1"
created: 2026-11-18
updated: 2026-11-18
parent: "gem59b"
doing_since: "2026-11-18T23:37"
stage: "active"
model: "claude-opus-5-5"
branch: "ynh84t"
worktree: "/Users/jhf/.jcode/scratch/ynh84t"
---

## Goal
Hooks and MCP entries that hug writes into users' harness configs run and name hug: hug mem install writes 'hug mem hook claude|codex|jcode' (not '<path>/muninn hook ...'), hug task install names its jcode MCP entry hug (not huginn), and hug upgrade --apply --machine moves existing configs over with a preview.

## Purpose
p277bp check 2 (crab 22:19): users type only hug, but a user who opens ~/.claude/settings.json, ~/.codex/hooks.json, a jcode config or mcp.json sees muninn and huginn, written by hug itself. Found in the p277bp check 2 sweep.

## Principle
Behaviour that edits users' configs needs an upgrade path: preview first, --apply --machine moves them, backups kept, old commands keep working as compatibility links. No alias added; nothing is changed without the upgrade preview.

## Relevant details
muninn/run/surfaces.go:105 selfPath: 'writing .../hug into a hook would run hug hook jcode, which hug does not know' (so hook commands are <dir>/muninn hook X). muninn/internal/harness/install.go:149-185 InstallClaude/Codex/Jcode use bin+' hook claude|codex|jcode'. huginn/internal/cli/install.go:67 'MCP config runs huginn mcp (task tools only), not hug mcp'; hug task help install: 'Adds huginn -> hug task mcp to Jcode mcp.json'. huginn/internal/cli/upgrade_env.go ours()/renameJSON/renameTOML already recognise hug|muninn|huginn commands, a place to hang the move. Evidence: /Users/jhf/.jcode/scratch/p277bp-check2.md.

## Depends on

## Acceptance
- [x] hug mem install claude|codex|jcode writes a hook command that runs hug (hug mem hook ...), test
- [x] hug task install writes a jcode MCP entry named hug running hug, test
- [x] hug upgrade (preview) lists each existing muninn/huginn hook and MCP entry it would move; --apply --machine moves them with backups, and the hooks still recall (test with a fake HOME)
- [ ] On Jørgen's machine after the release, no harness config names muninn or huginn (grep in a note)

## Notes


INCIDENT 2026-11-19 00:32 (duckling): a ynh84t upgrade test run on WIP a24cb26c reached the real home and rewrote 7 live configs to the hug form, each with a FILE.bak-hug-upgrade: ~/.jcode/config.toml, ~/.claude/settings.json, and .jcode/config.toml of ~/Projects/Brovs, ~/Projects/GenderedGames, ~/Projects/truth_tribe_and_treasure, ~/ssb, ~/Veridit/src/seid-lang. Cause: hooks.go took the machine home from os.UserHomeDir() (a24cb26c hooks.go:533) and envFiles read the process's JCODE_HOME, not the App's; a test that runs upgrade --apply --machine with only a.Env[HOME] set reached the real files (moveNames itself honours dry: --apply without --machine does not write, now tested). Undo: each restored from its .bak-hug-upgrade at once; crab verified 7/7 cmp-equal to their backups, 0 'hug mem' lines. Stray backups: 7 deleted 2026-11-19 00:37 after cmp equal to the live file, per crab ruling 1; find ~ -maxdepth 6 -name '*.bak-hug-upgrade' outside scratch: 0. Prevention: a.homeDir()/a.getenv(JCODE_HOME); envFiles(home, jcodeHome) (TestUpgradeReadsJcodeHomeFromTheApp, fails on main 356fb240); internal/homeguard TestMain in cmd/hug, huginn/internal/cli, huginn/internal/harness, muninn/internal/harness, muninn/run: HOME, USERPROFILE, JCODE_HOME, XDG_CONFIG_HOME, CODEX_HOME to a temp dir, sha256 of the real harness configs plus every JCODE_INSTANCES config before and after, run fails naming changed files. Live sha guard over 11 files: 0 changed after each later run.
check 1: c95fc32d, CI 44436073964 8/8: TestInstallWritesHugForms (muninn/internal/harness/hugforms_test.go) holds claude, codex and jcode writing '<hug> mem hook X', replacing the old muninn hook in place, uninstall removing either form
check 2: c95fc32d, CI 44436073964 8/8: TestInstallThrowawayHomes (huginn/internal/cli/install_test.go) installs jcode --all-instances, hug on PATH, no huginn entry left; servers_test.go holds one hug entry, a foreign hug entry refused with a fix
check 3: c95fc32d, CI 44436073964 8/8: TestUpgradeMovesOldNamesToHug (preview, --apply alone writes nothing exit 5, --apply --machine moves with one backup per file, second run nothing to do, foreign hug skipped). Real check on temp HOME with copies of ~/.jcode/config.toml, ~/.claude/settings.json, ~/.claude.json, Brovs .jcode/{config.toml,mcp.json}: preview exit 5 names 3 files, --apply exit 5 writes 0, --apply --machine exit 0 moves 3 with 3 backups, second run 'names: nothing to do'. Moved claude hook run as written returned the remembered marker (1 hit). Live sha guard 11/11 unchanged. Files: scratch/ynh84t-real2/{preview,apply,second}.txt, hook-out.txt
crab: stacked as ci/ynh84t-rebased cda4d231 at the top of the v0.7.75 stack (CODE-DIFF IDENTICAL). For Jørgen, check 4: after v0.7.75, run '/opt/homebrew/bin/hug upgrade --apply --machine' (the full path), so the hooks point at the stable brew link, not the Cellar path. ~/.local/bin/muninn has no hug beside it.
landed cda4d23 on main 2026-11-19: Name hook and MCP entries hug; guard tests from the real home (ynh84t)
shipped in v0.7.75 (cda4d23)
crab: landed on main (stack tip fbe3a412, CI 44438198799 success 8/8); released in v0.7.75. The open check waits on the live machine after brew upgrade.
## History
- 2026-11-18T23:37 todo -> doing/active (take)
- 2026-11-19T04:01 handed to crab: duckling's session ended (closed 2026-11-19 02:31)
