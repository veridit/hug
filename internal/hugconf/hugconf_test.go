package hugconf

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestHookForms(t *testing.T) {
	if got := HookCommand("/opt/homebrew/bin/hug", "claude"); got != "/opt/homebrew/bin/hug mem hook claude" {
		t.Fatalf("HookCommand = %q", got)
	}
	if got := HookCommand("/old/bin/muninn", "jcode"); got != "/old/bin/muninn hook jcode" {
		t.Fatalf("a --bin muninn keeps the old form: %q", got)
	}
	if got := HookCommand("/My Apps/hug", "codex"); got != "'/My Apps/hug' mem hook codex" {
		t.Fatalf("quoted: %q", got)
	}
	if got := HookCommand("HUG_HOME=/m /x/muninn", "jcode"); got != "HUG_HOME=/m /x/muninn hook jcode" {
		t.Fatalf("a kept prefix: %q", got)
	}
	if got := HookPrefix("HUG_HOME=/m '/My Apps/hug' mem hook jcode", "jcode"); got != "HUG_HOME=/m '/My Apps/hug'" || HookCommand(got, "jcode") != "HUG_HOME=/m '/My Apps/hug' mem hook jcode" {
		t.Fatalf("prefix round trip: %q", got)
	}
	for cmd, want := range map[string]bool{
		"/x/muninn hook jcode":              true,
		"/x/hug mem hook jcode":             true,
		"HUG_HOME=/m /x/hug mem hook jcode": true,
		"'/My Apps/hug' mem hook jcode":     true,
		"/x/muninn capture --hook":          false,
		"/x/hug mem capture --hook":         false,
		"/x/hug mem hook claude":            false,
		"/x/other hook jcode":               false,
		"echo muninn hook jcode":            false,
	} {
		if IsRecallHook(cmd, "jcode") != want {
			t.Errorf("IsRecallHook(%q) = %v", cmd, !want)
		}
	}
}

func TestMoveCommand(t *testing.T) {
	dir := filepath.ToSlash(t.TempDir()) // as a hook command spells it
	withHug := dir + "/bin"
	os.MkdirAll(withHug, 0o755)
	os.WriteFile(withHug+"/hug", nil, 0o755)
	for in, want := range map[string]string{
		withHug + "/muninn hook jcode":              withHug + "/hug mem hook jcode",
		"HUG_HOME=/m " + withHug + "/muninn hook c": "HUG_HOME=/m " + withHug + "/hug mem hook c",
		"/no/hug/here/muninn capture --hook":        "/fallback/hug mem capture --hook",
		"/no/hug/here/huginn hook post-tool":        "/fallback/hug task hook post-tool",
		"muninn hook claude":                        "/fallback/hug mem hook claude",
	} {
		got, ok := MoveCommand(in, "/fallback/hug")
		if !ok || got != want {
			t.Errorf("MoveCommand(%q) = %q %v, want %q", in, got, ok, want)
		}
	}
	for _, in := range []string{"/x/hug mem hook jcode", "~/.jcode/bin/disk_turn_end.sh", "echo muninn"} {
		if got, ok := MoveCommand(in, "/f/hug"); ok || got != in {
			t.Errorf("MoveCommand(%q) moved it: %q", in, got)
		}
	}
}

func TestOwned(t *testing.T) {
	for raw, want := range map[string]bool{
		`{"command":"/b/muninn","args":["mcp"],"env":{}}`:                  true,
		`{"command":"/b/huginn","args":["mcp"]}`:                           true,
		`{"command":"/b/hug","args":["mcp","--jcode-home","/j"],"env":{}}`: true,
		`{"type":"stdio","command":"hug","args":["mcp"]}`:                  true,
		`{"command":"/b/muninn","args":["mcp"],"env":{"HUG_HOME":"/m"}}`:   true,
		`{"command":"/b/muninn","args":["mcp"],"env":{"OTHER":"1"}}`:       false,
		`{"command":"/b/muninn","args":["serve"]}`:                         false,
		`{"command":"/b/other","args":["mcp"]}`:                            false,
		`{"command":"/b/hug","args":["mcp"],"shared":true}`:                false,
		`{"type":"http","url":"https://example.invalid/mcp"}`:              false,
	} {
		if _, ok := Owned(json.RawMessage(raw)); ok != want {
			t.Errorf("Owned(%s) = %v", raw, ok)
		}
	}
	if h, _ := Owned(json.RawMessage(`{"command":"/b/muninn","args":["mcp","--jcode-home","/j"]}`)); h != "/j" {
		t.Errorf("jcode home %q", h)
	}
}

func TestToolPermission(t *testing.T) {
	for in, want := range map[string]string{
		"mcp__muninn__memory_search": "mcp__hug__memory_search",
		"mcp__muninn__*":             "mcp__hug__*",
		"mcp__muninn":                "mcp__hug",
		"mcp__huginn__take":          "mcp__hug__task_take",
		"mcp__huginn__task_take":     "mcp__hug__task_take",
		"mcp__huginn__*":             "mcp__hug__*",
	} {
		if got, ok := ToolPermission(in); !ok || got != want {
			t.Errorf("ToolPermission(%q) = %q %v, want %q", in, got, ok, want)
		}
	}
	for _, in := range []string{"mcp__hug__memory_search", "mcp__backlog__task_list", "Bash(muninn:*)", "mcp__muninnx__a"} {
		if got, ok := ToolPermission(in); ok || got != in {
			t.Errorf("ToolPermission(%q) changed it: %q", in, got)
		}
	}
}
