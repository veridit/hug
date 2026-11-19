package harness

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ynh84t check 1: hug mem install writes hook commands that run hug
// (`hug mem hook claude|codex|jcode`) and one MCP server named hug. A
// config an older hug wrote (muninn hook ..., a "muninn" server) is
// replaced in place, never duplicated, and uninstall takes off either form.
func TestInstallWritesHugForms(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	hug := "/opt/homebrew/bin/hug"
	settings := filepath.Join(home, ".claude", "settings.json")
	os.MkdirAll(filepath.Dir(settings), 0o700)
	old := `{"hooks":{"UserPromptSubmit":[{"hooks":[{"type":"command","command":"/Users/x/.local/bin/muninn hook claude","timeout":15}]},{"hooks":[{"type":"command","command":"~/bin/mine.sh"}]}]}}`
	os.WriteFile(settings, []byte(old), 0o600)
	if _, err := InstallClaude(hug, true, false); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(settings)
	if !strings.Contains(string(b), `"command": "/opt/homebrew/bin/hug mem hook claude"`) || strings.Contains(string(b), "muninn") || !strings.Contains(string(b), "mine.sh") {
		t.Fatalf("claude settings:\n%s", b)
	}
	if strings.Count(string(b), "hook claude") != 1 {
		t.Fatalf("the hook is there twice:\n%s", b)
	}
	if r, _ := InstallClaude(hug, true, false); r.Changed {
		t.Fatalf("second install changed it: %v", r)
	}
	if _, err := InstallCodex(hug, true, false); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(filepath.Join(home, ".codex", "hooks.json")); !strings.Contains(string(b), `"/opt/homebrew/bin/hug mem hook codex"`) {
		t.Fatalf("codex hooks:\n%s", b)
	}

	jh := filepath.Join(home, ".jcode")
	os.MkdirAll(jh, 0o700)
	cfg := "[hooks]\n# muninn: automatic local memory recall\nturn_start = \"/Users/x/.local/bin/muninn hook jcode\"\n" +
		"turn_end = [\"~/.jcode/bin/disk_turn_end.sh\",\"/Users/x/.local/bin/muninn capture --hook\",\"/Users/x/.local/bin/muninn hook jcode\"]\n"
	os.WriteFile(filepath.Join(jh, "config.toml"), []byte(cfg), 0o600)
	mcp := `{"servers":{"muninn":{"command":"/Users/x/.local/bin/muninn","args":["mcp","--jcode-home","` + filepath.ToSlash(jh) + `"],"env":{}},"vikunja":{"type":"http","url":"https://example.invalid/mcp"}}}`
	os.WriteFile(filepath.Join(jh, "mcp.json"), []byte(mcp), 0o600)
	if _, err := InstallJcodeHome(jh, "jcode", hug, true, false); err != nil {
		t.Fatal(err)
	}
	b, _ = os.ReadFile(filepath.Join(jh, "config.toml"))
	for _, want := range []string{
		`turn_start = "/opt/homebrew/bin/hug mem hook jcode"`,
		`post_tool_feedback = "/opt/homebrew/bin/hug mem hook jcode"`,
		// capture is not the recall hook: install leaves it; upgrade moves it.
		`turn_end = ["~/.jcode/bin/disk_turn_end.sh","/Users/x/.local/bin/muninn capture --hook","/opt/homebrew/bin/hug mem hook jcode"]`,
		"# hug: automatic local memory recall",
	} {
		if !strings.Contains(string(b), want) {
			t.Errorf("config.toml lacks %s:\n%s", want, b)
		}
	}
	if strings.Contains(string(b), "muninn hook") || strings.Contains(string(b), "# muninn:") {
		t.Errorf("old recall hook left:\n%s", b)
	}
	var doc struct {
		Servers map[string]json.RawMessage `json:"servers"`
	}
	raw, _ := os.ReadFile(filepath.Join(jh, "mcp.json"))
	json.Unmarshal(raw, &doc)
	if _, ok := doc.Servers["muninn"]; ok || doc.Servers["vikunja"] == nil {
		t.Fatalf("mcp.json servers:\n%s", raw)
	}
	var e struct {
		Command string   `json:"command"`
		Args    []string `json:"args"`
	}
	json.Unmarshal(doc.Servers["hug"], &e)
	if e.Command != hug || strings.Join(e.Args, " ") != "mcp --jcode-home "+jh {
		t.Fatalf("hug server: %s", doc.Servers["hug"])
	}
	if !JcodeHookInstalled(jh) {
		t.Fatal("JcodeHookInstalled does not see the hug hook")
	}
	if r, _ := InstallJcodeHome(jh, "jcode", hug, true, false); r.Changed {
		t.Fatalf("second jcode install changed it: %v", r)
	}
	if _, err := InstallJcodeHome(jh, "jcode", hug, false, false); err != nil {
		t.Fatal(err)
	}
	b, _ = os.ReadFile(filepath.Join(jh, "config.toml"))
	raw, _ = os.ReadFile(filepath.Join(jh, "mcp.json"))
	if strings.Contains(string(b), "hook jcode") || strings.Contains(string(raw), `"hug"`) || !strings.Contains(string(raw), "vikunja") {
		t.Fatalf("uninstall left:\n%s\n%s", b, raw)
	}
}
