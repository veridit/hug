package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// m1c28r check 2: hug upgrade renames old variable names where hug wrote
// them (its hook commands, its MCP server entries) in a fixture home, keeps
// a backup, and leaves the user's own entries alone. Old names are spelt
// in parts: the code holds none.
func TestUpgradeRenamesEnvInHugConfigs(t *testing.T) {
	home := "MUN" + "INN_HOME"
	user := "HUG" + "INN_USER"
	h := t.TempDir()
	t.Setenv("HOME", h)
	t.Setenv("USERPROFILE", h)
	t.Setenv("JCODE_HOME", "")
	write := func(rel, body string) string {
		p := filepath.Join(h, rel)
		if e := os.MkdirAll(filepath.Dir(p), 0o755); e != nil {
			t.Fatal(e)
		}
		if e := os.WriteFile(p, []byte(body), 0o644); e != nil {
			t.Fatal(e)
		}
		return p
	}
	claude := write(".claude/settings.json", `{"hooks":{"UserPromptSubmit":[{"hooks":[{"type":"command","command":"`+home+`=/m /opt/homebrew/bin/muninn hook claude","timeout":15}]},{"hooks":[{"type":"command","command":"`+home+`=/x other-tool"}]}]},"theme":"dark"}`)
	mcp := write(".jcode/mcp.json", `{"servers":{"hug":{"command":"/opt/homebrew/bin/hug","args":["mcp"],"env":{"`+user+`":"poodle"}},"theirs":{"command":"/bin/tool","env":{"`+user+`":"keep"}}}}`)
	toml := write(".jcode/config.toml", "[hooks]\nturn_start = \""+home+"=/m /opt/homebrew/bin/muninn hook jcode\"\nother = \""+home+"=/x mytool\"\n\n[mcp_servers.hug]\ncommand = \"/opt/homebrew/bin/hug\"\nenv = { "+user+" = \"p\" }\n")
	codex := write(".codex/config.toml", "[mcp_servers.theirs]\nenv = { "+home+" = \"/x\" }\n")

	a := app(t)
	r := execute(t, a, "upgrade", "--apply")
	did := strings.Join(r["upgrade"].([]string), "\n")
	for _, f := range []string{claude, mcp, toml} {
		if !strings.Contains(did, "variables renamed to HUG_* in "+f) {
			t.Errorf("upgrade did not report %s:\n%s", f, did)
		}
		if _, e := os.Stat(f + ".bak-hug-upgrade"); e != nil {
			t.Errorf("no backup of %s: %v", f, e)
		}
	}
	read := func(p string) string { b, _ := os.ReadFile(p); return string(b) }
	if c := read(claude); !strings.Contains(c, "HUG_HOME=/m /opt/homebrew/bin/muninn hook claude") || !strings.Contains(c, home+"=/x other-tool") || !strings.Contains(c, `"theme": "dark"`) {
		t.Errorf("claude settings:\n%s", c)
	}
	if c := read(mcp); !strings.Contains(c, `"HUG_USER": "poodle"`) || !strings.Contains(c, `"`+user+`": "keep"`) {
		t.Errorf("jcode mcp.json:\n%s", c)
	}
	if c := read(toml); !strings.Contains(c, `turn_start = "HUG_HOME=/m /opt/homebrew/bin/muninn hook jcode"`) || !strings.Contains(c, `other = "`+home+`=/x mytool"`) || !strings.Contains(c, "env = { HUG_USER = \"p\" }") {
		t.Errorf("jcode config.toml:\n%s", c)
	}
	if c := read(codex); !strings.Contains(c, home) || strings.Contains(did, codex) {
		t.Errorf("a server that is not hug's was changed:\n%s\n%s", c, did)
	}
	// Again: nothing left to rename.
	r = execute(t, a, "upgrade", "--apply")
	if did := strings.Join(r["upgrade"].([]string), "\n"); strings.Contains(did, "variables renamed") {
		t.Errorf("second upgrade renamed again:\n%s", did)
	}
}
