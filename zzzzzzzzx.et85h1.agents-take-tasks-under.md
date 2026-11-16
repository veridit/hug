---
id: et85h1
title: "Agents name themselves (and their model) when they take a task"
status: "doing"
type: bug
priority: "high"
assignee: ["dragon"]
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
- [x] hug task take ID with no model fails: says no model was given and ends with fix: hug task take ID --model MODEL (or set HUG_MODEL); git user.name is never used (test)
- [x] hug task take ID --model gpt-6.1-sol with no name records the model as assignee (test)
- [x] hug task take ID --as poodle --model claude-opus-5-5 records assignee poodle and model claude-opus-5-5 on the task; HUG_USER/HUG_MODEL do the same (test)
- [x] inside jcode with no flag, take uses the session name from JCODE_SESSION_ID (test)
- [ ] MCP task_take takes agent and model and records them; show, ls and the TUI show the model next to the assignee
- [x] hug setup and the agent instructions tell agents to name themselves and their model on take


## Notes
2026-11-16 decided (Jørgen): a person taking a task gives --model human.

check 1 (hare): /tmp/et85smoke repo, git user.name 'Git Person', HUG_USER/HUG_MODEL/JCODE_SESSION_ID unset: 'hug task take s834q1' printed 'no model given: say which model takes s834q1 (claude-opus-5-5, gpt-6.1-sol, ...; a person: --model human), with --model or HUG_MODEL' / 'fix: hug task take s834q1 --model MODEL', exit 2; file unchanged. Tests: TestTakeWithoutModelFailsWithFix (cli; also asserts 'Test' git user.name never appears and HUGINN_USER is not read), TestTakeAsModelRequiredAndRecorded (task), TestTakeRecordsAgentAndModel (mcp: fix 'hug task take ID --model MODEL').

check 2: 'hug task take 2knt1d --model gpt-6.1-sol' (no name) -> file 'assignee: ["gpt-6.1-sol"]' / 'model: "gpt-6.1-sol"'; ls row '2knt1d  doing    feature  smoke-two  Smoke two  [gpt-6.1-sol]'. Test TestTakeModelOnlyAssigneeIsModel.

check 3: TestTakeAsAndModelRecordsBoth/flags and /HUG_USER_and_HUG_MODEL: file 'assignee: ["poodle"]' + 'model: "claude-opus-5-5"'; show 'taken by: poodle · claude-opus-5-5'; ls '... Take me  [poodle · claude-opus-5-5]'; board --wide same; next --mine --as poodle finds it; take --as crab -> taken.

check 4: in this jcode session (JCODE_SESSION_ID=session_hare_...), 'hug task take s834q1' with no flags -> 'fix: hug task take s834q1 --as hare --model MODEL'; with '--model claude-opus-5-5' -> 'taken by: hare · claude-opus-5-5', file 'assignee: ["hare"]' 'model: "claude-opus-5-5"'. Test TestTakeUsesJcodeSessionName (session name; HUG_USER wins over it; no model still fails) and TestHarnessName.

check 5 partial: MCP task_take has 'model' and 'agent' (TestTakeRecordsAgentAndModel: assignee [poodle], model claude-opus-5-5 in reply, show and file); show/ls/board text and JSON carry the model. TUI display not done: huginn/internal/tui is poodle's lane; crab DM'd 14:05 to sequence (tui/detail.go:70 'assigned:' line).

check 6: hug setup agentSection and AGENTS.md: 'Start: hug task next, then hug task take ID --model MODEL (the model you are, e.g. claude-opus-5-5 or gpt-6.1-sol; a person: --model human; --as NAME adds your agent or session name, optional)'; 'task_take (model required, agent optional)'; setup intro 'hug task next  pick the top task, then hug task take ID --model MODEL'; MCP instructions 'take (with your model, and your agent name if you have one)'; hug task help take documents --model/--as. Full niced suite on HEAD 61d35c4 + my files: all ok, last 'ok  	github.com/veridit/hug/muninn/run	10.153s'.

2026-11-16 15:39 crab: check 5 (model in the TUI detail) is uncommitted in the tree from hare: huginn/internal/tui/detail.go (+8) and new detail_model_test.go. Hare's session ended at the 15:21 server reload while running the tui tests.

check 5 TUI (dragon): detail header 'assigned:poodle · claude-opus-5-5', model alone when no name (TestDetailHeaderShowsModel, 5 cases incl. model shown once). 'GOMAXPROCS=2 GOFLAGS=-p=2 nice -n 19 go test ./huginn/internal/tui/' -> 'ok  	github.com/veridit/hug/huginn/internal/tui	45.787s', no golden changes. Commit d3cc92f.
