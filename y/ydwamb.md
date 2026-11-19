---
id: ydwamb
title: "Lint reads a glob citation (compare-*.png) as a missing file"
state: "done"
type: bug
priority: "medium"
assignee: ["duckling"]
labels: []
group: ""
rank: "zzzzzzzzzzzk"
created: 2026-11-18
updated: 2026-11-18
model: "claude-opus-5-5"
branch: "ydwamb"
worktree: "/Users/jhf/.jcode/scratch/ydwamb"
done_at: "2026-11-18T23:51"
done_release: "v0.7.72"
---

## Goal
hug task lint treats a citation with a shell glob, like .hug/tasks/9/911j70/compare-*.png, as the pattern it is: fine when it matches at least one attached file, a finding naming the pattern when it matches none.

## Purpose
911j70 note (crab, 2026-11-18) tells Jørgen to run: open -a Preview .hug/tasks/9/911j70/compare-*.png. Lint cut the citation at * and reports '.hug/tasks/9/911j70/compare-: file not found', fix: hug task attach 911j70 FILE --name compare-. That is a false broken finding with a fix that would attach a junk file. Ten compare-*.png files are attached.

## Principle
Lint never writes. A real missing file is still found. No special case for 911j70: any citation with * ? or [..] in the last path part is a glob, matched against the task's attached files.

## Relevant details
Finding: hug task lint 911j70 -> 9/911j70.md:0: broken [missing-file] .hug/tasks/9/911j70/compare-: file not found. The citation parser is in huginn/internal/task lint (missing-file). Run tests niced: GOMAXPROCS=2 GOFLAGS=-p=2 nice -n 19 go test ./huginn/...

## Depends on

## Acceptance
- [x] a test: a note citing .hug/tasks/X/ID/compare-*.png with two matching files gives no finding (fails before)
- [x] a test: a glob citation matching no attached file gives one missing-file finding naming the whole pattern, and its fix: line does not suggest --name with a cut-off name
- [x] hug task lint on this board shows no missing-file finding for 911j70

## Notes


49249f5f on main 8a45b0eb (ydwamb, ci/ydwamb). Fail-first: TestGlobCitationMatchingFilesIsFine got 'task:kdap66/compare-: file not found' and '.hug/tasks/k/kdap66/compare-1: file not found'; TestGlobCitationMatchingNothingNamesThePattern got 'shot-: file not found ... | fix: hug task attach hc607d FILE --name shot-'. After: both pass, plus TestCitedNameKeepsProseOut; task ok 25.624s, mcp ok 267.517s, cache ok 2.709s locally. Check 3, read-only with the built binary on this board: old 'hug task lint 911j70' -> '9/911j70.md:0: broken [missing-file] .hug/tasks/9/911j70/compare-: file not found'; new -> 'no findings'; board missing-file count 2 -> 1 (the one left is ydwamb's own goal text, which cites the cut form 'compare-' literally... it reads '.hug/tasks/9/911j70/compare-: file not found' in the task's relevant details). .hug/tasks git status: 0 changed before and after (lint never writes).
check 1: 49249f5f TestGlobCitationMatchingFilesIsFine: compare-80.png and compare-140.png attached, notes cite .hug/tasks/G/ID/compare-*.png, compare-1?0.png, compare-[18]*.png, file:compare-*.png, task:ID/compare-*.png -> no finding. Failed first: '.hug/tasks/k/kdap66/compare-1: file not found' and 'task:kdap66/compare-: file not found'
check 2: TestGlobCitationMatchingNothingNamesThePattern: shot-*.png matching nothing -> one 'missing-file: .hug/tasks/G/ID/shot-*.png: no attached file matches (attach one, or mark the citation (lost))', fix 'hug task attach ID FILE' (no --name); plain gone.tsv still 'file not found'. Failed first: 'shot-: file not found ... --name shot-'
check 3: Built 49249f5f, read-only on this board: 'hug task lint 911j70' old -> '9/911j70.md:0: broken [missing-file] .hug/tasks/9/911j70/compare-: file not found'; new -> 'no findings'. .hug/tasks git status 0 changes before and after. The one missing-file left on the board is ydwamb's own text quoting the cut 'compare-' form (line 30), a real non-glob citation.
CI 44406523669 on 49249f5f: completed/success, 8/8 jobs success (secrets, bench, test macos/windows/ubuntu, test-windows-cli sync+rest, test-windows-cli-shards).
landed 3c1b937 on main 2026-11-18: lint: a glob citation (compare-*.png) is a pattern matched against the task's…
crab 22:51: landed on main as 3c1b9375 (rebased stack f5a2493a, CI 44411349216 8/8); ships in v0.7.72
shipped in v0.7.72 (3c1b937)
## History
- 2026-11-18T22:34 todo -> doing/active (take)
- 2026-11-18T23:24 doing/active -> doing/waiting (wait)
- 2026-11-18T23:51 doing/waiting -> done (done); was doing_since: 2026-11-18T22:34; waiting_asked_by: duckling; waiting_on: reviewed; crab lands it as ci/ydwamb-rebased 878e4464 after v0.7.71; waiting_since: 2026-11-18T23:24
