---
id: 5mmqjk
title: "hug completion: one script for task, mem and the top level"
state: "doing"
type: feature
priority: ""
assignee: ["mizaru"]
labels: []
group: ""
rank: "zzzzzzzzzzzo"
created: 2026-11-18
updated: 2026-11-18
parent: "gem59b"
doing_since: "2026-11-18T23:58"
doing_with: "p277bp"
stage: "active"
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
---

## Goal
hug completion bash|zsh|fish|powershell prints one completion script for the hug command: top-level commands (task, mem, tui, mcp, apps, setup, upgrade, version), hug task's commands and flags, and hug mem's cobra commands and flags.

## Purpose
p277bp check 2 removed hug mem completion (crab 22:21, option A): it completed the old command name muninn. hug has no completion of its own.

## Principle
One script for the one command users type (hug). Scripts already installed for muninn keep working through the hidden __complete command. No completion per part.

## Relevant details
cobra's completion scripts take the program name from c.Root().Name() (cobra v1.10.2 completions.go, lines 774/824/861/889/911) and call '<name> __complete ...'; muninn's root is Use: "muninn" (muninn/run/main.go:60). cmd/hug/main.go dispatches an unknown first word (such as __complete) to huginn.Main, so 'hug __complete mem ...' reaches huginn, not muninn. hug task (huginn) is not cobra: it has its own Commands list (huginn/internal/cli/cli.go:61) and commandHelp usage map, so a hug script must combine both.

## Depends on

## Acceptance
- [x] hug completion zsh output, sourced in zsh, completes 'hug ta<Tab>' to task, 'hug task ta<Tab>' to take, 'hug mem se<Tab>' to search (test drives __complete)
- [x] muninn __complete still answers for scripts installed before (test)
- [x] README Install names the completion line for each shell

## Notes


Built at 18cfcbde (branch 5mmqjk, CI on ci/5mmqjk). hug completion bash|zsh|fish prints cobra v1.10.2's script for a root named hug; hug __complete dispatch: first word from cmd/hug topCommands; mem|memory|muninn words -> muninn.Main(__complete ...); task|tasks|huginn -> cli.Complete (Commands, allowedFlags hoisted from Execute unchanged, --type/--state values, open ids with titles where the usage line says ID, FILE/DIRECTORY -> file completion); the huginn link completes hug task, the muninn link stays hug mem. Ids come from cache.Peek: walkMarkers + cache rows read through an immutable SQLite open (no lock, no -wal/-shm write); a file whose size+mtime match its row outside the racy window takes id/title/state/rank from the row, any other file is parsed. Tests: cmd/hug TestCompletionInEachShell (real bash 5.3 / zsh / fish drive the generated scripts; fish skips locally, CI ubuntu installs zsh+fish and sets HUG_REQUIRE_SHELLS=1), TestCompleteDispatch, cli TestCompleteTaskTree, TestCompleteNeverWrites (broken remote, stale cache: answer is the file's new title, no file under the repo or .git changed), cache TestPeekUsesFreshRecords. Failed first: 'hug: unknown command "completion"'; the first never-writes run caught cache-28.db-wal and -shm changed by a mode=ro open, hence immutable=1. Mutants 15 run, 14 killed + m3 killed in its home package: /Users/jhf/.jcode/scratch/5mmqjk-mut/results.txt. Seid copy (3156 files, 2537 tasks, warm cache), 15 runs, load avg ~500: __complete task show '' wall median 159-200 ms, min 95-120 ms; process CPU median 76 ms (version alone 26 ms CPU); in-process Peek 23-43 ms (rows 8-16 ms warm, walk 8 ms, 2537/2537 rows fresh). task ls --json for comparison: wall median 547-589 ms, CPU 294 ms.
CI 44432165025 on 8610a01a (rebased on main bf170d46): 8/8 jobs success. ubuntu installs zsh 5.9 + fish 4.2.1 (log: 'Setting up fish (4.2.1-3.2)') and runs with HUG_REQUIRE_SHELLS=1, so a missing shell fails, never skips; run 44430041895 had failed on fish itself ('fish "hug task sh": exit status 1': string replace exits 1 when no line has a description), fixed in the test driver. apt step: 12 s on run 44430041895 (00:37:52-00:38:04), 119 s on run 44432165025 (00:51:37-00:53:36: 'Fetched 13.1 MB in 1min 41s (130 kB/s)', mirror speed).
check 1: cmd/hug TestCompletionInEachShell drives the real scripts: zsh sources 'hug completion zsh' and runs _hug (stand-in _describe prints the offers): 'hug ta'->task (not tui), 'hug task ta'->take, 'hug mem se'->search, plus 'hug task sh'->show, 'hug mem rec'->recall, 'hug task take --mo'->--model, 'hug task show <2 chars>'->the id, 'hug task ls --state d'->doing; same lines in bash 5 and fish. Passed in CI 44432165025 (ubuntu with HUG_REQUIRE_SHELLS=1) at 8610a01a; failed first with 'hug: unknown command "completion"'.
check 2: cmd/hug TestCompletionGoneInstalledScriptsAnswer (p277bp) still passes, and TestCompleteDispatch adds: run("muninn", __complete se) prints 'search\t...' and :4; the muninn-link case is handled first in run(), before hug's own __complete. Mutant m11-style removal of the link cases is killed (results /Users/jhf/.jcode/scratch/5mmqjk-mut/results.txt). CI 44432165025 8/8 at 8610a01a.
check 3: README.md Install, after the setup block: 'Tab completion for hug, hug task (commands, flags, open task ids) and hug mem:' then 'source <(hug completion zsh)  # in ~/.zshrc; bash 4.4+: source <(hug completion bash) in ~/.bashrc' and 'hug completion fish > ~/.config/fish/completions/hug.fish'. hug help and hug completion --help give the same lines. Commit 8610a01a.
Correction to check 2's evidence: the muninn-link mutant is m14 (m11 removed the huginn link). m14 killed: completion_test.go:187 'muninn ["__complete" "se"]: exit 0' and oldnames_test.go:121 'muninn __complete se: exit 0'. Mutants now 16, all killed (m3 in its home package cache). /Users/jhf/.jcode/scratch/5mmqjk-mut/results.txt
## History
- 2026-11-18T23:58 todo -> doing/active (take with p277bp)
