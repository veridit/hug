package cli

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/veridit/hug/huginn/gitbranch"
)

// sharesMemories makes this test's repositories opt in to sharing project
// memories (34r1ft), for tests of the shared rename itself.
func sharesMemories(t *testing.T) {
	t.Helper()
	old := memoryShares
	memoryShares = func(string) (bool, error) { return true, nil }
	t.Cleanup(func() { memoryShares = old })
}

func memoryLayoutApp(t *testing.T) (*App, string, string) {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("JCODE_HOME", filepath.Join(home, "jcode"))
	t.Setenv("HUG_HOME", filepath.Join(home, "hug"))
	t.Setenv("HUG_SYNC", "off")
	a := app(t)
	// Preserve fixture bytes through later Git checkouts even when the
	// machine-wide Windows default enables core.autocrlf.
	git(t, a.Cwd, "config", "core.autocrlf", "false")
	a.Env["HOME"], a.Env["JCODE_HOME"], a.Env["HUG_HOME"], a.Env["HUG_SYNC"] = home, filepath.Join(home, "jcode"), filepath.Join(home, "hug"), "off"
	execute(t, a, "init")
	old := filepath.Join(a.Cwd, ".muninn-notes")
	git(t, a.Cwd, "worktree", "add", "--detach", old, "HEAD")
	git(t, old, "checkout", "--orphan", "muninn-notes")
	raw := "{\"id\":\"memory-one\",\"content\":\"exact active text\",\"active\":true,\"created\":\"2026-11-17T12:00:00Z\",\"updated\":\"2026-11-17T12:00:00Z\"}\n{\"id\":\"memory-two\",\"content\":\"exact deleted text\",\"active\":false,\"created\":\"2026-11-17T12:00:00Z\",\"updated\":\"2026-11-17T12:00:00Z\"}\n"
	if err := os.WriteFile(filepath.Join(old, "memories.jsonl"), []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}
	git(t, old, "add", "memories.jsonl")
	git(t, old, "commit", "-qm", "project memory, exact bytes")
	return a, old, raw
}

func TestMemoryLayoutBothPresentPreservesDifferingData(t *testing.T) {
	a, old, raw := memoryLayoutApp(t)
	canonical := filepath.Join(a.Cwd, ".hug", "memory")
	git(t, a.Cwd, "worktree", "add", "--detach", canonical, "HEAD")
	git(t, canonical, "checkout", "--orphan", "hug/memory")
	other := "different canonical memory\n"
	if err := os.WriteFile(filepath.Join(canonical, "memories.jsonl"), []byte(other), 0o644); err != nil {
		t.Fatal(err)
	}
	git(t, canonical, "add", "memories.jsonl")
	git(t, canonical, "commit", "-qm", "different canonical data")
	refs := git(t, a.Cwd, "show-ref")
	if code, out := runText(a, "upgrade", "--apply"); code != 1 || !strings.Contains(out, "both branches muninn-notes and hug/memory exist") || !strings.Contains(out, "fix:") {
		t.Fatalf("differing branches were not safely refused: exit%d\n%s", code, out)
	}
	if got := git(t, a.Cwd, "show-ref"); got != refs {
		t.Fatalf("refusal changed refs: %s", got)
	}
	for path, want := range map[string]string{old: raw, canonical: other} {
		if b, err := os.ReadFile(filepath.Join(path, "memories.jsonl")); err != nil || string(b) != want {
			t.Fatalf("refusal changed %s: %v %q", path, err, b)
		}
	}
}

func TestMemoryLayoutRemoteOnlyNeverReportsNothingToDo(t *testing.T) {
	a, old, _ := memoryLayoutApp(t)
	head := git(t, old, "rev-parse", "HEAD")
	git(t, a.Cwd, "update-ref", "refs/remotes/origin/muninn-notes", head)
	git(t, a.Cwd, "worktree", "remove", old)
	git(t, a.Cwd, "branch", "-D", "muninn-notes")
	refs := git(t, a.Cwd, "show-ref")
	code, out := runText(a, "upgrade", "--apply")
	if code == 0 || !strings.Contains(out, "muninn-notes") || !strings.Contains(out, "fix:") {
		t.Fatalf("known legacy remote memory was silently missed: exit%d\n%s", code, out)
	}
	if got := git(t, a.Cwd, "show-ref"); got != refs {
		t.Fatalf("remote-only refusal changed refs: %s", got)
	}
}

func TestMemoryLayoutRemoteRecoveryNeverOffersUnsafeAttach(t *testing.T) {
	for _, kind := range []string{"old-path", "new-path", "registered", "ambiguous-remotes"} {
		t.Run(kind, func(t *testing.T) {
			a, old, _ := memoryLayoutApp(t)
			head := git(t, old, "rev-parse", "HEAD")
			remote := "origin"
			if kind == "ambiguous-remotes" {
				remote = "one"
			}
			git(t, a.Cwd, "update-ref", "refs/remotes/"+remote+"/muninn-notes", head)
			git(t, a.Cwd, "worktree", "remove", old)
			if kind == "registered" {
				elsewhere := filepath.Join(t.TempDir(), "elsewhere")
				git(t, a.Cwd, "worktree", "add", elsewhere, "muninn-notes")
			}
			git(t, a.Cwd, "update-ref", "-d", "refs/heads/muninn-notes")
			switch kind {
			case "old-path", "new-path":
				target := old
				if kind == "new-path" {
					target = filepath.Join(a.Cwd, ".hug", "memory")
				}
				if err := os.MkdirAll(target, 0o755); err != nil {
					t.Fatal(err)
				}
			case "ambiguous-remotes":
				git(t, a.Cwd, "update-ref", "refs/remotes/two/muninn-notes", head)
			}
			refs := git(t, a.Cwd, "show-ref")
			for _, args := range [][]string{{"upgrade"}, {"upgrade", "--apply"}} {
				code, out := runText(a, args...)
				if code != 1 || !strings.Contains(out, "fix:") || strings.Contains(out, "worktree add -b") {
					t.Fatalf("unsafe recovery instruction for %s: exit%d\n%s", kind, code, out)
				}
				if got := git(t, a.Cwd, "show-ref"); got != refs {
					t.Fatalf("refusal changed refs: %s", got)
				}
			}
		})
	}
}

func TestMemoryLayoutUnregisteredDirectoryIsNotSilentlyIgnored(t *testing.T) {
	a, old, raw := memoryLayoutApp(t)
	git(t, a.Cwd, "worktree", "remove", old)
	git(t, a.Cwd, "branch", "-D", "muninn-notes")
	if err := os.MkdirAll(old, 0o755); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(old, "memories.jsonl")
	if err := os.WriteFile(file, []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}
	refs := git(t, a.Cwd, "show-ref")
	for _, args := range [][]string{{"upgrade"}, {"upgrade", "--apply"}} {
		code, out := runText(a, args...)
		shown, hasPrefix := strings.CutPrefix(out, "project memory path ")
		shown, _, hasSuffix := strings.Cut(shown, " is not a readable registered Git worktree;")
		shownInfo, shownErr := os.Stat(shown)
		oldInfo, oldErr := os.Stat(old)
		// Git resolves temporary-directory aliases such as RUNNER~1. Require
		// the exact directory, rather than one spelling of its path.
		if code != 1 || !strings.Contains(out, "fix:") || !hasPrefix || !hasSuffix || shownErr != nil || oldErr != nil || !os.SameFile(shownInfo, oldInfo) {
			t.Fatalf("unregistered memory directory silently ignored: exit%d\n%s", code, out)
		}
		if b, err := os.ReadFile(file); err != nil || string(b) != raw {
			t.Fatalf("refusal changed memory data: %v %q", err, b)
		}
		if got := git(t, a.Cwd, "show-ref"); got != refs {
			t.Fatalf("refusal changed refs: %s", got)
		}
	}
}

func TestMemoryLayoutLinkedWorktreeUsesOneCanonicalCheckout(t *testing.T) {
	a, old, raw := memoryLayoutApp(t)
	main := a.Cwd
	linked := filepath.Join(t.TempDir(), "linked")
	git(t, main, "worktree", "add", "-b", "linked", linked)
	a.Cwd = linked
	if code, out := runText(a, "upgrade", "--apply"); code != 0 {
		t.Fatalf("linked memory migration: exit%d\n%s", code, out)
	}
	if b, err := os.ReadFile(filepath.Join(main, ".hug", "memory", "memories.jsonl")); err != nil || string(b) != raw {
		t.Fatalf("canonical shared memory lost: %v %q", err, b)
	}
	for _, absent := range []string{old, filepath.Join(linked, ".hug", "memory")} {
		if _, err := os.Stat(absent); !os.IsNotExist(err) {
			t.Fatalf("competing memory checkout %s: %v", absent, err)
		}
	}
}

func TestMemoryLayoutTwoClonesKeepUnpushedMemory(t *testing.T) {
	// A repository that shares its memories (34r1ft opt-in): the rename
	// reaches the remote. Not parallel (t.Setenv), so the swap is safe.
	sharesMemories(t)
	// Exercise the Windows Git default on every OS. The byte-preservation
	// fixture must explicitly select its checkout policy, not normalize data.
	config := filepath.Join(t.TempDir(), "global.gitconfig")
	if err := os.WriteFile(config, []byte("[core]\n\tautocrlf = true\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GIT_CONFIG_GLOBAL", config)
	a, old, raw := memoryLayoutApp(t)
	remote := filepath.Join(t.TempDir(), "remote.git")
	git(t, a.Cwd, "init", "--bare", "-b", "main", remote)
	quietRemote(t, remote)
	git(t, a.Cwd, "remote", "add", "origin", remote)
	git(t, a.Cwd, "push", "-u", "origin", "main")
	git(t, old, "push", "-u", "origin", "muninn-notes")
	b := *a
	b.Cwd = filepath.Join(t.TempDir(), "two")
	git(t, a.Cwd, "clone", remote, b.Cwd)
	git(t, b.Cwd, "config", "core.autocrlf", "false")
	git(t, b.Cwd, "config", "user.name", "Test B")
	git(t, b.Cwd, "config", "user.email", "b@example.invalid")
	oldB := filepath.Join(b.Cwd, ".muninn-notes")
	git(t, b.Cwd, "worktree", "add", "-b", "muninn-notes", oldB, "origin/muninn-notes")
	extra := "{\"id\":\"local-b\",\"content\":\"unpushed memory from B\",\"active\":true}\n"
	if err := os.WriteFile(filepath.Join(oldB, "memories.jsonl"), []byte(raw+extra), 0o644); err != nil {
		t.Fatal(err)
	}
	git(t, oldB, "commit", "-qam", "B unpushed memory")
	headB := git(t, oldB, "rev-parse", "HEAD")
	for _, clone := range []*App{a, &b} {
		if code, out := runText(clone, "upgrade", "--apply"); code != 0 {
			t.Fatalf("memory migration in %s: exit%d\n%s", clone.Cwd, code, out)
		}
	}
	canonicalB := filepath.Join(b.Cwd, ".hug", "memory")
	if b, err := os.ReadFile(filepath.Join(canonicalB, "memories.jsonl")); err != nil || string(b) != raw+extra {
		t.Fatalf("unpushed B memory lost: %v %q", err, b)
	}
	if got := git(t, canonicalB, "rev-parse", "HEAD"); got != headB {
		t.Fatalf("B commit was rewritten: %s != %s", got, headB)
	}
	git(t, canonicalB, "push", "origin", "HEAD")
	canonicalA := filepath.Join(a.Cwd, ".hug", "memory")
	git(t, canonicalA, "pull", "--ff-only")
	if b, err := os.ReadFile(filepath.Join(canonicalA, "memories.jsonl")); err != nil || string(b) != raw+extra {
		t.Fatalf("ordinary Git fast-forward lost migrated memory: %v %q", err, b)
	}
	if heads := git(t, remote, "show-ref"); strings.Contains(heads, "refs/heads/muninn-notes") || !strings.Contains(heads, "refs/heads/hug/memory") {
		t.Fatalf("remote layout incoherent: %s", heads)
	}
}

func TestUpgradeMovesProjectMemoryBranchOnlyExplicitly(t *testing.T) {
	a, old, raw := memoryLayoutApp(t)
	head := git(t, old, "rev-parse", "HEAD")
	refs := git(t, a.Cwd, "show-ref")
	code, out := runText(a, "upgrade")
	want := "layout: would: .muninn-notes/ moves to .hug/memory/, branch muninn-notes becomes hug/memory"
	if code != UpgradePending || !strings.Contains(out, want) {
		t.Fatalf("memory preview: exit %d, want %q:\n%s", code, want, out)
	}
	if got := git(t, a.Cwd, "show-ref"); got != refs {
		t.Fatalf("preview changed refs:\n%s\n%s", refs, got)
	}
	if b, err := os.ReadFile(filepath.Join(old, "memories.jsonl")); err != nil || string(b) != raw {
		t.Fatalf("preview changed source: %v %q", err, b)
	}
	if code, out = runText(a, "upgrade", "--apply"); code != 0 {
		t.Fatalf("memory apply: exit %d\n%s", code, out)
	}
	canonical := filepath.Join(a.Cwd, ".hug", "memory")
	if b, err := os.ReadFile(filepath.Join(canonical, "memories.jsonl")); err != nil || string(b) != raw {
		t.Fatalf("memory bytes lost: %v %q", err, b)
	}
	if got := git(t, canonical, "rev-parse", "HEAD"); got != head {
		t.Fatalf("memory history changed: %s != %s", got, head)
	}
	if got := git(t, canonical, "branch", "--show-current"); got != "hug/memory" {
		t.Fatalf("memory branch %q", got)
	}
	if _, err := os.Stat(old); !os.IsNotExist(err) {
		t.Fatalf("old mount remains: %v", err)
	}
	afterRefs := git(t, a.Cwd, "show-ref")
	if code, out = runText(a, "upgrade", "--apply"); code != 0 || strings.Contains(out, "would:") {
		t.Fatalf("second memory apply: exit %d\n%s", code, out)
	}
	if got := git(t, a.Cwd, "show-ref"); got != afterRefs {
		t.Fatalf("second apply changed refs: %s", got)
	}
}

// The approved memory migration reuses this shared mechanism. Its failure
// must not rename a branch before discovering an occupied destination.
func TestMemoryLayoutSharedUpgradeRollbackOnDestinationCollision(t *testing.T) {
	a, old, raw := memoryLayoutApp(t)
	canonical := filepath.Join(a.Cwd, ".hug", "memory")
	if err := os.MkdirAll(canonical, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(canonical, "memories.jsonl"), []byte("different destination\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	refs := git(t, a.Cwd, "show-ref")
	_, err := gitbranch.Upgrade(a.Cwd, "muninn-notes", ".muninn-notes", "hug/memory", ".hug/memory")
	if err == nil {
		t.Fatal("occupied destination was accepted")
	}
	if got := git(t, a.Cwd, "show-ref"); got != refs {
		t.Fatalf("shared Upgrade changed refs before refusing destination:\nbefore:\n%s\nafter:\n%s\nerror: %v", refs, got, err)
	}
	if b, err := os.ReadFile(filepath.Join(old, "memories.jsonl")); err != nil || string(b) != raw {
		t.Fatalf("failure changed source: %v %q", err, b)
	}
}

// 34r1ft (bonehound, wxsyc5): hug upgrade --apply renamed muninn-notes to
// hug/memory on the remote with no share check: the route by which a
// public repository published its memories. A repository that keeps its
// memories local (the default) renames only here: the remote's refs are
// byte-identical before and after, and the upgrade says why.
func TestMemoryLayoutUpgradeLeavesRemoteWhenRepoKeepsMemoriesLocal(t *testing.T) {
	a, old, raw := memoryLayoutApp(t)
	remote := filepath.Join(t.TempDir(), "remote.git")
	git(t, a.Cwd, "init", "--bare", "-b", "main", remote)
	git(t, a.Cwd, "remote", "add", "origin", remote)
	git(t, a.Cwd, "push", "-u", "origin", "main")
	git(t, old, "push", "-u", "origin", "muninn-notes")
	before := git(t, a.Cwd, "ls-remote", "origin")
	if !strings.Contains(before, "refs/heads/muninn-notes") {
		t.Fatalf("fixture: remote lacks muninn-notes:\n%s", before)
	}
	code, out := runText(a, "upgrade", "--apply")
	if code != 0 {
		t.Fatalf("upgrade: exit %d\n%s", code, out)
	}
	if after := git(t, a.Cwd, "ls-remote", "origin"); after != before {
		t.Fatalf("remote refs changed by a local-only upgrade:\nbefore:\n%s\nafter:\n%s", before, after)
	}
	if !strings.Contains(out, "remote muninn-notes kept; this repo keeps memories local") {
		t.Fatalf("upgrade did not say the remote was kept:\n%s", out)
	}
	canonical := filepath.Join(a.Cwd, ".hug", "memory")
	if got := git(t, canonical, "branch", "--show-current"); got != "hug/memory" {
		t.Fatalf("local branch not renamed: %q", got)
	}
	if b, err := os.ReadFile(filepath.Join(canonical, "memories.jsonl")); err != nil || string(b) != raw {
		t.Fatalf("memory bytes lost: %v %q", err, b)
	}
	// No upstream: the old one (origin/muninn-notes) is not carried over, so
	// a plain git push from the checkout has nowhere to go.
	if up, err := exec.Command("git", "-C", canonical, "rev-parse", "--abbrev-ref", "@{upstream}").CombinedOutput(); err == nil {
		t.Fatalf("hug/memory has an upstream after a local-only upgrade: %s", up)
	}
}
