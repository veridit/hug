package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/veridit/muninn/internal/app"
	"github.com/veridit/muninn/internal/config"
	"github.com/veridit/muninn/internal/scopetest"
	"github.com/veridit/muninn/internal/tidy"
)

func TestPlaceCLIToDryApplyReadOnly(t *testing.T) {
	f := scopetest.New(t)
	home := filepath.Join(t.TempDir(), "owner")
	t.Setenv("HOME", home)
	os.MkdirAll(filepath.Join(home, ".jcode"), 0700)
	root := filepath.Join(home, "Projects", "example")
	os.MkdirAll(filepath.Join(root, ".git"), 0700)
	os.MkdirAll(filepath.Join(root, ".muninn"), 0700)
	os.WriteFile(filepath.Join(root, ".muninn", "project"), []byte("example"), 0600)
	os.WriteFile(filepath.Join(root, "README.md"), []byte("# Example project\n"), 0600)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := json.Marshal([]map[string]any{{"sid": f.Units["Shared"].SID, "target": "project:example", "reason": "This is specific to Example", "safe_reason": "", "client_mention": false, "personal_preference": false, "multiple_projects": false, "dominant_project": false}})
		json.NewEncoder(w).Encode(map[string]any{"choices": []any{map[string]any{"message": map[string]string{"content": string(raw)}, "finish_reason": "stop"}}})
	}))
	defer srv.Close()
	cfg, _ := config.Load()
	cfg.Extract.Provider, cfg.Extract.BaseURL = "ollama", srv.URL
	if err := config.Save(cfg); err != nil {
		t.Fatal(err)
	}
	ids := filepath.Join(t.TempDir(), "sids.tsv")
	os.WriteFile(ids, []byte(f.Units["Shared"].SID+"\tignored preview\n"), 0600)
	path := filepath.Join(t.TempDir(), "place.json")
	before, _ := os.ReadFile(app.NativePath())
	dbBefore, _ := os.ReadFile(config.DBPath())
	cmd := tidyCmd()
	out := &bytes.Buffer{}
	cmd.SetOut(out)
	cmd.SetErr(out)
	cmd.SetArgs([]string{"place", "--sids", ids, "--from-scope", "global", "--out", path})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	p, err := tidy.Load(path)
	if err != nil || len(p.Entries) != 1 || p.Entries[0].Decision != "move" {
		t.Fatalf("plan=%+v err=%v", p, err)
	}
	cmd = tidyCmd()
	cmd.SetOut(out)
	cmd.SetArgs([]string{"apply", path, "--dry-run"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "Dry run: 1 moves, no writes.") {
		t.Fatal(out.String())
	}
	after, _ := os.ReadFile(app.NativePath())
	dbAfter, _ := os.ReadFile(config.DBPath())
	if !bytes.Equal(before, after) || !bytes.Equal(dbBefore, dbAfter) {
		t.Fatal("placement or dry apply wrote store")
	}
	for _, artifact := range []string{path, path + ".md"} {
		fi, err := os.Stat(artifact)
		if err != nil || fi.Mode().Perm() != 0600 {
			t.Fatal("review artifact not private")
		}
	}
	cmd = tidyCmd()
	cmd.SetArgs([]string{"place", "--out", path})
	if err := cmd.Execute(); err == nil {
		t.Fatal("existing plan accepted")
	}
}

func TestPlaceCLIRequiresOutputAndNonemptySIDFile(t *testing.T) {
	cmd := tidyCmd()
	cmd.SetArgs([]string{"place"})
	if err := cmd.Execute(); err == nil || !strings.Contains(err.Error(), "--out") {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "empty")
	os.WriteFile(path, []byte("# no SIDs\n"), 0600)
	if _, err := readPlaceSIDs(path); err == nil {
		t.Fatal("empty SID file would select all memories")
	}
}
