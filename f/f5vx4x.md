---
id: f5vx4x
title: "hug upgrade in a scratch clone changed the machine's jcode configs"
state: "done"
type: bug
priority: "high"
assignee: ["kestrel"]
labels: []
group: ""
rank: "zzzzzzzzzzsj"
created: 2026-11-18
updated: 2026-11-18
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
done_at: "2026-11-18T05:26"
done_release: "v0.7.56"
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
- [x] In a test, hug upgrade --apply run in a scratch clone with a fake HOME holding jcode configs changes only the clone's board unless the machine step is asked for (fails first on main)
- [x] The preview groups lines under 'this board:' and 'this machine:' (golden)

## Notes

Shape approved by crab (04:04Z) before building:
- hug upgrade previews under the headings 'this board (PATH):' and 'this machine (~/.jcode, ~/.claude, HUG_HOME):'.
- hug upgrade --apply changes only the board. While machine work is pending it exits 5 (crab: 5 means 'upgrade has changes pending', so a script never reads a half upgrade as done) and ends with 'fix: hug upgrade --apply --machine'.
- hug upgrade --apply --machine does the board and then the variables and memory parts, each with its backup as today.
- --machine is valid only with --apply. hug mem upgrade is unchanged.
- Test, failing first: a clone plus a fake HOME/JCODE_HOME with an old-hook jcode config. After --apply the config is byte-identical; after --apply --machine it is migrated and the .bak exists.
- The two headings get a golden.
- Order: kestrel builds this after w40cmk check 1.


Landed a13d7c74 on main.
- Failing first on main 07183cf5: a copy of TestMachineFirstOnMain, a fake HOME with an old-hook jcode config plus 'hug upgrade --apply', fails with 'machinefirst_test.go:21: hug upgrade --apply changed this machine's config:'.
- Now TestUpgradeTouchesMachineOnlyWithMachine, in a clone with a fake HOME (.jcode/config.toml with an old variable name) and a recording MemUpgrade, checks four runs:
  - hug upgrade: exit 5 and prints
    'this board (BOARD):' / '  layout: nothing to do' / '  tasks: would: 1 questions to a person moved ...'
    'this machine (hook and MCP configs in ~/.jcode, ~/.claude, ~/.codex; the memory store):' / '  variables: would: variables renamed to HUG_* in CFG' / '  memory: would deliver ...'
    'next: hug upgrade --apply' / 'then: hug upgrade --apply --machine  (this machine, once you have read it)' / 'fix: hug upgrade --apply'.
  - hug upgrade --apply: moves the board, leaves the config byte-identical with no .bak and no 'mem upgrade --apply' call, and exits 5 ending 'next: hug upgrade --apply --machine' / 'fix: hug upgrade --apply --machine'.
  - hug upgrade --machine: exit 2 with 'fix: hug upgrade --apply --machine'.
  - hug upgrade --apply --machine: exit 0; the config is renamed, the .bak equals the old bytes, and the next plain upgrade exits 0 with no would:.
- Mutants: 7 of 7 killed (machineDry=dry, vars use dry, mem uses dry, no exit 5 after apply, no then line, no headings, --machine alone allowed).
- Two existing tests are rewritten, not deleted: TestUpgradeRenamesEnvInHugConfigs uses --apply --machine; TestUpgradeMemoryPartRunsMemUpgrade asserts that --apply alone leaves the memory part pending with fix --apply --machine.
- Suites -short: all 12 ok. Race: cli 115.3s ok, cmd/hug 19.2s ok.
landed a13d7c7 on main 2026-11-18: hug upgrade --apply changes only this board; this machine's files need --apply…
Reflection (dismissed, 2026-11-18, kestrel): A command's reach has to match where it runs. hug upgrade mixed board steps and machine steps under one --apply, so testing one board touched every repo's hooks. Check 2's 'golden' is verbatim heading blocks asserted in TestUpgradeTouchesMachineOnlyWithMachine, not a golden file: the output holds temp paths.
landed 50747ea on main 2026-11-18: test: upgrade preview's board heading compared as the same folder, not the same…
shipped in v0.7.56 (a13d7c7)
Reflection 1 dismissed (tb0dra check 5 triage, crab 2026-11-19): G10, a lesson about one's own method, fixed in its own ticket; no instruction misled.
## History
- 2026-11-18T05:18 todo -> doing/active (take)
- 2026-11-18T05:26 doing/active -> done (done); was doing_since: 2026-11-18T05:18
