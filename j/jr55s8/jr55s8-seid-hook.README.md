jr55s8: switch Seid's commit-msg to hug's Model: <id>/<effort> gate (made against seid-lang e5e04f92c; touches .githooks/commit-msg, known-models.txt, the guard test, TOOLING.md and 2 skills).
Apply, in a Seid worktree on a lane branch: git am /Users/jhf/.jcode/scratch/jr55s8-seid-hook.patch (the commit carries Model: claude-opus-5-5/high; re-sign with git commit --amend -S --no-edit if needed).
Install the gate once per clone (the new hook refuses without it): hug task install commit-msg
Verify: bash compiler/test/infra/test_commit_msg_attribution_guard.sh prints 'all 17 commit-msg guard checks passed' (on the old hook 7 fail; the 10 exemption checks pass on both).
Decisions inside, for Jørgen (his go is check 4): Co-Authored-By dropped; a clone without the gate is refused; bypass is git commit --no-verify (SEID_ALLOW_NO_COAUTHOR removed).
