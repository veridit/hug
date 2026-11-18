---
id: agxqax
title: "Person questions written as plain waits, and asks to unknown names"
state: "doing"
type: bug
priority: ""
assignee: ["bonehound"]
labels: []
group: ""
rank: "zzzzzzzzzzzs"
created: 2026-11-18
updated: 2026-11-18
doing_since: "2026-11-18T15:55"
stage: "active"
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
---

## Goal
A question to a person can never sit in waiting or be asked of a name hug does not know: wait refuses a reason that addresses a person ('owner: ...', a [people] name or alias), ask --to refuses or resolves an unknown name, and lint flags both on existing boards with a runnable fix.

## Purpose
Jørgen 2026-11-18 on Statbus (screenshot): 5 waiting tasks are questions for him ('owner: check 4 on Norway rc.02 ...', j2g6wd 'the owner's decision on hug upgrade --apply'), and ezp3h0 is asked of 'jorgen', not 'jhf', so it is missing from his 'for you' list. Statbus was upgraded at 11:39 (4 moved to asked); hmgadt and mtw92d were written as waits at 12:28, after it.

## Principle
By design, not doctrine: the tool refuses the wrong shape at write time, with the right command, rather than relying on agents to remember. One person, one name: [people] keys are the names; aliases resolve to them.

## Relevant details
Statbus: hug.toml [people] jhf, erik, hhz. Plain wait refuses only --on NAME (cli.go:1184). Upgrade's MoveToAsked moves only waiting_for or 'NAME: what' with NAME in [people]; 'owner:' is not a name. Statbus waits: j2g6wd (hibiscus), cgtb4y, hmgadt, mtw92d, hzyejk (rabbit). ezp3h0 asked_for jorgen.

## Depends on

## Acceptance
- [ ] hug task wait ID 'owner: check 4 ...' (and 'jhf: ...', 'the owner's decision ...') is refused with the hug task ask fix; a test fails first on main.
- [ ] hug task ask --to jorgen on a board whose [people] has jhf with name Jørgen resolves to jhf or is refused naming jhf (test).
- [ ] Lint flags an existing waiting task whose reason addresses a person and an asked task whose asked_for is no known person, each with a runnable fix; on a copy of Statbus's board it finds these 6 (counts in a note).

## Notes


bonehound plan (worktree /Users/jhf/.jcode/scratch/agxqax from origin/main 62c20712 v0.7.64; Statbus board copy cloned read-only at $JCODE_SCRATCH_DIR/agxqax-statbus, push URL disabled). (1) task.AddressesPerson(people, text): a wait text addresses a person when it starts 'NAME:' or says 'NAME's' with NAME = owner / the owner / a [people] short name, alias or (diacritics-folded) name, or says 'the owner'. WaitUntil refuses it (CLI, MCP via CLI, TUI) with fix: hug task ask ID --to SHORT "what" (owner -> the first [people] entry; the 'NAME:' prefix dropped). (2) [people.X] gains optional alias = [...]; PersonMatch finds short, alias, or folded name (jorgen -> jhf). ask --to a name that matches a person resolves to the short name; one that is no person, no agent on the board and not the coordinator, on a board with [people], is refused naming the people. (3) lint: new code wait-on-person (fix: the ask), and unknown-person on a person's long name or alias gets fix hug task ask/design/review ... jhf, not people add. Failing-first tests in huginn/internal/cli.
## History
- 2026-11-18T15:55 todo -> doing/active (take)
