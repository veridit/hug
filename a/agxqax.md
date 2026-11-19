---
id: agxqax
title: "Person questions written as plain waits, and asks to unknown names"
state: "done"
type: bug
priority: ""
assignee: ["bonehound"]
labels: []
group: ""
rank: "zzzzzzzzzzxf"
created: 2026-11-18
updated: 2026-11-18
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
done_at: "2026-11-18T18:55"
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
- [x] hug task wait ID 'owner: check 4 ...' (and 'jhf: ...', 'the owner's decision ...') is refused with the hug task ask fix; a test fails first on main.
- [x] hug task ask --to jorgen on a board whose [people] has jhf with name Jørgen resolves to jhf or is refused naming jhf (test).
- [x] Lint flags an existing waiting task whose reason addresses a person and an asked task whose asked_for is no known person, each with a runnable fix; on a copy of Statbus's board it finds these 6 (counts in a note).

## Notes


bonehound plan (worktree /Users/jhf/.jcode/scratch/agxqax from origin/main 62c20712 v0.7.64; Statbus board copy cloned read-only at $JCODE_SCRATCH_DIR/agxqax-statbus, push URL disabled). (1) task.AddressesPerson(people, text): a wait text addresses a person when it starts 'NAME:' or says 'NAME's' with NAME = owner / the owner / a [people] short name, alias or (diacritics-folded) name, or says 'the owner'. WaitUntil refuses it (CLI, MCP via CLI, TUI) with fix: hug task ask ID --to SHORT "what" (owner -> the first [people] entry; the 'NAME:' prefix dropped). (2) [people.X] gains optional alias = [...]; PersonMatch finds short, alias, or folded name (jorgen -> jhf). ask --to a name that matches a person resolves to the short name; one that is no person, no agent on the board and not the coordinator, on a board with [people], is refused naming the people. (3) lint: new code wait-on-person (fix: the ask), and unknown-person on a person's long name or alias gets fix hug task ask/design/review ... jhf, not people add. Failing-first tests in huginn/internal/cli.
Red on main 62c20712 ($JCODE_SCRATCH_DIR/agxqax-red.txt): TestWaitAddressedToPersonIsRefused 'exit 0, want 2' on 'owner: check 4 on Norway rc.02 ...'; TestAskToResolvesOrRefusesName 'ask --to "jorgen" did not resolve to jhf'; TestLintFlagsPersonWaitsAndUnknownAsk 'lint lacks wait-on-person'. All 3 pass after, together with TestAddressesPerson (24 cases from real board waits) and TestPersonMatch.
check 3 counts: hug built from this branch, hug task lint --code wait-on-person / unknown-person run on a COPY of Statbus's hug/tasks branch (git clone into $JCODE_SCRATCH_DIR/agxqax-statbus, push URL disabled, inside a local repo with no remote; git status after: 0 changes). Result: 6 findings, exactly the six: wait-on-person on cgtb4y, hmgadt, hzyejk, j2g6wd and mtw92d (fix e.g. 'hug task ask hmgadt --to jhf --check 4 "check 4 on Norway rc.02 (remove a test record through the form)"'), and unknown-person on ezp3h0 (asked_for jorgen; fix 'hug task ask ezp3h0 --to jhf "owner confirms time frames on Norway after the release (rc.02 shipped)"'). hug's board: 0; sql_saga's: 0. Over 141 distinct waiting_on values in hug and Statbus git history, 24 are flagged, all questions to Jørgen; no thing or agent wait is flagged.
check 1 evidence (e886b474 on origin/agxqax, CI 44339826571 6/6): TestWaitAddressedToPersonIsRefused (huginn/internal/cli/personq_test.go): 8 person waits refused exit 2 with the ask fix, e.g. 'owner: check 4 on Norway rc.02 ...' -> fix: hug task ask ID --to jhf --check 4 "check 4 on Norway rc.02 (remove a test record through the form)"; 'jhf: check 2: ...' and "the owner's decision on hug upgrade --apply ..." too; nothing written; 5 thing/agent waits kept; the fix runs. Red on main: 'exit 0, want 2'.
check 2 evidence (e886b474): TestAskToResolvesOrRefusesName: ask --to jorgen, Jørgen, JHF, 'Jørgen H. Fjeld' write asked_for jhf (people jhf name 'Jørgen H. Fjeld'); alias = ["hedda-ssb"] resolves to hhz and survives people add; --to bob refused exit 2 listing 'jhf (Jørgen H. Fjeld), ...' with fix: hug task ask ID --to jhf "..."; an agent on the board and the coordinator pass. Red on main: 'ask --to "jorgen" did not resolve to jhf'.
check 3 evidence (e886b474): TestLintFlagsPersonWaitsAndUnknownAsk (a Statbus-shaped wait and asked_for jorgen; each lint fix runs; lint then 'no findings'). On the Statbus COPY: 6 findings, the six (counts in the note above); hug 0, sql_saga 0.
crab review 17:13 local: accepted. e886b474 on c84d3273: store-level refusal of person-addressed waits on boards with [people] (fix = matching ask), ask --to resolves any of a person's names or a new alias list, unknown names refused with the people listed; lint wait-on-person, unknown-person re-ask fix. Statbus copy exactly the 6; hug and sql_saga 0; 24 of 141 past waits flagged, all real. CI 44339826571 6/6. Choices 1-4 accepted. Lands after the current stack.
bonehound: rebased onto crab's landing stack 80c8ac28 at crab's request: origin/agxqax = 5da549a7. Conflicts in help.go (wait/ask), codes.go and CHANGELOG kept both sides. Niced go build ./... ok; cli ok 295.641s, task ok 37.598s, mcp ok 38.354s. Accepted by crab 16:13.
landed b878852 on main 2026-11-18: task: a question to a person is asked, never waited on, and only of someone hug…
shipped in v0.7.67 (b878852)
Reflection (unanswered, 2026-11-18, crab)
## History
- 2026-11-18T15:55 todo -> doing/active (take)
- 2026-11-18T17:29 doing/active -> doing/waiting (wait)
- 2026-11-18T18:55 doing/waiting -> done (done); was doing_since: 2026-11-18T15:55; waiting_asked_by: bonehound; waiting_on: landing: accepted by crab 16:13 as 5da549a7, in crab's next stack after 80c8ac28; waiting_since: 2026-11-18T17:29
