package cli

import "path/filepath"

// globID finds the task file of id below dir in either layout: C/ID.md
// (layout 3, w40cmk) or [rank.]id.slug.md, so a test reads the file
// wherever the board keeps it.
func globID(dir, id string) ([]string, error) {
	m, e := filepath.Glob(filepath.Join(dir, "*"+id+"*.md"))
	if e != nil {
		return nil, e
	}
	more, e := filepath.Glob(filepath.Join(dir, "?", id+".md"))
	return append(m, more...), e
}
