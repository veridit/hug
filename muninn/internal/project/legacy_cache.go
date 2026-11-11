package project

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const LegacyCacheFile = "legacy-folders.json"
const legacyCacheVersion = 1
const legacyCacheMaxAge = 24 * time.Hour

// LegacySource is derived provenance, never a memory or a synced artifact.
// An empty Dir records a hash that was unresolved at the last background walk.
type LegacySource struct {
	Path   string `json:"path"`
	Kind   string `json:"kind"`
	Client string `json:"client"`
	Root   string `json:"root"`
	Hash   string `json:"hash"`
	Dir    string `json:"dir,omitempty"`
	Scope  string `json:"scope"`
}

type LegacyCache struct {
	Version int                     `json:"version"`
	BuiltAt time.Time               `json:"built_at"`
	Sources map[string]LegacySource `json:"sources"`
}

// LoadLegacyCache is read-only. Missing, corrupt or expired derived state is
// indistinguishable from no cache. Hooks never repair or regenerate it.
func LoadLegacyCache(dataDir string) *LegacyCache {
	data, err := os.ReadFile(filepath.Join(dataDir, LegacyCacheFile))
	if err != nil || len(data) > 8<<20 {
		return nil
	}
	var c LegacyCache
	if json.Unmarshal(data, &c) != nil || c.Version != legacyCacheVersion || c.BuiltAt.IsZero() || time.Since(c.BuiltAt) > legacyCacheMaxAge || c.BuiltAt.After(time.Now().Add(time.Minute)) {
		return nil
	}
	return &c
}

// LegacyStoreHash recognizes jcode's physical memory/projects/<hash>.json.
func LegacyStoreHash(path, kind string) string {
	if kind != "jcode" {
		return ""
	}
	if physical, err := filepath.EvalSymlinks(path); err == nil {
		path = physical
	}
	if filepath.Base(filepath.Dir(path)) != "projects" || filepath.Base(filepath.Dir(filepath.Dir(path))) != "memory" || filepath.Ext(path) != ".json" {
		return ""
	}
	hash := strings.TrimSuffix(filepath.Base(path), ".json")
	if len(hash) != 16 || strings.Trim(hash, "0123456789abcdef") != "" {
		return ""
	}
	return hash
}

// ClientSourceScope is the common root/owner and nested-session read policy.
// Own contains only cheaply computed cwd/root hashes, with no walk or git probe.
// A missing cache under-shows unresolved stores in nested sessions, never leaks
// a sibling's store by guessing that its unknown hash is client-wide.
func ClientSourceScope(c *LegacyCache, path, kind, client, root, hash string, all bool, own map[string]string) (string, bool) {
	top := "client:" + client
	if hash == "" {
		return top, true
	}
	if scope, ok := own[hash]; ok {
		return scope, true
	}
	if c != nil {
		if e, ok := c.Sources[path]; ok && e.Path == path && e.Kind == kind && e.Client == client && e.Root == root && e.Hash == hash && (e.Scope == top || strings.HasPrefix(e.Scope, top+":")) {
			if e.Dir != "" {
				rel, err := filepath.Rel(root, e.Dir)
				if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
					return "", false
				}
				if fi, err := os.Stat(e.Dir); err != nil || !fi.IsDir() {
					return top, true
				}
				if name, boundary, err := ClientRoot(e.Dir); err != nil || name != client || boundary != root {
					return top, true
				}
			}
			return e.Scope, true
		}
	}
	if all {
		return top, true
	}
	return "", false
}

// BuildLegacyCache runs only in explicit background maintenance (index/serve).
// Sources are caller-supplied index provenance, not discovered harness contents.
func BuildLegacyCache(sources []LegacySource) (*LegacyCache, error) {
	c := &LegacyCache{Version: legacyCacheVersion, BuiltAt: time.Now().UTC(), Sources: map[string]LegacySource{}}
	var resolver ClientScopes
	for _, s := range sources {
		s.Hash = LegacyStoreHash(s.Path, s.Kind)
		if s.Hash == "" {
			continue
		}
		var err error
		s.Client, s.Root, err = SourceClientRoot(s.Path)
		if err != nil {
			return nil, err
		}
		if s.Client == "" {
			continue
		}
		s.Scope = resolver.Scope(s.Path, s.Kind, s.Client, s.Root)
		alias := sourceRootAlias(s.Path, s.Root)
		s.Dir = resolver.folders[s.Root+"\x00"+alias][s.Hash]
		c.Sources[s.Path] = s
	}
	return c, nil
}

// Save atomically replaces one owner-only derived file. Deleting it is safe.
func (c *LegacyCache) Save(dataDir string) error {
	path := filepath.Join(dataDir, LegacyCacheFile)
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	if previous, err := os.ReadFile(path); err == nil && bytes.Equal(previous, data) {
		return nil
	}
	if err := os.MkdirAll(dataDir, 0700); err != nil {
		return err
	}
	f, err := os.CreateTemp(dataDir, ".legacy-folders-*")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	if _, err = f.Write(data); err != nil {
		f.Close()
		return err
	}
	if err = f.Sync(); err != nil {
		f.Close()
		return err
	}
	if err = f.Close(); err != nil {
		return err
	}
	return os.Rename(f.Name(), path)
}
