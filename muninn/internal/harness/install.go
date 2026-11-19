package harness

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/veridit/hug/internal/hugconf"
)

// InstallResult describes what an installer did (or would do).
type InstallResult struct {
	Harness string
	File    string
	Backup  string
	Changed bool
	Note    string
}

func (r InstallResult) String() string {
	s := fmt.Sprintf("%s: %s", r.Harness, r.File)
	switch {
	case !r.Changed:
		s += " (already up to date)"
	case r.Backup != "":
		s += " (backup " + r.Backup + ")"
	}
	if r.Note != "" {
		s += "\n  " + r.Note
	}
	return s
}

func backup(path string, dry bool) (string, error) {
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		return "", nil
	}
	b := path + ".muninn-bak-" + time.Now().Format("20060102-150405")
	if dry {
		return b, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return b, os.WriteFile(b, data, 0o600)
}

func writeAtomic(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	mode := os.FileMode(0o600)
	if fi, err := os.Stat(path); err == nil {
		mode = fi.Mode().Perm()
	}
	tmp := path + ".muninn-tmp"
	if err := os.WriteFile(tmp, data, mode); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// isMuninnGroup reports whether a {"hooks":[{type,command,timeout}]} group
// in a Claude Code settings.json or a Codex hooks.json is hug's recall
// hook for harness h, in either form (muninn hook h, hug mem hook h).
func isMuninnGroup(g any, h string) bool {
	m, ok := g.(map[string]any)
	if !ok {
		return false
	}
	hs, _ := m["hooks"].([]any)
	for _, x := range hs {
		if hm, ok := x.(map[string]any); ok {
			if c, _ := hm["command"].(string); hugconf.IsRecallHook(c, h) {
				return true
			}
		}
	}
	return false
}

// editPromptHook adds (install) or removes (uninstall) hug's
// UserPromptSubmit group in a JSON hooks file, preserving everything else.
func editPromptHook(path, h, command string, timeout int, install, dry bool) (InstallResult, error) {
	res := InstallResult{File: path}
	root := map[string]any{}
	if b, err := os.ReadFile(path); err == nil && len(bytes.TrimSpace(b)) > 0 {
		if err := json.Unmarshal(b, &root); err != nil {
			return res, fmt.Errorf("%s is not valid JSON: %w", path, err)
		}
	} else if err != nil && !errors.Is(err, os.ErrNotExist) {
		return res, err
	}
	hooks, _ := root["hooks"].(map[string]any)
	if hooks == nil {
		hooks = map[string]any{}
	}
	groups, _ := hooks["UserPromptSubmit"].([]any)
	var kept []any
	found := false
	for _, g := range groups {
		if isMuninnGroup(g, h) {
			found = true
			continue
		}
		kept = append(kept, g)
	}
	want := map[string]any{"hooks": []any{map[string]any{"type": "command", "command": command, "timeout": timeout}}}
	if install {
		kept = append(kept, want)
	}
	if install && found {
		// Replace only if the command changed.
		for _, g := range groups {
			if isMuninnGroup(g, h) {
				old, _ := json.Marshal(g)
				nw, _ := json.Marshal(want)
				if bytes.Equal(old, nw) {
					return res, nil
				}
			}
		}
	}
	if !install && !found {
		return res, nil
	}
	if len(kept) == 0 {
		delete(hooks, "UserPromptSubmit")
	} else {
		hooks["UserPromptSubmit"] = kept
	}
	root["hooks"] = hooks
	out, _ := json.MarshalIndent(root, "", "  ")
	out = append(out, '\n')
	res.Changed = true
	var err error
	if res.Backup, err = backup(path, dry); err != nil {
		return res, err
	}
	if dry {
		return res, nil
	}
	return res, writeAtomic(path, out)
}

// InstallClaude wires `hug mem hook claude` into ~/.claude/settings.json.
func InstallClaude(bin string, install, dry bool) (InstallResult, error) {
	home, _ := os.UserHomeDir()
	r, err := editPromptHook(filepath.Join(home, ".claude", "settings.json"), "claude", hugconf.HookCommand(bin, "claude"), 15, install, dry)
	r.Harness = "claude"
	return r, err
}

// InstallCodex wires `hug mem hook codex` into ~/.codex/hooks.json. Codex
// requires the user to trust a new hook before it runs.
func InstallCodex(bin string, install, dry bool) (InstallResult, error) {
	home, _ := os.UserHomeDir()
	r, err := editPromptHook(filepath.Join(home, ".codex", "hooks.json"), "codex", hugconf.HookCommand(bin, "codex"), 15, install, dry)
	r.Harness = "codex"
	if install && r.Changed {
		r.Note = "Codex runs new hooks only after you trust them: open Codex and approve it in /hooks."
	}
	return r, err
}

// InstallJcode wires `hug mem hook jcode` into [hooks] turn_start,
// post_tool_feedback and turn_end of the global ~/.jcode/config.toml with a
// minimal line edit that keeps other hooks.
func InstallJcode(bin string, install, dry bool) (InstallResult, error) {
	home, _ := os.UserHomeDir()
	return InstallJcodeHome(filepath.Join(home, ".jcode"), "jcode", bin, install, dry)
}

// InstallJcodeHome does the same for one instance's JCODE_HOME.
func InstallJcodeHome(jcodeHome, label, bin string, install, dry bool) (InstallResult, error) {
	path := filepath.Join(jcodeHome, "config.toml")
	res := InstallResult{Harness: label, File: path}
	b, err := os.ReadFile(path)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return res, err
	}
	cmd := hugconf.HookCommand(bin, "jcode")
	text, changed, err := editJcodeHooks(string(b), cmd, install)
	if err != nil {
		return res, err
	}
	text2, changed2, err := editJcodeMemoryTakeover(text, install)
	if err != nil {
		return res, err
	}
	if changed2 {
		text, changed = text2, true
		if install {
			res.Note = "jcode's own memory off; "
		} else {
			res.Note = "jcode's own memory back on; "
		}
	}
	if changed {
		res.Changed = true
		if res.Backup, err = backup(path, dry); err != nil {
			return res, err
		}
		if !dry {
			if err := writeAtomic(path, []byte(text)); err != nil {
				return res, err
			}
		}
	}
	// MCP: the model saves and corrects through muninn's tools instead.
	notes, err := takeOverJcodeMemory(jcodeHome, bin, install, dry)
	if err != nil {
		return res, err
	}
	for _, n := range notes {
		if strings.Contains(n, "mcp.json") {
			res.Changed = true
			res.Note += n + "; "
		}
	}
	res.Note = strings.TrimSuffix(res.Note, "; ")
	return res, nil
}

// jcodeMarker heads the recall hook lines hug adds to [hooks];
// oldJcodeMarker is what hug wrote before ynh84t, still recognised.
const (
	jcodeMarker    = "# hug: automatic local memory recall"
	oldJcodeMarker = "# mun" + "inn: automatic local memory recall"
)

func isMarker(l string) bool {
	t := strings.TrimSpace(l)
	return t == jcodeMarker || t == oldJcodeMarker
}

// JcodeRecallNeedsMigration reports whether the jcode config in jcodeHome
// runs hug's recall hook but not on every key that delivers it now
// (before hug 0.7.53 it ran on turn_start alone and typed the block into
// the person's input box). bin is what that config runs before the hook
// words (a muninn link, or hug), so the migration keeps it.
func JcodeRecallNeedsMigration(jcodeHome string) (bin string, stale bool) {
	b, err := os.ReadFile(filepath.Join(jcodeHome, "config.toml"))
	if err != nil {
		return "", false
	}
	text := string(b)
	cmd := ""
	for _, key := range jcodeRecallKeys {
		cmds, err := jcodeHookCommands(text, key)
		if err != nil {
			return "", false
		}
		for _, c := range cmds {
			if isRecallHook(c) && cmd == "" {
				cmd = c
			}
		}
	}
	if cmd == "" {
		return "", false
	}
	bin = hugconf.HookPrefix(cmd, "jcode")
	_, changed, err := editJcodeHooks(text, cmd, true)
	return bin, err == nil && changed
}

// jcodeHookCommands returns the commands on one [hooks] key, read the way
// editJcodeHook reads them.
func jcodeHookCommands(text, key string) ([]string, error) {
	lines := strings.Split(text, "\n")
	inHooks := false
	for i := 0; i < len(lines); i++ {
		t := strings.TrimSpace(lines[i])
		if strings.HasPrefix(t, "[") {
			inHooks = t == "[hooks]"
			continue
		}
		if inHooks && (strings.HasPrefix(t, key+"=") || strings.HasPrefix(t, key+" ")) {
			end := i
			for end < len(lines)-1 && !tomlBalanced(strings.Join(lines[i:end+1], "\n")) {
				end++
			}
			raw := strings.Join(lines[i:end+1], "\n")
			return tomlStrings(strings.TrimSpace(raw[strings.Index(raw, "=")+1:]))
		}
	}
	return nil, nil
}

// JcodeHookInstalled reports whether a jcode config.toml already runs hug's
// recall hook (either form).
func JcodeHookInstalled(jcodeHome string) bool {
	b, err := os.ReadFile(filepath.Join(jcodeHome, "config.toml"))
	if err != nil {
		return false
	}
	// Removing the hook would change the file exactly when it is installed.
	_, changed, err := editJcodeHooks(string(b), "", false)
	return err == nil && changed
}

// jcodeRecallKeys are the [hooks] keys that run muninn's recall hook: the
// turn's start computes and parks the block, the first successful tool call
// delivers it inside its result, and the turn's end drops what was not
// taken (z9q42h). The command switches on JCODE_HOOK_EVENT.
var jcodeRecallKeys = []string{"turn_start", "post_tool_feedback", "turn_end"}

// editJcodeHooks puts cmd on every recall key (install) or takes muninn's
// recall hook off them, keeping every other hook on those keys.
func editJcodeHooks(text, cmd string, install bool) (string, bool, error) {
	changed := false
	for _, key := range jcodeRecallKeys {
		next, c, err := editJcodeHook(text, key, cmd, install)
		if err != nil {
			return text, false, err
		}
		text, changed = next, changed || c
	}
	return text, changed, nil
}

// isRecallHook reports whether a hook command is hug's jcode recall hook
// (not, say, "muninn capture --hook", which shares turn_end).
func isRecallHook(e string) bool {
	return hugconf.IsRecallHook(e, "jcode")
}

// editJcodeHook edits one key inside [hooks]. jcode accepts a string or an
// array of strings, on one line or several, so an existing hook is kept
// alongside muninn.
func editJcodeHook(text, key, cmd string, install bool) (string, bool, error) {
	lines := strings.Split(text, "\n")
	inHooks, hooksAt, keyAt, keyEnd := false, -1, -1, -1
	for i := 0; i < len(lines); i++ {
		t := strings.TrimSpace(lines[i])
		if strings.HasPrefix(t, "[") {
			inHooks = t == "[hooks]"
			if inHooks {
				hooksAt = i
			}
			continue
		}
		if inHooks && keyAt < 0 && (strings.HasPrefix(t, key+"=") || strings.HasPrefix(t, key+" ")) {
			keyAt, keyEnd = i, i
			// A multi-line array runs until its brackets balance.
			for keyEnd < len(lines)-1 && !tomlBalanced(strings.Join(lines[keyAt:keyEnd+1], "\n")) {
				keyEnd++
			}
		}
	}
	var existing []string
	if keyAt >= 0 {
		raw := strings.Join(lines[keyAt:keyEnd+1], "\n")
		v := strings.TrimSpace(raw[strings.Index(raw, "=")+1:])
		arr, err := tomlStrings(v)
		if err != nil {
			return text, false, fmt.Errorf("cannot parse %s = %s (%v): edit it by hand", key, v, err)
		}
		existing = arr
	}
	var next []string
	has := false
	for _, e := range existing {
		if isRecallHook(e) {
			if has || !install {
				has = true
				continue
			}
			has = true
			e = cmd
		}
		next = append(next, e)
	}
	if install && !has {
		next = append(next, cmd)
	}
	if strings.Join(next, "\x00") == strings.Join(existing, "\x00") && len(next) == len(existing) {
		return text, false, nil
	}
	var value string
	switch len(next) {
	case 0:
		value = ""
	case 1:
		value = fmt.Sprintf("%q", next[0])
	default:
		b, _ := json.Marshal(next)
		value = string(b)
	}
	line := key + " = " + value
	switch {
	case keyAt >= 0 && value == "":
		lines = append(lines[:keyAt], lines[keyEnd+1:]...)
		if keyAt > 0 && isMarker(lines[keyAt-1]) && (keyAt >= len(lines) || !isRecallLine(lines[keyAt])) {
			lines = append(lines[:keyAt-1], lines[keyAt:]...)
		}
	case keyAt >= 0:
		lines = append(lines[:keyAt], append([]string{line}, lines[keyEnd+1:]...)...)
		if keyAt > 0 && strings.TrimSpace(lines[keyAt-1]) == oldJcodeMarker && isRecallLine(line) {
			lines[keyAt-1] = jcodeMarker
		}
	case hooksAt >= 0:
		// New keys go together under hug's marker, right after [hooks].
		at := hooksAt + 1
		if at < len(lines) && isMarker(lines[at]) {
			lines[at] = jcodeMarker
			for at++; at < len(lines) && isRecallLine(lines[at]); at++ {
			}
			lines = append(lines[:at], append([]string{line}, lines[at:]...)...)
		} else {
			lines = append(lines[:at], append([]string{jcodeMarker, line}, lines[at:]...)...)
		}
	default:
		lines = append(lines, "", "[hooks]", jcodeMarker, line)
	}
	return strings.Join(lines, "\n"), true, nil
}

// isRecallLine reports whether a config line sets a recall key to muninn's
// hook alone (the lines muninn adds under its marker).
func isRecallLine(l string) bool {
	t := strings.TrimSpace(l)
	for _, k := range jcodeRecallKeys {
		if strings.HasPrefix(t, k+" = \"") && isRecallHook(strings.TrimSuffix(strings.TrimPrefix(t, k+" = \""), "\"")) {
			return true
		}
	}
	return false
}

// tomlBalanced reports whether every [ in s outside strings and comments is
// closed.
func tomlBalanced(s string) bool {
	depth, quote, comment := 0, byte(0), false
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case comment:
			comment = c != '\n'
		case quote != 0:
			if c == '\\' && quote == '"' {
				i++
			} else if c == quote {
				quote = 0
			}
		case c == '"' || c == '\'':
			quote = c
		case c == '#':
			comment = true
		case c == '[':
			depth++
		case c == ']':
			depth--
		}
	}
	return depth <= 0 && quote == 0
}

// tomlStrings parses a TOML string or array of strings: basic ("...") and
// literal ('...') strings, commas, a trailing comma, newlines and comments.
func tomlStrings(v string) ([]string, error) {
	var out []string
	i, arr := 0, false
	skip := func() {
		for i < len(v) {
			switch c := v[i]; {
			case c == ' ' || c == '\t' || c == '\n' || c == '\r' || (arr && c == ','):
				i++
			case c == '#':
				for i < len(v) && v[i] != '\n' {
					i++
				}
			default:
				return
			}
		}
	}
	skip()
	if i < len(v) && v[i] == '[' {
		arr = true
		i++
	}
	for {
		skip()
		if i >= len(v) {
			if arr {
				return nil, errors.New("unclosed array")
			}
			return out, nil
		}
		switch v[i] {
		case ']':
			if !arr {
				return nil, errors.New("unexpected ]")
			}
			i++
			skip()
			if i < len(v) {
				return nil, errors.New("text after the array")
			}
			return out, nil
		case '"':
			j := i + 1
			for j < len(v) && v[j] != '"' {
				if v[j] == '\\' {
					j++
				}
				j++
			}
			if j >= len(v) {
				return nil, errors.New("unclosed string")
			}
			var s string
			if err := json.Unmarshal([]byte(v[i:j+1]), &s); err != nil {
				return nil, err
			}
			out, i = append(out, s), j+1
		case '\'':
			j := strings.IndexByte(v[i+1:], '\'')
			if j < 0 {
				return nil, errors.New("unclosed string")
			}
			out, i = append(out, v[i+1:i+1+j]), i+j+2
		default:
			return nil, fmt.Errorf("unexpected %q", v[i])
		}
		if !arr {
			skip()
			if i < len(v) {
				return nil, errors.New("text after the string")
			}
			return out, nil
		}
	}
}

func shellQuote(s string) string {
	if !strings.ContainsAny(s, " '\"$`\\") {
		return s
	}
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}
