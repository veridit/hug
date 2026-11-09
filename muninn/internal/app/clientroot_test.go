package app_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

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
	root := filepath.Join(home, "client")
	a := makePlacementRepo(t, filepath.Join(root, "project-a"))
	b := makePlacementRepo(t, filepath.Join(root, "project-b"))
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
	legacy("legacy-b", b.Root, b.Scope())
	legacy("legacy-folder", folder, folderScope)
	// A non-jcode source that happens to have a hashed name is not a folder store.
	add("client-note", "notes", filepath.Join(root, "note.md"), "markdown", "client:acme")
	top := []string{"top", "legacy-root", "legacy-unknown", "client-note"}
	ownA := []string{"native-a", "legacy-a", "legacy-a-sub", "legacy-a-alias", "legacy-a-tilde"}
	ownB := []string{"native-b", "legacy-b"}
	ownFolder := []string{"native-folder", "legacy-folder"}
	tests := []struct {
		name, cwd string
		groups    [][]string
	}{
		{"root", root, [][]string{top, ownA, ownB, ownFolder}},
		{"project", a.Root, [][]string{top, ownA}},
		{"project-child", sub, [][]string{top, ownA}},
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
}
