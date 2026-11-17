---
id: t709he
title: "Commits name the agent and model that made them (gate)"
status: "done"
type: feature
priority: "high"
assignee: ["wyvern"]
labels: []
created: 2026-11-16
updated: 2026-11-16
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
---

## Goal
Every commit in a hug repo, code and tasks, carries exactly one Model: trailer naming what made it: a model id (Model: claude-opus-5-5) or Model: human when the git author did it themselves. There is no Agent: trailer. A commit without Model: is refused with a fix: line, so later we can find all work done by a model with a known deficiency and act on it

## Purpose
Jørgen 2026-11-16: 'knowing the model that did the work sometimes lets you take action if you know a model had certain deficiencies'. Today nothing records it: git author is the machine user for every agent

## Principle
Git trailers, so git log --format=%(trailers) and git log --grep find them. hug setup installs a commit-msg hook that refuses a commit without Model: (Jørgen 2026-11-16: always declare something; agents are ephemeral, so no Agent: line; the git author is the person, Model: human means the author did it); hug's own commits on hug/tasks add them from the same identity as take (et85h1). No guessing: the agent states them.

## Relevant details
Depends on et85h1 for the identity source (HUG_USER, --as, JCODE_SESSION_ID; HUG_MODEL, --model). The hook must be cheap (under 50 ms) and work in linked worktrees (065wwn). From 2026-11-16 the coordinator asks workers to add the trailers by hand until this ships.

## Depends on
- et85h1

## Acceptance
- [x] hug setup installs a commit-msg hook; git commit without a Model: trailer is refused with a fix: line showing it (Model: human for a person); there is no Agent: trailer (test in a temp repo)
- [x] commits hug makes on hug/tasks carry Model: from the taker's identity (test)
- [x] hug task log or a documented git log line lists commits by model

## Notes

2026-11-16 Jørgen on his own commits: 'maybe the model is me'. Proposed: a person writes their own name as the model (Model: jhf). Git author cannot tell, because agents commit under his git user too. Awaiting his yes.

2026-11-16 decided (Jørgen): a person's own commit says Model: human. The hook accepts it like any model; the person may add Agent: <name>.

2026-11-16 superseded (Jørgen): no Agent: trailer at all, not even optional. One Model: line always; the git author is the person, Model: human means they did it themselves.

dragon, 0750044 on dragon/t709he (stacked on 0hkaa5; crab rulings 19:22 applied: no tracked-hook edits, no bypass var, exempt hug/tasks + muninn-notes, TUI Model: human, never guess for CLI/MCP, unpastable slot). Check 1: hug task install commit-msg -> <common-git-dir>/hooks/commit-msg (POSIX sh), chains a prior hook as commit-msg.pre-hug; core.hooksPath -> hooks/hug-commit-msg + 'hug setup' step 'Model: trailer gate' open until the project's commit-msg has the 2 lines. Refusal: 'commit refused: no Model: trailer' ... 'fix: git -c trailer.ifexists=replace commit --cleanup=strip -F '<.git/COMMIT_EDITMSG>' --trailer 'Model: <model-id-or-human>'' -- pasted as is: refused 'names no model (replace the <...> slot)'; slot filled: commit lands with Model: gpt-6.1-sol. Exempt: fixup!, merges/rebase (rebase --force-rebase test), muninn-notes branch, --no-verify; refused: 2 Model: lines, empty Model:, Model only in body prose. No Agent: anywhere. Cost: 'commit with gate 104.968917ms, --no-verify 65.92825ms' (39 ms) under load. Tests TestCommitMsgGate, TestCommitMsgGateChainsExistingHook, TestCommitMsgGateWithHooksPath PASS. Scratch trial (copies only, $JCODE_SCRATCH_DIR/t709he-trial.sh + t709he-seid.sh): statbus copy refuses no-Model commit, accepts Model:, its own bare-ticket rule still blocks 'closes #12'; seid copy: Co-Authored-By only -> 'commit refused: no Model: trailer'; both trailers rc=0; Model only -> seid's own 'REFUSED — missing Co-Authored-By trailer'. Gate alone 0.05-0.06 s. Check 2: TestBoardCommitsCarryModelTrailer PASS: board log 'hug task take ID|claude-opus-5-5', 'note|gpt-6.1-sol' (HUG_MODEL), 'new|test-model'. Check 3: README 'Who made a commit' + help: git log --format='%h %(trailers:key=Model,valueonly,separator=%x2C) %s' (verified on this repo: 'e6a135a claude-opus-5-5 hug tui: the detail header...'), git log --grep='^Model: gpt-6.1-sol$'. CI after 0hkaa5 lands.

dragon 3e8d6d3 on dragon/t709he: the gate's 50 ms test failed under load ('commit with gate 123.306ms, --no-verify 44.228042ms' -> 'the gate costs 79.077958ms per commit, over 50ms', 14 packages testing beside it). Fix: the gate drops dirname and sed (${0%/*}, read -r), leaving 2 git calls + 1 awk; the test takes the fastest of 7 interleaved commits each. Same load, -count=3: 'gate 102.017333ms, --no-verify 57.270375ms', 'gate 102.355041ms, --no-verify 65.981458ms', 'gate 72.293417ms, --no-verify 49.213125ms' (45/36/23 ms); TestCommitMsgGate, ChainsExistingHook, WithHooksPath PASS x3.

2026-11-17 09:00 crab: NOW waiting on 44w1ev's SQLITE_BUSY fix; stacked on it as ece85d9 (ci/m1c28r); wyvern lands it after 44w1ev.

Check 1: TestCommitMsgGate, TestCommitMsgGateChainsExistingHook, TestCommitMsgGateWithHooksPath (huginn/internal/cli/commitmsg_test.go:24,151,184) in a temp repo: hug task install commit-msg installs the hook, a commit without Model: is refused with a fix: line that shows Model: human for a person, no Agent: trailer. Check 2: TestBoardCommitsCarryModelTrailer (huginn/internal/cli/trailer_test.go:11): new, note and take commits on hug/tasks end with one Model: line from HUG_MODEL or --model. Check 3: README.md:58-60, git log --format='%h %(trailers:key=Model,valueonly,separator=%x2C) %s' lists every commit with its model; --grep='^Model: gpt-6.1-sol$' one model's commits; git -C .hug/tasks log for task changes. Shipped in a08bb34 (v0.7.40), CI 44065886978 green on ubuntu, macOS, Windows.

Gate timing test, landed as d74af5d on main (CI 44087396396 green on all 5 jobs). Before it, on f583cac under local load (go test ./huginn/..., niced, 13:15Z), the old test failed: 'commit with gate 151.380791ms, --no-verify 72.261209ms (fastest of 7, interleaved)' / 'the gate costs 79.119582ms per commit, over 50ms'. The new test counts processes (awk=1 git=2). A mutation adding 'git --version' to the hook fails it: 'the gate starts awk=1 git=3 per commit, want awk=1 git=2'.
