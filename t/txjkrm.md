---
id: txjkrm
title: "Tags: listed per board in hug.toml, shown on rows, Seid's labels cleaned up"
state: "done"
type: feature
priority: ""
assignee: ["crab"]
labels: []
group: ""
rank: "zzzzzzzzzz38"
created: 2026-11-17
updated: 2026-11-17
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
done_at: "2026-11-20T10:55"
---

## Goal
Each board lists its tags in hug.toml, rows show them, lint holds tasks to the list and hug upgrade cleans old labels.

## Purpose
Jørgen 2026-11-17 10:24-10:27: backlog's board on Seid shows [Compiler][Parser] on each row and hug's does not, so the area is lost at a glance. State and stage are hard-coded because they carry semantics; tags are the per-project freedom to organise and see where work belongs, and listing them gives the consistency people and AI need. Measured on Seid (2,516 tasks): 891 distinct labels, 420 used once, variants like perf/performance, backend-c/c-backend, compiler/Compiler, test/tests/testing, and labels that are not tags (bug 467 = type, design 244 = stage, task-632 = a reference, low-pri = priority).

## Principle
Tags carry no meaning to hug: display, filter, group only. The list in hug.toml is the truth; a tag not on it is a lint finding with the fix (add it to the list, or hug task set ID --tag). No backward compatibility: the field is tags, hug upgrade renames labels. The cleanup never writes before Jørgen approves the diff. Subtle on rows; tags are cut after the title, before the owner (102bc0 order).

## Relevant details
Labels become tags: each board lists its tags in hug.toml, every task's tags show dimmed on its row before the title (compiler parser  Compile-time bounds…), lint holds every task to the list, and hug upgrade cleans an existing board (merges spelling variants, moves labels that duplicate type, stage or priority into those fields, drops one-off tags) after showing the diff.

Data: Seid label counts in /Users/jhf/.jcode/scratch/seid-labels.txt (regenerate with the grep in the notes). Seid board: ~/Veridit/src/seid-lang/.hug/tasks. Related: 6ephqk (state and stage, which absorbs design/king-ruled), kx2ndm (g groups; by tag is a candidate view), 102bc0 (row layout). Preview first: a mock of the overview with tags, from real Seid tasks, for Jørgen's look before building.

## Depends on

## Acceptance
- [x] a preview of the Seid overview with tags on rows, built from real tasks, is shown to Jørgen and approved before the TUI work starts
- [x] hug.toml tags = [...] is read; lint flags a task tag not on the list with a fix: line (test)
- [x] rows show the task's tags dimmed before the title and cut by 102bc0's order (goldens at 80 and 140 columns)
- [x] filter tag:compiler and g grouping by tag work (test)
- [x] hug upgrade --tags on Seid proposes the merges, field moves and drops as a diff with counts, writes nothing until approved; after approval lint is clean of tag findings
- [x] Jørgen 19:35Z: tags sit right after the task ID (pie, id, tags, type, priority, stage, text), dim, at most 22 columns; this supersedes 'before the title' in check 3 (goldens at 80 and 140)
- [ ] ~~Jørgen confirms Seid's overview with tags right after the ID reads right (file seid-overview-tags-row.txt)~~ (not relevant: superseded by Jørgen's 911j70 answer 1 (2026-11-20 08:26Z): tags become a prefix of the title, not a slot right after the ID; built and confirmed under 911j70 check 5, crab, 2026-11-20)

## Questions
- 2026-11-18T23:38 question 1 by kikazaru to jhf (check 7, show seid-overview-tags-row.txt):
  Seid's overview with tags right after the ID, as you asked: does it read right?
  answered 2026-11-20T10:55 by jhf:
    I think this has been replaced by my next ruling that the tags should be at the start of the title and I think we just have to look at the design how we can visually see the tags when they're beginning the title such as by and I wonder if they should be a separate column or it should be tag and title with respect to sorting I think that's the only question but a visual I think showing them at the beginning with some kind of very subtle separator makes it very easy to see what this is about and then the details in the title. As such, I think this task should close and there will be other follow-up tasks to adjust this further, and that the question was asked before my new changes landed

## Notes

Jørgen 2026-11-17 10:28 (row layout): the state is the section header, so rows drop the status column. Row order: pie, id, type, priority, tags, stage, text. A stage with extra data shows 'stage: reason' (waiting: CI run N · 12 min; depends: 44w1ev (doing)) instead of the title; a stage without it shows the title. Mock in tmp/lanes/txjkrm-preview.md (Revision). Open: reason replaces the title fully, or reason then dim title.

Jørgen 2026-11-17 10:56, settles the open question: the reason replaces the title fully. The colon is the marker: 'waiting: CI run N' means the text is the stage's reason, a bare 'active' means the text is the title. No dim title after the reason.

file:txjkrm-preview.md (4770 bytes, sha256:ab7fc3a2) design, moved here from tmp/lanes (git-ignored, so other clones never saw it)

mizaru 14:45Z: reassigned from nautilus (session gone, crab 14:44Z said take it); take refused 'taken by nautilus (txjkrm)' with only 'fix: hug task next', so used the stale-owner fix hug task set txjkrm --assignee mizaru. Scope from crab: check 1 only (preview + question to jhf); no TUI code until approved.


file:seid-overview-tags.ans (3236 bytes, sha256:477072ab) check 1 preview: Seid's overview (ASKED, WAITING, first 12 of NEXT UP) with tags on rows, row order pie, id, type, priority, tags, stage, text (Jørgen 10:28/10:56). Built read-only from ~/Veridit/src/seid-lang/.hug/tasks at 2eef4a7f2 with hug's own Store.Tasks and Stage; tags after the proposed cleanup in seid-tags-proposal.txt
file:seid-tags-proposal.txt (4983 bytes, sha256:d1819ff2) check 1: proposed tag list for seid-lang: 38 tags kept (20+ tasks after merging, with what each merges), 14 labels moved to type/stage/priority, 133 dropped as provenance/rulings/references, 656 dropped as rare
Check 2 at c22587c6 (branch txjkrm on main 62c20712), per crab 14:52 rules. hug.toml top-level tags = [...] (multi-line ok) plus optional [tag_merges] OLD = "LISTED"; lint unlisted-tag (advice, open tasks, no list = not judged). Variant (case, -/_, stated merge) -> one fix 'hug task set ID --tag OLD=LISTED'; other -> 'hug task tags add NAME' or 'hug task set ID --tag NAME='. New: hug task tags [add NAME]; set --tag NAME|OLD=NEW|OLD= (idempotent, repeatable); MCP set tag; Finding.Fixes (alternatives, printed as 'or:' lines). Field stays labels; --label unchanged. Failing-first ($JCODE_SCRATCH_DIR/txjkrm-preview/check2-ff2.txt): task tags_test.go:18 undefined: Tags; cli tags_test.go:24 unknown command tags; mcp tags_test.go:32 unknown command tags. Mutant (variants by exact key only) fails task tags_test.go:61 and cli tags_test.go:39. Full suite niced -count=1: 34 ok, 0 FAIL. CI 44332113664 queued.
crab review 16:09 local: check 2 accepted. c22587c6 on 62c20712: task/tags.go (tags list + [tag_merges], TagFindings unlisted-tag Advice, open tasks, no list = not judged), set --tag NAME|OLD=NEW|OLD= idempotent, hug task tags [add], MCP set tag; Finding.fixes extra 'or:' lines, Fix first; --label/labels unchanged; no schema bump. 3 failing-first, 1 mutant. Lands after CI 44332113664 (exact SHA after rebase). Checks 3-5 wait on Jørgen's check 1.
Check 4 at 1506435b (on c22587c6). TUI filter 'label' renamed 'tag' (no alias; picker letter stays l): ': tag compiler' keeps tasks tagged compiler, chip tag:compiler, unknown tag refused. g cycles flat -> group -> epic -> tag; by tag each task sits under ONE heading: its first tag in hug.toml list order, else its own first tag, else 'no tag' (crab 15:29 ruling). Tests huginn/internal/tui/tags_test.go TestTagFilter + TestGroupingByTag failed first (check4-ff.txt: tags_test.go:58 tag compiler: Lex,Both,Codegen,Odd,Bare). Mutants (check4-mutants.txt): list order ignored -> tags_test.go:95 lacks '▾ parser  [2]'; no own-tag fallback -> lacks '▾ zzz  [1]'; filter ignored -> tags_test.go:58. Goldens: text only, 6 files 21+/21- (pickers 'l tag one tag', help 'tag <t>' and the g line). Local: build-ok, vet-ok, ok tui 130.750s. CI 44338943159 queued. Question for Jørgen with check 1: the tag picker letter stays l (t is type).
crab review 16:45 local: check 4 accepted. 1506435b on c22587c6: TUI label filter -> tag (no alias, letter l, one-letter fields resolve picker letters first), g cycles flat/group/epic/tag (first listed tag), tag order loaded with the snapshot. Failing-first + 3 mutants; goldens text-only 6 files 21+/21-. Lands after the current stack. Open for Jørgen with check 1: tag picker letter l (t is type), keep or swap.
Question for Jørgen (with check 1): the TUI tag filter's picker letter is l (f l, then the tag) because t is type. Keep l, or another letter?
crab 18:35 local: stacked as 0a1fdbe4 + 5924cfdb on ybbfsa b2232191 (ci/txjkrm-ec6z6j-rebased c7ccbca7), code identical to mizaru's f85ac6ec, build+vet ok. Includes the pw0t19 wrong-count fix in tags.go.
landed on main 8c26ec51 (CI 44363087479 6/6, windows-cli 549.357s)
landed 709e327 on main 2026-11-18: Tags listed per board: hug.toml tags, unlisted-tag lint, hug task tags, set…
landed dfe1aee on main 2026-11-18: TUI: tag filter (was label) and g grouping by tag, first listed tag (txjkrm…
check 5 (mizaru, branch txjkrm5 on 8c26ec51, uncommitted pending suite): hug upgrade --tags built to crab's rulings. Failing first: /Users/jhf/.jcode/scratch/txjkrm-preview/check5-ff.txt (exit 2, unknown flag). Tests TestUpgradeTagsPreviewThenApply + TestUpgradeTagsProposesAList pass; 13 mutants all killed (/Users/jhf/.jcode/scratch/txjkrm5-mut/mutants-round1.txt, mutants-round2.txt; overlay, no source edits). Seid copy (/Users/jhf/.jcode/scratch/txjkrm5-seid, proposal's 38 tags + design, 50 merges): lint before 1754 unlisted-tag; preview exit 5, 1451 tasks, 21 deferred, 196 design kept, 6 draft, 0 open disagreements, 1005 closed tasks merges-only; apply = one commit a29158c2b 'hug upgrade --tags: labels cleaned in 1451 tasks'; lint after: no findings with code unlisted-tag; second run: tags: nothing to do. Preview/apply output: /Users/jhf/.jcode/scratch/txjkrm-preview/seid-preview.txt, seid-apply.txt.
shipped in v0.7.68 (709e327)
check 5 committed: 782a807c on origin/main 6a6b6573 (v0.7.68), branches txjkrm5 + ci/txjkrm5. CI 44376607765 completed success 6/6: secrets, bench, test ubuntu/macos/windows, test-windows-cli (19:28:14Z). Local: task + mcp ok; TestUpgradeTags* ok; full cli suite did not finish locally (load average ~677: WaitDelay/git timeouts in sync tests, untouched by this change); CI is the clean run.
check 1: Jørgen 19:35Z (answer 1): the list and preview are approved with one change. Tags go right after the ticket ID, because they say quickly what the task is about. See check 6.
crab 20:36 local: Jørgen also floated auto-grouping tasks that share a tag; 'discuss later', so not in scope here (g grouping by tag already exists, check 4). txjkrm5 (check 5) 782a807c is rebased as 67569a0d on 477696be for landing.
landed 4dcbf91 on main 2026-11-18: hug upgrade --tags: labels cleaned against hug.toml tags, previewed, one commit…
shipped in v0.7.69 (4dcbf91)
file:seid-overview-tags-row.txt (10814 bytes, sha256:5fcea285) check 6 for Jørgen: Seid's overview at 80x24 and 140x40, before (origin/main 8bf24c02) and after (kikazaru/txjkrm6 54f31124), from the scratch copy txjkrm5-seid after hug upgrade --tags --apply; plain text
file:after-140x40.ans (9165 bytes, sha256:ec918981) check 6: the after capture at 140x40 with colour (cat it in a terminal): tags dim right after the id
kikazaru checks 3+6: 4d2da9f1 + 54f31124 (gofmt) on kikazaru/txjkrm6 / ci/txjkrm6, on origin/main 8bf24c02. Rows read pie, id, TAGS, type, priority, stage, text (Jørgen 19:35Z). Tags dim; listed tags first in hug.toml order, then unlisted in the task's order; one cell wide per board = widest tag text, at most 22, at least 4 (TAGS). Squeeze: tags cut first down to 8, drop last (after stage); a board column shows tags only while its title keeps 20 (so the board goldens are unchanged). TAGS is a column-cursor stop between ID and TYPE; o orders by the shown tag text, untagged last. Board without tags: no cell, rows as before (TestNoTagsNoCell). macaque OK'd (gutter-only, no overlap). Failing first ($JCODE_SCRATCH_DIR/txjkrm6-ff.txt): tagrow_test.go:67 header not ID TAGS TYPE…; :104 at 80 the tags do not follow the id; :133 the third column is title, want tags. Mutants: tagMax 99 -> tagrow_test.go:90 + golden tags-140; no listed-order sort -> :77, :120, :158. Goldens: new testdata/rows/tags-80.txt, tags-140.txt; updated text-only 26 files (cursor.txt, view-pending.txt, tmux 80x24/140x40 color+no-color chips/grab/list/overview/sort): each row gains the TAGS cell after the id; columns_test L counts +1 (TAGS stop). Local: go vet ok; full tui suite non-short 'ok github.com/veridit/hug/huginn/internal/tui 123.585s'. Seid captures attached (seid-overview-tags-row.txt, after-140x40.ans). CI 44396528366 pending.
kikazaru: CI 44396528366 on 54f31124 completed success 8/8 (secrets, bench, test ubuntu/macos/windows, test-windows-cli sync/rest/shards). Checks 3 and 6 ticked; Seid before/after captures attached for Jørgen's look.
mizaru 21:13Z: cleared the old wait (resume). It read 'check 1: Jørgen sees the Seid overview preview; check 3: rows show tags' and would become a wait-on-person finding under ebben8. Both are settled now: check 1 ticked 20:36 local (Jørgen's 19:35Z answer), checks 3 and 6 ticked 22:08 local after kikazaru's goldens (CI 44396528366). So there is nothing left to ask Jørgen. Every check holds: done is crab's.
landed 3ce9119 on main 2026-11-18: TUI: tags right after the id on every row (txjkrm checks 3, 6)
landed 3f5e89d on main 2026-11-18: gofmt tagrow_test.go (txjkrm)
landed 4d7eec2 on main 2026-11-18: tui: tmux palette goldens with tags column (txjkrm x 05r3pg)
shipped in v0.7.71 (3ce9119)
kikazaru 22:37Z: shipped in v0.7.71 (3ce9119a, 3f5e89d9, 4d7eec2d). The owed question for Jørgen ('Seid's overview with tags right after the ID, as you asked: does it read right?', --show seid-overview-tags-row.txt) was not asked: hug task ask refuses a done task ('done -> doing/asked is not a move a command makes'); reopening is crab's call.
Reflection (open, 2026-11-20, crab): kikazaru's question was overtaken by Jørgen's 911j70 ruling; he answered it and asked to close.
## History
- 2026-11-18T15:45 todo -> doing/active (take)
- 2026-11-18T15:49 doing/active -> doing/asked (ask)
- 2026-11-18T20:33 answered asked (check 1), by jhf, for mizaru (doing/asked -> doing/active):
  > Seid's real overview with tags on rows, in your 10:28 row order: pie, id, type, priority, tags, stage, text. Tags are dim, at most 22 columns. The proposed list keeps 38 tags used on 20+ tasks after merging (file seid-tags-proposal.txt: compiler 924, parser 341, and so on), moves bug/design/paused/low-pri to type, stage and priority, and drops rulings, provenance and rare labels. Does this look right, so I can build it? Ticking check 1 is yes; or answer with what to change (cutoff, a tag, the column).
  So I think the tags should be after the ticket ID because they allow you to quickly understand what this is all about and I think we can discuss later if we can make groupings for things that have the same tab somehow like an auto group on that
- 2026-11-18T20:33 answer 1 delivered to crab (session_crab_1793783281662_44ffaee927a8a9d5)
- 2026-11-18T20:35 answer 1 acknowledged by crab
- 2026-11-18T20:36 answer 1 in progress by crab: hug task check txjkrm 1 "Jørgen 19:35Z (answer 1): the list and preview are approved with one change. Tags go right after the ticket ID, because…"
- 2026-11-18T20:44 doing/active -> doing/waiting (wait)
- 2026-11-18T20:44 mizaru read the answer: hug task wait txjkrm "check 1: Jørgen sees the Seid overview preview; check 3: rows show tags (not started, mizaru moved to q0a8rx by crab 19…"
- 2026-11-18T22:13 doing/waiting -> doing/active (resume); was waiting_asked_by: mizaru; waiting_on: check 1: Jørgen sees the Seid overview preview; check 3: rows show tags (not started, mizaru moved to q0a8rx by crab 19:44Z); waiting_since: 2026-11-18T20:44
- 2026-11-18T22:21 doing/active -> doing/waiting (wait)
- 2026-11-18T23:24 doing/waiting -> done (done, by release v0.7.71: every check holds); was doing_since: 2026-11-18T15:45; waiting_asked_by: kikazaru; waiting_on: crab landing ci/txjkrm6-rebased 3f5e89d9 and a release; then ask Jørgen the check-6 question with --show (crab 21:09); waiting_since: 2026-11-18T22:21
- 2026-11-18T23:38 done -> todo/ready (reopen); was done_at: 2026-11-18T23:24; done_release: v0.7.71
- 2026-11-18T23:38 todo/ready -> todo/design (ask)
- 2026-11-20T09:28 todo/design -> doing/asked (take with 911j70, 1 open question(s) kept); was design_asked_by: kikazaru; design_for: jhf; design_question: Seid's overview with tags right after the ID, as you asked: does it read right?; design_show: seid-overview-tags-row.txt; design_since: 2026-11-18T23:38
- 2026-11-20T10:55 answered asked (check 7), by jhf, for kikazaru (doing/asked -> doing/active):
  > Seid's overview with tags right after the ID, as you asked: does it read right?
  I think this has been replaced by my next ruling that the tags should be at the start of the title and I think we just have to look at the design how we can visually see the tags when they're beginning the title such as by and I wonder if they should be a separate column or it should be tag and title with respect to sorting I think that's the only question but a visual I think showing them at the beginning with some kind of very subtle separator makes it very easy to see what this is about and then the details in the title. As such, I think this task should close and there will be other follow-up tasks to adjust this further, and that the question was asked before my new changes landed
- 2026-11-20T10:55 answer 2 delivered to crab (session_crab_1793783281662_44ffaee927a8a9d5)
- 2026-11-20T10:55 doing/active -> done (done); was doing_since: 2026-11-20T09:28; doing_with: 911j70
- 2026-11-20T10:55 answer 2 acknowledged by crab
