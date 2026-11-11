package app_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/veridit/muninn/internal/config"
	"github.com/veridit/muninn/internal/harness"
	"github.com/veridit/muninn/internal/project"
	"github.com/veridit/muninn/internal/scopetest"
	"github.com/veridit/muninn/internal/store"
	"github.com/veridit/muninn/internal/tidy"
)

func TestClientRootAndLegacyFolderVisibility(t *testing.T) {
	f := scopetest.New(t)
	home, err := filepath.EvalSymlinks(f.Root)
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home) // os.UserHomeDir on Windows
	root := filepath.Join(home, "client")
	a := makePlacementRepo(t, filepath.Join(root, "project-a"))
	b := makePlacementRepo(t, filepath.Join(root, "project-b"))
	nestedProject := makePlacementRepo(t, filepath.Join(a.Root, "nested-project"))
	sub := filepath.Join(a.Root, "sub")
	folder := filepath.Join(root, "plain'folder")
	child := filepath.Join(folder, "child")
	alias := filepath.Join(root, "alias-a")
	for _, dir := range []string{sub, child, filepath.Join(root, ".jcode", "memory", "projects")} {
		if err := os.MkdirAll(dir, 0700); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Symlink(a.Root, alias); err != nil {
		t.Fatal(err)
	}
	crossAlias := filepath.Join(b.Root, "link-to-a-sub")
	if err := os.Symlink(sub, crossAlias); err != nil {
		t.Fatal(err)
	}
	native := f.Units["Client"].SourceID
	units := map[string]store.Unit{}
	effective := map[string]string{}
	add := func(name, scope, path, kind, want string) {
		t.Helper()
		source := native
		if path != "" {
			if err := os.WriteFile(path, []byte("{}"), 0600); err != nil {
				t.Fatal(err)
			}
			var err error
			source, _, _, err = f.App.DB.Source(kind, path)
			if err != nil {
				t.Fatal(err)
			}
		}
		id, err := f.App.DB.UpsertUnit(store.Unit{SourceID: source, Key: name, Scope: scope, Body: "quartz deployment policy " + name, Active: true, Kind: "memory"})
		if err != nil {
			t.Fatal(err)
		}
		units[name], err = f.App.DB.GetUnit(id)
		if err != nil {
			t.Fatal(err)
		}
		effective[name] = want
	}
	legacy := func(name, path, want string) {
		hash := harness.JcodeProjectHash(path)
		add(name, "jcode@old-name:"+hash, filepath.Join(root, ".jcode", "memory", "projects", hash+".json"), "jcode", want)
	}
	add("top", "client:acme", "", "", "client:acme")
	add("native-a", a.Scope(), "", "", a.Scope())
	add("native-b", b.Scope(), "", "", b.Scope())
	folderScope := "client:acme:folder:" + folder
	add("native-folder", folderScope, "", "", folderScope)
	add("other-client", "client:other", "", "", "client:other")
	add("user-global", "global", "", "", "global")
	legacy("legacy-root", root, "client:acme")
	legacy("legacy-unknown", filepath.Join(root, "gone"), "client:acme")
	legacy("legacy-a", a.Root, a.Scope())
	legacy("legacy-a-sub", sub, a.Scope())
	legacy("legacy-a-alias", alias, a.Scope())
	legacy("legacy-a-tilde", "~/client/project-a", a.Scope())
	rootAlias := filepath.Join(home, "client-link")
	if err := os.Symlink(root, rootAlias); err != nil {
		t.Fatal(err)
	}
	aliasHash := harness.JcodeProjectHash(filepath.Join(rootAlias, "project-a"))
	add("legacy-root-alias", "jcode@old-name:"+aliasHash, filepath.Join(rootAlias, ".jcode", "memory", "projects", aliasHash+".json"), "jcode", a.Scope())
	fileAlias := filepath.Join(home, "external-source.json")
	if err := os.Symlink(filepath.Join(root, ".jcode", "memory", "projects", harness.JcodeProjectHash(a.Root)+".json"), fileAlias); err != nil {
		t.Fatal(err)
	}
	add("legacy-file-alias", "jcode@old-name", fileAlias, "jcode", a.Scope())
	add("hashed-markdown", "notes", filepath.Join(root, ".jcode", "memory", "projects", harness.JcodeProjectHash(filepath.Join(root, "markdown-only"))+".json"), "markdown", "client:acme")
	legacy("legacy-b", b.Root, b.Scope())
	legacy("legacy-folder", folder, folderScope)
	// A non-jcode source that happens to have a hashed name is not a folder store.
	add("client-note", "notes", filepath.Join(root, "note.md"), "markdown", "client:acme")
	if err := f.App.RebuildLegacyFolders(context.Background()); err != nil {
		t.Fatal(err)
	}
	top := []string{"top", "legacy-root", "legacy-unknown", "client-note", "hashed-markdown"}
	ownA := []string{"native-a", "legacy-a", "legacy-a-sub", "legacy-a-alias", "legacy-a-tilde", "legacy-root-alias", "legacy-file-alias"}
	ownB := []string{"native-b", "legacy-b"}
	ownFolder := []string{"native-folder", "legacy-folder"}
	tests := []struct {
		name, cwd string
		groups    [][]string
	}{
		{"root", root, [][]string{top, ownA, ownB, ownFolder}},
		{"project", a.Root, [][]string{top, ownA}},
		{"project-child", sub, [][]string{top, ownA}},
		{"cross-project-symlink", crossAlias, [][]string{top, ownA}},
		{"nested-project", nestedProject.Root, [][]string{top}},
		{"sibling", b.Root, [][]string{top, ownB}},
		{"folder", folder, [][]string{top, ownFolder}},
		{"folder-child", child, [][]string{top, ownFolder}},
		{"other-client", filepath.Join(f.Root, "other"), [][]string{{"other-client"}}},
		{"global", f.Cwd, [][]string{{"user-global"}}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			binding := f.App.Bound(filepath.Join(f.Root, ".jcode"), tc.cwd)
			if binding.Err != nil {
				t.Fatal(binding.Err)
			}
			want := map[string]bool{}
			for _, group := range tc.groups {
				for _, name := range group {
					want[name] = true
				}
			}
			where, args := binding.SQL("u.scope", "u.source_id")
			for name, u := range units {
				shown, visible := binding.Present(u)
				if visible != want[name] {
					t.Errorf("%s visible=%t want=%t scope=%s", name, visible, want[name], shown.Scope)
				}
				if visible && shown.Scope != effective[name] {
					t.Errorf("%s scope=%s want=%s", name, shown.Scope, effective[name])
				}
				var n int
				err := f.App.DB.QueryRow("SELECT count(*) FROM units u WHERE id=? AND ("+where+")", append([]any{u.ID}, args...)...).Scan(&n)
				if err != nil {
					t.Fatal(err)
				}
				if (n == 1) != visible {
					t.Errorf("SQL/Go mismatch %s SQL=%d Go=%t", name, n, visible)
				}
				if visible {
					var scope string
					err = f.App.DB.QueryRow("SELECT "+binding.ScopeExpr("u.scope", "u.source_id")+" FROM units u WHERE id=?", u.ID).Scan(&scope)
					if err != nil || scope != shown.Scope {
						t.Fatalf("SQL effective %s=%q want=%q err=%v", name, scope, shown.Scope, err)
					}
				}
			}
			if tc.name == "root" {
				place, err := binding.Placement("", "here")
				if err != nil || place.Scope != "client:acme" {
					t.Fatalf("root placement %+v %v", place, err)
				}
				if _, err := binding.WriteScope(a.Scope()); err == nil {
					t.Fatal("root reads widened writes")
				}
			}
			scopetest.Pass(t, "client-root-legacy.SQL-Go", tc.name)
		})
	}
	// Source paths are unique in the index, but their importer kind can change.
	// Refresh must not reuse a previous kind's folder-store classification.
	for _, tc := range []struct {
		kind, scope string
	}{
		{"markdown", "client:acme"},
		{"jcode", b.Scope()},
	} {
		if _, err := f.App.DB.Exec("UPDATE sources SET kind=? WHERE id=?", tc.kind, units["legacy-b"].SourceID); err != nil {
			t.Fatal(err)
		}
		binding := f.App.Bound(filepath.Join(home, ".jcode"), root)
		if scope, ok := binding.ScopeFor(units["legacy-b"]); !ok || scope != tc.scope {
			t.Fatalf("changed source kind=%s scope=%s visible=%t want=%s", tc.kind, scope, ok, tc.scope)
		}
	}
	// Tidy uses exactly the same resolver without changing stored scopes or ids.
	graph, err := tidy.ReadGraph(context.Background(), f.App.DB.Path)
	if err != nil {
		t.Fatal(err)
	}
	for name, u := range units {
		if graph.EffectiveScopes[u.ID] != effective[name] {
			t.Errorf("tidy %s=%s want=%s", name, graph.EffectiveScopes[u.ID], effective[name])
		}
	}
	for name, u := range units {
		stored, err := f.App.DB.GetUnit(u.ID)
		if err != nil || stored.Scope != u.Scope || stored.SID != u.SID || stored.Key != u.Key {
			t.Fatalf("identity changed %s: %+v %v", name, stored, err)
		}
	}
	if _, err := project.Find(folder, false); err == nil {
		t.Fatal("plain folder identity was created")
	}
	scopetest.Pass(t, "client-root-legacy.tidy-identity", "all")
	cachePath := filepath.Join(config.DataDir(), project.LegacyCacheFile)
	if err := os.Remove(cachePath); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		cwd  string
		sees map[string]bool
	}{
		{root, map[string]bool{"legacy-a": true, "legacy-b": true, "legacy-folder": true, "legacy-unknown": true, "legacy-root": true}},
		{a.Root, map[string]bool{"legacy-a": true, "legacy-root": true}},
		{sub, map[string]bool{"legacy-a": true, "legacy-a-sub": true, "legacy-root": true}},
		{crossAlias, map[string]bool{"legacy-a": true, "legacy-a-sub": true, "legacy-root": true}},
		{nestedProject.Root, map[string]bool{"legacy-root": true}},
		{folder, map[string]bool{"legacy-folder": true, "legacy-root": true}},
	} {
		binding := f.App.Bound(filepath.Join(home, ".jcode"), tc.cwd)
		where, args := binding.SQL("u.scope", "u.source_id")
		for _, name := range []string{"legacy-a", "legacy-b", "legacy-folder", "legacy-unknown", "legacy-root"} {
			u := units[name]
			_, visible := binding.Present(u)
			if visible != tc.sees[name] {
				t.Errorf("cache absent cwd=%s %s visible=%t", tc.cwd, name, visible)
			}
			var n int
			if err := f.App.DB.QueryRow("SELECT count(*) FROM units u WHERE id=? AND ("+where+")", append([]any{u.ID}, args...)...).Scan(&n); err != nil || (n == 1) != visible {
				t.Fatalf("absent SQL/Go %s %d %t %v", name, n, visible, err)
			}
		}
	}
	if _, err := os.Stat(cachePath); !os.IsNotExist(err) {
		t.Fatal("read path regenerated cache", err)
	}
	scopetest.Pass(t, "client-root-legacy.missing-cache", "root/project/folder")
}
