package cli

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/veridit/hug/huginn/internal/task"
)

func catchupPair(t *testing.T, f func(a, b *App, ba, bb string, ids []string)) {
	t.Helper()
	// Each pair owns its repositories, App environments and child processes.
	// Use the existing bounded test pool, just like the older clones tests,
	// instead of delaying every parallel test behind these serial fixtures.
	t.Parallel()
	a, b, _ := clones(t)
	ba, bb := filepath.Join(a.Cwd, TaskDir), filepath.Join(b.Cwd, TaskDir)
	offline(t, a, func() {
		offline(t, b, func() {
			ids := oldLayoutBoard(t, a)
			git(t, ba, "push", "-q", "origin", "HEAD:"+TaskBranch)
			git(t, bb, "pull", "-q", "--rebase", "origin", TaskBranch)
			f(a, b, ba, bb, ids)
		})
	})
}

func applyCatchup(t *testing.T, a *App) string {
	t.Helper()
	code, out := runText(a, "upgrade", "--apply")
	if code != 0 && code != UpgradePending {
		t.Fatalf("upgrade: exit %d\n%s", code, out)
	}
	return out
}

func writeCatchupFile(t *testing.T, board, rel string, raw []byte) {
	t.Helper()
	p := filepath.Join(board, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, raw, 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestCatchUpFirstParentRetainsMergeResolutionOnce(t *testing.T) {
	catchupPair(t, func(a, b *App, ba, bb string, ids []string) {
		p := oldPath(t, bb, ids[0])
		raw, err := os.ReadFile(filepath.Join(bb, p))
		if err != nil {
			t.Fatal(err)
		}
		git(t, bb, "checkout", "-qb", "local-side")
		oldWrite(t, bb, "side branch edit", func() {
			d := task.Parse(p, raw)
			d.SetField("title", "side title")
			d.Note("side branch content exactly once")
			writeCatchupFile(t, bb, p, d.Bytes())
		})
		sideRaw, err := os.ReadFile(filepath.Join(bb, p))
		if err != nil {
			t.Fatal(err)
		}
		git(t, bb, "checkout", "-q", TaskBranch)
		oldWrite(t, bb, "mainline edit", func() {
			d := task.Parse(p, raw)
			d.SetField("title", "mainline title")
			writeCatchupFile(t, bb, p, d.Bytes())
		})
		cmd := exec.Command("git", "merge", "--no-ff", "--no-commit", "local-side")
		cmd.Dir = bb
		if out, err := cmd.CombinedOutput(); err == nil || !strings.Contains(string(out), "CONFLICT") {
			t.Fatalf("expected real merge conflict: %v\n%s", err, out)
		}
		d := task.Parse(p, sideRaw)
		d.SetField("title", "resolved merge title")
		d.Note("merge-only resolution exactly once")
		writeCatchupFile(t, bb, p, d.Bytes())
		git(t, bb, "add", "-A")
		git(t, bb, "commit", "-qm", "merge conflict resolution")
		author := git(t, bb, "log", "-1", "--format=%an <%ae> %aI")
		applyCatchup(t, a)
		execute(t, a, "sync")
		git(t, bb, "fetch", "-q", "origin")
		out := applyCatchup(t, b)
		got := taskFile(t, b, ids[0])
		for _, text := range []string{"resolved merge title", "side branch content exactly once", "merge-only resolution exactly once"} {
			if strings.Count(got, text) != 1 {
				t.Fatalf("%q not retained exactly once:\n%s", text, got)
			}
		}
		if !strings.Contains(out, "2 commits of this clone replayed") {
			t.Fatalf("wrong first-parent count:\n%s", out)
		}
		if got := git(t, bb, "log", "--format=%s", "origin/"+TaskBranch+"..HEAD"); got != "merge conflict resolution\nmainline edit" {
			t.Fatalf("replayed side branch twice or lost merge: %s", got)
		}
		if got := git(t, bb, "log", "-1", "--format=%an <%ae> %aI"); got != author {
			t.Fatalf("author changed: %q != %q", got, author)
		}
	})
}

func TestCatchUpOrdinaryOfflineUpgradeDoesNotFetch(t *testing.T) {
	catchupPair(t, func(a, b *App, ba, bb string, ids []string) {
		git(t, bb, "remote", "set-url", "origin", filepath.Join(t.TempDir(), "missing-remote"))
		applyCatchup(t, b)
		if !strings.Contains(git(t, bb, "show", "HEAD:hug.toml"), "layout = 3") {
			t.Fatal("offline board did not migrate")
		}
	})
}

func TestCatchUpKnownRemoteFailureKeepsEverything(t *testing.T) {
	catchupPair(t, func(a, b *App, ba, bb string, ids []string) {
		oldWrite(t, bb, "local note", func() { oldNote(t, bb, oldPath(t, bb, ids[0]), "local exact note") })
		applyCatchup(t, a)
		execute(t, a, "sync")
		git(t, bb, "fetch", "-q", "origin")
		git(t, bb, "remote", "set-url", "origin", filepath.Join(t.TempDir(), "missing-remote"))
		head := git(t, bb, "rev-parse", "HEAD")
		code, out := runText(b, "upgrade", "--apply")
		if code == 0 || !strings.Contains(out, "fix:") || git(t, bb, "rev-parse", "HEAD") != head || git(t, bb, "status", "--porcelain") != "" {
			t.Fatalf("network failure not safely retriable: exit%d\n%s", code, out)
		}
	})
}

func TestCatchUpBinaryAndConfigRecovery(t *testing.T) {
	for _, existing := range []string{"absent", "same", "collision", "same-change"} {
		t.Run(existing, func(t *testing.T) {
			catchupPair(t, func(a, b *App, ba, bb string, ids []string) {
				oldAttachment := ids[0] + "/proof.bin"
				oldWrite(t, ba, "base attachment and config", func() {
					writeCatchupFile(t, ba, oldAttachment, []byte{0, 1, 2})
					config, err := os.ReadFile(filepath.Join(ba, "hug.toml"))
					if err != nil {
						t.Fatal(err)
					}
					writeCatchupFile(t, ba, "hug.toml", append(config, []byte("\n[recovery_test]\nvalue = \"base\"\n")...))
				})
				git(t, ba, "push", "-q", "origin", "HEAD:"+TaskBranch)
				git(t, bb, "pull", "-q", "--rebase", "origin", TaskBranch)
				localBytes := []byte{0, 255, 3, 10, 0}
				oldWrite(t, bb, "local binary and config", func() {
					writeCatchupFile(t, bb, oldAttachment, localBytes)
					if err := os.Chmod(filepath.Join(bb, oldAttachment), 0o755); err != nil {
						t.Fatal(err)
					}
					config, err := os.ReadFile(filepath.Join(bb, "hug.toml"))
					if err != nil {
						t.Fatal(err)
					}
					writeCatchupFile(t, bb, "hug.toml", bytes.Replace(config, []byte(`"base"`), []byte(`"local"`), 1))
					git(t, bb, "add", "-A")
					git(t, bb, "update-index", "--chmod=+x", "--", oldAttachment)
				})
				local := git(t, bb, "rev-parse", "HEAD")
				localConfig, err := os.ReadFile(filepath.Join(bb, "hug.toml"))
				if err != nil {
					t.Fatal(err)
				}
				applyCatchup(t, a)
				attachment := task.FolderFor(ids[0]) + "/proof.bin"
				recovered := attachment + ".local-" + local
				configRecovery := "hug.toml.local-" + local
				remoteBytes := []byte{0, 254, 7}
				remoteValue := `"remote"`
				if existing == "same-change" {
					remoteBytes, remoteValue = localBytes, `"local"`
				}
				oldWrite(t, ba, "remote binary and config", func() {
					writeCatchupFile(t, ba, attachment, remoteBytes)
					config, err := os.ReadFile(filepath.Join(ba, "hug.toml"))
					if err != nil {
						t.Fatal(err)
					}
					writeCatchupFile(t, ba, "hug.toml", bytes.Replace(config, []byte(`"base"`), []byte(remoteValue), 1))
					if existing == "same" || existing == "collision" {
						content := localBytes
						if existing == "collision" {
							content = []byte("keep existing recovery")
						}
						writeCatchupFile(t, ba, recovered, content)
						if err := os.Chmod(filepath.Join(ba, recovered), 0o755); err != nil {
							t.Fatal(err)
						}
						git(t, ba, "add", "-A")
						git(t, ba, "update-index", "--chmod=+x", "--", recovered)
					}
				})
				execute(t, a, "sync")
				git(t, bb, "fetch", "-q", "origin")
				code, out := runText(b, "upgrade", "--apply")
				if existing == "collision" {
					if code == 0 || !strings.Contains(out, recovered) || !strings.Contains(out, "collision") || git(t, bb, "rev-parse", "HEAD") != local || git(t, bb, "status", "--porcelain") != "" {
						t.Fatalf("recovery collision not safely stopped: exit%d\n%s", code, out)
					}
					return
				}
				if code != 0 && code != UpgradePending {
					t.Fatalf("catch-up exit%d\n%s", code, out)
				}
				if existing == "same-change" {
					if paths := git(t, bb, "ls-tree", "-r", "--name-only", "HEAD"); strings.Contains(paths, ".local-") {
						t.Fatalf("identical changes made a duplicate: %s", paths)
					}
					return
				}
				for _, p := range []string{attachment, recovered, configRecovery, local} {
					if !strings.Contains(out, p) {
						t.Fatalf("summary lacks %q:\n%s", p, out)
					}
				}
				for p, want := range map[string][]byte{attachment: remoteBytes, recovered: localBytes, configRecovery: localConfig} {
					got, err := os.ReadFile(filepath.Join(bb, p))
					if err != nil || !bytes.Equal(got, want) {
						t.Fatalf("%s not byte exact: %v\ngot%q\nwant%q", p, err, got, want)
					}
				}
				if mode := git(t, bb, "ls-tree", "HEAD", "--", recovered); !strings.HasPrefix(mode, "100755 ") {
					t.Fatalf("recovery mode lost: %s", mode)
				}
				active, err := os.ReadFile(filepath.Join(bb, "hug.toml"))
				if err != nil || !bytes.Contains(active, []byte(`"remote"`)) || bytes.Contains(active, []byte(`"local"`)) {
					t.Fatalf("local config became active: %v\n%s", err, active)
				}
				head := git(t, bb, "rev-parse", "HEAD")
				listCode, listOut, listErr := runJSON(b, "ls", "--jsonl")
				if rows := strings.Split(strings.TrimSpace(listOut), "\n"); listCode != 0 || len(rows) != len(ids) {
					t.Fatalf("recovered files misparsed as tasks: %d != %d\nstderr:\n%s", len(rows), len(ids), listErr)
				}
				applyCatchup(t, b)
				if git(t, bb, "rev-parse", "HEAD") != head {
					t.Fatal("second apply made another recovery")
				}
			})
		})
	}
}

func TestCatchUpDeletedTaskStopsWithExactRecoverableBlob(t *testing.T) {
	catchupPair(t, func(a, b *App, ba, bb string, ids []string) {
		oldWrite(t, bb, "first safe local note", func() { oldNote(t, bb, oldPath(t, bb, ids[1]), "first local note") })
		p := oldPath(t, bb, ids[0])
		oldWrite(t, bb, "local edit of deleted task", func() { oldNote(t, bb, p, "exact retained local content") })
		local := git(t, bb, "rev-parse", "HEAD")
		raw, err := os.ReadFile(filepath.Join(bb, p))
		if err != nil {
			t.Fatal(err)
		}
		applyCatchup(t, a)
		git(t, ba, "rm", "--", task.PathFor(ids[0]))
		git(t, ba, "commit", "-qm", "remote deletion")
		execute(t, a, "sync")
		git(t, bb, "fetch", "-q", "origin")
		code, out := runText(b, "upgrade", "--apply")
		for _, want := range []string{ids[0], "deleted", "2 local commits kept", local + ":" + p} {
			if code == 0 || !strings.Contains(out, want) {
				t.Fatalf("missing safe deletion evidence %q: exit%d\n%s", want, code, out)
			}
		}
		if git(t, bb, "rev-parse", "HEAD") != local || git(t, bb, "status", "--porcelain") != "" {
			t.Fatal("failed deletion replay changed live board")
		}
		blob := exec.Command("git", "-C", bb, "show", local+":"+p)
		got, err := blob.Output()
		if err != nil || !bytes.Equal(raw, got) {
			t.Fatalf("original exact bytes not recoverable: %v", err)
		}
	})
}
