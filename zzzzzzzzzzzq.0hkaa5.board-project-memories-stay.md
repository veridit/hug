---
id: 0hkaa5
title: "Board and project memories stay in sync across clones, in the background"
status: todo
type: feature
priority: "high"
assignee: []
labels: []
created: 2026-11-16
updated: 2026-11-16
---

## Goal
Every clone of a project sees other clones' task changes and project memories within seconds, without anyone running sync: writes are pushed in batches in the background (fetch, rebase, push), reads trigger a background fetch when the last one is stale, and nothing an agent runs waits on the network.

## Purpose
Jørgen 2026-11-16, for Statbus, which has many clones with branches and worktrees: the task board and memory must stay up to date across all of them. Today only hug task take pulls and pushes (huginn/internal/cli/take.go); new, note, check, done only commit locally (cli.go Commit calls); project memories move only on a manual hug mem sync (muninn/run/project.go). Linked worktrees on one machine already share the main checkout (065wwn, nfv18p).

## Principle
Batched and in the background (Jørgen): a write commits locally, then starts or nudges one detached syncer per board; it waits a few seconds to gather a burst into one push, then fetch, rebase, push, and loops while new commits arrived. One syncer at a time per board (lock file). Reads never block on the network: a read older than ~30 s since the last fetch starts a background fetch, and the TUI refreshes by itself when it lands (nfv18p). take keeps its synchronous push (it is the race lock). Offline or a failed push never fails a command: hug task and hug mem report 'N unpushed' with a fix: line and catch up at the next write. A rebase conflict leaves local commits intact and surfaces in lint and the TUI with fix: hug task sync. Same mechanism for the muninn-notes branch (project memories after remember/forget, and at session start).

## Relevant details
gitbranch: Sync/pull/push/UpdateMessage in huginn/gitbranch/gitbranch.go; memory branch: muninn/internal/project/branch.go SyncBranch. Commands are short-lived processes, so the syncer must be a detached child (setsid on unix, DETACHED_PROCESS on Windows). Must respect the write turn (me51r2) and m8shdj. Final check is a real trial on Statbus clones.

## Depends on

## Acceptance
- [ ] two clones on one remote: hug task note in clone A shows up in hug task show in clone B within 10 s, without any manual sync (test with a local bare remote)
- [ ] a burst of 20 notes in 2 s produces at most 2 pushes (test counts pushes)
- [ ] with the remote unreachable, hug task note succeeds in under 400 ms and hug task ls reports '1 unpushed' with a fix: line; it is pushed after the remote returns and the next write happens (test)
- [ ] hug mem remember of a project memory in clone A is recallable in clone B within 10 s (test)
- [ ] a rebase conflict keeps both clones' commits, and lint shows it with fix: hug task sync (test)
- [ ] real trial on two Statbus clones, noted with times

## Notes

Jørgen 2026-11-16: graceful detection of network failure, never hang (every git network call has a timeout and runs non-interactively: GIT_TERMINAL_PROMPT=0, ssh BatchMode=yes, connect timeout), and sensible architecture. Good old engineering.
