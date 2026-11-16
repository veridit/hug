package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAppsInstall(t *testing.T) {
	dir := t.TempDir()
	js := filepath.Join(dir, "claude.json")
	os.WriteFile(js, []byte(`{"other":1,"mcpServers":{"muninn":{"command":"x"},"keep":{"command":"k"}}}`), 0o600)
	tm := filepath.Join(dir, "config.toml")
	os.WriteFile(tm, []byte("model = \"x\"\n\n[mcp_servers.keep]\ncommand = \"k\"\n"), 0o600)
	for _, tg := range []appTarget{{Name: "Claude Code", Path: js, kind: "json"}, {Name: "Codex", Path: tm, kind: "toml"}} {
		if got := inspectApp(tg, "/bin/hug"); got.Status != "missing" {
			t.Fatal(tg.Name, got.Status)
		}
		if msg, e := installApp(tg, "/bin/hug", true); e != nil || msg != "would update" {
			t.Fatal(msg, e)
		}
		before, _ := os.ReadFile(tg.Path)
		if _, e := installApp(tg, "/bin/hug", false); e != nil {
			t.Fatal(e)
		}
		after, _ := os.ReadFile(tg.Path)
		if string(before) == string(after) {
			t.Fatal("unchanged", tg.Name)
		}
		if got := inspectApp(tg, "/bin/hug"); got.Status != "ok" {
			t.Fatal(tg.Name, "after install", got.Status, string(after))
		}
		if !strings.Contains(string(after), "keep") {
			t.Fatal("other server lost", string(after))
		}
		backups, _ := filepath.Glob(tg.Path + ".bak-*")
		if len(backups) != 1 {
			t.Fatal("backup", backups)
		}
	}
	b, _ := os.ReadFile(js)
	if strings.Contains(string(b), "muninn") || !strings.Contains(string(b), `"other": 1`) {
		t.Fatal(string(b))
	}
}
