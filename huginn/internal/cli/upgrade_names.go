package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/veridit/hug/huginn/internal/harness"
	"github.com/veridit/hug/internal/hugconf"
)

// hug upgrade moves the names an older hug wrote into this machine's
// harness configs (ynh84t): hook commands that run the muninn or huginn
// link run hug (`hug mem hook claude`), MCP entries hug wrote under the
// names muninn and huginn become one entry named hug running hug mcp, and
// Claude permissions for those servers (mcp__muninn__*) name hug. The old
// links stay, so configs not moved yet, and MCP servers still running,
// keep working until they are moved or reloaded.

// moveNamesJSON moves hug's names in a JSON harness config (Claude
// settings.json, ~/.claude.json, a Codex hooks.json, a jcode or Claude
// Desktop MCP config). foreign is set when an MCP map holds a "hug" entry
// hug did not write: that map is left alone.
func moveNamesJSON(b []byte, hug string) (out []byte, changed, foreign bool, err error) {
	var root map[string]json.RawMessage
	if e := json.Unmarshal(b, &root); e != nil || root == nil {
		return b, false, false, e
	}
	// MCP server maps: the top level and, in ~/.claude.json, each project's.
	edit := func(obj map[string]json.RawMessage) bool {
		var servers map[string]json.RawMessage
		for _, k := range []string{"servers", "mcpServers"} {
			if raw, ok := obj[k]; ok && json.Unmarshal(raw, &servers) == nil {
				break
			}
			servers = nil
		}
		if servers == nil {
			return false
		}
		var old json.RawMessage
		jcodeHome := ""
		for _, name := range hugconf.OldServers {
			if raw, ok := servers[name]; ok {
				if h, mine := hugconf.Owned(raw); mine {
					old = raw
					if h != "" {
						jcodeHome = h
					}
				}
			}
		}
		if old == nil {
			return false
		}
		var e hugconf.Entry
		json.Unmarshal(old, &e)
		entry := hugconf.MCPEntry(hugconf.HugBeside(e.Command, hug), jcodeHome)
		if len(e.Env) > 0 {
			var m map[string]any
			json.Unmarshal(entry, &m)
			m["env"] = e.Env
			entry, _ = json.Marshal(m)
		}
		var typ struct {
			Type string `json:"type"`
		}
		if json.Unmarshal(old, &typ) == nil && typ.Type != "" {
			var m map[string]any
			json.Unmarshal(entry, &m)
			m["type"] = typ.Type
			entry, _ = json.Marshal(m)
		}
		ch, e2 := hugconf.EditServers(obj, entry, true)
		if errors.Is(e2, hugconf.ErrForeign) {
			foreign = true
			return false
		}
		return e2 == nil && ch
	}
	if edit(root) {
		changed = true
	}
	if raw, ok := root["projects"]; ok {
		var projects map[string]map[string]json.RawMessage
		if json.Unmarshal(raw, &projects) == nil {
			touched := false
			for _, p := range projects {
				if p != nil && edit(p) {
					touched = true
				}
				if moveTools(p, "allowedTools") {
					touched = true
				}
			}
			if touched {
				root["projects"], _ = json.Marshal(projects)
				changed = true
			}
		}
	}
	// Hooks: hooks.<event>[].hooks[].command.
	if raw, ok := root["hooks"]; ok {
		var hooks map[string][]map[string]any
		if json.Unmarshal(raw, &hooks) == nil {
			touched := false
			for _, groups := range hooks {
				for _, g := range groups {
					hs, _ := g["hooks"].([]any)
					for _, h := range hs {
						hm, _ := h.(map[string]any)
						if c, _ := hm["command"].(string); c != "" {
							if n, ok := hugconf.MoveCommand(c, hug); ok {
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
	// Claude permissions: permissions.allow / deny / ask entries.
	if raw, ok := root["permissions"]; ok {
		var perms map[string]json.RawMessage
		if json.Unmarshal(raw, &perms) == nil {
			touched := false
			for _, k := range []string{"allow", "deny", "ask"} {
				if moveTools(perms, k) {
					touched = true
				}
			}
			if touched {
				root["permissions"], _ = json.Marshal(perms)
				changed = true
			}
		}
	}
	if !changed {
		return b, false, foreign, nil
	}
	out, err = json.MarshalIndent(root, "", "  ")
	return append(out, '\n'), true, foreign, err
}

// moveTools renames old server names in obj[key], a list of tool names or
// permission rules, dropping a rule the new name already holds.
func moveTools(obj map[string]json.RawMessage, key string) bool {
	raw, ok := obj[key]
	if !ok {
		return false
	}
	var list []string
	if json.Unmarshal(raw, &list) != nil {
		return false
	}
	seen := map[string]bool{}
	for _, s := range list {
		seen[s] = true
	}
	var out []string
	touched := false
	for _, s := range list {
		if n, ok := hugconf.ToolPermission(s); ok {
			touched = true
			if seen[n] {
				continue
			}
			seen[n] = true
			s = n
		}
		out = append(out, s)
	}
	if !touched {
		return false
	}
	if out == nil {
		out = []string{}
	}
	obj[key], _ = json.Marshal(out)
	return true
}

// The comment hug mem install puts above its jcode recall hooks, before
// and after ynh84t (old name spelt in parts).
const (
	oldMarker = "# mun" + "inn: automatic local memory recall"
	newMarker = "# hug: automatic local memory recall"
)

// moveNamesTOML moves hug's names in a TOML config: jcode's [hooks] (a
// string or an array of strings per key, on one or more lines) and
// Codex's [mcp_servers.muninn|huginn] tables whose command is hug's,
// renamed to [mcp_servers.hug]. A file that already has an
// [mcp_servers.hug] table keeps its old tables (foreign).
func moveNamesTOML(b []byte, hug string) ([]byte, bool, bool) {
	text := string(b)
	lines := strings.Split(text, "\n")
	// Each table's header line and its command value.
	headerAt := map[string]int{}
	command := map[string]string{}
	section := ""
	for i, l := range lines {
		t := strings.TrimSpace(l)
		if strings.HasPrefix(t, "[") {
			section = strings.Trim(t, "[] ")
			headerAt[section] = i
			continue
		}
		if key, v, ok := strings.Cut(l, "="); ok && strings.TrimSpace(key) == "command" {
			var s string
			if json.Unmarshal([]byte(strings.TrimSpace(v)), &s) == nil {
				command[section] = s
			}
		}
	}
	_, hasHug := headerAt["mcp_servers."+hugconf.Server]
	foreign := false
	rename := map[string]bool{} // old table names to move
	for _, old := range hugconf.OldServers {
		p := "mcp_servers." + old
		if _, ok := headerAt[p]; !ok || !hugconf.Ours(command[p]) {
			continue
		}
		if hasHug || len(rename) > 0 {
			foreign = hasHug || foreign
			continue
		}
		rename[p] = true
	}
	section = ""
	for i := 0; i < len(lines); i++ {
		l := lines[i]
		t := strings.TrimSpace(l)
		if strings.HasPrefix(t, "[") {
			section = strings.Trim(t, "[] ")
			for p := range rename {
				if section == p || strings.HasPrefix(section, p+".") {
					lines[i] = strings.Replace(l, p, "mcp_servers."+hugconf.Server, 1)
				}
			}
			continue
		}
		if section == "hooks" && t == oldMarker {
			lines[i] = strings.Replace(l, oldMarker, newMarker, 1)
			continue
		}
		key, v, ok := strings.Cut(l, "=")
		if !ok {
			continue
		}
		v = strings.TrimSpace(v)
		if section == "hooks" {
			// A multi-line array runs until it parses.
			end := i
			if strings.HasPrefix(v, "[") {
				for end < len(lines)-1 && !parsesTOMLArray(joinValue(v, lines[i+1:end+1])) {
					end++
				}
			}
			if n, ok := moveTOMLValue(joinValue(v, lines[i+1:end+1]), hug); ok {
				lines = append(lines[:i], append([]string{strings.TrimRight(key, " ") + " = " + n}, lines[end+1:]...)...)
			}
			continue
		}
		// command = ".../muninn" in a server table that moves to hug.
		if strings.TrimSpace(key) == "command" && (rename[section] || section == "mcp_servers."+hugconf.Server) {
			var s string
			if json.Unmarshal([]byte(v), &s) == nil && hugconf.Ours(s) && hugconf.Program(s) != "" && !isHugBin(s) {
				lines[i] = strings.TrimRight(key, " ") + " = " + tomlString(hugconf.HugBeside(s, hug))
			}
		}
	}
	out := strings.Join(lines, "\n")
	return []byte(out), out != text, foreign
}

func isHugBin(p string) bool {
	b := strings.TrimSuffix(p, ".exe")
	return b == "hug" || strings.HasSuffix(b, "/hug") || strings.HasSuffix(b, `\hug`)
}

// joinValue is a TOML value that starts with v and runs on over more.
func joinValue(v string, more []string) string {
	return strings.Join(append([]string{v}, more...), "\n")
}

// tomlArray reads a TOML array of basic strings, on one line or several,
// with a trailing comma or # comments between the items.
func tomlArray(v string) ([]string, bool) {
	var keep []string
	for _, l := range strings.Split(v, "\n") {
		if i := commentAt(l); i >= 0 {
			l = l[:i]
		}
		keep = append(keep, l)
	}
	s := strings.TrimSpace(strings.Join(keep, " "))
	if strings.HasSuffix(s, "]") {
		inner := strings.TrimSpace(strings.TrimSuffix(s, "]"))
		inner = strings.TrimSuffix(inner, ",")
		s = inner + "]"
	}
	var many []string
	if json.Unmarshal([]byte(s), &many) != nil {
		return nil, false
	}
	return many, true
}

func parsesTOMLArray(v string) bool { _, ok := tomlArray(v); return ok }

// commentAt is where a # comment starts in a line, outside "strings".
func commentAt(l string) int {
	in := false
	for i := 0; i < len(l); i++ {
		switch l[i] {
		case '\\':
			if in {
				i++
			}
		case '"':
			in = !in
		case '#':
			if !in {
				return i
			}
		}
	}
	return -1
}

// tomlString writes s as a TOML basic string (JSON escapes, no HTML ones).
func tomlString(s string) string {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.Encode(s)
	return strings.TrimSpace(buf.String())
}

// moveTOMLValue moves the commands in one [hooks] value: "cmd" or
// ["cmd", ...]. An array comes back on one line, as hug mem install
// writes it.
func moveTOMLValue(v string, hug string) (string, bool) {
	var one string
	if json.Unmarshal([]byte(v), &one) == nil {
		if n, ok := hugconf.MoveCommand(one, hug); ok {
			return tomlString(n), true
		}
		return v, false
	}
	many, ok := tomlArray(v)
	if !ok {
		return v, false
	}
	moved := false
	for i, c := range many {
		if n, ok := hugconf.MoveCommand(c, hug); ok {
			many[i], moved = n, true
		}
	}
	if !moved {
		return v, false
	}
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.Encode(many)
	return strings.TrimSpace(buf.String()), true
}

// homeDir is this machine's home as the App sees it: HOME (USERPROFILE on
// Windows) from its own environment first, so a test's App never reaches
// the real machine's configs, then the process's.
func (a *App) homeDir() (string, error) {
	for _, k := range []string{"HOME", "USERPROFILE"} {
		if v, ok := a.Env[k]; ok && v != "" {
			return v, nil
		}
	}
	return os.UserHomeDir()
}

// nameFiles are the configs whose old names upgrade moves: the ones the
// variables step reads, Claude's local settings, and every jcode instance
// this machine registers (each one's config.toml and mcp.json).
func nameFiles(home, jcodeHome string) []string {
	files := append(envFiles(home, jcodeHome), filepath.Join(home, ".claude", "settings.local.json"))
	var extras []string
	if jcodeHome != "" {
		extras = append(extras, jcodeHome)
	}
	for _, i := range harness.DiscoverJcode(home, extras) {
		files = append(files, filepath.Join(i.Home, "config.toml"), filepath.Join(i.Home, "mcp.json"))
	}
	return files
}

// nameMove is one machine config upgrade would change.
type nameMove struct {
	File    string
	Foreign bool // a "hug" MCP entry hug did not write: its servers stay
}

// moveNames moves old names in files (preview with dry). Each changed file
// is backed up once as FILE.bak-hug-upgrade, unless the variables step
// already backed it up this run (backedUp).
func moveNames(files []string, hug string, dry bool, backedUp map[string]bool) ([]nameMove, error) {
	var moves []nameMove
	seen := map[string]bool{}
	for _, f := range files {
		if seen[f] {
			continue
		}
		seen[f] = true
		b, e := os.ReadFile(f)
		if e != nil || !bytes.Contains(b, []byte("mun"+"inn")) && !bytes.Contains(b, []byte("hug"+"inn")) {
			continue
		}
		var out []byte
		var did, foreign bool
		if strings.HasSuffix(f, ".toml") {
			out, did, foreign = moveNamesTOML(b, hug)
		} else if out, did, foreign, e = moveNamesJSON(b, hug); e != nil {
			continue // not JSON hug can read: leave it
		}
		if foreign {
			moves = append(moves, nameMove{File: f, Foreign: true})
		}
		if !did {
			continue
		}
		moves = append(moves, nameMove{File: f})
		if dry {
			continue
		}
		st, e := os.Stat(f)
		if e != nil {
			return moves, e
		}
		if !backedUp[f] {
			if e = os.WriteFile(f+".bak-hug-upgrade", b, 0o600); e != nil {
				return moves, e
			}
		}
		tmp := f + ".hug-tmp"
		if e = os.WriteFile(tmp, out, st.Mode().Perm()); e != nil {
			return moves, e
		}
		if e = os.Rename(tmp, f); e != nil {
			return moves, e
		}
	}
	return moves, nil
}
