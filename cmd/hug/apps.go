package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"

	"time"
)

const appsHelp = `hug apps [install] [--dry-run]: desktop and CLI apps that should start hug mcp.

Checks Claude Desktop, Claude Code (~/.claude.json) and ChatGPT/Codex
(~/.codex/config.toml): whether each has a "hug" MCP server and whether it
points at this hug. "hug apps install" adds or updates the entry, keeping a
timestamped backup and every other server; --dry-run shows what would change.
Restart an app after installing so it starts the new server.
jcode instances are handled by "hug mem install jcode --all-instances".
`

type appTarget struct {
	Name   string `json:"name"`
	Path   string `json:"path"`
	Found  bool   `json:"found"`
	Status string `json:"status"` // ok, missing, stale, absent
	Detail string `json:"detail,omitempty"`
	kind   string // json-servers (key mcpServers) or toml
}

func appTargets(home string) []appTarget {
	var out []appTarget
	if runtime.GOOS == "darwin" {
		out = append(out, appTarget{Name: "Claude Desktop", Path: filepath.Join(home, "Library/Application Support/Claude/claude_desktop_config.json"), kind: "json"})
	} else if runtime.GOOS == "windows" {
		out = append(out, appTarget{Name: "Claude Desktop", Path: filepath.Join(os.Getenv("APPDATA"), "Claude/claude_desktop_config.json"), kind: "json"})
	} else {
		out = append(out, appTarget{Name: "Claude Desktop", Path: filepath.Join(home, ".config/Claude/claude_desktop_config.json"), kind: "json"})
	}
	out = append(out,
		appTarget{Name: "Claude Code", Path: filepath.Join(home, ".claude.json"), kind: "json"},
		appTarget{Name: "ChatGPT / Codex", Path: filepath.Join(home, ".codex/config.toml"), kind: "toml"},
	)
	return out
}

// hugBin is the path apps should run: the stable link (e.g.
// /opt/homebrew/bin/hug), never a versioned Cellar path.
func hugBin() string {
	for _, p := range []string{"/opt/homebrew/bin/hug", "/usr/local/bin/hug", "/home/linuxbrew/.linuxbrew/bin/hug"} {
		if _, e := os.Stat(p); e == nil {
			return p
		}
	}
	if p, e := os.Executable(); e == nil {
		return p
	}
	return "hug"
}

var tomlHugSection = regexp.MustCompile(`(?m)^\[mcp_servers\.hug\]\s*$`)

func inspectApp(t appTarget, bin string) appTarget {
	b, e := os.ReadFile(t.Path)
	if e != nil {
		t.Status, t.Detail = "absent", "app config not found"
		return t
	}
	t.Found = true
	switch t.kind {
	case "json":
		var doc map[string]json.RawMessage
		if json.Unmarshal(b, &doc) != nil {
			t.Status, t.Detail = "error", "config is not valid JSON; not touched"
			return t
		}
		var servers map[string]json.RawMessage
		_ = json.Unmarshal(doc["mcpServers"], &servers)
		raw, ok := servers["hug"]
		if !ok {
			t.Status = "missing"
			if _, old := servers["muninn"]; old {
				t.Detail = "has old muninn/huginn servers"
			}
			return t
		}
		if !bytes.Contains(raw, []byte(bin)) && !bytes.Contains(raw, []byte(`"hug"`)) {
			t.Status, t.Detail = "stale", "hug entry points elsewhere: "+string(raw)
			return t
		}
		t.Status = "ok"
	case "toml":
		if !tomlHugSection.Match(b) {
			t.Status = "missing"
			return t
		}
		t.Status = "ok"
	}
	return t
}

func installApp(t appTarget, bin string, dry bool) (string, error) {
	b, e := os.ReadFile(t.Path)
	if e != nil {
		return "", e
	}
	var out []byte
	switch t.kind {
	case "json":
		var doc map[string]json.RawMessage
		if e = json.Unmarshal(b, &doc); e != nil {
			return "", e
		}
		servers := map[string]json.RawMessage{}
		if raw, ok := doc["mcpServers"]; ok {
			if e = json.Unmarshal(raw, &servers); e != nil {
				return "", e
			}
		}
		entry := map[string]any{"command": bin, "args": []string{"mcp"}}
		if t.Name == "Claude Code" {
			entry["type"] = "stdio"
		}
		servers["hug"], _ = json.Marshal(entry)
		// The old part servers are replaced by hug (same tools, task_ prefix).
		delete(servers, "muninn")
		delete(servers, "huginn")
		doc["mcpServers"], _ = json.Marshal(servers)
		out, e = json.MarshalIndent(doc, "", "  ")
		if e != nil {
			return "", e
		}
		out = append(out, '\n')
	case "toml":
		if tomlHugSection.Match(b) {
			return "already present", nil
		}
		out = append(bytes.TrimRight(b, "\n"), []byte(fmt.Sprintf("\n\n[mcp_servers.hug]\ncommand = %q\nargs = [\"mcp\"]\nstartup_timeout_sec = 60\n", bin))...)
	}
	if dry {
		return "would update", nil
	}
	backup := t.Path + ".bak-" + time.Now().Format("20060102T150405")
	if e = os.WriteFile(backup, b, 0o600); e != nil {
		return "", e
	}
	info, _ := os.Stat(t.Path)
	tmp := t.Path + ".tmp"
	if e = os.WriteFile(tmp, out, info.Mode().Perm()); e != nil {
		return "", e
	}
	if e = os.Rename(tmp, t.Path); e != nil {
		return "", e
	}
	return "updated (backup " + filepath.Base(backup) + "); restart the app", nil
}

func runApps(args []string) int {
	install, dry, jsonOut := false, false, false
	for _, a := range args {
		switch a {
		case "install":
			install = true
		case "--dry-run":
			dry = true
		case "--json":
			jsonOut = true
		case "-h", "--help", "help":
			fmt.Print(appsHelp)
			return 0
		default:
			fmt.Fprintf(os.Stderr, "hug apps: unknown argument %q\nfix: hug apps --help\n", a)
			return 2
		}
	}
	home, e := os.UserHomeDir()
	if e != nil {
		fmt.Fprintln(os.Stderr, "hug apps:", e)
		return 1
	}
	bin := hugBin()
	var results []map[string]any
	open := 0
	for _, t := range appTargets(home) {
		t = inspectApp(t, bin)
		r := map[string]any{"name": t.Name, "path": t.Path, "status": t.Status}
		if t.Detail != "" {
			r["detail"] = t.Detail
		}
		if install && (t.Status == "missing" || t.Status == "stale") {
			msg, err := installApp(t, bin, dry)
			if err != nil {
				r["error"] = err.Error()
			} else {
				r["action"] = msg
			}
		} else if t.Status == "missing" || t.Status == "stale" {
			open++
		}
		results = append(results, r)
	}
	if jsonOut {
		next := "hug apps install"
		if open == 0 {
			next = "hug setup"
		}
		_ = json.NewEncoder(os.Stdout).Encode(map[string]any{"apps": results, "open": open, "next": next})
		return 0
	}
	for _, r := range results {
		line := fmt.Sprintf("%-16s %-8s %s", r["name"], r["status"], r["path"])
		if d, ok := r["detail"]; ok {
			line += "  (" + d.(string) + ")"
		}
		if a, ok := r["action"]; ok {
			line += "  -> " + a.(string)
		}
		if er, ok := r["error"]; ok {
			line += "  error: " + er.(string)
		}
		fmt.Println(line)
	}
	if open > 0 {
		fmt.Printf("next: hug apps install --dry-run, then hug apps install\n")
	} else {
		fmt.Println("next: restart any app that was just updated")
	}
	return 0
}
