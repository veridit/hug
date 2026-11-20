---
id: 9vp9by
title: "Memory layout migration commit runs no repo hooks"
state: "done"
type: feature
priority: ""
assignee: ["tigress"]
labels: []
group: ""
rank: "zzzzzzzzzzs1"
created: 2026-11-18
updated: 2026-11-18
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
done_at: "2026-11-18T23:24"
done_release: "v0.7.71"
---

## Goal
muninn's layout migration (project/branch.go migrationGit rm/commit/restore of .muninn/memories.jsonl) commits without running the code repository's hooks, so a commit-msg gate (Seid's Model: gate, jr55s8) or a hook that leaves a background process holding stdout can neither refuse nor hang it

## Purpose
In Seid the commit-msg hook refuses a commit without a Model: trailer, so the migration would fail; a hook's background job holding git's stdout would block CombinedOutput forever (WaitDelay 0). Same lesson as the 2026-11-09 seid-lang incident that made runAt pass core.hooksPath=/dev/null

## Principle
hug's own commits never run the host project's hooks; reuse gitbranch's runner semantics, no timeout raised, no retry

## Relevant details
Audit 2026-11-18 (tigress, origin/main 8a45b0eb): migrationGit at muninn/internal/project/branch.go:159-166 runs exec.Command + CombinedOutput with no core.hooksPath override; callers :135-139. runAt does it at huginn/gitbranch/gitbranch.go:170. Tests: a repo whose commit-msg hook exits 1, and one whose hook leaves 'sleep' in the background holding stdout; fail first on main, niced.

## Depends on

## Acceptance
- [x] A test with a commit-msg hook that exits 1 fails on main and passes after the fix
- [x] A test with a hook leaving a background sleep holding stdout fails (hangs past a bound) on main and completes after the fix
- [x] No timeout raised, no retry; niced -count=50 of the new tests passes

## Notes


check 1: Fail first at 21b1ea39 on main 8a45b0eb: 'migration_hooks_test.go:64: migration with a commit-msg hook: migrated=true err=migration git commit: refused: no Model: trailer: exit status 1'. After b6848d6e: --- PASS commit-msg_refuses (0.48s). Mutation (flag removed): fails the same way again.
check 2: Fail first: 'migration_hooks_test.go:67: the migration ran the code repository's post-commit hook (took 20.338774334s)' (the hook's background sleep 20 held CombinedOutput). After b6848d6e: --- PASS post-commit_leaves_a_process_holding_output (0.48s), and the hook never runs (marker file absent).
check 3: The fix is one flag, -c core.hooksPath=/dev/null, in migrationGit (branch.go:166). No timeout, no retry. TestMigrationCommitRunsNoRepoHooks -count=50 niced: 'ok github.com/veridit/hug/muninn/internal/project 61.782s'. Package: 'ok 11.210s'.
Branch 9vp9by / ci/9vp9by on main 8a45b0eb: 21b1ea39 (failing test), b6848d6e (fix + CHANGELOG). CI 44399475509 queued.
CI 44399475509 on b6848d6e: completed success, 9/9 jobs (ubuntu, windows, macos, both windows-cli shards, bench, secrets). All 3 checks ticked; not closing it myself (crab lands it).
landed 5281611 on main 2026-11-18: project: failing test, the memory migration commit runs no repo hooks (9vp9by)
landed 0b724ec on main 2026-11-18: project: the memory migration commit runs no repo hooks (9vp9by)
shipped in v0.7.71 (5281611)
crab: on main (ancestor of fac02994 verified): fix 0b724ec3. Closed by the coordinator.
Reflection (open, 2026-11-19, crab): nothing (coordinator closing on verified landing)
## History
- 2026-11-18T22:11 todo -> doing/active (take)
- 2026-11-18T22:38 doing/active -> doing/waiting (wait)
- 2026-11-18T23:24 doing/waiting -> done (done, by release v0.7.71: every check holds); was doing_since: 2026-11-18T22:11; waiting_asked_by: tigress; waiting_on: 0b724ec3 (ci/9vp9by-rebased) to land on main; all 3 checks ticked; close when landed; waiting_since: 2026-11-18T22:38
