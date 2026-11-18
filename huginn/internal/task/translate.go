package task

import (
	"path"
	"strings"
)

// Layout3Names maps every file of a tree (its paths) to its layout-3 name
// (w40cmk check 9): an old-named task file to C/ID.md, a file in its files
// folder [group/]ID/ to C/ID/, anything else to itself.
func Layout3Names(paths []string) map[string]string {
	owner := map[string]string{} // old files folder -> new one
	for _, p := range paths {
		if OldName(p) {
			i, _ := IDOf(p)
			owner[path.Join(path.Dir(p), i)] = FolderFor(i)
		}
	}
	out := make(map[string]string, len(paths))
	for _, p := range paths {
		to := p
		if OldName(p) {
			i, _ := IDOf(p)
			to = PathFor(i)
		} else if len(owner) > 0 {
			for dir := path.Dir(p); dir != "." && dir != "/"; dir = path.Dir(dir) {
				if nd, ok := owner[dir]; ok {
					to = nd + strings.TrimPrefix(p, dir)
					break
				}
			}
		}
		out[p] = to
	}
	return out
}

// Layout3Content is a file's text at its layout-3 name: an old-named task
// file gets rank: and group: from its name (as hug upgrade writes them),
// hug.toml says layout 3, anything else is as it was. A task file hug
// cannot read keeps its bytes: the merge stops on it only if both sides
// changed it.
func Layout3Content(p string, b []byte) []byte {
	switch {
	case p == ConfigName:
		return []byte(WithLayout(string(b), Layout))
	case OldName(p):
		r, i, _, _ := ParseName(p)
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
