// Command muninn is the memory part of hug, kept as its own command for
// existing installs, hooks and MCP configs.
package main

import (
	"os"

	"github.com/veridit/hug/muninn/run"
)

func main() { os.Exit(run.Main(os.Args[1:])) }
