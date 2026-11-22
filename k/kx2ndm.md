---
id: kx2ndm
title: "TUI: groups and epics visible and collapsible"
state: "done"
type: feature
priority: "high"
assignee: ["nautilus"]
labels: ["needs-discussion"]
group: ""
rank: "o"
created: 2026-11-16
updated: 2026-11-16
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
done_at: "2026-11-18T00:40"
done_release: "v0.7.51"
---

## Goal
In overview, list and board a reader sees which group (e.g. later) and which epic each task belongs to, and can fold a group or epic away

## Purpose
Jørgen 2026-11-16: e4w9ya is in group later, but the overview shows it among NEXT UP with nothing saying so; he pressed Space on a row expecting a group to expand or collapse

## Principle
Compact: no extra line per task. Group and epic are shown as section headings or a short dim cell, not repeated prose. Space folds or unfolds a heading; it never ticks a check outside the detail view. Only show keys that work

## Relevant details
CLI already shows epic:/child: lines and has --epic. TUI has filter fields g (group) and e (epic) only. Code: huginn/internal/tui/render.go (overviewText, listText, taskRowMark), tui.go (paneIndices, move). task.ChildIndex resolves parents once; do not resolve per row (ls on 2,498 tasks went 19 s to 0.8 s that way). Decide with Jørgen via the coordinator: view by group vs by epic as a toggle, and whether later/ groups start folded. Goldens: HUGINN_UPDATE_GOLDEN=1 go test -run TestTmuxGolden ./huginn/internal/tui/

## Depends on

## Acceptance
- [x] list and overview show each task's group and epic without adding a line per task
- [x] a key toggles grouping by group, by epic, or flat; the header says which
- [x] Space on a group or epic heading folds and unfolds it, and the count stays visible when folded
- [ ] ~~tasks in group later are visibly set apart from NEXT UP~~ (not relevant: no magic group names; deferred is a stage (6ephqk), nautilus, 2026-11-17)
- [x] tests cover grouping, folding and the 80-column width; goldens reviewed

## Notes

2026-11-16 Jørgen on the overview: qawmnb (doing, with an unmet dependency) shows in IN PROGRESS, again in BLOCKED and again in LINT. A task must appear in one section only. Under discussion, coordinator's proposal: a task someone is working on stays in IN PROGRESS with its status drawn as blocked (red) and what it waits on; BLOCKED holds only tasks nobody has started that cannot start; LINT lists the findings (problem and fix), not the task rows again. Not decided yet.

2026-11-16 Jørgen answered 'I like your lint plan. Yes' right after the overview proposal; taken as approval of the severity plan (pbh9xq). The overview LINT section question (list findings there, or drop the section) is still open with him.

2026-11-16 Jørgen: the IN PROGRESS row already carries everything (the ! lint mark and the blocked status), but the overview has no legend for '!'. Add one, compact, no extra line: the list view footer already says '! lint finding · status blocked = unmet dependency', the overview footer does not, and the header only says '! 1 findings'. Coordinator's reading of the overview layout, stated to him: one row per task, in IN PROGRESS when taken (status drawn blocked, with what it waits on); BLOCKED only for tasks nobody started; the LINT section shows the finding text and fix, not the task row again.

vole 2026-11-17 question for Jørgen (also on tmp/lanes/morning.md). Proposal, building the first part now: (A) every row gets one short dim cell after the title's start, no extra line: 'later · ↳gem59b' (group, then the epic id with ↳); shown only when set, cut first when the row is tight (102bc0 order). (B) NEXT UP leaves out tasks in group 'later' (like hug task next should), and a folded heading 'LATER [3]' under NEXT UP holds them; Space on it unfolds. Questions: 1) Should 'later' be special by name, or should any group be markable as deferred (e.g. hug.toml deferred_groups = ["later"])? 2) The grouping toggle (check 2): one key, g, cycling flat → by group → by epic, header says 'grouped by epic'. g is free outside gg; or use z (vim's fold prefix)? 3) Do group headings start folded for 'later' only, or for all groups except the one holding the selection?

vole: check 1 built on branch wip/kx2ndm (3c2f123, on top of zhvkmj): row cell 'later·↳<epic>' before the title, dim, max 24 cells; m.epicOf built once per load from task.ChildIndex, rows never resolve parents. TestRowsShowGroupAndEpic: '--- PASS' (overview, list at 80 and 140, 30 lines, no ↳ on a task without epic). Not landed: it changes every tmux golden, and checks 2-4 (toggle key, folding, later apart from NEXT UP) wait on the three questions above. Not ticked.

Jørgen 2026-11-17: no magic group names. 'later' is not special; 'not now' is the todo stage deferred in 6ephqk. Groups are folders with no meaning.

Jørgen 2026-11-17 10:25 (ruling): the grouping toggle key is g (grouped); it cycles flat -> by group -> by epic. gg stays 'top'.

vole 10:55: built per the rulings so far, on ci/kx2ndm: row cell '<group>·↳<epic>' before the title (no special group names; any group shows the same way) and the overview footer legend '! lint finding · ↳ epic'. Open question for Jørgen on the g ruling: a single g is already the pending prefix of gg (top). Either (a) g toggles at once and a second g undoes it and goes to the top (the grouping flickers on every gg), or (b) g waits for the next key: g again = top, any other key = cycle grouping, then that key runs (the toggle shows one key late), or (c) the pending g shows its choices in place, like the header-tree idea in 0sw0dd: 'g: g top · r grouping'. Proposal: (c). Question 3 (what starts folded) is still open; with no magic names, the default would be nothing folded.

nautilus (was vole) 11:03, on hold until 6ephqk (crab, Jørgen's earlier ruling). Branch wip/kx2ndm, commit 12fa729, 3 commits on main f37a528: (1) 0cd9b7c row cell '<group>·↳<epic>' before the title, dim, max 24 cells, m.epicOf from task.ChildIndex once per load; (2) 42e1ebd overview footer legend '! lint finding · ↳ epic'; (3) 12fa729 goldens regenerated with HUG_UPDATE_GOLDEN=1 (the variable was renamed from HUGINN_UPDATE_GOLDEN on main), 25 files; suite 'ok github.com/veridit/hug/huginn/internal/tui 83.334s'. Tests: TestRowsShowGroupAndEpic, TestOverviewFooterLegend. Not on main; nothing ticked. When it comes back: it will need the 6ephqk row order (pie, id, type, priority, tags, stage, text), so the cell likely becomes the tags slot or sits beside it. Open for Jørgen (crab brings it): a single g is the pending prefix of gg (top); toggle at once with gg undoing it, toggle on the next key, or show choices in place 'g: g top · r grouping' (proposed). Also: with no magic group names, does anything start folded?


Jørgen 23:35 local chose the trie for prefixes (qh9ep1: 'view: ...', vo/vl/vb/v!). Read by crab as the g/gg answer: (c), a pending g shows its choices 'g: g top · r grouping'. Built with qh9ep1's prefix mechanism.
Jørgen 23:38 local: gg and G are dropped (qh9ep1), so the g/gg question is gone: g toggles grouping at once, flat -> by group -> by epic, no pending state.
Landed on main 1fda4f82..8d409e2d (a648520 rows show group and epic, fbb38da reload keeps the heading, c5e058e g toggles grouping and gg/G go, 8d409e2 tmux goldens). Evidence: 1: TestRowsShowGroupAndEpic (dim cell 'later·↳id' before the title, no extra line). 2: TestGroupingCyclesAndHeaderSays (g and ': grouping' cycle flat/group/epic; header '· grouped by X'). 3: TestSpaceFoldsAHeading (Space/Enter fold, '▸ later  [3]' keeps the count). 5: TestGroupedViewsFitAndGolden (testdata/group/views.txt at 80 and 140, reviewed), tmux goldens reviewed (group cell, '< > Home End', g in help). Gates: CI 44198908062 at 30a2b12 success on ubuntu (-race, tmux), macos, windows, bench, secrets; local -race tui ok 91.8s and go test ./... all-exit 0 (skip tmux, load ~660); after rebase onto f557613+scrxak local tui ok 53.6s. (nautilus)
landed a648520 on main 2026-11-18: hug tui: rows show group and epic; : grouping cycles flat, by group, by epic…
landed fbb38da on main 2026-11-18: hug tui: a reload keeps the heading under the cursor (kx2ndm)
landed c5e058e on main 2026-11-18: hug tui: g groups rows at once; gg and G go, top and bottom are < > Home End…
landed 8d409e2 on main 2026-11-18: hug tui: tmux goldens for grouping and g (kx2ndm)
Reflection (dismissed, 2026-11-18, nautilus): 'No magic group names' (Jørgen) guided well: it struck check 4 instead of special-casing 'later'. Gap: the ticket never named the key; the g/gg/G ruling took a park-and-wait round trip. The tmux load guard could not finish locally at load ~660, so the goldens came from CI GOT screens. (nautilus)
shipped in v0.7.51 (a648520)
## History
- 2026-11-17T21:59 todo -> doing/active (take)
- 2026-11-17T23:26 doing/active -> doing/waiting (wait)
- 2026-11-18T00:02 answered waiting: "the g key: Jørgen's g/gg ruling (a, b or c, crab asked 21:05); parked for e3sevr. Built and committed on nautilus/kx2ndm (2238d78, 079bc70): checks 1, 2 (as : grouping), 3, 5, not landed yet" -> "Jørgen 23:38 local: drop gg and G (never used, vi specific). So g is no prefix: g toggles grouping at once, flat -> by group -> by epic. Recorded by crab from his spoken answer.", by crab, for nautilus (doing/waiting -> doing/active)
- 2026-11-18T00:40 nautilus read the answer
- 2026-11-18T00:40 doing/active -> done (done); was doing_since: 2026-11-17T21:59
