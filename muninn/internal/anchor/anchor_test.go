package anchor

import (
	"reflect"
	"testing"
)

// n7jbmx check 5: anchors are the exact things a tool call and a memory can
// share: a path of two or more segments, a file name, a command head of two
// or three words. A memory states each anchor once; a call states every
// contiguous run of its paths, so a memory naming a folder matches a file
// in it.
func TestMemory(t *testing.T) {
	for text, want := range map[string][]string{
		"Run `go test ./muninn/run/` niced; muninn/run/main.go holds the hook.":       {"cmd:go test", "file:main.go", "path:muninn/run", "path:muninn/run/main.go"},
		"Never edit .hug/tasks/ files by hand; use `hug task note`.":                  {"cmd:hug task note", "path:.hug/tasks"},
		"The cache schema is in huginn/internal/cache/cache.go.":                      {"file:cache.go", "path:huginn/internal/cache/cache.go"},
		"Jørgen prefers short answers.":                                               nil,
		"See https://example.com/a/b.html for details.":                               nil,
		"Use `git rebase origin/main` then `git push`; `forget` alone is no command.": {"cmd:git push", "cmd:git rebase"},
		"Rebase on origin/main, then check muninn/run/ and and/or docs.":              {"path:muninn/run"},
		"Update CHANGELOG.md and ~/Projects/hug/AGENTS.md.":                           {"file:AGENTS.md", "file:CHANGELOG.md", "path:Projects/hug/AGENTS.md"},
	} {
		if got := Memory(text); !reflect.DeepEqual(got, want) {
			t.Errorf("Memory(%q)\n got %v\nwant %v", text, got, want)
		}
	}
}

func TestCall(t *testing.T) {
	home, root := "/Users/x", "/Users/x/Projects/hug"
	got := Call("bash", `cd /Users/x/Projects/hug && GOMAXPROCS=2 nice -n 19 go test ./muninn/run/ | tail -3; hug task done abc123`, nil, root, home)
	want := []string{"cmd:go test", "cmd:hug task", "cmd:hug task done", "path:muninn/run"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("bash anchors\n got %v\nwant %v", got, want)
	}
	got = Call("edit", "", []string{"/Users/x/Projects/hug/muninn/run/main.go"}, root, home)
	want = []string{"file:main.go", "path:muninn/run", "path:muninn/run/main.go", "path:run/main.go"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("edit anchors\n got %v\nwant %v", got, want)
	}
	got = Call("bash", `sed -n 1,20p "$HOME/x.txt" huginn/internal/task/questions.go`, nil, root, home)
	want = []string{"file:questions.go", "path:huginn/internal", "path:huginn/internal/task", "path:huginn/internal/task/questions.go", "path:internal/task", "path:internal/task/questions.go", "path:task/questions.go"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("bash path anchors\n got %v\nwant %v", got, want)
	}
}
