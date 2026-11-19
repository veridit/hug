package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// completions are the words Complete offers (without descriptions).
func completions(t *testing.T, cwd string, words ...string) ([]string, int) {
	t.Helper()
	got, directive := Complete(context.Background(), cwd, words)
	var out []string
	for _, c := range got {
		w, _, _ := strings.Cut(c, "\t")
		out = append(out, w)
	}
	return out, directive
}

// 5mmqjk: hug task completion comes from hug task's own tree: its
// commands, each command's flags, task ids where the usage line says ID,
// and the values a flag takes from the model (types, states).
func TestCompleteTaskTree(t *testing.T) {
	t.Parallel()
	a := app(t)
	execute(t, a, "init")
	open := execute(t, a, "new", "Open task", "--goal", "g", "--purpose", "p", "--principle", "q", "--accept", "one")["task"].(Reply)["id"].(string)
	closed := execute(t, a, "new", "Closed task", "--goal", "g", "--purpose", "p", "--principle", "q", "--accept", "one")["task"].(Reply)["id"].(string)
	execute(t, a, "drop", closed, "not needed")
	for _, tc := range []struct {
		words     []string
		want      []string
		not       []string
		directive int
	}{
		{[]string{"sh"}, []string{"show"}, []string{"setup", "take"}, NoFileComp},
		{[]string{""}, Commands, nil, NoFileComp},
		{[]string{"take", "--"}, []string{"--model", "--as", "--with", "--json", "--help"}, []string{"--goal"}, NoFileComp},
		{[]string{"ls", "--state", ""}, []string{"todo", "doing", "done", "dropped"}, nil, NoFileComp},
		{[]string{"set", "x", "--type", "b"}, []string{"bug"}, []string{"feature"}, NoFileComp},
		{[]string{"show", ""}, []string{open}, []string{closed}, NoFileComp},
		{[]string{"show", open[:2]}, []string{open}, nil, NoFileComp},
		{[]string{"note", open, ""}, nil, []string{open}, NoFileComp},
		{[]string{"take", "x", "--with", ""}, []string{open}, []string{closed}, NoFileComp},
		{[]string{"take", "--model", "m", ""}, []string{open}, nil, NoFileComp},
		{[]string{"memory", ""}, []string{"share", "local"}, nil, NoFileComp},
		{[]string{"install", "jcode", ""}, nil, []string{"jcode", "commit-msg"}, NoFileComp},
		{[]string{"reflections", "resolve", open, "1", "a"}, []string{"applied"}, []string{"dismissed"}, NoFileComp},
		{[]string{"help", "ta"}, []string{"take", "tags"}, []string{"show"}, NoFileComp},
		{[]string{"attach", open, ""}, nil, nil, FileComp},
		// The typed literal picks the alternative: backlog DIRECTORY is
		// files, github OWNER/REPO is not.
		{[]string{"import", "backlog", ""}, nil, nil, FileComp},
		{[]string{"import", "github", ""}, nil, nil, NoFileComp},
		{[]string{"bogus", ""}, nil, nil, NoFileComp},
	} {
		got, directive := completions(t, a.Cwd, tc.words...)
		for _, w := range tc.want {
			if !slices.Contains(got, w) {
				t.Errorf("%q: %q missing from %q", tc.words, w, got)
			}
		}
		for _, w := range tc.not {
			if slices.Contains(got, w) {
				t.Errorf("%q: %q offered: %q", tc.words, w, got)
			}
		}
		if directive != tc.directive {
			t.Errorf("%q: directive %d, want %d", tc.words, directive, tc.directive)
		}
	}
	// Outside a repository: commands still, ids none, no error.
	outside := t.TempDir()
	if got, _ := completions(t, outside, "show", ""); len(got) != 0 {
		t.Errorf("show ids outside a repository: %q", got)
	}
	if got, _ := completions(t, outside, "ne"); !slices.Equal(got, []string{"new", "next"}) {
		t.Errorf("ne outside a repository: %q", got)
	}
}

// 5mmqjk (crab): completion runs on every TAB and must never write: no
// sync, no cache rebuild, no lock. A board whose remote is broken and whose
// cache is stale (a task file edited after the cache last looked) answers
// with the file's current state, and no file anywhere in the repository,
// its git folder included, changes.
func TestCompleteNeverWrites(t *testing.T) {
	t.Parallel()
	a := app(t)
	execute(t, a, "init")
	id := execute(t, a, "new", "Before", "--goal", "g", "--purpose", "p", "--principle", "q", "--accept", "one")["task"].(Reply)["id"].(string)
	execute(t, a, "ls") // the cache knows the board
	git(t, a.Cwd, "remote", "add", "origin", filepath.Join(t.TempDir(), "missing.git"))
	path := taskPath(t, a, id)
	b, e := os.ReadFile(path)
	if e != nil {
		t.Fatal(e)
	}
	if e = os.WriteFile(path, []byte(strings.Replace(string(b), "title: \"Before\"", "title: \"After\"", 1)), 0o644); e != nil {
		t.Fatal(e)
	}
	before := snapshotTree(t, a.Cwd)
	got, _ := Complete(context.Background(), a.Cwd, []string{"show", ""})
	if !slices.Contains(got, id+"\tAfter") {
		t.Errorf("stale cache: want %q, got %q", id+"\tAfter", got)
	}
	Complete(context.Background(), a.Cwd, []string{"take", ""})
	after := snapshotTree(t, a.Cwd)
	for p, s := range after {
		if before[p] != s {
			t.Errorf("completion changed %s", p)
		}
	}
	for p := range before {
		if _, ok := after[p]; !ok {
			t.Errorf("completion removed %s", p)
		}
	}
}

// snapshotTree is every file below root (the .git folder too) with its
// size, modification time and mode.
func snapshotTree(t *testing.T, root string) map[string]string {
	t.Helper()
	out := map[string]string{}
	e := filepath.Walk(root, func(p string, info os.FileInfo, e error) error {
		if e != nil {
			return e
		}
		out[p] = fmt.Sprint(info.Mode(), " ", info.ModTime().UnixNano(), " ", info.Size())
		return nil
	})
	if e != nil {
		t.Fatal(e)
	}
	return out
}
