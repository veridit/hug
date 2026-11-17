---
id: t3fqaw
title: "set --title leaves the old multi-line title's continuation lines"
status: "doing"
type: bug
priority: "high"
assignee: ["kestrel"]
labels: []
created: 2026-11-17
updated: 2026-11-17
doing_since: "2026-11-17T18:56"
stage: "active"
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
---

## Goal
hug task set ID --title replaces the whole title value, including a folded or multi-line YAML title with indented continuation lines, so the front matter stays valid.

## Purpose
Seid foreman (orangutan) requests 2026-11-17, file: seid-lang .jcode/scratch/TASK-2057/hug-agent-requests.md, read by crab 15:07, item 5. A Seid worker shortening titles left the old continuation lines, giving 24 frontmatter, unknown-status and 72 missing-field findings until removed by hand. Rule: fixes never need an editor.

## Principle
Edit the field, not the line: any front-matter write replaces the whole value. A test with an imported multi-line title fails first.

## Relevant details
Imported backlog titles can be multi-line. Check every field writer (set, take, wait, design), not only title.

## Depends on

## Acceptance
- [ ] a test sets --title on a task whose title spans 3 lines: the file has one title line and lints clean; it fails on main
- [ ] the same for every other front-matter field a command writes (test)

## Notes


Ready on branch kestrel/t3fqaw, commit 9d42d03, not pushed: main is held from 18:05Z for nautilus's rename (crab 17:56); it will be rebased onto that. Cause: frontmatter.go trimmed '#' lines back from the next key as YAML comments, so an indented block-scalar line starting with '#' stayed. Real case: Seid t6j13s, dc0e38964 (2026-11-16 21:22, v0.7.35) left '  #return/#break clauses (test_handler_return.sd)' under the new title; reproduced on main with that exact file. Tests fail first on main: TestSetTitleReplacesFoldedTitle 'continuation line left: "  #return/#break clauses (test_handler_return.sd)"'; TestEveryFieldWriteReplacesMultiLineValue 6 of 7 writers (type, priority, assignee, labels, waiting_on, design_question) leave '"  #second line"' '"  #third line"'. -race: task ok 60.738s, cli ok 138.441s, importer ok 2.849s.
## History
- 2026-11-17T18:56 todo -> doing/active (take)
