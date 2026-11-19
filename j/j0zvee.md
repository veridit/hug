---
id: j0zvee
title: "Two clones closing one release write the release line twice"
state: "doing"
type: feature
priority: ""
assignee: ["iwazaru"]
labels: []
group: ""
rank: "zzzzzzzzzzzw"
created: 2026-11-19
updated: 2026-11-19
doing_since: "2026-11-19T02:06"
stage: "active"
model: "claude-opus-5-5"
branch: "awht8w"
worktree: "/Users/jhf/.jcode/scratch/awht8w"
---

## Goal
TestReleaseCloseTwoClones holds: when two clones deduce the same release tag offline and then sync, the task's History carries one 'done, by release v0.1.1' line, as yf39ya check 2 promises; find why the merge keeps both clones' identical transition lines a minute apart

## Purpose
Two machines closing the same task by the same release is the normal case on a shared board; a doubled History line is a real merge defect (not only a test flake) and fails CI and local suites

## Principle
Cause first, with a fail-first test; no timeout raised, no retry, no skip; the merge decides, not the clock

## Relevant details
CI census 2026-11-19 (iwazaru, scratch/ci-census.md). 2 hits on 2 different bases: CI 44412479085 (ci/hpqt99 32ed02dd, ubuntu, 2026-11-18T22:32) 'release_test.go:148: /tmp/TestReleaseCloseTwoClones69224392/001/one: 2 release lines' with History '- 2026-11-18T22:34 doing/waiting -> done (done, by release v0.1.1: every check holds)' and the same line at 22:35; and iwazaru's niced local cli run on c186f78f+awht8w at load 137-266 (scratch/awht8w-ev/pkg-full1.txt:3023) '--- FAIL: TestReleaseCloseTwoClones (296.05s)' 'release_test.go:148: .../001/one: 2 release lines', History lines at 23:43 and 23:44. The two lines differ only in their timestamp minute. shipped/landed lines were single both times (line 145 passed).

## Depends on

## Acceptance
- [x] Cause named in a note with the History of a failing run or a reproduction
- [ ] A test fails first, then TestReleaseCloseTwoClones passes -count=20 niced under load
- [ ] No timeout raised and no retry added

## Notes


check 1: Cause: History merges entries by exact text (merge.go mergeAppend 'seen[key]'). Two clones deducing the same tag offline each run closeShipped at their own minute, so the two 'doing/waiting -> done (done, by release v0.1.1: every check holds)' lines differ in the minute only and both stay; done_at differs by that minute too and was noted as a conflict ('kestrel set done_at 2026-11-18T23:44 ... kept 2026-11-18T23:43'). Deterministic reproduction, no load: TestReleaseCloseTwoClonesInTwoMinutes (board clock pinned, A deduces at 12:01, B at 12:02, background off) on main 50fa8681: 'release_minutes_test.go:54: .../one: 2 release lines' with History '- 2026-01-31T12:01 doing/waiting -> done (done, by release v0.1.1 ...' and '- 2026-01-31T12:02 ...'; unit TestMergeSameReleaseCloseInAnotherMinuteCountsOnce: 'merge_release_test.go:52: 2 release lines, want 1'. Failing commit 763979f2.
## History
- 2026-11-19T02:06 todo -> doing/active (take)
