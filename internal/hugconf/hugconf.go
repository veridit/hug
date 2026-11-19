// Package hugconf is the one rule for the entries hug writes into harness
// configs (ynh84t): hook commands run hug (`hug mem hook claude`), MCP
// servers are one entry named "hug" running `hug mcp`. The installers
// (hug mem install, hug task install) write these forms; hug upgrade moves
// the old ones (a muninn or huginn link, an MCP entry named muninn or
// huginn, mcp__muninn__* permissions) onto them. Old forms keep working:
// muninn and huginn stay links to the same binary.
package hugconf

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
)

// Server is the MCP server name hug installs.
const Server = "hug"

// OldServers are the server names older installs wrote.
var OldServers = []string{"muninn", "huginn"}

// partOf maps an old binary name to the hug level it stands for.
var partOf = map[string]string{"muninn": "mem", "huginn": "task"}

// base is a command word's program name: hug, muninn, huginn (no .exe).
func base(word string) string {
	return strings.TrimSuffix(filepath.Base(strings.Trim(word, `'"`)), ".exe")
}

// Ours reports whether a binary path is hug or one of its old names.
func Ours(path string) bool {
	b := base(path)
	return b == "hug" || partOf[b] != ""
}

// HugPath is the hug binary hooks and MCP entries should run: next to the
// running program as invoked (a stable link such as /opt/homebrew/bin/hug,
// never the versioned Cellar path it resolves to), else hug on PATH, else
// the running program when it is hug or an old link, else plain "hug"
// (a test binary is neither: an entry naming it would not read as hug's).
func HugPath() string {
	p, err := os.Executable()
	if err != nil {
		return "hug"
	}
	if base(p) == "hug" {
		return p
	}
	sib := filepath.Join(filepath.Dir(p), "hug"+filepath.Ext(p))
	if _, err := os.Stat(sib); err == nil {
		return sib
	}
	if lp, err := exec.LookPath("hug"); err == nil {
		return lp
	}
	if Ours(p) {
		return p
	}
	return "hug"
}

// HugBeside is the hug that replaces old, a muninn or huginn path: hug in
// the same folder when it is there, else fallback.
func HugBeside(old, fallback string) string {
	o := strings.Trim(old, `'"`)
	if strings.ContainsAny(o, `/\`) {
		// Swap the last path part as written, keeping its separators.
		b := filepath.Base(o)
		if i := strings.LastIndexAny(o, `/\`); i >= 0 {
			b = o[i+1:]
		}
		sib := o[:len(o)-len(b)] + "hug" + filepath.Ext(b)
		if _, err := os.Stat(sib); err == nil {
			return sib
		}
	}
	return fallback
}

// Quote quotes a path for a shell hook command when it needs it.
func Quote(s string) string {
	if !strings.ContainsAny(s, " '\"$`\\") {
		return s
	}
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// HookCommand is the recall hook command for harness h (claude, codex,
// jcode) run by bin: `bin mem hook h`, or the old `bin hook h` when bin is
// a muninn link (--bin pointing at one). A bin that is already a command
// prefix (VAR=value assignments, or a quoted program, as a migration keeps
// it from the old hook) is used as written.
func HookCommand(bin, h string) string {
	assign, prog, _ := words(bin)
	if len(assign) > 0 || strings.HasPrefix(prog, "'") || strings.HasPrefix(prog, `"`) {
		return bin + HookTail(prog, h)
	}
	return Quote(bin) + HookTail(bin, h)
}

// HookTail is what follows the program in bin's recall hook for h:
// " mem hook h", or " hook h" for a muninn link.
func HookTail(bin, h string) string {
	if base(bin) == "muninn" {
		return " hook " + h
	}
	return " mem hook " + h
}

// HookPrefix is everything before the hook words in a recall hook command
// for h (VAR=value assignments and the program, as written), "" when cmd
// is not one. A reinstall or a migration keeps it.
func HookPrefix(cmd, h string) string {
	if !IsRecallHook(cmd, h) {
		return ""
	}
	for _, tail := range []string{" mem hook " + h, " memory hook " + h, " hook " + h} {
		if p, ok := strings.CutSuffix(strings.TrimSpace(cmd), tail); ok {
			return p
		}
	}
	return ""
}

// Program is the program path of a command prefix (VAR=value assignments
// dropped, quotes taken off).
func Program(prefix string) string {
	_, prog, _ := words(prefix)
	if strings.HasPrefix(prog, "'") {
		return strings.ReplaceAll(strings.Trim(prog, "'"), `'\''`, "'")
	}
	return strings.Trim(prog, `"`)
}

// MCPCommand is what the hug MCP entry runs for an install given bin (a
// path or a hook's command prefix): hug itself, since muninn mcp and
// huginn mcp each serve one part only. A muninn or huginn bin gives the hug
// beside it, else HugPath.
func MCPCommand(bin string) string {
	p := Program(bin)
	if base(p) == "hug" {
		return p
	}
	return HugBeside(p, HugPath())
}

// fields splits cmd on whitespace outside single or double quotes, keeping
// each word as written (quotes included).
func fields(cmd string) []string {
	var out []string
	var cur strings.Builder
	var q byte
	in := false
	for i := 0; i < len(cmd); i++ {
		c := cmd[i]
		switch {
		case q != 0:
			if c == q {
				q = 0
			}
		case c == '\'' || c == '"':
			q = c
		case c == ' ' || c == '\t' || c == '\n':
			if in {
				out = append(out, cur.String())
				cur.Reset()
				in = false
			}
			continue
		}
		cur.WriteByte(c)
		in = true
	}
	if in {
		out = append(out, cur.String())
	}
	return out
}

// words splits a hook command into VAR=value assignments, the program and
// its arguments.
func words(cmd string) (assign []string, prog string, args []string) {
	f := fields(cmd)
	i := 0
	for ; i < len(f); i++ {
		name, _, ok := strings.Cut(f[i], "=")
		if !ok || name == "" || strings.ContainsAny(name, `/\'"`) {
			break
		}
	}
	if i >= len(f) {
		return f, "", nil
	}
	return f[:i], f[i], f[i+1:]
}

// IsRecallHook reports whether cmd is hug's recall hook for harness h, in
// either form: `.../muninn hook h` or `.../hug mem hook h` (any VAR=value
// before it). Not `muninn capture --hook`, which shares jcode's turn_end.
func IsRecallHook(cmd, h string) bool {
	_, prog, args := words(cmd)
	switch base(prog) {
	case "muninn":
		return len(args) == 2 && args[0] == "hook" && args[1] == h
	case "hug":
		return len(args) == 3 && (args[0] == "mem" || args[0] == "memory") && args[1] == "hook" && args[2] == h
	}
	return false
}

// MoveCommand rewrites a hook command an old install wrote: its program
// muninn (or huginn) becomes hug at the same level (hug mem ..., hug task
// ...), hug taken from beside the old link or else hugPath. Anything that
// is not a muninn or huginn command comes back unchanged, false.
func MoveCommand(cmd, hugPath string) (string, bool) {
	assign, prog, args := words(cmd)
	level := partOf[base(prog)]
	if level == "" {
		return cmd, false
	}
	out := append([]string{}, assign...)
	out = append(out, Quote(HugBeside(prog, hugPath)), level)
	out = append(out, args...)
	return strings.Join(out, " "), true
}

// Entry is an MCP server entry in a jcode or Claude config.
type Entry struct {
	Command string            `json:"command"`
	Args    []string          `json:"args"`
	Env     map[string]string `json:"env"`
}

// MCPEntry is the "hug" server entry for bin; jcodeHome pins one jcode
// instance (hug mcp --jcode-home), "" for none.
func MCPEntry(bin, jcodeHome string) json.RawMessage {
	args := []string{"mcp"}
	if jcodeHome != "" {
		args = append(args, "--jcode-home", jcodeHome)
	}
	b, _ := json.Marshal(Entry{Command: bin, Args: args, Env: map[string]string{}})
	return b
}

// Owned reports whether raw is an MCP entry hug wrote: its command is hug,
// muninn or huginn, its args are mcp [--jcode-home DIR], it has no other
// fields but env (and type stdio), and env holds only HUG_ variables.
// jcodeHome is the pinned instance, "" if none. Another tool's entry, or
// one a person changed, is not hug's to move.
func Owned(raw json.RawMessage) (jcodeHome string, ok bool) {
	var m map[string]json.RawMessage
	if json.Unmarshal(raw, &m) != nil {
		return "", false
	}
	for k, v := range m {
		switch k {
		case "command", "args", "env":
		case "type":
			var t string
			if json.Unmarshal(v, &t) != nil || t != "stdio" {
				return "", false
			}
		default:
			return "", false
		}
	}
	var e Entry
	if json.Unmarshal(raw, &e) != nil || !Ours(e.Command) {
		return "", false
	}
	switch {
	case reflect.DeepEqual(e.Args, []string{"mcp"}):
	case len(e.Args) == 3 && e.Args[0] == "mcp" && e.Args[1] == "--jcode-home":
		jcodeHome = e.Args[2]
	default:
		return "", false
	}
	for k := range e.Env {
		if !strings.HasPrefix(k, "HUG_") {
			return "", false
		}
	}
	return jcodeHome, true
}

// ToolPermission rewrites a Claude permission or tool name that names an
// old server (mcp__muninn__memory_search, mcp__huginn__*): the server part
// becomes hug, and huginn's unprefixed task tools gain hug mcp's task_
// prefix (mcp__huginn__take -> mcp__hug__task_take).
func ToolPermission(p string) (string, bool) {
	rest, ok := strings.CutPrefix(p, "mcp__")
	if !ok {
		return p, false
	}
	server, tool, _ := strings.Cut(rest, "__")
	switch server {
	case "muninn":
	case "huginn":
		if tool != "" && tool != "*" && !strings.HasPrefix(tool, "task_") {
			tool = "task_" + tool
		}
	default:
		return p, false
	}
	if tool == "" && !strings.Contains(rest, "__") {
		return "mcp__" + Server, true
	}
	return "mcp__" + Server + "__" + tool, true
}

// ErrForeign: the config has a "hug" MCP entry hug did not write (another
// command, extra fields). hug leaves that file's servers alone.
var ErrForeign = errors.New(`its "hug" MCP entry is not one hug wrote; hug leaves it as it is`)

func equalJSON(a, b json.RawMessage) bool {
	var x, y any
	return json.Unmarshal(a, &x) == nil && json.Unmarshal(b, &y) == nil && reflect.DeepEqual(x, y)
}

// EditServers puts want under "hug" in a jcode or Claude MCP config
// (install), replacing hug's own entries under the old names muninn and
// huginn, or takes every entry hug wrote off (uninstall). Jcode reads
// "servers" or its alias "mcpServers", never both: the shape the file has
// is kept (servers preferred) and a mixed file is consolidated. Entries
// that are not hug's stay byte for byte. A "hug" entry hug did not write
// stops it with ErrForeign, changing nothing.
func EditServers(root map[string]json.RawMessage, want json.RawMessage, install bool) (bool, error) {
	key := "servers"
	if _, ok := root[key]; !ok {
		if _, ok := root["mcpServers"]; ok {
			key = "mcpServers"
		}
	}
	maps := map[string]map[string]json.RawMessage{}
	for _, k := range []string{"servers", "mcpServers"} {
		if raw, ok := root[k]; ok {
			var entries map[string]json.RawMessage
			if err := json.Unmarshal(raw, &entries); err != nil || entries == nil {
				return false, fmt.Errorf("%s must be an object", k)
			}
			maps[k] = entries
		}
	}
	servers := map[string]json.RawMessage{}
	for _, k := range []string{"servers", "mcpServers"} {
		for name, raw := range maps[k] {
			if old, ok := servers[name]; ok && !equalJSON(old, raw) {
				return false, fmt.Errorf("conflicting server %s in servers and mcpServers; reconcile it before retrying", name)
			}
			servers[name] = raw
		}
	}
	if cur, ok := servers[Server]; ok {
		if _, mine := Owned(cur); !mine {
			return false, ErrForeign
		}
	}
	before, _ := json.Marshal(servers)
	for _, name := range append([]string{Server}, OldServers...) {
		if raw, ok := servers[name]; ok {
			if _, mine := Owned(raw); mine {
				delete(servers, name)
			}
		}
	}
	if install {
		servers[Server] = want
	}
	after, _ := json.Marshal(servers)
	same := equalJSON(before, after)
	if same && (len(maps) <= 1 || !install) {
		return false, nil
	}
	delete(root, "servers")
	delete(root, "mcpServers")
	root[key] = after
	return true, nil
}
