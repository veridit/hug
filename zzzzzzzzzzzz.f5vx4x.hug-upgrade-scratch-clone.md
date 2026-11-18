---
id: f5vx4x
title: "hug upgrade in a scratch clone changed the machine's jcode configs"
state: todo
type: bug
priority: "high"
assignee: []
labels: []
created: 2026-11-18
updated: 2026-11-18
---

## Goal
hug upgrade says which changes are this board's and which are this machine's, and a run meant for one board never changes machine-wide files (jcode configs, the memory store) unless the person asked for those.

## Purpose
2026-11-18 04:53 local: kestrel ran 'hug upgrade --apply' in ~/.jcode/scratch/kestrel-w40cmk-clone (HUG_SYNC=off, to test the w40cmk layout move). Besides the clone's board it applied the z9q42h jcode hook change to all 6 of Jørgen's live jcode configs (~/.jcode/config.toml backup config.toml.muninn-bak-20261118-045310), a change crab had previewed to Jørgen at 03:19 local and was still waiting for his yes on. Rules: upgrades preview by default; nothing writes silently; the person approves machine changes.

## Principle
A command's reach must match where it is run: a repo's upgrade changes that repo; machine-wide changes are shown under their own heading ('this machine:') and need their own explicit step. No flag soup: propose the smallest shape to crab before building.

## Relevant details
Code: cmd/hug upgrade dispatch, muninn/run/upgrade.go (jcode hook migration), huginn task upgrade. Test sandboxes must also isolate HOME/JCODE_HOME/HUG_HOME (the bench and test harnesses).

## Depends on

## Acceptance
- [ ] In a test, hug upgrade --apply run in a scratch clone with a fake HOME holding jcode configs changes only the clone's board unless the machine step is asked for (fails first on main)
- [ ] The preview groups lines under 'this board:' and 'this machine:' (golden)

## Notes
