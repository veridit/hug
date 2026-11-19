package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ynh84t check 3: hug upgrade previews every hook and MCP entry an older
// hug wrote under the muninn and huginn names; --apply --machine moves
// them to hug with one backup per file; a second run has nothing to do. A
// "hug" entry hug did not write is reported and left alone (crab 22:47:
// 2), Claude permissions for the old servers follow (1), and the report
// says the old links keep working until a reload (4). Old names spelt in
// parts.
func TestUpgradeMovesOldNamesToHug(t *testing.T) {
	mun, hgn := "mun"+"inn", "hug"+"inn"
	h := t.TempDir()
	t.Setenv("HOME", h)
	t.Setenv("USERPROFILE", h)
	t.Setenv("JCODE_HOME", "")
	bin := filepath.ToSlash(filepath.Join(h, "bin")) // as configs spell it (JSON-safe on Windows)
	os.MkdirAll(bin, 0o755)
	os.WriteFile(filepath.Join(bin, "hug"), nil, 0o755) // hug beside the old links
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
	old := bin + "/" + mun
	settings := write(".claude/settings.json", `{"hooks":{"UserPromptSubmit":[{"hooks":[{"type":"command","command":"`+old+` hook claude","timeout":15}]},{"hooks":[{"type":"command","command":"~/bin/mine.sh"}]}]},"permissions":{"allow":["mcp__`+mun+`__memory_search","mcp__`+hgn+`__take","Bash(git status)"]}}`)
	claudeJSON := write(".claude.json", `{"mcpServers":{"`+mun+`":{"type":"stdio","command":"`+old+`","args":["mcp"]}},"projects":{"/p":{"allowedTools":["mcp__`+mun+`__*"],"mcpServers":{}}},"numStartups":3}`)
	cfg := write(".jcode/config.toml", "[hooks]\n# "+mun+": automatic local memory recall\nturn_start = \""+old+" hook jcode\"\n"+
		"turn_end = [\n  \"~/.jcode/bin/disk_turn_end.sh\",\n  \""+old+" capture --hook\", # capture\n  \""+old+" hook jcode\",\n]\nsession_end = \""+old+" capture --hook\"\n\n[ambient]\nx = 1\n")
	mcp := write(".jcode/mcp.json", `{"servers":{"`+mun+`":{"command":"`+old+`","args":["mcp","--jcode-home","`+filepath.ToSlash(h)+`/.jcode"],"env":{}},"vikunja":{"type":"http","url":"https://example.invalid/mcp"}}}`)
	codex := write(".codex/config.toml", "[mcp_servers."+mun+"]\ncommand = \""+old+"\"\nargs = [\"mcp\"]\n\n[mcp_servers.theirs]\ncommand = \"/bin/tool\"\n")
	// Another tool's "hug" entry beside hug's old one: report, do not merge.
	proj := filepath.Join(h, "proj", ".jcode")
	foreign := write("proj/.jcode/mcp.json", `{"servers":{"hug":{"command":"/elsewhere/proxy","args":["serve"]},"`+hgn+`":{"command":"`+bin+"/"+hgn+`","args":["mcp"]}}}`)
	write(".zshrc", "typeset -gA JCODE_INSTANCES=(\n  proj "+filepath.Dir(proj)+"\n)\n")

	read := func(p string) string { b, _ := os.ReadFile(p); return string(b) }
	before := map[string]string{}
	for _, f := range []string{settings, claudeJSON, cfg, mcp, codex, foreign} {
		before[f] = read(f)
	}

	a := app(t)
	a.Env["HOME"], a.Env["JCODE_HOME"] = h, "" // this test's machine
	execute(t, a, "init")
	code, out := runText(a, "upgrade")
	for _, f := range []string{settings, claudeJSON, cfg, mcp, codex} {
		if !strings.Contains(out, "names: would: hooks and MCP entries run hug in "+f+"\n") {
			t.Errorf("preview does not name %s:\n%s", f, out)
		}
	}
	if !strings.Contains(out, "names: skipped "+foreign+`: its "hug" MCP entry is not hug's`) || !strings.Contains(out, "mcp action=reload server=hug") {
		t.Errorf("preview lacks the skipped line or the reload fix:\n%s", out)
	}
	if code != UpgradePending {
		t.Errorf("preview exit %d, want %d", code, UpgradePending)
	}
	for f, b := range before {
		if read(f) != b {
			t.Fatalf("the preview changed %s", f)
		}
	}

	// --apply without --machine is a board's upgrade: it never writes a
	// machine file (f5vx4x, crab 23:36 hole a) and says how to.
	code, out = runText(a, "upgrade", "--apply")
	if code != UpgradePending || !strings.Contains(out, "fix: hug upgrade --apply --machine") {
		t.Errorf("--apply alone: exit %d, want %d with the --machine fix:\n%s", code, UpgradePending, out)
	}
	for f, b := range before {
		if read(f) != b {
			t.Fatalf("--apply without --machine changed %s", f)
		}
		if _, e := os.Stat(f + ".bak-hug-upgrade"); e == nil {
			t.Fatalf("--apply without --machine left a backup of %s", f)
		}
	}

	code, out = runText(a, "upgrade", "--apply", "--machine")
	if code != 0 {
		t.Fatalf("--apply --machine: exit %d\n%s", code, out)
	}
	for _, f := range []string{settings, claudeJSON, cfg, mcp, codex} {
		if b, e := os.ReadFile(f + ".bak-hug-upgrade"); e != nil || string(b) != before[f] {
			t.Errorf("no backup of %s holding the old file: %v", f, e)
		}
		if strings.Contains(read(f), mun) || strings.Contains(read(f), hgn) {
			t.Errorf("%s still names an old part:\n%s", f, read(f))
		}
	}
	hug := bin + "/hug"
	for f, wants := range map[string][]string{
		settings:   {`"command": "` + hug + ` mem hook claude"`, "mine.sh", `"mcp__hug__memory_search"`, `"mcp__hug__task_take"`, `"Bash(git status)"`},
		claudeJSON: {`"hug": {`, `"command": "` + hug + `"`, `"type": "stdio"`, `"mcp__hug__*"`, `"numStartups": 3`},
		cfg: {"# hug: automatic local memory recall", `turn_start = "` + hug + ` mem hook jcode"`,
			`turn_end = ["~/.jcode/bin/disk_turn_end.sh","` + hug + ` mem capture --hook","` + hug + ` mem hook jcode"]`,
			`session_end = "` + hug + ` mem capture --hook"`, "[ambient]\nx = 1\n"},
		mcp:   {`"hug": {`, `"--jcode-home"`, "vikunja"},
		codex: {"[mcp_servers.hug]\ncommand = \"" + hug + "\"", "[mcp_servers.theirs]\ncommand = \"/bin/tool\""},
	} {
		for _, w := range wants {
			if !strings.Contains(read(f), w) {
				t.Errorf("%s lacks %s:\n%s", f, w, read(f))
			}
		}
	}
	if read(foreign) != before[foreign] {
		t.Errorf("merged into a hug entry that is not hug's:\n%s", read(foreign))
	}

	// Again: nothing to move; the skipped file is still reported.
	code, out = runText(a, "upgrade", "--apply", "--machine")
	if code != 0 || strings.Contains(out, "names: hooks and MCP entries") || !strings.Contains(out, "names: skipped "+foreign) {
		t.Fatalf("second run: exit %d\n%s", code, out)
	}
	code, out = runText(a, "upgrade")
	if strings.Contains(out, "names: would") {
		t.Fatalf("preview after the move still has work: exit %d\n%s", code, out)
	}
}
