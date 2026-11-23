---
id: tfe9nr
title: "TestTmuxAutoRefresh parses stderr as JSON from hug task new"
state: "done"
type: feature
priority: ""
assignee: ["camel"]
labels: []
group: ""
rank: "zzzzzzzzzzz9"
created: 2026-11-21
updated: 2026-11-21
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
done_at: "2026-11-23T15:02"
---

## Goal
TestTmuxAutoRefresh reads only stdout from 'hug task new --json', so a stderr line can never break its parse.

## Purpose
ladybug 2026-11-21 (4ddcwa): tmux_refresh_test.go:65: invalid character 'h' looking for beginning of value, once, passed on rerun. The run helper uses CombinedOutput (tmux_refresh_test.go ~49), so any stderr line (a git hint:, a hug notice) lands before the JSON.

## Principle
Fix the cause, not the retry: take stdout for parsing, keep stderr for the failure message. Find which line starting with 'h' was printed and say whether it should print at all.

## Relevant details
Same pattern may exist in other tests that json.Unmarshal CombinedOutput: grep for it.

## Depends on

## Acceptance
- [x] A test that makes hug task new print a stderr line still parses (fails first with CombinedOutput)
- [x] The stderr line starting with 'h' is named, and either fixed or shown to be correct
- [x] Other tests that unmarshal CombinedOutput are listed and fixed

## Notes


Inventory (check 3), branch tfe9nr 940afa84. Tests that parsed JSON out of a stdout+stderr mix, now stdout only, stderr shown on failure: (1) huginn/internal/tui/tmux_refresh_test.go:57 run used CombinedOutput, parsed at :65 (now :60 cmd.Output, parse :78). (2) cmd/hug/memory_layout_test.go:38 try/run CombinedOutput -> json.Unmarshal of mem search --json at :76 (now stdout helper :53, parse :95). (3) cmd/hug/sync_test.go:112 task new --json via CombinedOutput, id by substring between() (now cmd.Output + json.Unmarshal :113-125; between() removed, no other caller). (4) huginn/internal/cli runText (take_test.go:35) sets a.Out=a.Err=one buffer; 10 --json/--jsonl call sites parsed it, now runJSON (take_test.go:44): busy_test.go:31, catchup_preservation_test.go:235, detour_test.go:31 and :54, lsasked_test.go:91 and :104, states_test.go:29, strike_test.go:26, takewith_test.go:112, upgrade_layout3_test.go:80. Checked and left alone: mcp/*_test.go use CombinedOutput only for git fixtures and parse the MCP reply text; muninn/run runCLI (concurrent_writes_test.go:19) already splits stdout/stderr; notstored_test.go:24 merges but parses no JSON; install_test.go:30 merges but parses no JSON; memory_local_test.go:87 task memory --json only scanned for fix:/next: lines; deliverhook_test.go:105 captureSurface is stdout-only.
check 1: TestTmuxAutoRefresh now sets an old variable name so hug task new --json prints 'hug: hug reads only HUG_* variables' on stderr. Failing first on 9d215df7 with CombinedOutput: tmux_refresh_test.go:72: invalid character 'h' looking for beginning of value. Fixed in 940afa84: run uses cmd.Output, stderr kept for the failure message and asserted to carry the warning. Mutant CombinedOutput back -> tmux_refresh_test.go:78 same error.
check 3: Listed in the note on tfe9nr: 13 parse sites in 4 places fixed in 940afa84 (tmux_refresh_test.go:78, memory_layout_test.go:95, sync_test.go:113, and 10 cli call sites moved from runText to runJSON take_test.go:44). runjson_test.go pins runJSON with a real stderr line (hug setup: comparing .backlog/). Mutants: memory_layout CombinedOutput -> :102 invalid character 'h'; sync_test CombinedOutput -> :124 fails; runJSON merging -> runjson_test.go:34 fails.
check 2: Named: hug's stale-name warning, cmd/hug/main.go:86 warnStale(os.Stderr) -> cmd/hug/env.go:19 'hug: hug reads only HUG_* variables: HUGINN_UPDATE_GOLDEN is now HUG_UPDATE_GOLDEN (the old name is ignored)' plus a fix: line, exit 0. Evidence: ladybug's failing command (session ladybug_1795278907122, task 447038fcqb 'Update goldens across tui') was 'HUG_UPDATE_GOLDEN=1 HUGINN_UPDATE_GOLDEN=1 ... go test ./huginn/internal/tui/'; the test's children inherit os.Environ(); reproduced byte-exact: same error at tmux_refresh_test.go:72 on 9d215df7. It is correct on stderr: m1c28r put it there on purpose (main.go:83-85 'on stderr (stdout may be JSON or a hook's reply)'), the --json stdout stays clean; no product change. Likely source of the old name: ~/Projects/hug/AGENTS.md:25 still says HUGINN_UPDATE_GOLDEN (origin/main AGENTS.md says HUG_UPDATE_GOLDEN).
CI 8/8 green on 940afa84: https://github.com/veridit/hug/actions/runs/45188277140 (secrets, bench, test ubuntu/macos/windows, test-windows-cli rest/sync/shards). Failing-first 9d215df7 failed only ubuntu: tmux_refresh_test.go:72 invalid character 'h' (run 45185260103). Tests-only, no CHANGELOG entry.
Reflection (open, 2026-11-23, camel): One gap: /Users/jhf/Projects/hug/AGENTS.md:25 (the checkout agents read) still says 'HUGINN_UPDATE_GOLDEN=1 go test -run TestTmuxGolden'; main says HUG_UPDATE_GOLDEN. Following it caused this flake: every hug child of the test printed the stale-name warning on stderr. The 'never run git in ~/Projects/hug' rule kept me from reading main's AGENTS.md there; the worktree copy was the right one.
## History
- 2026-11-23T14:24 todo -> doing/active (take)
- 2026-11-23T15:02 doing/active -> done (done); was doing_since: 2026-11-23T14:24
