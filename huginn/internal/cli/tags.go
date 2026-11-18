package cli

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/veridit/hug/huginn/internal/task"
)

// hug task tags lists the board's tags (hug.toml tags = [...]); hug task
// tags add NAME adds one (txjkrm check 2), the fix an unlisted-tag finding
// gives when the tag belongs on the list.
func (a *App) runTags(s *session, f flags) (Reply, error) {
	root := s.r.Dir
	if len(f.args) == 0 {
		list, e := task.Tags(root)
		if e != nil {
			return nil, e
		}
		tags := list.List
		if tags == nil {
			tags = []string{}
		}
		next := "hug task lint --code unlisted-tag"
		if len(tags) == 0 {
			next = "hug task tags add NAME"
		}
		return Reply{"tags": tags, "next": next}, nil
	}
	if f.args[0] != "add" || len(f.args) != 2 {
		return nil, usage("tags takes no arguments, or add NAME", "hug task tags add NAME")
	}
	name := strings.TrimSpace(f.args[1])
	if name == "" || strings.ContainsAny(name, " \t,[]\"=") {
		return nil, usage("a tag is one word, as rows show it: "+name, "hug task tags add "+strings.Join(strings.FieldsFunc(name, func(r rune) bool { return strings.ContainsRune(" \t,[]\"=", r) }), "-"))
	}
	path := filepath.Join(root, task.ConfigName)
	b, e := os.ReadFile(path)
	if e != nil && !os.IsNotExist(e) {
		return nil, e
	}
	text := task.WithTag(string(b), name)
	if text == string(b) {
		return Reply{"tags": mustTags(root), "changed": false, "next": "hug task lint --code unlisted-tag"}, nil
	}
	if e = os.WriteFile(path, []byte(text), 0o644); e != nil {
		return nil, e
	}
	if e = s.commit("hug task tags add " + name); e != nil {
		return nil, e
	}
	return Reply{"tags": mustTags(root), "changed": true, "next": "hug task lint --code unlisted-tag"}, nil
}

func mustTags(root string) []string {
	list, _ := task.Tags(root)
	if list.List == nil {
		return []string{}
	}
	return list.List
}
