---
id: t3fqaw
title: "set --title leaves the old multi-line title's continuation lines"
status: todo
type: bug
priority: "high"
assignee: []
labels: []
created: 2026-11-17
updated: 2026-11-17
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
