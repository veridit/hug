---
id: fthher
title: "hug task ls cannot list the questions waiting for someone"
state: "doing"
type: feature
priority: ""
assignee: ["crab"]
labels: []
group: ""
rank: "zzzzzzzzzzza"
created: 2026-11-21
updated: 2026-11-21
doing_since: "2026-11-23T13:00"
stage: "asked"
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
parent: "j2v40p"
asked_for: "jhf"
asked_since: "2026-11-23T14:23"
asked_by: "crab"
asked_check: "2"
asked_what: "Run hug task ls --asked (v0.7.110). Same question as release ticket j2v40p; one answer ticks both."
---

## Goal
One shell command lists every open question asked of a person, with task, check and question.

## Purpose
crab 2026-11-21 19:03Z told Jørgen 'hug task ls --asked lists the ones waiting for you'; the flag does not exist (unknown flag --asked). Only the TUI's ASKED section and '? N for you' show them. An agent guessed the flag, so a human will too.

## Principle
Reuse ls: a filter, not a new command. Its error for --asked must point to it.

## Relevant details
Candidate: hug task ls --asked [--to NAME] (default: the caller), one line per question: id, check, first line of the question, asked since.

## Depends on

## Acceptance
- [x] hug task ls --asked lists the open questions asked of the caller, one per line with id and check (test)
- [ ] Jørgen runs it and gets his list

## Questions
- 2026-11-23T14:23 question 1 by crab to jhf (check 2):
  Run hug task ls --asked (v0.7.110). Same question as release ticket j2v40p; one answer ticks both.

## Notes


Real board copy (git ls-files of .hug/tasks, 635 files, rsynced into scratch repo on hug/tasks; HUG_USER=jhf HUG_SYNC=off isolated HOME; binary from 97f74c64). hug task ls --asked: 40 open questions for jhf, on 38 tasks. TUI on the same copy (tui.newModel User jhf): countForYou=38 forYouCount=38, so '? 38 for you' = 38 tasks in ls --asked (a task may ask jhf twice: hpqt99 asks 3). Brief said 37 at 08:20Z; the board has moved since. Same 40/38 with HUG_USER unset (person by git email). First 10 lines:
bxqtkt  -    approve the design (attached: bxqtkt.md), check 1, and answer Q1-Q6 at its end       calf      since 17 Nov
p277bp  3    Mascot artwork: the drafts are in your Grok Imagine library (prompts in docs/art/d…  crab      since 20 Nov
hpqt99  2    Decision 1, which client owns an imported memory: 1A keep today (you rescope by ha…  crab      since 20 Nov
hpqt99  5    Import and export (Decision 3 in Details): import previews by default, --apply imp…  crab      since 20 Nov
hpqt99  3    Decision 2, jcode memories: 2A keep reading jcode's files live, 2B import once and…  crab      since 20 Nov
rnpdnm  3    5 of the 8 should never have been memories (memory-qualifiers.md). Option D: tight…  crab      since 20 Nov
b8mk9s  5    Six layouts in Details, measured on your 18,615 memories. A day holds 38 KB typica…  crab      since 20 Nov
kka3b1  1    v0.7.91: with rows marked, r opens one request box under the list and Enter sends …  crab      since 20 Nov
dswkgj  1    v0.7.92: answering a withdrawn question now says when and why it was withdrawn and…  crab      since 20 Nov
24403z  6    After you answer a task in ASKED it moves (here to NEXT UP). The breadcrumb keeps …  eagle     since 20 Nov
check 1: 97f74c64 on fthher and ci/fthher; CI https://github.com/veridit/hug/actions/runs/45179618266 8/8 success. Tests: TestLsAskedListsQuestionsForCaller, TestLsAskedEmptySaysSo, TestInstructionsNameLsAsked (cli), TestAskedAgreesWithForYou (tui: per-task count equals countForYou on stage/asked/withdrawn boards for jhf, Jørgen, nautilus, nobody), TestListTakesAsked (mcp). 13/13 mutants killed. Real board copy: 40 questions for jhf on 38 tasks, TUI countForYou 38.
## History
- 2026-11-23T13:00 todo -> doing/active (take)
- 2026-11-23T14:23 doing/active -> doing/asked (ask)
- 2026-11-23T15:33 handed to crab: orangutan's session ended (closed 2026-11-23 14:13)
