package project

import (
	"github.com/veridit/muninn/internal/testutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/veridit/muninn/internal/harness"
)

func TestLegacyCacheLifecycleAndReadFallback(t *testing.T) {
	root := t.TempDir()
	root, _ = filepath.EvalSymlinks(root)
	for _, dir := range []string{filepath.Join(root, ".muninn"), filepath.Join(root, "plain"), filepath.Join(root, ".jcode", "memory", "projects")} {
		if err := os.MkdirAll(dir, 0700); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(root, ".muninn", "client"), []byte("acme"), 0600); err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(root, "plain")
	hash := harness.JcodeProjectHash(dir)
	path := filepath.Join(root, ".jcode", "memory", "projects", hash+".json")
	unknownHash := harness.JcodeProjectHash(filepath.Join(root, "gone"))
	unknownPath := filepath.Join(filepath.Dir(path), unknownHash+".json")
	sources := []LegacySource{{Path: path, Kind: "jcode"}, {Path: unknownPath, Kind: "jcode"}}
	cache, err := BuildLegacyCache(sources)
	if err != nil {
		t.Fatal(err)
	}
	dataDir := t.TempDir()
	if err := cache.Save(dataDir); err != nil {
		t.Fatal(err)
	}
	saved := filepath.Join(dataDir, LegacyCacheFile)
	testutil.AssertPrivate(t, saved, "legacy cache")
	loaded := LoadLegacyCache(dataDir)
	if loaded == nil {
		t.Fatal("saved cache missing")
	}
	scope, ok := ClientSourceScope(loaded, path, "jcode", "acme", root, hash, false, nil)
	if !ok || scope != "client:acme:folder:"+dir {
		t.Fatal(scope, ok)
	}
	scope, ok = ClientSourceScope(loaded, unknownPath, "jcode", "acme", root, unknownHash, false, nil)
	if !ok || scope != "client:acme" {
		t.Fatal("known unresolved not top", scope, ok)
	}
	for _, bad := range []struct{ kind, client, root, hash string }{{"markdown", "acme", root, hash}, {"jcode", "other", root, hash}, {"jcode", "acme", root + "x", hash}, {"jcode", "acme", root, "bad"}} {
		if _, ok := ClientSourceScope(loaded, path, bad.kind, bad.client, bad.root, bad.hash, false, nil); ok {
			t.Fatal("stale provenance accepted", bad)
		}
	}
	// Disappearing folders lose their project mapping at read time. No write is
	// needed on the hook path, and the next background rebuild drops the entry.
	if err := os.Remove(dir); err != nil {
		t.Fatal(err)
	}
	scope, ok = ClientSourceScope(loaded, path, "jcode", "acme", root, hash, false, nil)
	if !ok || scope != "client:acme" {
		t.Fatal("deleted dir not top", scope, ok)
	}
	rebuilt, err := BuildLegacyCache(sources)
	if err != nil {
		t.Fatal(err)
	}
	if rebuilt.Sources[path].Dir != "" || rebuilt.Sources[path].Scope != "client:acme" {
		t.Fatal("deleted dir retained", rebuilt.Sources[path])
	}
	cache.BuiltAt = time.Now().Add(-25 * time.Hour)
	if err := cache.Save(dataDir); err != nil {
		t.Fatal(err)
	}
	if LoadLegacyCache(dataDir) != nil {
		t.Fatal("expired cache accepted")
	}
	if err := os.WriteFile(saved, []byte("{corrupt"), 0600); err != nil {
		t.Fatal(err)
	}
	if LoadLegacyCache(dataDir) != nil {
		t.Fatal("corrupt cache accepted")
	}
	if err := os.Remove(saved); err != nil {
		t.Fatal(err)
	}
	if LoadLegacyCache(dataDir) != nil {
		t.Fatal("absent cache accepted")
	}
	for _, all := range []bool{true, false} {
		scope, ok = ClientSourceScope(nil, path, "jcode", "acme", root, hash, all, nil)
		if ok != all || all && scope != "client:acme" {
			t.Fatal("absent cache boundary", scope, ok, all)
		}
	}
	scope, ok = ClientSourceScope(nil, path, "jcode", "acme", root, hash, false, map[string]string{hash: "client:acme:own"})
	if !ok || scope != "client:acme:own" {
		t.Fatal("own fallback lost", scope, ok)
	}
	if _, err := os.Stat(saved); !os.IsNotExist(err) {
		t.Fatal("read regenerated derived file", err)
	}
}
