# Seid: switch .githooks/commit-msg to hug's Model: gate (jr55s8 check 4)

Proposal for Jørgen's approval. Nothing is committed in seid-lang.

## Today

`.githooks/commit-msg` (bash, 131 lines) requires one
`Co-Authored-By: <Name> <email>` trailer. The name must match a regex in
`.githooks/known-models.txt` (25 family regexes). The trailer names no effort.

Last 5 commits on Seid main:
- 4 carry only Co-Authored-By.
- e10b80734 carries both `Model: claude-opus-5-5` and Co-Authored-By.

## Proposed

1. `hug task install commit-msg` in each Seid clone. Since core.hooksPath is
   `.githooks`, it writes `<git-common-dir>/hooks/hug-commit-msg` and never
   touches the tracked hook.
2. `.githooks/commit-msg` becomes:

   ```sh
   #!/bin/sh
   # commit-msg: every commit names the model that made it and the effort it
   # ran at, Model: <model-id>/<effort>, or Model: human (jr55s8, Jørgen
   # 2026-11-17). The gate is hug's: hug task help install.
   h="$(git rev-parse --git-common-dir)/hooks/hug-commit-msg"
   [ ! -x "$h" ] || "$h" "$@" || exit 1
   ```

3. `git rm .githooks/known-models.txt`. The roster moves to hug.toml [models]
   on hug/tasks, one line per model with its own levels. For example (the
   levels are to be confirmed per provider):

   ```
   hug task models add claude-opus-5-5 low medium high max
   hug task models add gpt-6.1-sol none low medium high xhigh
   hug task models add k3-256k low medium high
   hug task models add deepseek-v4-flash low medium high
   ```

   Until the roster exists, any of none, minimal, low, medium, high, xhigh
   or max is accepted as the effort, and a typo is refused.

## Decisions for Jørgen

- **Drop Co-Authored-By.** The proposal drops it: one trailer line, as ruled.
  Keeping both means two hooks and two rosters.
- **A clone without the hug gate.** The two-line call passes when
  `hug-commit-msg` is not installed (`[ ! -x "$h" ]`), so a fresh clone or a
  machine without hug accepts any message. The stricter form refuses instead,
  with the fix:

  ```sh
  [ -x "$h" ] || { echo "commit refused: hug's Model: gate is not installed" >&2; echo "fix: hug task install commit-msg" >&2; exit 1; }
  "$h" "$@" || exit 1
  ```

  Recommendation: use the stricter form in Seid, since every Seid machine has hug.
- **Bypass.** The bypass changes from `SEID_ALLOW_NO_COAUTHOR=1` to
  `git commit --no-verify`. hug has no environment bypass.

## Exemptions kept

These are the same in both hooks: merges, cherry-picks, reverts, rebases,
fixup!/squash!/amend!. hug also exempts its own branches (hug/tasks,
muninn-notes).
