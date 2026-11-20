---
id: a0p5kt
title: "hug refuses an unknown HUG_* variable instead of ignoring it"
state: "done"
type: bug
priority: "high"
assignee: ["wyvern"]
labels: []
group: ""
rank: "zzzzzzzzzzlv"
created: 2026-11-18
updated: 2026-11-18
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
done_at: "2026-11-18T04:12"
done_release: "v0.7.55"
---

## Goal
Any hug command run with an environment variable named HUG_* that hug does not know stops before doing anything, names the variable, and gives the closest real one as a fix line.

## Purpose
wyvern 2026-11-18 03:40 local: a forget benchmark set HUG_NO_SYNC=1 (not a real variable; the real one is HUG_SYNC=off), so hug ignored it and synced 11 forgets from a store copy into origin/muninn-notes between 03:19 and 03:37 local. Repaired by hand (muninn-notes 76b153c6). A misspelt safety switch must never fail open; Jørgen's rule: nothing writes silently.

## Principle
Fail closed on a misspelt switch: unknown HUG_* is a usage error (exit 2) with a fix line. The list of known variables lives in one place in the code and is what hug help prints; tests and the release check that every HUG_ name in the code is on it. No allowlist of extras.

## Relevant details
Known today (git grep on main): HUG_HOME, HUG_SYNC, HUG_MODEL, HUG_USER, HUG_NO_EMBED, HUG_FULL_SPEED, HUG_BIN_DIR, HUG_SERVE, HUG_CPUPROFILE, HUG_BENCH, HUG_MCP_RESUME, HUG_TUI_STATE, HUG_TUI_ARTIFACTS, HUG_MIGRATION_COPY, HUG_REPAIR_COPY, HUG_UPDATE_GOLDEN, plus HUG_TEST_* and HUG_CAPTURE_KILL_* used by tests. Check at startup in cmd/hug before dispatch (also hug mcp and hooks). Example: 'hug mem forget: HUG_NO_SYNC is not a hug variable' then 'fix: HUG_SYNC=off hug mem forget ...'.

## Depends on

## Acceptance
- [x] HUG_NO_SYNC=1 hug mem forget X exits 2, writes nothing, and the fix line names HUG_SYNC=off (test, fails first on main)
- [x] A test greps the code for every HUG_ name and fails if one is missing from the known list, so a new variable cannot be forgotten
- [x] hug help lists every user-facing variable with one line each

## Notes


Landed on main 085a81ce (unchecked until CI is green). Check 1: TestUnknownHUGVariableRefused (cmd/hug/env_test.go) runs hug mem forget abcd with HUG_NO_SYNC=1 (spelt in parts) and gets exit 2, the line 'hug mem forget: not a hug variable: HUG_NO_SYNC (nearest: HUG_SYNC)', 'fix: unset HUG_NO_SYNC; export HUG_SYNC=off', and HUG_HOME not created. On main the same test got: 'exit 1, want 2', stderr missing both lines, 'the refused command touched the store', 'the refused hook touched the store', 'hug help does not list HUG_HOME'. Check 2: TestEveryHUGNameIsKnown (internal/env/known_test.go) walks the repo (code, scripts, .github, docs; not CHANGELOG.md or testdata). Before the fix it failed on 5 lines: 4 HUG_NO_SYNC comments and capture-e2e.py's HUG_SYNTHETIC_MISSING_CREDENTIAL, which is renamed SYNTHETIC_MISSING_CREDENTIAL since it isn't a hug variable. Check 3: hug help ends with 'Environment (any other HUG_* variable is refused):' and 9 lines (HUG_HOME USER MODEL SYNC NO_EMBED FULL_SPEED CPUPROFILE BIN_DIR VERSION); the test asserts each user-set Known entry is listed and HUG_TEST_SETTLE is not. Hooks (mem hook, mem capture --hook) refuse silently with exit 0. -race: cmd/hug 21.3s, internal/env 4.8s, muninn/run 34.1s, all ok. Running processes carried one more unknown name, my own w40cmk bench's HUG_BENCH_GITLOG, now renamed BENCH_GITLOG.
landed 085a81c on main 2026-11-18: hug refuses an unknown HUG_* variable instead of ignoring it (a0p5kt)
landed 0a2f06a on main 2026-11-18: hug: a hook refused for an unknown HUG_ variable says so in the hook log…
Checks 1-3 green on main CI 44226874819 (085a81ce): ubuntu, macos, windows, test-windows-cli, bench, secrets all success. Hook-log addition (crab 03:58: a refused hook must leave a hook.log line) is 0a2f06ae, CI 44227585322.
Hook-log line green on main CI 44227585322 (0a2f06ae): ubuntu, macos, windows, test-windows-cli, bench, secrets success.
Reflection (dismissed, 2026-11-18, wyvern): A safety switch that is only a convention (a variable hug happens to read) fails open on a typo, so the one list of known names plus a repo scan now makes a typo loud. The incident also exposed two harness gaps of mine: a benchmark cwd with a live remote, and a copied repo whose worktree pointed at the original (ygay9c). Next time a bench runs from a remote-less clone with HUG_SYNC=off from the start.
shipped in v0.7.55 (085a81c)
Reflection 1 dismissed (tb0dra check 5 triage, crab 2026-11-19): G10, a lesson about one's own method, fixed in its own ticket; no instruction misled.
## History
- 2026-11-18T03:41 todo -> doing/active (take)
- 2026-11-18T03:57 doing/active -> doing/waiting (wait)
- 2026-11-18T04:12 doing/waiting -> done (done); was doing_since: 2026-11-18T03:41; waiting_asked_by: wyvern; waiting_on: main CI 44227585322 on 0a2f06ae (hook log line); waiting_since: 2026-11-18T04:04
