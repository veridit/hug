package project

import (
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/veridit/muninn/internal/harness"
)

// ClientScopes resolves legacy jcode folder stores without changing the index.
// Directory/hash and effective-scope caches belong to this resolver's process
// lifetime. Callers can resolve only their own client, keeping global hooks and
// other clients independent of a large client's directory tree.
type ClientScopes struct {
	mu       sync.Mutex
	folders  map[string]map[string]string
	scopes   map[string]string
	projects map[string]string
}

// SourceClientRoot is SourceClient with the physical boundary included.
func SourceClientRoot(path string) (client, root string, err error) {
	if physical, e := filepath.EvalSymlinks(path); e == nil {
		path = physical
	}
	return ClientRoot(filepath.Dir(path))
}

// Scope overlays a non-native source's client boundary. Only jcode's
// memory/projects/<hash>.json stores have folder ownership. Root, unknown and
// deleted folders retain the client top level, as do all other source kinds.
// An identity lookup never creates a marker or writes a harness file.
func (r *ClientScopes) Scope(path, kind, client, root string) string {
	top := "client:" + client
	if kind != "jcode" {
		return top
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.scopes == nil {
		r.scopes = map[string]string{}
	}
	key := top + "\x00" + root + "\x00" + path
	if scope, ok := r.scopes[key]; ok {
		return scope
	}
	original := path
	if physical, err := filepath.EvalSymlinks(path); err == nil {
		path = physical
	}
	if filepath.Base(filepath.Dir(path)) != "projects" || filepath.Base(filepath.Dir(filepath.Dir(path))) != "memory" || filepath.Ext(path) != ".json" {
		r.scopes[key] = top
		return top
	}
	hash := strings.TrimSuffix(filepath.Base(path), ".json")
	if len(hash) != 16 || strings.Trim(hash, "0123456789abcdef") != "" {
		r.scopes[key] = top
		return top
	}
	if r.folders == nil {
		r.folders = map[string]map[string]string{}
	}
	alias := sourceRootAlias(original, root)
	folderKey := root + "\x00" + alias
	folders, ok := r.folders[folderKey]
	if !ok {
		folders = clientFolders(root, alias)
		r.folders[folderKey] = folders
	}
	scope := top
	if dir := folders[hash]; dir != "" && dir != root {
		// A nested client's folder or an invalid marker never becomes this
		// client's project, even if the parent walk encountered its hash.
		if name, boundary, err := ClientRoot(dir); err == nil && name == client && boundary == root {
			scope = r.folderScope(dir, root, client)
		}
	}
	r.scopes[key] = scope
	return scope
}

// Recover the lexical root spelling from a source under a symlinked client.
// An arbitrary file symlink outside a client is not a root alias.
func sourceRootAlias(path, root string) string {
	if strings.HasPrefix(path, root+string(filepath.Separator)) {
		return ""
	}
	for d := filepath.Dir(path); ; d = filepath.Dir(d) {
		if physical, err := filepath.EvalSymlinks(d); err == nil && physical == root {
			return d
		}
		if filepath.Dir(d) == d {
			return ""
		}
	}
}

// Match the same identity as Find, sharing git identities between hashes
// within one repository. Plain folders retain Find's existing fallback rules.
func (r *ClientScopes) folderScope(dir, root, client string) string {
	if r.projects == nil {
		r.projects = map[string]string{}
	}
	key := "client:" + client + "\x00" + dir
	for d := dir; ; d = filepath.Dir(d) {
		if _, err := os.Stat(filepath.Join(d, ".git")); err == nil {
			key = "client:" + client + "\x00" + d
			break
		}
		if d == root || filepath.Dir(d) == d {
			break
		}
	}
	if scope, ok := r.projects[key]; ok {
		return scope
	}
	if p, err := Find(dir, false); err == nil && p.Root != root {
		r.projects[key] = p.Scope()
		return p.Scope()
	}
	return "client:" + client + ":folder:" + dir
}

// clientFolders bounds both depth (three levels below the client root) and
// work (16,384 entries). Symlinks are hashed in their original spelling and
// physical spelling, but followed only inside this root and never in a cycle.
// Unresolved/out-of-budget hashes intentionally retain client-top ownership.
func clientFolders(root string, aliases ...string) map[string]string {
	out := map[string]string{}
	home, _ := os.UserHomeDir()
	addPath := func(path, physical string) {
		out[harness.JcodeProjectHash(path)] = physical
		if rel, err := filepath.Rel(home, path); home != "" && err == nil && rel != "." && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			out[harness.JcodeProjectHash("~/"+filepath.ToSlash(rel))] = physical
		}
	}
	add := func(path, physical string) {
		addPath(path, physical)
		for _, alias := range aliases {
			if alias == "" {
				continue
			}
			if rel, err := filepath.Rel(root, path); err == nil {
				addPath(filepath.Join(alias, rel), physical)
			}
		}
	}
	budget := 16384
	type folder struct {
		path, physical string
		depth          int
	}
	// Breadth first: a large repository cannot exhaust the budget before
	// its sibling repositories' root folders have been named.
	queue := []folder{{root, root, 0}}
	add(root, root)
	for len(queue) > 0 && budget > 0 {
		f := queue[0]
		queue = queue[1:]
		if f.depth == 3 {
			continue
		}
		dir, err := os.Open(f.path)
		if err != nil {
			continue
		}
		for budget > 0 {
			entries, err := dir.ReadDir(min(256, budget))
			for _, e := range entries {
				budget--
				switch e.Name() {
				case ".git", "node_modules", "vendor", ".muninn", ".muninn-notes":
					continue
				}
				next := filepath.Join(f.path, e.Name())
				target := filepath.Join(f.physical, e.Name())
				if e.Type()&os.ModeSymlink != 0 {
					info, err := os.Stat(next)
					if err != nil || !info.IsDir() {
						continue
					}
					target, err = filepath.EvalSymlinks(next)
					if err != nil || !strings.HasPrefix(target, root+string(filepath.Separator)) || target == f.physical || strings.HasPrefix(f.physical, target+string(filepath.Separator)) {
						continue
					}
				} else if !e.IsDir() {
					continue
				}
				// Hash immediately, including folders queued just as the budget
				// runs out. Traversal, not ownership, is bounded by the queue.
				add(next, target)
				if next != target {
					add(target, target)
				}
				queue = append(queue, folder{next, target, f.depth + 1})
			}
			if err != nil {
				break
			}
		}
		dir.Close()
	}
	return out
}
