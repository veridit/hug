package cli

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// xxvm6t check 2: every command that takes a ref accepts task:ID, the form
// running text uses, as well as the bare id. Check 3: what it prints keeps
// the bare id, so the next command runs when pasted into a shell.
func TestEveryRefCommandAcceptsTaskColon(t *testing.T) {
	t.Parallel()
	a := app(t)
	execute(t, a, "init")
	newID := func(args ...string) string {
		r := execute(t, a, append([]string{"new"}, args...)...)
		return r["task"].(Reply)["id"].(string)
	}
	epic := newID("Epic", "--goal", "g", "--accept", "e")
	beta := newID("Beta", "--goal", "g", "--accept", "b")
	// new: --dep and --after take refs.
	gamma := newID("Gamma", "--goal", "g", "--accept", "c", "--accept", "d", "--dep", "task:"+beta, "--after", "task:"+epic)
	src := filepath.Join(t.TempDir(), "notes.md")
	if e := os.WriteFile(src, []byte("x\n"), 0o644); e != nil {
		t.Fatal(e)
	}
	useEditor(t, a, "", "") // edit leaves the file as it is
	T := func(id string) string { return "task:" + id }
	for _, args := range [][]string{
		{"show", T(beta)},
		{"note", T(beta), "evidence"},
		{"check", T(beta), "1"},
		{"set", T(beta), "--parent", T(epic)},
		{"ls", "--epic", T(epic)},
		{"next", "--epic", T(epic)},
		{"board", "--epic", T(epic)},
		{"move", T(gamma), "--before", T(epic)},
		{"move", T(gamma), "--after", T(beta)},
		{"dep", T(gamma), "--remove", T(beta)},
		{"dep", T(gamma), "--add", T(beta)},
		{"attach", T(gamma), src, "--name", "notes.md"},
		{"show", T(gamma), "--file", "notes.md"},
		{"detach", T(gamma), "notes.md"},
		{"edit", T(gamma)},
		{"promote", T(gamma), "2"},
		{"mv", T(gamma), "other"},
		{"rebalance", T(epic), T(gamma)},
		{"take", T(beta)},
		{"done", T(beta)},
		{"drop", T(gamma), "superseded"},
		{"set", T(gamma), "--status", "todo"},
		{"design", T(gamma), "which way?"},
		{"design", T(gamma), "--clear"},
		{"take", T(gamma)},
		{"wait", T(gamma), "CI"},
		{"resume", T(gamma)},
	} {
		out := &bytes.Buffer{}
		a.Out, a.Err = out, &bytes.Buffer{}
		if code := a.Run(context.Background(), args); code != 0 {
			t.Fatalf("%v: exit %d\n%s%s", args, code, out, a.Err)
		}
		for _, line := range strings.Split(out.String(), "\n") {
			if (strings.HasPrefix(line, "next: ") || strings.HasPrefix(line, "fix: ")) && strings.Contains(line, "task:") {
				t.Errorf("%v printed a command with task:ID, which a shell runs but reads oddly: %q", args, line)
			}
		}
	}
	r := execute(t, a, "show", beta)
	if tk := r["task"].(Reply); tk["status"] != "done" || tk["parent"] != epic {
		t.Fatalf("beta after task:ID commands: status %v parent %v", tk["status"], tk["parent"])
	}
	// A typo still says not-found, naming the bare id.
	if _, e := a.Execute(context.Background(), []string{"show", "task:zzzzzz"}); e == nil {
		t.Fatal("typo found a task")
	} else if _, problem, _ := ErrorDetails(e); problem != "not-found: zzzzzz" {
		t.Fatalf("typo: %q", problem)
	}
}
