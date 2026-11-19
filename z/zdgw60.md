---
id: zdgw60
title: "A set that wrote says what it wrote, before lint advice"
state: "done"
type: feature
priority: ""
assignee: ["piglet"]
labels: []
group: ""
rank: "zzzzzzzzzzza"
created: 2026-11-19
updated: 2026-11-19
model: "claude-opus-5-5"
branch: "zdgw60"
worktree: "/Users/jhf/.jcode/scratch/zdgw60"
done_at: "2026-11-19T01:59"
done_release: "v0.7.74"
---

## Goal
hug task set (and MCP task_update set) prints one line per field it changed, e.g. 'added check 2: VALUE', before any lint: block, so a reply that ends in a lint fix never reads as a failed write.

## Purpose
Reflection asr4t2 (nautilus, 2026-11-17): set --accept 'VALUE' added the check but the reply ended with a lint fix about a long title, which read as if the set had failed, so the check was added twice. Reproduced 2026-11-18 by piglet in a scratch repo: the reply lists the task, '2. [ ] VALUE', then 'lint: 1 finding ... fix: hug task set ID --title "..."', with no line saying what was written. Triage tb0dra check 5, group G1 (crab approved 2026-11-19).

## Principle
Say what changed first, then advice. No change to what lint finds or how set writes. One line per changed field, short.

## Relevant details
Triage: /Users/jhf/.jcode/scratch/reflections-triage.md G1. Related reflections 05r3pg (done on todo; now prints the take fix) and 3nhp67 (fix line a no-op; fixed in its ticket).

## Depends on

## Acceptance
- [x] set --accept X on a task with a lint finding prints 'added check N: X' before the lint: block (test, failing first, from the reproduction)
- [x] each other field set changes gets its own line (title, goal, purpose, principle, details, tag, parent, state) (test)
- [x] MCP task_update set's reply says the same (test)

## Notes


Branch zdgw60 / ci/zdgw60 at 1ca07644 on main 6bf6a928; tests-first c777466f failed: TestSetSaysWhatItWroteBeforeLint 'want added check 2: VALUE before the lint block' (reply was the task, '2. [ ] VALUE', lint, fix:, no wrote line), TestSetSaysEachFieldItWrote 'wrote lacks title: Old title -> New title', MCP TestNewAndUpdateReturnFindingsAndOneFix 'update set wrote: <nil>'. Fix: cli/setwrote.go (snapshot before/after set), reply['wrote'] printed before lint:, help and MCP description updated, CHANGELOG line. 6 mutants all fail (no added check; no title line; no unchanged line; wrote printed after lint; no tags line; no wrote in reply -> MCP). Local targeted: cli ok 170 s, mcp ok 154 s. CI 44429491626 running.
CI 44429491626 (1ca07644) success on all 8 jobs: secrets, bench, test ubuntu/macos/windows, test-windows-cli sync/rest/shards. Checks stay open until it lands (ticks with the main SHA).
check 1: On main as e75e1593 (tests 4edade75), code-identical to 1ca07644 (diff of huginn/ between them empty). TestSetSaysWhatItWroteBeforeLint failed first ('want added check 2: VALUE before the lint block'), passes: 'added check 2: VALUE' prints before 'lint: 1 finding'. CI 44429491626 success, all 8 jobs.
check 2: On main as e75e1593. TestSetSaysEachFieldItWrote: 11 lines for 11 changed fields (title, state, type, priority, assignee, parent, tags, goal, purpose, principle, relevant details) and 'unchanged: ...' on a repeat; failed first ('wrote lacks title: Old title -> New title'). CI 44429491626 success.
check 3: On main as e75e1593. MCP TestNewAndUpdateReturnFindingsAndOneFix: task_update set returns wrote ['title: ...','purpose: Why now','added check 2: D']; failed first ('update set wrote: <nil>'). CI 44429491626 success.
Reflection (open, 2026-11-19, piglet): nothing misled: the reflection's own reproduction (set --accept then a lint fix) made the failing-first test directly; the only surprise was that --state todo also clears the assignee, which the reply now says.
landed 4edade7 on main 2026-11-19: test: a set says what it wrote before lint advice (zdgw60, failing)
landed e75e159 on main 2026-11-19: hug task set says what it wrote before lint advice (zdgw60)
shipped in v0.7.74 (4edade7)
## History
- 2026-11-19T01:16 todo -> doing/active (take)
- 2026-11-19T01:40 doing/active -> doing/waiting (wait)
- 2026-11-19T01:59 doing/waiting -> done (done); was doing_since: 2026-11-19T01:16; waiting_asked_by: piglet; waiting_on: crab lands ci/zdgw60-rebased e75e1593 (on main bf170d46, CI running); then tick checks 1-3 with the main SHA; waiting_since: 2026-11-19T01:40
