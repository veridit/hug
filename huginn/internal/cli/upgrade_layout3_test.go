package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/veridit/hug/huginn/internal/task"
)

// oldLayoutBoard makes a board as a layout-2 hug left it: four tasks named
// rank.id.slug.md, one in group later/ with its files folder beside it,
// and hug.toml saying layout = 2. It returns the ids in board order.
func oldLayoutBoard(t *testing.T, a *App) []string {
	t.Helper()
	execute(t, a, "init")
	board := filepath.Join(a.Cwd, TaskDir)
	var ids []string
	for _, title := range []string{"First look", "Second look", "Third look", "Later look"} {
		ids = append(ids, execute(t, a, "new", title, "--goal", "g", "--purpose", "p", "--principle", "q", "--accept", "a")["task"].(Reply)["id"].(string))
	}
	execute(t, a, "attach", ids[3], "--name", "notes.txt", "--content", "kept\n")
	ranks := []string{"a", "b", "c", "d"}
	for n, id := range ids {
		from := filepath.Join(board, filepath.FromSlash(task.PathFor(id)))
		raw, err := os.ReadFile(from)
		if err != nil {
			t.Fatal(err)
		}
		var keep []string
		for _, l := range strings.SplitAfter(string(raw), "\n") {
			if !strings.HasPrefix(l, "rank: ") && !strings.HasPrefix(l, "group: ") {
				keep = append(keep, l)
			}
		}
		group := ""
		if n == 3 {
			group = "later"
		}
		to := filepath.Join(board, group, task.BuildName(ranks[n], id, task.Slug(strings.SplitN(strings.SplitN(string(raw), "title: ", 2)[1], "\n", 2)[0])))
		os.MkdirAll(filepath.Dir(to), 0o755)
		if err := os.WriteFile(to, []byte(strings.Join(keep, "")), 0o644); err != nil {
			t.Fatal(err)
		}
		os.Remove(from)
		if dir := filepath.Join(board, filepath.FromSlash(task.FolderFor(id))); n == 3 {
			if err := os.Rename(dir, filepath.Join(board, group, id)); err != nil {
				t.Fatal(err)
			}
		}
		os.Remove(filepath.Dir(from))
	}
	if err := os.WriteFile(filepath.Join(board, "hug.toml"), []byte("layout = 2\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	git(t, board, "add", "-A")
	git(t, board, "commit", "-qm", "a board from before layout 3")
	return ids
}

// w40cmk check 1 (and 5's write guard): on a layout-2 board, a write is
// refused with fix hug upgrade --apply; the preview names every move and
// changes nothing, exit 5; --apply moves every task to C/ID.md with its
// rank and group in the file and its folder to C/ID/, in one commit with
// layout = 3; ls, next and lint show the same order and groups after.
func TestUpgradeMovesBoardToLayout3(t *testing.T) {
	t.Parallel()
	a := app(t)
	a.Env["HOME"] = t.TempDir()
	ids := oldLayoutBoard(t, a)
	board := filepath.Join(a.Cwd, TaskDir)
	head := git(t, board, "rev-parse", "HEAD")

	code, out := runText(a, "note", ids[0], "x")
	if code != 1 || !strings.Contains(out, "this board has layout 2 (hug.toml on branch hug/tasks) and this hug writes layout 3: 4 task files get fixed names (C/ID.md, rank and group in the file), one commit; it writes nothing until then") || !strings.HasSuffix(out, "fix: hug upgrade\n") {
		t.Fatalf("a write on a layout-2 board: exit %d\n%s", code, out)
	}
	ls := func() string {
		_, out, _ := runJSON(a, "ls", "--jsonl")
		var rows []string
		for _, l := range strings.Split(strings.TrimSpace(out), "\n") {
			for _, k := range []string{`"id":`, `"rank":`, `"group":`, `"state":`} {
				i := strings.Index(l, k)
				if i < 0 {
					t.Fatalf("ls row has no %s: %s", k, l)
				}
				j := strings.IndexAny(l[i+len(k):], ",}")
				rows = append(rows, l[i:i+len(k)+j])
			}
		}
		return strings.Join(rows, " ")
	}
	beforeLs := ls()
	_, beforeNext := runText(a, "next")
	if !strings.Contains(beforeLs, `"group":"later"`) {
		t.Fatalf("fixture has no group: %s", beforeLs)
	}

	code, out = runText(a, "upgrade")
	if code != UpgradePending || !strings.Contains(out, "tasks: would: 4 tasks move to C/ID.md (the path never changes again), rank: and group: set in each file, 1 files folders to C/ID/, group folders later become group: values") {
		t.Fatalf("preview: exit %d\n%s", code, out)
	}
	for _, id := range ids {
		if !strings.Contains(out, id+": ") || !strings.Contains(out, " -> "+task.PathFor(id)) {
			t.Fatalf("preview does not name %s's move:\n%s", id, out)
		}
	}
	if git(t, board, "rev-parse", "HEAD") != head || git(t, board, "status", "--porcelain") != "" {
		t.Fatal("the preview changed the board")
	}

	code, out = runText(a, "upgrade", "--apply")
	if code != 0 || !strings.Contains(out, "tasks: 4 tasks move to C/ID.md") || !strings.Contains(out, "tasks: board layout 2 -> 3 in hug.toml, in the same commit") {
		t.Fatalf("--apply: exit %d\n%s", code, out)
	}
	if log := git(t, board, "log", "--format=%s", head+"..HEAD"); log != "hug upgrade: 4 tasks to C/ID.md, rank: and group: in the file (layout 3)" {
		t.Fatalf("commits after the preview: %q", log)
	}
	if st := git(t, board, "status", "--porcelain"); st != "" {
		t.Fatalf("not committed: %s", st)
	}
	if toml, _ := os.ReadFile(filepath.Join(board, "hug.toml")); string(toml) != "layout = 3\n" {
		t.Fatalf("hug.toml: %s", toml)
	}
	for n, id := range ids {
		f, err := os.ReadFile(filepath.Join(board, filepath.FromSlash(task.PathFor(id))))
		if err != nil {
			t.Fatal(err)
		}
		group := ""
		if n == 3 {
			group = "later"
		}
		if !strings.Contains(string(f), "\ngroup: \""+group+"\"\nrank: \""+[]string{"a", "b", "c", "d"}[n]+"\"\n") {
			t.Fatalf("%s:\n%s", id, f)
		}
	}
	if b, err := os.ReadFile(filepath.Join(board, filepath.FromSlash(task.FolderFor(ids[3])), "notes.txt")); err != nil || string(b) != "kept\n" {
		t.Fatalf("files folder: %q %v", b, err)
	}
	if _, err := os.Stat(filepath.Join(board, "later")); !os.IsNotExist(err) {
		t.Fatalf("later/ is still there: %v", err)
	}
	if got := ls(); got != beforeLs {
		t.Fatalf("ls changed:\n%s\nwas\n%s", got, beforeLs)
	}
	if _, got := runText(a, "next"); got != beforeNext {
		t.Fatalf("next changed:\n%s\nwas\n%s", got, beforeNext)
	}
	if code, got := runText(a, "show", "a."+ids[0]+".first-look"); code != 0 || !strings.Contains(got, "First look  "+ids[0]) {
		t.Fatalf("the old name is no ref: exit %d\n%s", code, got)
	}
	execute(t, a, "note", ids[0], "a layout-3 board takes writes")
	if code, out = runText(a, "upgrade"); code != 0 || !strings.Contains(out, "tasks: nothing to do") {
		t.Fatalf("second upgrade: exit %d\n%s", code, out)
	}
}
