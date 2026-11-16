---
id: 0hkaa5
title: "Board and project memories stay in sync across clones, in the background"
status: "doing"
type: feature
priority: "high"
assignee: ["dragon"]
labels: []
created: 2026-11-16
updated: 2026-11-16
model: "claude-opus-5-5"
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
- [x] two clones on one remote: hug task note in clone A shows up in hug task show in clone B within 10 s, without any manual sync (test with a local bare remote)
- [x] a burst of 20 notes in 2 s produces at most 2 pushes (test counts pushes)
- [ ] with the remote unreachable, hug task note succeeds in under 400 ms and hug task ls reports '1 unpushed' with a fix: line; it is pushed after the remote returns and the next write happens (test)
- [x] hug mem remember of a project memory in clone A is recallable in clone B within 10 s (test)
- [x] a rebase conflict keeps both clones' commits, and lint shows it with fix: hug task sync (test)
- [ ] real trial on two Statbus clones, noted with times

## Notes

Jørgen 2026-11-16: graceful detection of network failure, never hang (every git network call has a timeout and runs non-interactively: GIT_TERMINAL_PROMPT=0, ssh BatchMode=yes, connect timeout), and sensible architecture. Good old engineering.

design (dragon 18:10): (1) runAt: network verbs (ls-remote/fetch/push/pull/clone) get -c http.lowSpeedLimit=1000 -c http.lowSpeedTime=15, GIT_ASKPASS= SSH_ASKPASS_REQUIRE=never GCM_INTERACTIVE=never, ssh BatchMode=yes ConnectTimeout=10 unless the user has an ssh command, and a 60 s kill (WaitDelay 2 s). (2) gitbranch/background.go: one detached syncer per board, with two flocks in <git-common-dir>/hug/sync/ (queue and run): a nudged child takes the queue lock (else exits, the queued one will see its commit), then the run lock, frees the queue, waits 2 s (debounce), then fetch, rebase under the board turn, push; no lost wakeups. State JSON: last fetch, unpushed, error, conflict. (3) board turn: write commands hold <common>/hug/sync/<branch>.turn for the whole read-modify-commit, and the syncer only for rebase, so a background rebase can never revert a foreground edit (m8shdj check 1 follows). (4) reads nudge when the last fetch attempt is >5 s old (stat only; the 10 s acceptance needs <30 s). (5) the reply carries sync {unpushed, error, fix: hug task sync} only after a failed sync (no git walk normally). (6) spawning only from the real hug binary or with HUG_TEST_SYNC_CHILD=1 (test binaries never fork-bomb); HUG_SYNC=off disables it. (7) mem: hug mem sync --background via the same protocol, nudged after remember/forget of project memories and on recall when stale.

dragon, commits 86e21ef + 43b6a6a on branch dragon/0hkaa5 (ci/0hkaa5). Evidence (GOMAXPROCS=2 GOFLAGS=-p=2 nice -n 19 go test -count=3 -run TestBackgroundSync -v ./huginn/internal/cli/): 3 of 3 rounds PASS for every test: NoteReachesOtherClone 'note reached clone B in 3.2s / 3.98s / 3.26s'; BurstIsBatched (20 notes, pushes counted by a post-receive hook on the bare remote, want 1..2) PASS; OfflineCatchesUp PASS, ls said 'sync: 1 unpushed: ls-remote --heads origin refs/heads/hug/tasks: the remote is unreachable: fatal: '...remote.git' does not appear to be a git repository' + 'fix: hug task sync', pushed after reconnect and the next note; ConflictKeepsCommits PASS; ParallelNotesAllLand (20 parallel notes, all 20 reach clone B) PASS. Real binary, real detached syncers (go test -run TestBackgroundSyncTwoClonesRealBinary ./cmd/hug/): 'clone B saw the note after 3.92s and the memory after 6.2s'. Package runs: ok gitbranch 13.0s, cli 93.3s, mcp 5.8s, cmd/hug 10.5s, muninn project 3.7s, mcpserver 3.3s, run 12.5s, store 3.7s. Note cost with vs without background sync (10 notes each, niced): 443.6 ms / 466.1 ms / 441.9 ms / 444.3 ms per note (spawn off/on/off/on): the board itself costs ~440 ms here, the sync adds ~0-20 ms. Check 3's '<400 ms' is therefore not met in absolute terms on this machine by any note, online or off; the test asserts offline <= baseline+150 ms (measured: offline 272 ms vs 275 ms without sync).
