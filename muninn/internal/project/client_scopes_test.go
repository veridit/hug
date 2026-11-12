package project

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/veridit/hug/muninn/internal/harness"
)

func TestClientFoldersBoundsAndSymlinks(t *testing.T) {
	root := t.TempDir()
	var err error
	root, err = filepath.EvalSymlinks(root)
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", root)
	t.Setenv("USERPROFILE", root) // os.UserHomeDir on Windows
	for _, path := range []string{"visible/child/grandchild", "visible/child/grandchild/too-deep", ".git/hidden", "node_modules/hidden", "vendor/hidden"} {
		if err := os.MkdirAll(filepath.Join(root, path), 0700); err != nil {
			t.Fatal(err)
		}
	}
	outside := t.TempDir()
	if err := os.Symlink(outside, filepath.Join(root, "outside")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(root, "visible"), filepath.Join(root, "alias")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(root, filepath.Join(root, "visible", "cycle")); err != nil {
		t.Fatal(err)
	}
	folders := clientFolders(root)
	for _, path := range []string{"visible", "visible/child", "visible/child/grandchild", "alias", "alias/child"} {
		dir := filepath.Join(root, path)
		physical, err := filepath.EvalSymlinks(dir)
		if err != nil {
			t.Fatal(err)
		}
		for _, spelling := range []string{dir, "~/" + path} {
			if got := folders[harness.JcodeProjectHash(spelling)]; got != physical {
				t.Errorf("%s = %s want %s", spelling, got, physical)
			}
		}
	}
	for _, path := range []string{"visible/child/grandchild/too-deep", ".git", ".git/hidden", "node_modules", "node_modules/hidden", "vendor", "outside", "visible/cycle"} {
		if got := folders[harness.JcodeProjectHash(filepath.Join(root, path))]; got != "" {
			t.Errorf("out-of-bounds folder %s resolved as %s", path, got)
		}
	}
}

func TestClientScopesKeepUnknownAndNestedBoundaryAtTop(t *testing.T) {
	root := t.TempDir()
	root, _ = filepath.EvalSymlinks(root)
	marker := func(dir, name string) {
		t.Helper()
		if err := os.MkdirAll(filepath.Join(dir, ".muninn"), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, ".muninn", "client"), []byte(name), 0600); err != nil {
			t.Fatal(err)
		}
	}
	marker(root, "acme")
	nested := filepath.Join(root, "nested")
	marker(nested, "other")
	plain := filepath.Join(root, "plain")
	os.MkdirAll(plain, 0700)
	var resolver ClientScopes
	for _, dir := range []string{root, nested, filepath.Join(root, "missing")} {
		path := filepath.Join(root, ".jcode", "memory", "projects", harness.JcodeProjectHash(dir)+".json")
		if got := resolver.Scope(path, "jcode", "acme", root); got != "client:acme" {
			t.Errorf("%s: %s", dir, got)
		}
	}
	path := filepath.Join(root, ".jcode", "memory", "projects", harness.JcodeProjectHash(plain)+".json")
	if got := resolver.Scope(path, "jcode", "acme", root); got != "client:acme:folder:"+plain {
		t.Fatal(got)
	}
	if got := resolver.Scope(path, "markdown", "acme", root); got != "client:acme" {
		t.Fatal("non-jcode source reclassified", got)
	}
	if _, err := os.Stat(filepath.Join(plain, ".muninn")); !os.IsNotExist(err) {
		t.Fatal("resolver wrote project metadata", err)
	}
}
