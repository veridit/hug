package cli

import (
	"bufio"
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"

	"github.com/veridit/hug/internal/env"
)

// UpgradePending is hug upgrade's exit code when its preview found changes
// to make (asr4t2): 0 means nothing to do, and hug upgrade --apply makes
// them. Distinct from 1-4 so a script can tell "behind" from "failed".
const UpgradePending = 5

// hug upgrade renames old variable names where hug wrote them (m1c28r):
// the env of hug's own MCP server entries and hug's hook commands, in the
// harness and app configs hug installs into. Nothing else in those files
// changes (a ~/.claude.json also holds the user's history).

// envFiles are the configs hug installs into under home, and in jcodeHome
// (the App's JCODE_HOME) when set.
func envFiles(home, jcodeHome string) []string {
	files := []string{
		filepath.Join(home, ".jcode", "config.toml"),
		filepath.Join(home, ".jcode", "mcp.json"),
		filepath.Join(home, ".claude", "settings.json"),
		filepath.Join(home, ".claude.json"),
		filepath.Join(home, ".codex", "hooks.json"),
		filepath.Join(home, ".codex", "config.toml"),
	}
	switch runtime.GOOS {
	case "darwin":
		files = append(files, filepath.Join(home, "Library/Application Support/Claude/claude_desktop_config.json"))
	case "windows":
		files = append(files, filepath.Join(os.Getenv("APPDATA"), "Claude/claude_desktop_config.json"))
	default:
		files = append(files, filepath.Join(home, ".config/Claude/claude_desktop_config.json"))
	}
	if jcodeHome != "" {
		files = append(files, filepath.Join(jcodeHome, "config.toml"), filepath.Join(jcodeHome, "mcp.json"))
	}
	return files
}

// oldName matches an old variable name (spelt in parts: the code holds none).
var oldName = regexp.MustCompile(`\b(?:HUG` + `INN|MUN` + `INN)_[A-Z0-9_]+\b`)

func renameIn(s string) string {
	return oldName.ReplaceAllStringFunc(s, env.Renamed)
}

// ours is a command hug installed: hug, muninn or huginn, as a path or the
// first word of a hook command after any VAR=value assignments.
func ours(command string) bool {
	for _, w := range strings.Fields(strings.Trim(command, `'"`)) {
		w = strings.Trim(w, `'"`)
		if name, _, ok := strings.Cut(w, "="); ok && name != "" && !strings.ContainsAny(name, `/\`) {
			continue // an assignment before the command
		}
		base := strings.TrimSuffix(filepath.Base(w), ".exe")
		return base == "hug" || base == "muninn" || base == "huginn"
	}
	return false
}

// renameJSON renames in hug's MCP server entries (mcpServers, servers) and
// hug's hook commands (hooks.*[].hooks[].command).
func renameJSON(b []byte) ([]byte, bool, error) {
	var root map[string]json.RawMessage
	if e := json.Unmarshal(b, &root); e != nil || root == nil {
		return b, false, e
	}
	changed := false
	for _, key := range []string{"mcpServers", "servers"} {
		raw, ok := root[key]
		if !ok {
			continue
		}
		var servers map[string]map[string]any
		if json.Unmarshal(raw, &servers) != nil {
			continue
		}
		touched := false
		for _, entry := range servers {
			cmd, _ := entry["command"].(string)
			envMap, _ := entry["env"].(map[string]any)
			if !ours(cmd) || len(envMap) == 0 {
				continue
			}
			for k, v := range envMap {
				if n := renameIn(k); n != k {
					delete(envMap, k)
					envMap[n] = v
					touched = true
				}
			}
		}
		if touched {
			root[key], _ = json.Marshal(servers)
			changed = true
		}
	}
	if raw, ok := root["hooks"]; ok {
		var hooks map[string][]map[string]any
		if json.Unmarshal(raw, &hooks) == nil {
			touched := false
			for _, groups := range hooks {
				for _, g := range groups {
					hs, _ := g["hooks"].([]any)
					for _, h := range hs {
						hm, _ := h.(map[string]any)
						if c, _ := hm["command"].(string); ours(c) {
							if n := renameIn(c); n != c {
								hm["command"] = n
								touched = true
							}
						}
					}
				}
			}
			if touched {
				root["hooks"], _ = json.Marshal(hooks)
				changed = true
			}
		}
	}
	if !changed {
		return b, false, nil
	}
	out, e := json.MarshalIndent(root, "", "  ")
	return append(out, '\n'), true, e
}

// renameTOML renames in [mcp_servers.hug|muninn|huginn] sections and on
// lines that run hug (jcode's [hooks] entries).
func renameTOML(b []byte) ([]byte, bool) {
	var out bytes.Buffer
	section := ""
	sc := bufio.NewScanner(bytes.NewReader(b))
	sc.Buffer(make([]byte, 1<<20), 1<<24)
	for sc.Scan() {
		line := sc.Text()
		if t := strings.TrimSpace(line); strings.HasPrefix(t, "[") {
			section = strings.Trim(t, "[] ")
		}
		mine := strings.HasPrefix(section, "mcp_servers.hug") || strings.HasPrefix(section, "mcp_servers.muninn") || strings.HasPrefix(section, "mcp_servers.huginn")
		if !mine {
			if _, v, ok := strings.Cut(line, "="); ok && ours(strings.TrimSpace(v)) {
				mine = true
			}
		}
		if mine {
			line = renameIn(line)
		}
		out.WriteString(line)
		out.WriteByte('\n')
	}
	if !bytes.HasSuffix(b, []byte("\n")) {
		out.Truncate(out.Len() - 1)
	}
	return out.Bytes(), !bytes.Equal(out.Bytes(), b)
}

// renameEnv rewrites old names in files, keeping a backup of each changed
// file next to it (FILE.bak-hug-upgrade). It returns the files changed.
func renameEnv(files []string, dry bool) ([]string, error) {
	var changed []string
	seen := map[string]bool{}
	for _, f := range files {
		if seen[f] {
			continue
		}
		seen[f] = true
		b, e := os.ReadFile(f)
		if e != nil || !oldName.Match(b) {
			continue
		}
		var out []byte
		var did bool
		if strings.HasSuffix(f, ".toml") {
			out, did = renameTOML(b)
		} else if out, did, e = renameJSON(b); e != nil {
			continue // not JSON hug can read: leave it
		}
		if !did {
			continue
		}
		changed = append(changed, f)
		if dry {
			continue
		}
		st, e := os.Stat(f)
		if e != nil {
			return changed, e
		}
		if e = os.WriteFile(f+".bak-hug-upgrade", b, 0o600); e != nil {
			return changed, e
		}
		tmp := f + ".hug-tmp"
		if e = os.WriteFile(tmp, out, st.Mode().Perm()); e != nil {
			return changed, e
		}
		if e = os.Rename(tmp, f); e != nil {
			return changed, e
		}
	}
	return changed, nil
}
