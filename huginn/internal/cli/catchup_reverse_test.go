package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/veridit/hug/huginn/internal/task"
)

func TestCatchUpRemoteOldLayoutIsTranslatedAtomically(t *testing.T) {
	catchupPair(t, func(a, b *App, ba, bb string, ids []string) {
		applyCatchup(t, b)
		execute(t, b, "note", ids[0], "local layout3 content")
		attachment := "later/" + ids[3] + "/remote.bin"
		binary := []byte{0, 1, 255, '\r', '\n', 128}
		oldWrite(t, ba, "remote layout2 note and metadata", func() {
			oldNote(t, ba, oldPath(t, ba, ids[0]), "remote-only layout2 content")
			p := oldPath(t, ba, ids[2])
			raw, err := os.ReadFile(filepath.Join(ba, p))
			if err != nil {
				t.Fatal(err)
			}
			d := task.Parse(p, raw)
			for k, v := range map[string]any{"title": "Remote changed third title", "aliases": []string{"legacy-third-ref"}, "updated": "2020-01-02"} {
				if err := d.SetField(k, v); err != nil {
					t.Fatal(err)
				}
			}
			writeCatchupFile(t, ba, p, d.Bytes())
			writeCatchupFile(t, ba, attachment, binary)
			if err := os.Chmod(filepath.Join(ba, attachment), 0o755); err != nil {
				t.Fatal(err)
			}
			git(t, ba, "add", "--", attachment)
			git(t, ba, "update-index", "--chmod=+x", "--", attachment)
		})
		git(t, ba, "push", "-q", "origin", "HEAD:"+TaskBranch)
		remoteHead := git(t, ba, "rev-parse", "HEAD")
		git(t, bb, "fetch", "-q", "origin")
		out := applyCatchup(t, b)
		got := taskFile(t, b, ids[0])
		for _, text := range []string{"local layout3 content", "remote-only layout2 content"} {
			if strings.Count(got, text) != 1 {
				t.Fatalf("lost or repeated %q:\n%s", text, got)
			}
		}
		rows := execute(t, b, "ls")["tasks"].([]Reply)
		if len(rows) != len(ids) {
			t.Fatalf("duplicated old+canonical tasks: %d != %d", len(rows), len(ids))
		}
		paths := strings.Split(git(t, bb, "ls-tree", "-r", "--name-only", "HEAD"), "\n")
		for _, p := range paths {
			if task.OldName(p) {
				t.Fatalf("old name survived translation: %s", p)
			}
		}
		for _, id := range ids {
			if got := git(t, bb, "ls-files", "--", task.PathFor(id)); got != task.PathFor(id) {
				t.Fatalf("canonical task absent: %s", id)
			}
		}
		metadata := taskFile(t, b, ids[2])
		if task.Parse(task.PathFor(ids[2]), []byte(metadata)).Task.Updated != "2020-01-02" || !strings.Contains(metadata, "Remote changed third title") {
			t.Fatalf("remote metadata was not preserved:\n%s", metadata)
		}
		for _, ref := range []string{"legacy-third-ref", "third-look"} {
			if code, out := runText(b, "show", ref); code != 0 || !strings.Contains(out, ids[2]) {
				t.Fatalf("remote alias/old slug %s lost: exit%d\n%s", ref, code, out)
			}
		}
		canonicalAttachment := task.FolderFor(ids[3]) + "/remote.bin"
		if raw, err := os.ReadFile(filepath.Join(bb, canonicalAttachment)); err != nil || !bytes.Equal(raw, binary) {
			t.Fatalf("remote binary attachment changed: %x %v", raw, err)
		}
		if entry := git(t, bb, "ls-tree", "HEAD", "--", canonicalAttachment); !strings.HasPrefix(entry, "100755 blob ") {
			t.Fatalf("remote executable mode changed: %s", entry)
		}
		if !strings.Contains(git(t, bb, "show", "HEAD:hug.toml"), "layout = 3") {
			t.Fatal("incoherent board metadata")
		}
		if strings.Contains(out, "remote board moved to layout 3") {
			t.Fatalf("summary invents a remote move:\n%s", out)
		}
		// Every installed replay commit, not only the last one, is coherent.
		for _, sha := range strings.Fields(git(t, bb, "rev-list", remoteHead+"..HEAD")) {
			if !strings.Contains(git(t, bb, "show", sha+":hug.toml"), "layout = 3") {
				t.Fatalf("commit %s is not layout3", sha)
			}
			for _, p := range strings.Split(git(t, bb, "ls-tree", "-r", "--name-only", sha), "\n") {
				if task.OldName(p) {
					t.Fatalf("commit%s has old name%s", sha, p)
				}
			}
		}
		if git(t, ba, "rev-parse", "HEAD") != remoteHead || git(t, ba, "status", "--porcelain") != "" {
			t.Fatal("remote source clone changed")
		}
		head := git(t, bb, "rev-parse", "HEAD")
		applyCatchup(t, b)
		execute(t, b, "sync")
		applyCatchup(t, b)
		execute(t, b, "sync")
		if git(t, bb, "rev-parse", "HEAD") != head || git(t, bb, "status", "--porcelain") != "" {
			t.Fatal("second apply/sync was not idempotent")
		}
	})
}

func TestCatchUpReverseTranslationFailureKeepsClones(t *testing.T) {
	for _, kind := range []string{"ambiguous", "unreadable"} {
		t.Run(kind, func(t *testing.T) {
			catchupPair(t, func(a, b *App, ba, bb string, ids []string) {
				applyCatchup(t, b)
				execute(t, b, "note", ids[0], "keep local layout3 content")
				p := oldPath(t, ba, ids[2])
				oldWrite(t, ba, "remote invalid translation", func() {
					if kind == "unreadable" {
						writeCatchupFile(t, ba, p, []byte("no front matter\n"))
						return
					}
					raw, err := os.ReadFile(filepath.Join(ba, p))
					if err != nil {
						t.Fatal(err)
					}
					writeCatchupFile(t, ba, "later/z."+ids[2]+".duplicate.md", raw)
				})
				git(t, ba, "push", "-q", "origin", "HEAD:"+TaskBranch)
				git(t, bb, "fetch", "-q", "origin")
				ha, hb := git(t, ba, "rev-parse", "HEAD"), git(t, bb, "rev-parse", "HEAD")
				code, out := runText(b, "upgrade", "--apply")
				if code == 0 || !strings.Contains(out, "fix:") || !strings.Contains(out, ids[2]) {
					t.Fatalf("unsafe reverse translation: exit%d\n%s", code, out)
				}
				if git(t, ba, "rev-parse", "HEAD") != ha || git(t, bb, "rev-parse", "HEAD") != hb || git(t, ba, "status", "--porcelain") != "" || git(t, bb, "status", "--porcelain") != "" {
					t.Fatal("failed translation changed a source clone")
				}
			})
		})
	}
}
