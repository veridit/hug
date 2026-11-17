---
id: sfa8w4
title: "Acceptance checks can be struck out as not relevant, with a reason"
status: "doing"
type: feature
priority: "high"
assignee: ["kestrel"]
labels: ["needs-discussion"]
created: 2026-11-16
updated: 2026-11-16
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
---

## Goal
A check that no longer applies can be struck out with a reason, shown crossed through, and a task closes only when every check is either ticked or struck out

## Purpose
Jørgen 2026-11-16: closing a task without ticking its boxes, or marking them obsolete, is broken: otherwise there is no accountability. Today a check has two states (open, ticked), so the only ways to close a task whose check stopped applying are ticking something that was not done or leaving it open, which is what 432 done tasks in the seid copy did

## Principle
Three states: open, ticked, struck out. Striking out always carries a reason and who and when, so it is a recorded decision, not a way around the check. Done counts struck-out checks as settled. A done task with an open check is a broken finding, not advice (ticket pbh9xq). The file stays plain Markdown that renders correctly anywhere

## Relevant details
Today: huginn/internal/task/document.go Acceptance{Text, Checked, Line}; parse accepts '- [ ] ', '- [x] ', '- [X] ' (line 321); check rewrites '[ ]' to '[x]' (line 458); done refuses on unchecked items with kind unchecked-acceptance (store.go 357-364); lint code done-unchecked (document.go 44). Proposed file form, standard Markdown strikethrough: '- [ ] ~~text~~ (not relevant: reason, name, date)'. Proposed command: hug task strike ID N --reason "..." and an undo (hug task strike ID N --undo); MCP task_update action strike. Show: struck-out items print crossed through or as '[-]' in plain text. TUI: draw with strikethrough and dim, a key to strike with a reason prompt; Space still ticks. Promote already turns an item into 'done when task:X is done'; keep that working. Decide with Jørgen via the coordinator: the command name, and whether a reason is mandatory (proposal: yes)

## Depends on

## Acceptance
- [ ] hug task strike ID N --reason R marks item N struck out, keeps its text, and records reason, name and date in the file
- [ ] hug task done succeeds when every check is ticked or struck out, and still refuses with a fix line when one is open
- [ ] show, ls --jsonl and MCP report the three states; the TUI draws a struck-out check crossed through
- [ ] a done task with an open check is reported by lint as broken
- [ ] the stored form renders as a crossed-out checklist line in a standard Markdown viewer, and older files parse unchanged (test)

## Notes

2026-11-16 Jørgen: striking out a check with a reason must be available to him in the TUI too, not only to agents over MCP and the CLI. Already in check 3 ('the TUI draws a struck-out check crossed through'); made explicit: with a check focused, a key strikes it out and asks for the reason in a prompt, and the same key undoes it. Build CLI, MCP and TUI in the same ticket.

kestrel 12:41, built (branch kestrel/sfa8w4, rebased on 6d88d74; suite running, then ci/sfa8w4). Decisions taken from the ticket's proposals (Jørgen may overrule): command hug task strike ID N --reason R, reason mandatory, --undo restores; MCP task_update action strike (item, reason, undo, agent, model); TUI key '-' on the focused check opens 'strike out, reason:' prompt, Enter strikes (empty reason strikes nothing), '-' on a struck check asks 'Restore struck check: ☒ N. text' and Enter restores; a ticked check cannot be struck, a struck one must be restored before ticking. File form '- [ ] ~~TEXT~~ (not relevant: REASON, NAME, YYYY-MM-DD)'. Evidence: task/strike_test.go TestStrikeOutCheck (file line '- [ ] ~~second, no longer~~ (not relevant: replaced by check 3, see note, kestrel, DATE)'; done refused with check 3 open, listing only '3. third' and 'or strike one that no longer applies: hug task strike ID 3'; done succeeds with ticked + struck; lint clean); TestDoneWithOpenCheckIsBroken ('done with unchecked acceptance: broken, k7m2qx is done but check 1 is open; tick it ... or strike it out with hug task strike k7m2qx 1 --reason ...', struck check not flagged); TestStruckFormParsesAndOldFilesUnchanged (sample round-trips byte-identical); TestStruckFormRendersCrossedOut (goldmark GFM: '<li><input disabled="" type="checkbox"> <del>no longer applies</del> (not relevant: ...)</li>'); cli/strike_test.go TestStrikeThroughCLI (show '2. [-] ~~no longer applies~~ (not relevant: the API changed, kestrel, DATE)', ls --jsonl state/reason, lint --all '[done-unchecked] ... broken'); mcp/strike_test.go TestStrikeOverMCP (states ticked,struck,open; reason required; undo); tui/strike_test.go TestStrikeInTUI ('☒ 2. second check (not relevant: covered by the first, tester, DATE)', dim + strikethrough). Goldens: 6 tmux files, only the key bar ('· - strike out') and help line ('Space/-/Enter   tick / strike out check / follow dep >').

kestrel 12:58: 4eb3e6e on ci/sfa8w4 (rebased on 893cd50), CI 44078742157 running; full niced suite on 4eb3e6e exit 0. TUI key bar shows '- strike out' only at >= 120 columns (TestVersionCorner: at 100 it pushed '? help · q quit' off); the Acceptance heading always says '· - strikes out' and ? help lists 'Space/-/Enter   tick / strike out check / follow dep >'. Goldens: 5 files (heading and the 80-col bar). Open question for Jørgen: command name 'strike' and mandatory reason were taken from the ticket's proposal.
