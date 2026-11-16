---
id: et85h1
title: "Agents name themselves (and their model) when they take a task"
status: todo
type: bug
priority: "high"
assignee: []
labels: []
created: 2026-11-16
updated: 2026-11-16
---

## Goal
hug never assigns a task to a person by guessing. Whoever takes a task must state the model (claude-opus-5-5, gpt-6.1-sol, or human); the agent or session name (poodle, crab) is recorded when known but is optional, because agents are ephemeral and models are not (Jørgen 2026-11-16). Both are recorded on the task, so the board shows who is working and later tells us which model did which work.

## Purpose
Jørgen 2026-11-16: every doing task showed him as assignee because take fell back to git user.name while agents did the work. 'Let's not put my name automatically as the assigned user. Let's require the agent to tell us what agent or session is running this, and record what model if we know, because that tells us something about intelligence for later.'

## Principle
No git user.name fallback and no backwards compatibility. The model is required: from --model or HUG_MODEL; take without it fails with a fix: line. The name comes from --as, HUG_USER or the harness and is optional; with no name the assignee is the model. The harness may fill it in where it knows (jcode: JCODE_SESSION_ID session_<name>_...), but never from the OS or git user. The model is never guessed. MCP task_take gets the same fields. The setup/hook text tells agents to name themselves.

## Relevant details
huginn/internal/cli/cli.go:180 identity() reads HUGINN_USER, else git config user.name. take.go calls it; MCP take and the TUI callback share it. --mine uses the same identity. 2026-11-16 qawmnb and ksv323 were reassigned by hand to crab and poodle.

## Depends on

## Acceptance
- [ ] hug task take ID with no model fails: says no model was given and ends with fix: hug task take ID --model MODEL (or set HUG_MODEL); git user.name is never used (test)
- [ ] hug task take ID --model gpt-6.1-sol with no name records the model as assignee (test)
- [ ] hug task take ID --as poodle --model claude-opus-5-5 records assignee poodle and model claude-opus-5-5 on the task; HUG_USER/HUG_MODEL do the same (test)
- [ ] inside jcode with no flag, take uses the session name from JCODE_SESSION_ID (test)
- [ ] MCP task_take takes agent and model and records them; show, ls and the TUI show the model next to the assignee
- [ ] hug setup and the agent instructions tell agents to name themselves and their model on take


## Notes
2026-11-16 decided (Jørgen): a person taking a task gives --model human.
