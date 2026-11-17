# ggydj2: hug task import backlog .backlog --cleanup --dry-run on a scratch clone of seid-lang

Made 2026-11-17 16:38 by kestrel, binary from kestrel/ggydj2. The clone (no remote, core.hooksPath=.githooks like Seid) first had 'hug task import files .backlog/evidence --cite .backlog/evidence' run, because without it the dry run refuses: 'not retiring .backlog: the import is not complete: .backlog/evidence: 4738 files to copy and 2766 citations to rewrite'. Seid itself was not changed.

```
would git rm -r .backlog: 7658 files (git history keeps them)
would commit that as one commit, then record the import in hug.toml on the task branch
61 documents cite .backlog/ paths; those links stay readable in git history
the agent: lines below still use backlog or read .backlog/: hug does not edit them; change them first, or a hook or script may fail once .backlog/ is gone
agent: remove from .githooks/pre-commit:40: if ! python3 "$REPO_ROOT/tools/backlog_integrity.py" --cached ||
agent: remove from .githooks/pre-commit:41: ! python3 "$REPO_ROOT/tools/backlog_integrity.py"; then
agent: remove from .githooks/pre-commit:45: Run: python3 tools/backlog_integrity.py
agent: remove from .claude/hooks/remind-backlog-on-pick.sh:31: {"additionalContext": "PICK/PUSH detected — backlog ritual (King directive 2026-07-26): task closure joins the pick ritual. In the same motion as this landing: update or complete the backlog task it advances (status, comment with the commit hash, finalSummary if done). Statuses lie when landings don't close tasks — the 2026-07-26 audit found 6 tasks In Progress weeks after their work landed."}
agent: remove from .claude/hooks/test-check-syntax-drift.sh:149: "$(payload_edit "$REPO_ROOT/.backlog/tasks/task-297.md" 'support Eq for Color')"
agent: remove from .claude/settings.json:140: "command": "/Users/jhf/Veridit/src/seid-lang/.claude/hooks/remind-backlog-on-pick.sh",
agent: remove from compiler/bin/seid.ml:10074: if starts_with file ".backlog/tasks/"
agent: remove from compiler/bin/seid.ml:10075: || starts_with file ".backlog/completed/"
agent: remove from compiler/bin/seid.ml:10076: || starts_with file ".backlog/archive/tasks/" then "backlog"
agent: remove from compiler/bin/seid.ml: 2 more lines that read .backlog/ (git grep -n '.backlog' -- compiler/bin/seid.ml)
agent: remove from compiler/test/check/tuple_ord_raw_binop_fallback_error.sd:8: -- The doc-004 tuple-canon ruling (.backlog/docs/design/doc-004) is the source:
agent: remove from compiler/test/lang/perceus/task1496_cross_call_reuse_rt.sh:9: SD="$ROOT/.backlog/evidence/TASK-1496/cross_call_reuse.sd"
agent: remove from compiler/test/lang/perceus/task1496_main_entry_abi_bb.sh:9: SD="$ROOT/.backlog/evidence/TASK-1496/main_entry_reuse.sd"
agent: remove from compiler/test/lang/perceus/task1496_main_entry_abi_bb.sh:25: TEST_SD="$ROOT/.backlog/evidence/TASK-1496/test_entry_reuse.sd"
agent: remove from compiler/test/lang/perceus/task1556_loop_fresh_slot.sd:8: -- .backlog/evidence/TASK-1556/loop-fresh/).
agent: remove from compiler/test/lang/perceus/task1889_extern_string_rt.sd:6: --        keeps the pointer (audit: .backlog/evidence/TASK-1889/extern-audit.md).
agent: remove from compiler/test/test_search_pipeline_stability.sh:17: mkdir -p "$SANDBOX/.backlog/tasks" "$SANDBOX/.backlog/completed" \
agent: remove from compiler/test/test_search_pipeline_stability.sh:18: "$SANDBOX/.backlog/archive/tasks" "$SANDBOX/design" \
agent: remove from compiler/test/test_search_pipeline_stability.sh:23: cat > "$SANDBOX/.backlog/tasks/task-active.md" <<'EOF'
agent: remove from compiler/test/test_search_pipeline_stability.sh: 8 more lines that read .backlog/ (git grep -n '.backlog' -- compiler/test/test_search_pipeline_stability.sh)
agent: remove from compiler/test/test_task1938_parity_selftest.sh:25: cp "$ROOT/.backlog/evidence/TASK-1937/p9_outer_abort_pending_suffix.sd" "$W/p/mutant_c_silent.sd"
agent: remove from compiler/test/test_task2041_tcc_dir.sh:78: #     gave identical results with either dir (.backlog/evidence/TASK-2041/measure-result.txt). The
agent: remove from reports/task1581-reconciliation-2026-10-02/materialize.py:138: destination = '.backlog/archive/tasks/' + source.name
agent: remove from reports/task1581-reconciliation-2026-10-02/relocate.py:66: source = exact_path(root, choice['source_path'], '.backlog/tasks')
agent: remove from reports/task1581-reconciliation-2026-10-02/relocate.py:67: canonical = exact_path(root, choice['canonical_path'], '.backlog/completed')
agent: remove from reports/task1581-reconciliation-2026-10-02/relocate.py:68: destination = exact_path(root, choice['archive_path'], '.backlog/archive/tasks')
agent: remove from scripts/syntax-drift-verify.sh:49: # Everything else (notes/, Articles/, scratch/, .backlog/, integration/,
agent: remove from tools/backlog_integrity.py:5: Historical .backlog/archive is intentionally outside the active lookup scope.
agent: remove from tools/backlog_integrity.py:18: SCOPES = ('.backlog/tasks', '.backlog/completed', '.backlog/drafts')
agent: remove from tools/backlog_union_merge.py:17: if "/.backlog/" not in "/" + path:
agent: remove from tools/evidence.py:4: .backlog/evidence/TASK-<id>/<file>     committed; text only, each file <= 256 KiB
agent: remove from tools/evidence.py:29: TASKS = REPO / ".backlog" / "tasks"
agent: remove from tools/evidence.py:30: EVID = REPO / ".backlog" / "evidence"
agent: remove from tools/evidence.py: 2 more lines that read .backlog/ (git grep -n '.backlog' -- tools/evidence.py)
agent: remove from tools/gate_process_guard.py:273: # showing it.  Evidence (.backlog/evidence/TASK-1964/): see the report there for observed maxima.
agent: remove from tools/land.sh:322: { [ -n "$c" ] && ! echo "$c" | grep -qv '^\.backlog/' && (cd "$RP" && echo "$c" | tr '\n' '\0' | xargs -0 python3 "$R/tools/backlog_union_merge.py") >> "$LOG" 2>&1 \
agent: remove from tools/land.sh:399: #  1. every path changed between the measured master and today's master is under .backlog/;
agent: remove from tools/land.sh:400: #  2. today's replay of the lane equals the measured replay outside .backlog/.
agent: remove from tools/land.sh: 7 more lines that read .backlog/ (git grep -n '.backlog' -- tools/land.sh)
agent: remove from tools/land_cancel.sh:3: # Observed cause (TASK-2027 AC#1, .backlog/evidence/TASK-2027/): a TERM ran land.sh's EXIT trap, and a
agent: remove from tools/land_known_red_check.sh:10: #   - ticket is TASK-<id>, the ticket EXISTS in TREE/.backlog/tasks, is not Done or Cancelled
agent: remove from tools/land_known_red_check.sh:21: TASKS=$TREE/.backlog/tasks
agent: remove from tools/lane_worktree.sh:20: TF=$(ls "$R/.backlog/tasks/task-$TID - "*.md 2>/dev/null | head -1 || true)
agent: remove from tools/scratch_check.py:29: TASKS = REPO / ".backlog" / "tasks"
agent: remove from tools/scratch_check.py:30: COMPLETED = REPO / ".backlog" / "completed"
agent: remove from tools/snapshot_hygiene.py:45: (.backlog/evidence/TASK-1958/callers.md) runs this tool from post-checkout or by hand, never
agent: remove from tools/snapshot_hygiene.py:300: tasks = repo_root() / ".backlog" / "tasks"
agent: remove from tools/test_backlog_integrity.py:55: source = f'.backlog/tasks/task-{number:03} - historical Ω.md'
agent: remove from tools/test_backlog_integrity.py:96: self.write('.backlog/tasks/task-007 - a.md', record('TASK-007'))
agent: remove from tools/test_backlog_integrity.py:97: self.write('.backlog/completed/task-7 - b.md', record('task-7', 'Done'))
agent: remove from tools/test_backlog_integrity.py: 28 more lines that read .backlog/ (git grep -n '.backlog' -- tools/test_backlog_integrity.py)
agent: remove from tools/test_known_red_check.sh:10: D=$T/compiler/test/known-red; mkdir -p "$D" "$T/tools" "$T/.backlog/tasks"
agent: remove from tools/test_known_red_check.sh:12: ticket() { printf -- '---\nid: TASK-%s\nstatus: %s\n---\n' "$1" "$2" > "$T/.backlog/tasks/task-$1 - t.md"; }
agent: remove from tools/test_land_known_red.sh:83: V=$T/V; rm -rf "$V"; mkdir -p "$V/P" "$V/L/tools" "$V/L/.backlog/tasks" "$V/L/compiler/test/known-red"
agent: remove from tools/test_land_known_red.sh:85: printf -- '---\nid: TASK-7\nstatus: %s\n---\n' "To Do" > "$V/L/.backlog/tasks/task-7 - t.md"
agent: remove from tools/test_land_known_red.sh:95: printf -- '---\nid: TASK-7\nstatus: %s\n---\n' "Done" > "$V/L/.backlog/tasks/task-7 - t.md"
agent: remove from tools/test_land_known_red.sh: 1 more lines that read .backlog/ (git grep -n '.backlog' -- tools/test_land_known_red.sh)
agent: remove from tools/test_land_reuse.sh:3: # master moved only in .backlog/ AND the lane's replay is identical outside .backlog/.
agent: remove from tools/test_land_reuse.sh:18: mkdir -p .backlog/tasks compiler/lib tools
agent: remove from tools/test_land_reuse.sh:19: echo board > .backlog/tasks/t.md; echo code > compiler/lib/a.ml; echo tool > tools/land.sh
agent: remove from tools/test_land_reuse.sh: 5 more lines that read .backlog/ (git grep -n '.backlog' -- tools/test_land_reuse.sh)
agent: remove from tools/test_snapshot_hygiene.py:102: task = root / ".backlog" / "tasks" / "task-open.md"
agent: remove from tools/worktree_hygiene.py:70: no caller runs this tool from pre-commit (.backlog/evidence/TASK-1958/callers.md), and every
next: show this diff to the person; once they agree: hug task import backlog .backlog --cleanup
```

A real run on the clone was refused by Seid's pre-commit and changed nothing (tools/evidence.py: FileNotFoundError: .backlog/tasks). So the agent: lines must be handled first: the pre-commit hook lines 40-45, tools/evidence.py, tools/scratch_check.py and the rest.
