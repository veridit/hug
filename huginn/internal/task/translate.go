package task

import (
	"path"
	"sort"
	"strings"
)

// Layout3Names maps every file of a tree (its paths) to its layout-3 name
// (w40cmk check 9): an old-named task file to C/ID.md, a file in its files
// folder [group/]ID/ to C/ID/, anything else to itself.
func Layout3Names(paths []string) map[string]string {
	owner := map[string]string{} // old files folder -> new one
	ownerOf := func(p string) (string, string, bool) {
		for dir := path.Dir(p); dir != "." && dir != "/"; dir = path.Dir(dir) {
			if nd, ok := owner[dir]; ok {
				return dir, nd, true
			}
		}
		return "", "", false
	}
	ordered := append([]string(nil), paths...)
	sort.Slice(ordered, func(i, j int) bool {
		a, b := strings.Count(ordered[i], "/"), strings.Count(ordered[j], "/")
		if a != b {
			return a < b
		}
		return ordered[i] < ordered[j]
	})
	for _, p := range ordered {
		if _, _, owned := ownerOf(p); owned {
			continue
		}
		if i, ok := IDPath(p); ok {
			owner[FolderFor(i)] = FolderFor(i)
		} else if OldName(p) {
			i, _ := IDOf(p)
			owner[path.Join(path.Dir(p), i)] = FolderFor(i)
		}
	}
	out := make(map[string]string, len(paths))
	for _, p := range paths {
		to := p
		if dir, nd, owned := ownerOf(p); owned {
			to = nd + strings.TrimPrefix(p, dir)
		} else if OldName(p) {
			i, _ := IDOf(p)
			to = PathFor(i)
		}
		out[p] = to
	}
	return out
}

// Layout3Content is a file's text at its layout-3 name: an old-named task
// file gets rank: and group: from its name (as hug upgrade writes them),
// hug.toml says layout 3, anything else is as it was. A task file hug
// cannot read keeps its bytes here: the task merge validates them before
// they can be written.
func Layout3Content(p, to string, b []byte) []byte {
	switch {
	case p == ConfigName:
		return []byte(WithLayout(string(b), Layout))
	case OldName(p):
		r, i, _, _ := ParseName(p)
		if to != PathFor(i) {
			return b // task-named attachment, not a task
		}
		group := path.Dir(p)
		if group == "." {
			group = ""
		}
		if out, e := Layout3Fields(b, LayoutMove{ID: i, From: p, To: PathFor(i), Rank: r, Group: group}); e == nil {
			return out
		}
	}
	return b
}
