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
	brew := hugBin(t, h) // a folder with hug beside the old links
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
	claude := write(".claude/settings.json", `{"hooks":{"UserPromptSubmit":[{"hooks":[{"type":"command","command":"`+home+`=/m `+brew+`/muninn hook claude","timeout":15}]},{"hooks":[{"type":"command","command":"`+home+`=/x other-tool"}]}]},"theme":"dark"}`)
	mcp := write(".jcode/mcp.json", `{"servers":{"hug":{"command":"`+brew+`/hug","args":["mcp"],"env":{"`+user+`":"poodle"}},"theirs":{"command":"/bin/tool","env":{"`+user+`":"keep"}}}}`)
	toml := write(".jcode/config.toml", "[hooks]\nturn_start = \""+home+"=/m "+brew+"/muninn hook jcode\"\nother = \""+home+"=/x mytool\"\n\n[mcp_servers.hug]\ncommand = \""+brew+"/hug\"\nenv = { "+user+" = \"p\" }\n")
	codex := write(".codex/config.toml", "[mcp_servers.theirs]\nenv = { "+home+" = \"/x\" }\n")

	a := app(t)
	a.Env["HOME"], a.Env["JCODE_HOME"] = h, "" // this test's machine
	// This machine's files change with --machine only (f5vx4x).
	r := execute(t, a, "upgrade", "--apply", "--machine")
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
	if c := read(claude); !strings.Contains(c, "HUG_HOME=/m "+brew+"/hug mem hook claude") || !strings.Contains(c, home+"=/x other-tool") || !strings.Contains(c, `"theme": "dark"`) {
		t.Errorf("claude settings:\n%s", c)
	}
	if c := read(mcp); !strings.Contains(c, `"HUG_USER": "poodle"`) || !strings.Contains(c, `"`+user+`": "keep"`) {
		t.Errorf("jcode mcp.json:\n%s", c)
	}
	if c := read(toml); !strings.Contains(c, `turn_start = "HUG_HOME=/m `+brew+`/hug mem hook jcode"`) || !strings.Contains(c, `other = "`+home+`=/x mytool"`) || !strings.Contains(c, "env = { HUG_USER = \"p\" }") {
		t.Errorf("jcode config.toml:\n%s", c)
	}
	// One backup holds the file as it was before both steps (ynh84t).
	if b, _ := os.ReadFile(toml + ".bak-hug-upgrade"); !strings.Contains(string(b), home+"=/m "+brew+"/mun"+"inn hook jcode") {
		t.Errorf("backup is not the original:\n%s", b)
	}
	if c := read(codex); !strings.Contains(c, home) || strings.Contains(did, codex) {
		t.Errorf("a server that is not hug's was changed:\n%s\n%s", c, did)
	}
	// Again: nothing left to rename.
	r = execute(t, a, "upgrade", "--apply", "--machine")
	if did := strings.Join(r["upgrade"].([]string), "\n"); strings.Contains(did, "variables renamed") {
		t.Errorf("second upgrade renamed again:\n%s", did)
	}
}

// hug upgrade reads JCODE_HOME from the App it runs in, not from the test
// or host process (ynh84t, crab 23:36 ruling 3): with the process's
// JCODE_HOME pointing at one jcode home and the App's at another, only the
// App's is previewed. Before, envFiles read os.Getenv, so a test reached
// whatever jcode home its shell had set.
func TestUpgradeReadsJcodeHomeFromTheApp(t *testing.T) {
	home := "MUN" + "INN_HOME"
	h := t.TempDir()
	proc, mine := filepath.Join(h, "proc-jcode"), filepath.Join(h, "app-jcode")
	for _, d := range []string{proc, mine} {
		os.MkdirAll(d, 0o755)
		os.WriteFile(filepath.Join(d, "config.toml"), []byte("[hooks]\nturn_start = \""+home+"=/m /opt/homebrew/bin/hug mem hook jcode\"\n"), 0o644)
	}
	t.Setenv("HOME", h)
	t.Setenv("USERPROFILE", h)
	t.Setenv("JCODE_HOME", proc)
	a := app(t)
	a.Env["HOME"], a.Env["JCODE_HOME"] = h, mine
	_, out := runText(a, "upgrade")
	if !strings.Contains(out, filepath.Join(mine, "config.toml")) || strings.Contains(out, proc) {
		t.Errorf("upgrade did not preview the App's JCODE_HOME (%s) alone:\n%s", mine, out)
	}
}

// hugBin is a folder under h holding a hug, spelt with forward slashes as
// a config spells it: the hug an old muninn link beside it moves to, on
// every OS and whether or not hug is on PATH (ynh84t: CI has none).
func hugBin(t *testing.T, h string) string {
	t.Helper()
	bin := filepath.ToSlash(filepath.Join(h, "brew-bin"))
	if e := os.MkdirAll(bin, 0o755); e != nil {
		t.Fatal(e)
	}
	if e := os.WriteFile(bin+"/hug", nil, 0o755); e != nil {
		t.Fatal(e)
	}
	return bin
}
