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

	"github.com/spf13/cobra"

	huginn "github.com/veridit/hug/huginn/run"
	"github.com/veridit/hug/internal/env"
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
  hug upgrade [--apply [--machine]]
                            what every migration to the current hug would change (exit 5
                            if any); --apply does this board's, --apply --machine this
                            machine's too (hook and MCP configs, the memory store)
  hug version
  hug completion bash|zsh|fish
                            the shell completion script; source <(hug completion zsh)
                            in ~/.zshrc (hug completion --help for bash and fish)

Details: hug task help, hug mem --help.
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
	// Shell completion (5mmqjk) answers on every TAB, before anything else:
	// never a refusal or a warning on the shell's line. Through the muninn
	// link it stays hug mem's (scripts installed before keep answering).
	if len(args) > 0 && (args[0] == cobra.ShellCompRequestCmd || args[0] == cobra.ShellCompNoDescRequestCmd) {
		switch strings.TrimSuffix(name, ".exe") {
		case "muninn":
			return muninn.Main(args)
		case "huginn":
			return complete(append([]string{args[0], "task"}, args[1:]...))
		}
		return complete(args)
	}
	// Old huginn and muninn variables are not read (m1c28r): say so, once
	// per command, on stderr (stdout may be JSON or a hook's reply).
	warnStale(os.Stderr, nil)
	// A HUG_ variable hug does not know is a misspelt setting: stop before
	// anything runs (a0p5kt). A hook does nothing and exits 0, so the
	// harness is never broken (exit 2 blocks a Claude Code prompt).
	if !helpOrVersion(args) && refuseUnknown(os.Stderr, nil, commandName(name, args)) {
		if isHook(name, args) {
			// Not silent: the hook log says which variable and the fix.
			problem, fix := env.Refusal(env.Unknowns(nil))
			muninn.LogHook("%s: refused, did nothing: %s; fix: %s", commandName(name, args), problem, fix)
			return 0
		}
		return 2
	}
	// Called through a link named after a part: behave exactly as that part.
	switch strings.TrimSuffix(name, ".exe") {
	case "huginn":
		return huginn.Main(args)
	case "muninn":
		return muninn.Main(args)
	}
	if len(args) == 0 {
		fmt.Print(help)
		fmt.Print("\n" + env.HelpText())
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
	case "completion":
		return runCompletion(args[1:])
	case "setup", "tui", "upgrade":
		return huginn.Main(args)
	case "version", "--version":
		fmt.Printf("hug %s\n", Version)
		return 0
	case "help", "-h", "--help":
		fmt.Print(help)
		fmt.Print("\n" + env.HelpText())
		return 0
	}
	fmt.Fprintf(os.Stderr, "hug: unknown command %q\nfix: hug help\n", args[0])
	return 2
}
