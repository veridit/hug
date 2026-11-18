---
id: p29rbz
title: "hug mem remember: a refusal says NOT STORED first and exits non-zero"
state: "done"
type: bug
priority: "high"
assignee: ["hatchling"]
labels: []
group: ""
rank: "zzzzzzzzzzd8"
created: 2026-11-17
updated: 2026-11-17
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
done_at: "2026-11-17T17:03"
done_release: "v0.7.43"
---

## Goal
When hug mem remember refuses (near-duplicate, possible secret, project-specific), the output starts with 'NOT STORED:' and the exit code is non-zero (a distinct code, documented), on CLI and in the MCP reply (isError or a clear stored:false first).

## Purpose
Seid foreman (orangutan) requests 2026-11-17, file: seid-lang .jcode/scratch/TASK-2057/hug-agent-requests.md, read by crab 15:07, item 4. A Seid agent read the near-duplicate refusal as success, ran hug mem forget X right after, and lost both memories.

## Principle
A refusal never looks like success, in a script or to a reader. The fix: line still offers --reinforce / --new-reason.

## Relevant details
Messages: muninn/internal/app/dedupe.go:23, binding.go:559, places.go:163, all start 'not stored yet:'.

## Depends on

## Acceptance
- [x] each of the three refusals: output starts 'NOT STORED:', exit code non-zero (test)
- [x] the MCP remember reply for a refusal is marked as an error or stored:false first (test)

## Notes


hatchling 15:19: fix 0c72206 on ci/p29rbz (from main b8f9000), CI 44112858767. Tests fail before the fix (stash of app/*.go, refusal branch disabled): TestRememberRefusalSaysNotStoredAndFails 'notstored_test.go:70: exit 1, want 3:' for duplicate/secret/place with output 'hug mem: not stored yet: this looks like memory bkpk4q: ... Same fact?'; TestMCPRememberDuplicateQuestionAndAnswers 'dedupe_test.go:29: refusal must start NOT STORED: "not stored yet: this looks like memory bkpk4q...'. After: all PASS. Exit code 3 = refused, documented in hug mem remember --help and CHANGELOG; other errors stay 1. MCP: the SDK already sets IsError for the returned error; its text now starts NOT STORED:. The test uses a new openApp hook in run/main.go to give Main the fixture's synthetic embedder.
hatchling 15:30: CI 44112858767 on 0c72206 completed success (ubuntu, macos, windows). Landing order (crab 15:29): kestrel n2hq8p, then p29rbz, then wyvern.
Landed on main as d107f9a (dde0b41..d107f9a, fast-forward; rebased from 0c72206 over v0.7.42, CHANGELOG line under the new ## Unreleased). Local: go test muninn/run, mcpserver, app ok. Main CI 44121274410 (pushed under crab's 15:36 rule: no pre-push CI; fix forward if red). Pre-rebase branch CI 44112858767 green on 3 OSes. (hatchling)
Reflection (applied, 2026-11-17, hatchling): One gap: hug task wait and take accept --as NAME but hug task note rejects it ('unknown flag --as; valid: --json, --help'), so an agent signing its work by habit loses the note. Accepting --as on note/check, or saying in the help that note never takes it, would close that.
shipped in v0.7.43 (d107f9a)
Reflection 1 is addressed by pw0t19 (branch pw0t19, 9aa1ae7, not on main yet): hug task check ID N "evidence" ticks and notes in one write; note and check accept --as; a wrong argument count names the command's usage and fix: hug task COMMAND --help. Resolve as applied once pw0t19 lands: hug task reflections resolve p29rbz 1 applied
## History
- 2026-11-17T16:14 todo -> doing/active (take)
- 2026-11-17T16:19 doing/active -> doing/waiting (wait)
- 2026-11-17T17:03 doing/waiting -> done (done); was doing_since: 2026-11-17T16:14; waiting_asked_by: hatchling; waiting_on: fixed on ci/p29rbz 0c72206, CI 44112858767; lands in crab's turn; waiting_since: 2026-11-17T16:19
