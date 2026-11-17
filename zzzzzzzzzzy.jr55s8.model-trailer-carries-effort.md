---
id: jr55s8
title: "Model trailer carries the effort: Model: id/effort, checked per model"
status: "doing"
type: feature
priority: "high"
assignee: ["wyvern"]
labels: []
created: 2026-11-17
updated: 2026-11-17
doing_since: "2026-11-17T18:16"
stage: "waiting"
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
waiting_on: "after 46ek1f (crab 17:52)"
waiting_since: "2026-11-17T18:53"
waiting_asked_by: "wyvern"
---

## Goal
hug task install commit-msg accepts exactly one 'Model: <model-id>/<effort>' trailer (or 'Model: human'), and checks the effort against that model's own levels listed in hug.toml ([models] 'claude-opus-5-5' = ['low','medium','high','max']). hug task take --model takes the same id/effort form and records both. Unknown model or effort: refused with a fix: naming the hug.toml line to add.

## Purpose
Seid foreman (orangutan) requests 2026-11-17, file: seid-lang .jcode/scratch/TASK-2057/hug-agent-requests.md, read by crab 15:07, item 2. Jørgen ruling 2026-11-17 in Seid: one trailer line Model: <model-id>/<effort>; effort can change a model's behaviour entirely (low effort: double-check the grounding).

## Principle
Declarative roster in hug.toml, no model list in code. '/' never appears in model ids, so the split is unambiguous. hug records what the agent declares; the docs say: declare what is actually running (a swarm may ignore the requested model).

## Relevant details
Current gate: hug task install commit-msg (t709he) puts a hook in <git-common-dir>/hooks and keeps an existing hook as commit-msg.pre-hug. Seid has a tracked hook .githooks/commit-msg with roster .githooks/known-models.txt requiring Co-Authored-By, being retired; Seid wants the gate as two lines in its tracked hook (hug task install commit-msg --print, or a documented 'hug task hook commit-msg FILE' call). hug's own AGENTS.md trailer 'Model: claude-opus-5-5' changes to id/effort too.

## Depends on

## Acceptance
- [ ] the hook accepts Model: claude-opus-5-5/high and Model: human, refuses a missing effort, an effort not listed for that model, and two Model lines, each with a fix: line (test)
- [ ] hug task take --model id/effort records model and effort, and the row/detail show both (test)
- [ ] a tracked hook can call the gate in two lines; the help shows them
- [ ] Seid's tracked hook switched, in a commit Jørgen approves

## Notes


## History
- 2026-11-17T18:16 todo -> doing/active (take)
- 2026-11-17T18:53 doing/active -> doing/waiting (wait)
