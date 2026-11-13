// Command hug is one tool for a project's memory and tasks, for people and
// AI agents in every harness. Its parts keep their raven names:
//
//	hug task ...   tasks on the project's git branch (huginn, the raven of thought)
//	hug mem ...    memory recall, capture and dreaming (muninn, the raven of memory)
//
// The muninn and huginn commands remain as short names for the same parts.
package main

import (
	"fmt"
	"os"
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
  hug version

Details: hug task help, hug mem --help.
The commands huginn (= hug task) and muninn (= hug mem) work too.
`

func main() { os.Exit(run(filepath.Base(os.Args[0]), os.Args[1:])) }

func run(name string, args []string) int {
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
	case "setup", "tui":
		return huginn.Main(args)
	case "version", "--version":
		fmt.Printf("hug %s (memory: muninn %s, tasks: huginn %s)\n", Version, muninn.Version, huginn.Version)
		return 0
	case "help", "-h", "--help":
		fmt.Print(help)
		return 0
	}
	fmt.Fprintf(os.Stderr, "hug: unknown command %q\nfix: hug help\n", args[0])
	return 2
}
