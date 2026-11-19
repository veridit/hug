---
id: jj7jpz
title: "A background pass that cannot open the board leaves .pending forever"
state: "done"
type: feature
priority: ""
assignee: ["crab"]
labels: []
group: ""
rank: "zzzzzzzzzzxc"
created: 2026-11-18
updated: 2026-11-18
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
done_at: "2026-11-18T17:46"
---

## Goal
A detached pass that returns before Background.Run (board Open fails, no remote, Background error) removes the .pending marker its Nudge wrote, so settled and Busy stop reporting a start that will never queue.

## Purpose
CI 44230245084 (ubuntu, d75c1691): 'sync_test.go:92: background pass still running after 30 s: queued false, busy false, pending true, turn held false', with the pass log line 'open: .../two/.hug/tasks is not a working checkout of hug/tasks (git no longer knows it as a worktree)'. backgroundPass returns at the Open error before b.Run, so the marker Run would remove stays. Pending() has no expiry, so a test waiting on it waits forever.

## Principle
Only the early-return paths remove the marker. After Run it is never removed again, because a newer Nudge's marker must stay visible (rt80x4). No timeouts raised.

## Relevant details
Found while working 6ep7y2 (tigress). Fix in huginn/internal/cli/sync.go plus gitbranch.BackgroundOf. Branch 6ep7y2 stack.

## Depends on

## Acceptance
- [x] a counting test that runs the real detached child against a board Open cannot open: N of N passes leave no .pending (fails first on origin/main)
- [x] mutation: dropping the removal makes the counting test fail
- [x] ordinary CI green on the fix commit (run id)

## Notes

Fix 41ba4ed1 on branch 6ep7y2 (based on origin/main c87da5a8, tigress). Failing first, before the production edit: 'sync_pending_test.go:61: 5 of 5 passes that could not open the board left their pending marker'. With the fix, -count=3: 'ok  	github.com/veridit/hug/huginn/internal/cli	5.744s'. Mutant without b.Abandon(born): '5 of 5 passes that could not open the board left their pending marker' again, then restored. Raw: scratch/6ep7y2-local/jj7jpz-*.txt. The test removes the board worktree's admin folder, so gitbranch.Open refuses it, as on CI 44230245084. CI pending on the 4th 6ep7y2 push.

probe

Ordinary CI on abc8a2a9 (branch ci/6ep7y2-h5; contains 41ba4ed1, plus 8023b9f1 test-only diagnostics and a branch-only experiment workflow): run 44318566065, all 6 jobs success: secrets 134851100953, bench 134851179930, test (windows-latest) 134851180031, test-windows-cli 134851180053, test (macos-latest) 134851180073, test (ubuntu-latest) 134851180204. Not merged to main (crab's call).

crab review 15:41 local: accepted. 41ba4ed1: BackgroundOf + Abandon(born) removes .pending only when not newer than the child's start (rt80x4 kept); failing-first 5/5, mutant fails 5/5. 8023b9f1 test-only trace2 stall diagnostics. CI 44318566065 6/6 ran these two inside abc8a2a9; I land land/6ep7y2-jj7jpz after a rebase and exact-SHA CI.

Landed on main via 56e2cc6f (stack rjk44w c2df86c4, jj7jpz 215df154, 2qndsd 56e2cc6f; CI 44332698559 exact SHA 6/6). Ships in the next release.

landed 0c9e122 on main 2026-11-18: sync: a pass that cannot open the board ends its pending marker (jj7jpz)

shipped in v0.7.65 (0c9e122)


Reflection (unanswered, 2026-11-18, crab)
## History
- 2026-11-18T17:46 todo -> doing/active (take)
- 2026-11-18T17:46 doing/active -> done (done); was doing_since: 2026-11-18T17:46
