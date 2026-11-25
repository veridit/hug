//go:build !windows

package project

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/veridit/hug/internal/timing"
)

// legacyRepo is a git repository whose code branch tracks the old
// .muninn/memories.jsonl, ready for SyncBranch --migrate.
func legacyRepo(t *testing.T) (root string, p Project, native string) {
	t.Helper()
	root = t.TempDir()
	gitInit(t, root)
	if err := os.MkdirAll(filepath.Join(root, ".muninn"), 0o700); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	data, _ := json.Marshal(Line{ID: "shared", Content: "Hooks of the code repo are not hug's.", Category: "decision", Active: true, Created: now, Updated: now})
	if err := os.WriteFile(filepath.Join(root, ".muninn", "memories.jsonl"), append(data, '\n'), 0o600); err != nil {
		t.Fatal(err)
	}
	branchGit(t, root, "add", ".muninn")
	branchGit(t, root, "commit", "-m", "synthetic legacy")
	p, err := Find(root, false)
	if err != nil {
		t.Fatal(err)
	}
	return root, p, filepath.Join(t.TempDir(), "native.json")
}

// 9vp9by: the layout migration's commit on the code branch runs none of
// the code repository's hooks, as hug's other commits do not (runAt passes
// core.hooksPath=/dev/null since the 2026-11-09 seid-lang incident). Seid's
// commit-msg hook is hug's own Model: gate (jr55s8), which refuses a
// commit without the trailer; and a hook that leaves a process holding
// git's output kept CombinedOutput waiting for it (WaitDelay 0).
func TestMigrationCommitRunsNoRepoHooks(t *testing.T) {
	timing.Budget(t)
	for _, tc := range []struct{ name, hook, script string }{
		{"commit-msg refuses", "commit-msg", "#!/bin/sh\necho ran > \"$MARK\"\necho 'refused: no Model: trailer' >&2\nexit 1\n"},
		{"post-commit leaves a process holding output", "post-commit", "#!/bin/sh\necho ran > \"$MARK\"\nsleep 20 &\nexit 0\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root, p, native := legacyRepo(t)
			mark := filepath.Join(t.TempDir(), "hook-ran")
			t.Setenv("MARK", mark)
			hooks := branchGit(t, root, "rev-parse", "--path-format=absolute", "--git-path", "hooks")
			if err := os.MkdirAll(hooks, 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(hooks, tc.hook), []byte(tc.script), 0o755); err != nil {
				t.Fatal(err)
			}
			before := branchGit(t, root, "rev-parse", "HEAD")
			start := time.Now()
			st, err := SyncBranch(p, native, BranchOptions{Migrate: true})
			took := time.Since(start)
			if err != nil || !st.Migrated {
				t.Fatalf("migration with a %s hook: migrated=%v err=%v", tc.hook, st.Migrated, err)
			}
			if _, e := os.Stat(mark); e == nil {
				t.Fatalf("the migration ran the code repository's %s hook (took %s)", tc.hook, took)
			}
			if took > 15*time.Second {
				t.Fatalf("the migration waited %s on a hook's process", took)
			}
			if got := branchGit(t, root, "rev-list", "--count", before+"..HEAD"); got != "1" {
				t.Fatalf("migration made %s code commits", got)
			}
			if got := branchGit(t, root, "ls-files", "--", ".muninn/memories.jsonl"); got != "" {
				t.Fatalf("legacy file still tracked: %s", got)
			}
			if got := branchGit(t, root, "status", "--porcelain"); got != "" {
				t.Fatalf("migration left the code worktree dirty: %s", got)
			}
			if !strings.Contains(branchGit(t, root, "log", "-1", "--format=%s"), "Move project memories") {
				t.Fatalf("unexpected commit: %s", branchGit(t, root, "log", "-1", "--format=%s"))
			}
		})
	}
}
