---
id: a0gwvg
title: "Repo tool: regenerate TUI goldens from a failed CI run's log"
state: "done"
type: feature
priority: ""
assignee: ["duckling"]
labels: []
group: ""
rank: "zzzzzzzzzzzm"
created: 2026-11-18
updated: 2026-11-18
model: "claude-opus-5-5"
branch: "ci-goldens"
worktree: "/Users/jhf/.jcode/scratch/ci-goldens"
done_at: "2026-11-18T23:50"
done_release: "v0.7.72"
---

## Goal
A scripts/ci-goldens.sh RUNID [JOB] in the repo rewrites the TUI goldens a CI run reported as 'golden drift', from that run's GOT blocks, so goldens can be refreshed when a local HUG_UPDATE_GOLDEN run cannot finish (machine load) or the drift is platform-specific

## Purpose
2026-11-18 stack 6448e7d5 failed CI 44403710185 on ubuntu only (tmux palette goldens, txjkrm TAGS x 05r3pg palette); crab's local regen could not finish at load ~225. A scratch script fixed it as 4d7eec2d in minutes

## Principle
Never writes unless every WANT block, decoded the same way, is byte-equal to the committed golden (proves the decoding on that run's own log). Refuses when two jobs disagree on a path. Runs on macOS bash 3.2. Errors end with a fix: line. It is a convenience next to HUG_UPDATE_GOLDEN, not a replacement: the diff is still reviewed

## Relevant details
Prototype attached (ci-goldens.sh, from $JCODE_SCRATCH_DIR). Format source: huginn/internal/tui/tmux_test.go t.Errorf("golden drift %s\nGOT:\n%s\nWANT:\n%s"); testing.decorate indents every line after the first and drops the final newline. First use: CI 44403710185 -> 4 palette.txt files, commit 4d7eec2d on ci/stack71. Other golden helpers (golden(), goldenFile() in alias_search_test.go, e3sevr_test.go) may print differently: check before claiming them

## Depends on

## Acceptance
- [x] scripts/ci-goldens.sh exists and AGENTS.md's TUI goldens line names it
- [x] a test feeds a recorded --log-failed excerpt (with a blank line and a last-line golden) and gets byte-identical files
- [x] a tampered WANT block makes it write nothing and exit non-zero with a fix: line

## Notes

file:ci-goldens.sh (4365 bytes, sha256:5e41e14e)


check 1: 917a3a28: scripts/ci-goldens.sh (shellcheck 0.11.0: 0 findings); AGENTS.md line 27 under TUI goldens: 'Under load, take the GOT screens from a failed CI run: scripts/ci-goldens.sh RUNID.'
check 2: TestCIGoldensRebuildsDriftedGoldens (internal/repocheck): --log testdata/ci-goldens/failed.log (2 blocks of CI 44403710185, 140x40 screen with 6 blank lines, last rows with trailing spaces) gives bytes equal to 4d7eec2d's goldens, with and without JOB. Failed first: the files came out one '\n' long, fixed. Real run: the script in a worktree of 6448e7d5 wrote 4 files, 'git diff --quiet 4d7eec2d -- huginn/internal/tui/testdata' exit 0.
check 3: Same test: one byte changed in the 140x40 golden (GLOBAL->GLOBAl) -> exit non-zero, 'ci-goldens: wrote nothing' and a 'fix:' line, 80x24 golden left as it was; an empty log -> "no 'golden drift' block" + fix: line.
CI 44411972128 on 917a3a28: completed/success, 8/8 jobs.
Landed on main as c53dae61 (crab), CI 44413400970 8/8.
Reflection (applied, 2026-11-18, duckling): One gap: the scratch prototype assumed bash 4 (declare -A) and failed on macOS bash 3.2 ('declare: -A: invalid option'); AGENTS.md says nothing about which shell scripts/ targets. Also, the tool gate refused file writes into a runtime-computed path ($T/...), which pushed the real-run check into a git worktree: better, since gh needs a repo to resolve the run.
landed c53dae6 on main 2026-11-18: scripts/ci-goldens.sh: rewrite drifted TUI goldens from a failed CI run (a0gwvg)
shipped in v0.7.72 (c53dae6)
Reflection 1 applied (tb0dra triage G14, crab 2026-11-19): AGENTS.md now says '- Shell scripts run on macOS's bash 3.2 too: no bash 4 features (declare -A, mapfile, ${x,,}).' (commit 0ba5e9e9, ci/g14, CI 44429351931). The tool-gate half is about the harness, not hug: dismissed.
## History
- 2026-11-18T23:24 todo -> doing/active (take)
- 2026-11-18T23:37 doing/active -> doing/waiting (wait)
- 2026-11-18T23:50 doing/waiting -> done (done); was doing_since: 2026-11-18T23:24; waiting_asked_by: duckling; waiting_on: reviewed by crab; he stacks it with ydwamb; waiting_since: 2026-11-18T23:37
