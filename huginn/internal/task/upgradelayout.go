package task

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/veridit/hug/internal/sharefile"
)

// Layout 3 upgrade (w40cmk check 1): every old-named task file
// ([group/]rank.id.slug.md) moves to C/ID.md, the rank and group its name
// said become rank: and group: in its front matter, and its files folder
// [group/]ID/ moves to C/ID/. No other line of the file changes.

// LayoutMove is one task's move to layout 3.
type LayoutMove struct {
	ID       string `json:"id"`
	From     string `json:"from"`
	To       string `json:"to"`
	Rank     string `json:"rank"`
	Group    string `json:"group"`
	FilesDir string `json:"files_from,omitempty"` // the old files folder, when there is one
}

// LayoutPlan is the whole move: the tasks, and the old group folders that
// are left empty.
type LayoutPlan struct {
	Moves  []LayoutMove `json:"moves"`
	Groups []string     `json:"groups"`
	Files  int          `json:"files"` // files folders moved
}

// identityKeys are the front-matter keys rank: and group: go after: the
// first found from the end (Jørgen 01:27 grouping: id, title, type,
// priority, labels, group, parent, rank).
var identityKeys = []string{"parent", "labels", "priority", "type", "title", "id"}

// PlanLayout3 lists the moves without changing anything. A move whose
// target is taken (two files with one id) stops the plan with the fix.
func (s *Store) PlanLayout3() (LayoutPlan, error) {
	var p LayoutPlan
	ts, e := s.Tasks()
	if e != nil {
		return p, e
	}
	to := map[string]string{}
	groups := map[string]bool{}
	for _, t := range ts {
		if !OldName(t.Path) {
			continue
		}
		dest := PathFor(t.ID)
		if prev, ok := to[dest]; ok {
			return p, &Error{Kind: "duplicate-id", ID: t.ID, Value: prev + ", " + t.Path, Fix: "hug task lint"}
		}
		to[dest] = t.Path
		m := LayoutMove{ID: t.ID, From: t.Path, To: dest, Rank: t.Rank, Group: t.Group}
		if dir, e := s.diskPath(filepath.Join(filepath.Dir(t.Path), t.ID)); e == nil {
			if st, e := os.Lstat(dir); e == nil && st.IsDir() {
				m.FilesDir = filepath.ToSlash(filepath.Join(filepath.Dir(t.Path), t.ID))
				p.Files++
			}
		}
		if t.Group != "" {
			groups[t.Group] = true
		}
		p.Moves = append(p.Moves, m)
	}
	// A target that is there already (a C/ID.md beside the old file, two
	// files for one id) stops the plan: the upgrade overwrites nothing.
	for _, m := range p.Moves {
		for _, rel := range []string{m.To, FolderFor(m.ID)} {
			if rel == FolderFor(m.ID) && m.FilesDir == "" {
				continue
			}
			disk, e := s.diskPath(rel)
			if e != nil {
				return p, e
			}
			if _, e := os.Lstat(disk); e == nil {
				return p, &Error{Kind: "path-exists", ID: m.ID, Value: rel + " (moving " + m.From + ")", Fix: "hug task lint"}
			}
		}
	}
	for g := range groups {
		p.Groups = append(p.Groups, g)
	}
	sort.Strings(p.Groups)
	return p, nil
}

// Layout3Fields gives the file's text as it is at C/ID.md: rank: and group:
// set after the identity keys, every other line as it was.
func Layout3Fields(raw []byte, m LayoutMove) ([]byte, error) {
	d := Parse(m.From, raw)
	if d.frontEnd < 0 || !d.frontmatterValid {
		return nil, &Error{Kind: "broken-file", ID: m.ID, Value: m.From, Fix: "hug task lint " + m.ID}
	}
	for _, k := range []string{"rank", "group"} {
		v := m.Rank
		if k == "group" {
			v = m.Group
		}
		if e := d.setFieldAfter(k, v, identityKeys); e != nil {
			return nil, e
		}
	}
	return d.Bytes(), nil
}

// setFieldAfter sets k to value: in place when k is there, else on a new
// line after the first of after that the file has (else last).
func (d *Document) setFieldAfter(k, value string, after []string) error {
	if _, ok := d.fields[k]; ok {
		return d.SetField(k, value)
	}
	at := d.frontEnd
	for _, a := range after {
		if _, ok := d.fields[a]; ok {
			at = d.fieldEnds[a]
			if at <= d.fields[a] {
				at = d.fields[a] + 1
			}
			break
		}
	}
	line := fmt.Sprintf("%s: %q%s", k, value, d.newline)
	lines := append([]string(nil), d.lines...)
	d.replace(append(lines[:at], append([]string{line}, lines[at:]...)...))
	return nil
}

// ApplyLayout3 does the plan's moves on disk and returns every path the
// commit must stage (old and new). It refuses a target that exists.
func (s *Store) ApplyLayout3(p LayoutPlan) ([]string, error) {
	var paths []string
	s.Invalidate()
	for _, m := range p.Moves {
		src, e := s.diskPath(m.From)
		if e != nil {
			return paths, e
		}
		dst, e := s.diskPath(m.To)
		if e != nil {
			return paths, e
		}
		if _, e := os.Lstat(dst); e == nil {
			return paths, &Error{Kind: "path-exists", ID: m.ID, Value: m.To, Fix: "hug task lint"}
		}
		raw, e := sharefile.ReadFile(src)
		if e != nil {
			return paths, e
		}
		out, e := Layout3Fields(raw, m)
		if e != nil {
			return paths, e
		}
		if e = AtomicWrite(dst, out); e != nil {
			return paths, e
		}
		if e = os.Remove(src); e != nil {
			return paths, e
		}
		paths = append(paths, m.From, m.To)
		if m.FilesDir != "" {
			from, e := s.diskPath(m.FilesDir)
			if e != nil {
				return paths, e
			}
			to, e := s.diskPath(FolderFor(m.ID))
			if e != nil {
				return paths, e
			}
			if _, e := os.Lstat(to); e == nil {
				return paths, &Error{Kind: "path-exists", ID: m.ID, Value: FolderFor(m.ID), Fix: "hug task lint"}
			}
			if e = os.Rename(from, to); e != nil {
				return paths, e
			}
			paths = append(paths, m.FilesDir, FolderFor(m.ID))
		}
	}
	// A group folder left with nothing in it goes (git keeps no empty
	// folders; on disk it would read as a group still).
	root, e := filepath.Abs(s.Root)
	if e != nil {
		return paths, e
	}
	for _, g := range p.Groups {
		dir, e := s.diskPath(g)
		if e != nil {
			continue
		}
		for d := dir; strings.HasPrefix(d, root+string(filepath.Separator)); d = filepath.Dir(d) {
			if os.Remove(d) != nil {
				break
			}
		}
	}
	s.Invalidate()
	return paths, nil
}
