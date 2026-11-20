---
id: tb0dra
title: "Agents reflect on the instructions and propose fixes"
state: "doing"
type: feature
priority: ""
assignee: ["crab"]
labels: []
group: ""
rank: "0r"
created: 2026-11-16
updated: 2026-11-16
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
stage: "asked"
asked_what: "Approve these three additions to the AGENTS.md section hug setup writes into every repo?"
asked_for: "jhf"
asked_since: "2026-11-19T00:58"
asked_by: "piglet"
asked_check: "5"
asked_show: "agentsection-proposal.md"
---

## Goal
Agents reflect at done on whether the instructions guided them, and each reflection becomes an instruction change or is dismissed.

## Purpose
Jørgen 2026-11-16: 'you have to be mindful of the instructions, if they're good, if they're guiding you and the agents. We have to observe ourselves and adjust. Some reflection should be built into us.' Today's evidence: the todo misuse (tzc9hb), the main-push collisions that needed hand-off DMs, and the check in 44w1ev that contradicted a later ruling were all instruction gaps found only by chance.

## Principle
One short question at done, never a form. Empty answers are allowed (nothing to report). A reflection names the instruction text it is about and what happened, with evidence; no vague 'could be better'. Agents propose; changes to installed instructions are reviewed by a human before hug setup ships them. Harness-neutral wording.

## Relevant details
Reflection is part of the loop. When an agent closes a ticket, hug task done asks it one question: did the instructions (AGENTS.md section, tool descriptions, hug messages) guide you well, or did one mislead you or leave a gap? The answer is stored with the ticket as a Reflection note naming the exact instruction. hug task reflections collects open ones across the board, and a coordinator or Jørgen turns them into an instruction change (shown as a diff) or dismisses them.

Builds on tzc9hb (the loop in the AGENTS.md section) and x325pp (nudges only between tasks). done already prints a next line; the question goes there, and MCP task_done accepts an optional reflection field.

## Depends on
- tzc9hb

## Acceptance
- [x] hug task done ID --reflect TEXT stores a Reflection note on the ticket; plain done prints the one-line question (test)
- [x] MCP task_done accepts reflection and its description asks the question (test)
- [x] hug task reflections lists open reflections with ticket, instruction quoted and date; resolve marks one applied or dismissed (test)
- [x] the hug setup AGENTS.md section tells agents to answer the question honestly, including 'nothing'
- [ ] after a week of use, one instruction change in this repo traced to a reflection (link in Notes)

## Questions
- 2026-11-19T00:58 question 1 by piglet to jhf (check 5, show agentsection-proposal.md):
  Approve these three additions to the AGENTS.md section hug setup writes into every repo?

## Notes

guppy 2026-11-17 10:09: take refused (blocked on tzc9hb). tzc9hb checks 1-3 are on ci/tzc9hb d355085; its check 4 needs a release with that commit (hug setup in this repo and seid-lang), which the releaser does. I build tb0dra on top of ci/tzc9hb meanwhile and take it once tzc9hb is done. Question for the coordinator: promote tzc9hb check 4 to a release-time task so tzc9hb can close at merge?

guppy 2026-11-17 10:40: rebased on main 71d7ff7 (tzc9hb), ci/tb0dra 6679ad8; earlier CI 44054747337 (585f775) green on all three. Conflicts cli.go and help.go resolved by hand, keeping main's sync --background-pass flag and longer sync help. Local: TestBackgroundSyncTwoClonesRealBinary failed once (TempDir cleanup: directory not empty), passed 3/3 on rerun here and 3/3 on main.

guppy 2026-11-17 11:01: CI 44057681493 (6679ad8): ubuntu, macos, secrets green; windows huginn/internal/cli hit the 10 m test timeout (goroutine in take_branch_test.go:57). Same timeout on main's own CI 44056677278 (93bee16, v0.7.39), so not this change; failed job rerun requested.

guppy (hatchling) 12:35: on main 6d88d74 (fast-forward after rebase over 0be2c34). CI 44069850437 (59262e3, same tree minus AGENTS.md) green ubuntu, macos, windows, secrets, bench. Tests: huginn/internal/task/reflect_test.go and cli/reflect_test.go (done --reflect stores 'Reflection (open, DATE, WHO): ...' only after done succeeds, 'nothing' stores none, plain done adds the reflect: question; reflections [--all]; resolve ID N applied|dismissed), mcp take_test (task_done reflection+agent, description asks). Setup section line: 'hug task done ID asks whether the instructions guided you well; answer honestly with --reflect "...", "nothing" is fine'. Check 5 needs a week of use.

guppy (hatchling) 12:41: stays doing, waits on check 5 (a week of use since landing 6d88d74 on 2026-11-17). Look again 2026-11-24.


retriever 2026-11-18 13:35 (board check for crab, hatchling's session is gone, journal last written 06:47): the wait is real and still holds. Check 5 needs a week of use since 6d88d74 landed on 2026-11-17, so it can next move on 2026-11-24. Who: a worker crab assigns that day; owner hatchling is no longer running. Material for check 5 today: hug task reflections lists 6 open reflections (6ephqk, kx2ndm, pbh9xq, h176dp, jmx10s, 44w1ev).
file:agentsection-proposal.md (3031 bytes, sha256:60579bed) Three additions to agentSection (G2b, G3 trimmed, G5) from the reflections triage; crab-reviewed 2026-11-19; for Jørgen
instruction change traced to reflections: daa4c889 on main (built as a0939668; G2 h176dp/t64v7d/bejd89 and G7 gjmqhn/n2hq8p/t3fqaw/r5r8c1/55a7xx)
landed daa4c88 on main 2026-11-19: AGENTS.md: write a superseding ruling on the earlier ticket; go vet after rebase…
landed 2320b92 on main 2026-11-19: AGENTS.md: shell scripts run on macOS's bash 3.2 too (tb0dra)
shipped in v0.7.74 (daa4c88)
file:screen-1057.ans (6373 bytes, sha256:0fd93a69) screen capture from hug tui
file:screen-1057-2.ans (24091 bytes, sha256:39c4a77d) screen capture from hug tui
## History
- 2026-11-17T14:36 doing -> doing/waiting (wait)
- 2026-11-19T00:58 doing/waiting -> doing/asked (ask); was waiting_asked_by: hatchling; waiting_on: a week of use, look again 2026-11-24; waiting_since: 2026-11-17T14:36
