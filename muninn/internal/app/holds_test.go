package app_test

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/veridit/hug/muninn/internal/app"
	"github.com/veridit/hug/muninn/internal/ingest"
	"github.com/veridit/hug/muninn/internal/recall"
	"github.com/veridit/hug/muninn/internal/scopetest"
)

func recalls(t *testing.T, f *scopetest.Fixture, msg, sid string) bool {
	t.Helper()
	b := f.App.Bound("", f.Cwd)
	r, err := f.App.Recall.Recall(context.Background(), recall.Request{Message: msg, Scopes: b.Scopes, ScopeFor: b.ScopeFor})
	if err != nil {
		t.Fatal(err)
	}
	for _, h := range r.Hits {
		if h.SID == sid {
			return true
		}
	}
	return false
}

// check 1 and 2 (ez50rb): remember stores a condition; a condition that
// has ended hides the memory from recall and becomes one dream item;
// nothing is deleted.
func TestHoldsRememberRecallDream(t *testing.T) {
	f := scopetest.New(t)
	ctx := context.Background()
	b := f.App.Bound("", f.Cwd)

	if _, _, err := b.RememberPlaced(ctx, f.App, "zebrafish gateway runs on port 9443", "fact", "", nil, "", "", "", app.RememberDecision{Holds: "while dragon runs"}); err == nil ||
		!strings.Contains(err.Error(), "\nfix: hug mem remember --holds") {
		t.Fatalf("malformed holds must fail with a fix line, got %v", err)
	}
	past := time.Now().Add(-48 * time.Hour).Format("2006-01-02")
	future := time.Now().Add(30 * 24 * time.Hour).Format("2006-01-02")
	gone, _, err := b.RememberPlaced(ctx, f.App, "zebrafish gateway runs on port 9443 during the migration", "fact", "", nil, "", "", "", app.RememberDecision{Holds: "Until  " + past, NewReason: "test"})
	if err != nil {
		t.Fatal(err)
	}
	kept, _, err := b.RememberPlaced(ctx, f.App, "zebrafish gateway freeze applies to the staging cluster", "fact", "", nil, "", "", "", app.RememberDecision{Holds: "until " + future, NewReason: "test"})
	if err != nil {
		t.Fatal(err)
	}
	mems, _ := ingest.LoadNative(app.NativePath())
	stored := map[string]string{}
	for _, m := range mems {
		stored[m.SID] = m.Holds
	}
	if stored[gone.SID] != "until "+past || stored[kept.SID] != "until "+future {
		t.Fatalf("stored holds: %q / %q", stored[gone.SID], stored[kept.SID])
	}
	if u, _ := f.App.DB.Resolve(gone.SID); u.Holds != "until "+past {
		t.Fatalf("index holds = %q", u.Holds)
	}
	if recalls(t, f, "zebrafish gateway port migration", gone.SID) {
		t.Fatal("a memory whose date has passed must not be recalled")
	}
	if !recalls(t, f, "zebrafish gateway freeze staging cluster", kept.SID) {
		t.Fatal("a memory whose condition still holds must be recalled")
	}
	before := 0
	f.App.DB.QueryRow(`SELECT count(*) FROM units WHERE active=1`).Scan(&before)
	for i := 0; i < 2; i++ {
		if _, err := f.App.Dream(ctx, false); err != nil {
			t.Fatal(err)
		}
	}
	var items int
	f.App.DB.QueryRow(`SELECT count(*) FROM dream_items WHERE kind='ended' AND sids=?`, gone.SID).Scan(&items)
	if items != 1 {
		t.Fatalf("ended memory: %d dream items, want exactly 1", items)
	}
	f.App.DB.QueryRow(`SELECT count(*) FROM dream_items WHERE kind='ended' AND sids=?`, kept.SID).Scan(&items)
	if items != 0 {
		t.Fatal("a memory that still holds must not be queued")
	}
	after := 0
	f.App.DB.QueryRow(`SELECT count(*) FROM units WHERE active=1`).Scan(&after)
	if after != before {
		t.Fatalf("nothing is deleted automatically: active %d -> %d", before, after)
	}
	got, _ := b.DreamItems(f.App, "jcode:h", 5)
	var it app.DreamItem
	for _, x := range got {
		if x.Kind == "ended" {
			it = x
		}
	}
	if it.ID == 0 || !strings.Contains(it.Instruction, "has passed") || len(it.SIDs) != 1 {
		t.Fatalf("ended item not handed out: %+v", got)
	}
}

func git(t *testing.T, dir string, args ...string) {
	t.Helper()
	c := exec.Command("git", append([]string{"-C", dir}, args...)...)
	c.Env = append(os.Environ(), "GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@e", "GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@e")
	if out, err := c.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

// A task or branch condition ends when hug sees the task closed or the
// branch merged; a recorded branch alone never hides a memory, it only
// asks (crab, 2026-11-16).
func TestHoldsBranchAndTask(t *testing.T) {
	f := scopetest.New(t)
	ctx := context.Background()
	repo := filepath.Join(t.TempDir(), "repo")
	git(t, filepath.Dir(repo), "init", "-q", "-b", "main", repo)
	git(t, repo, "commit", "-q", "--allow-empty", "-m", "init")
	git(t, repo, "branch", "feat/merged")
	git(t, repo, "branch", "feat/live")
	git(t, repo, "commit", "-q", "--allow-empty", "-m", "ahead")
	git(t, repo, "checkout", "-q", "feat/live")
	git(t, repo, "commit", "-q", "--allow-empty", "-m", "live work")
	git(t, repo, "checkout", "-q", "main")
	// A board: dn0001 is done, op0001 is doing.
	git(t, repo, "worktree", "add", "-q", "--orphan", "-b", "hug/tasks", ".hug/tasks")
	for id, status := range map[string]string{"dn0001": "done", "op0001": "doing"} {
		task := "---\nid: \"" + id + "\"\ntitle: \"T\"\nstate: \"" + status + "\"\n---\n\n## Goal\nG\n"
		if err := os.WriteFile(filepath.Join(repo, ".hug/tasks", "a."+id+".t.md"), []byte(task), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	git(t, filepath.Join(repo, ".hug/tasks"), "add", "-A")
	git(t, filepath.Join(repo, ".hug/tasks"), "commit", "-q", "-m", "t")
	mems, err := ingest.LoadNative(app.NativePath())
	if err != nil {
		t.Fatal(err)
	}
	add := func(id, text, holds, branch string) {
		mems = append(mems, ingest.NativeMemory{ID: id, Content: text, Category: "fact", Scope: "project:matrix-project", Active: true, Strength: 1,
			Created: time.Now(), Updated: time.Now(), Holds: holds, Branch: branch, Repo: repo})
	}
	add("mem_hb1", "walrus parser cache lives in tmp while the branch is open", "while branch feat/merged unmerged", "")
	add("mem_hb2", "walrus lexer flag stays on during the live branch", "while branch feat/live unmerged", "")
	add("mem_hb3", "walrus lesson learned on a merged branch, generally true", "", "feat/merged")
	add("mem_hb4", "walrus note about an unknown task", "while task:zz9999 open", "")
	add("mem_hb5", "walrus note while the done task was open", "while task:dn0001 open", "")
	add("mem_hb6", "walrus note while the doing task is open", "while task:op0001 open", "")
	if err := ingest.SaveNative(app.NativePath(), mems); err != nil {
		t.Fatal(err)
	}
	if _, err := f.App.Reindex(ctx); err != nil {
		t.Fatal(err)
	}
	sid := map[string]string{}
	mems, _ = ingest.LoadNative(app.NativePath())
	for _, m := range mems {
		sid[m.ID] = m.SID
	}
	if _, err := f.App.Dream(ctx, false); err != nil {
		t.Fatal(err)
	}
	kind := func(id string) string {
		var k string
		f.App.DB.QueryRow(`SELECT group_concat(kind) FROM dream_items WHERE sids=?`, sid[id]).Scan(&k)
		return k
	}
	if kind("mem_hb1") != "ended" || kind("mem_hb2") != "" || kind("mem_hb3") != "branch" || kind("mem_hb4") != "" ||
		kind("mem_hb5") != "ended" || kind("mem_hb6") != "" {
		t.Fatalf("items: hb1=%q hb2=%q hb3=%q hb4=%q hb5=%q hb6=%q (want ended, none, branch, none (not on the board: unknown), ended, none)",
			kind("mem_hb1"), kind("mem_hb2"), kind("mem_hb3"), kind("mem_hb4"), kind("mem_hb5"), kind("mem_hb6"))
	}
	if recalls(t, f, "walrus parser cache tmp branch open", sid["mem_hb1"]) {
		t.Fatal("an ended branch condition must hide the memory")
	}
	if !recalls(t, f, "walrus lesson learned merged branch generally true", sid["mem_hb3"]) {
		t.Fatal("a recorded branch alone must never hide a memory")
	}
}
