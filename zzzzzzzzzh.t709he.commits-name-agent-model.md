---
id: t709he
title: "Commits name the agent and model that made them (gate)"
status: todo
type: feature
priority: "high"
assignee: []
labels: []
created: 2026-11-16
updated: 2026-11-16
---

## Goal
Every commit in a hug repo, code and tasks, carries a Model: trailer naming the model that made it (Model: claude-opus-5-5), plus Agent: when an agent name is known. A commit without Model: is refused with a fix: line, so later we can find all work done by a model with a known deficiency and act on it

## Purpose
Jørgen 2026-11-16: 'knowing the model that did the work sometimes lets you take action if you know a model had certain deficiencies'. Today nothing records it: git author is the machine user for every agent

## Principle
Git trailers, so git log --format=%(trailers) and git log --grep find them. hug setup installs a commit-msg hook that refuses a commit without Model: (Agent: is optional: agents are ephemeral, models are not, Jørgen 2026-11-16); hug's own commits on hug/tasks add them from the same identity as take (et85h1). No guessing: the agent states them. Open: how a human commit passes (e.g. Agent: jhf, Model: human) - ask Jørgen.

## Relevant details
Depends on et85h1 for the identity source (HUG_USER, --as, JCODE_SESSION_ID; HUG_MODEL, --model). The hook must be cheap (under 50 ms) and work in linked worktrees (065wwn). From 2026-11-16 the coordinator asks workers to add the trailers by hand until this ships.

## Depends on
- et85h1

## Acceptance
- [ ] hug setup installs a commit-msg hook; git commit without a Model: trailer is refused with a fix: line showing it; Agent: is accepted but not required (test in a temp repo)
- [ ] commits hug makes on hug/tasks carry Model: (and Agent: when known) from the taker's identity (test)
- [ ] hug task log or a documented git log line lists commits by model

## Notes

2026-11-16 Jørgen on his own commits: 'maybe the model is me'. Proposed: a person writes their own name as the model (Model: jhf). Git author cannot tell, because agents commit under his git user too. Awaiting his yes.
