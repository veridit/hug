// Command hug is one tool for a project's memory and tasks, for people and
// AI agents in every harness. Its parts keep their raven names:
//
//	hug task ...   tasks on the project's git branch (huginn, the raven of thought)
//	hug mem ...    memory recall, capture and dreaming (muninn, the raven of memory)
//
// The muninn and huginn commands remain as short names for the same parts.
package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	huginn "github.com/veridit/hug/huginn/run"
	muninn "github.com/veridit/hug/muninn/run"
)

// Version is set by the release build (-X main.Version=...).
var Version = "dev"

const help = `hug: one tool for a project's memory and tasks, for people and AI agents.

Usage:
  hug setup                 what this repository still needs, with exact commands
  hug task <command> ...    tasks on the git branch (next, take, show, done, new, ls, tui, ...)
  hug mem <command> ...     memory (recall, search, remember, forget, show, dream, tui, ...)
  hug tui                   task board (hug mem tui for memories)
  hug mcp                   one MCP server: memory_* and task_* tools
  hug apps [install]        Claude Desktop, Claude Code, ChatGPT/Codex: start hug mcp
  hug upgrade [--apply]     what every migration to the current hug would change (exit 5
                            if any); --apply does it
  hug version

Details: hug task help, hug mem --help.
The commands huginn (= hug task) and muninn (= hug mem) work too.
`

func main() { os.Exit(run(filepath.Base(os.Args[0]), os.Args[1:])) }

// memUpgrade runs this binary's hug mem upgrade for hug upgrade's memory
// part (5sc5rw): its own process, its output captured.
func memUpgrade(args []string) (int, string) {
	exe, err := os.Executable()
	if err != nil {
		return 1, err.Error()
	}
	out, err := exec.Command(exe, append([]string{"mem", "upgrade"}, args...)...).CombinedOutput()
	var ee *exec.ExitError
	if errors.As(err, &ee) {
		return ee.ExitCode(), string(out)
	}
	if err != nil {
		return 1, err.Error()
	}
	return 0, string(out)
}

func init() { huginn.MemUpgrade = memUpgrade }

func run(name string, args []string) int {
	// Old huginn and muninn variables are not read (m1c28r): say so, once
	// per command, on stderr (stdout may be JSON or a hook's reply).
	warnStale(os.Stderr, nil)
	// Called through a link named after a part: behave exactly as that part.
	switch strings.TrimSuffix(name, ".exe") {
	case "huginn":
		return huginn.Main(args)
	case "muninn":
		return muninn.Main(args)
	}
	if len(args) == 0 {
		fmt.Print(help)
		return 0
	}
	switch args[0] {
	case "task", "tasks", "huginn":
		return huginn.Main(args[1:])
	case "mem", "memory", "muninn":
		return muninn.Main(args[1:])
	case "mcp":
		return runMCP(args[1:])
	case "apps":
		return runApps(args[1:])
	case "setup", "tui", "upgrade":
		return huginn.Main(args)
	case "version", "--version":
		fmt.Printf("hug %s\n", Version)
		return 0
	case "help", "-h", "--help":
		fmt.Print(help)
		return 0
	}
	fmt.Fprintf(os.Stderr, "hug: unknown command %q\nfix: hug help\n", args[0])
	return 2
}
