// Package run is the huginn command line, callable from the hug binary.
package run

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"slices"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/veridit/hug/huginn/internal/cli"
	"github.com/veridit/hug/huginn/internal/mcp"
)

// Version is set by the release build (-X).
var Version = "v0.1.6"

// Main runs huginn with args (without the program name) and returns the exit code.
func Main(args []string) int {
	if len(args) == 1 && args[0] == "version" {
		fmt.Println("huginn " + Version)
		return 0
	}
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()
	cwd, e := os.Getwd()
	if e != nil {
		fmt.Fprintln(os.Stderr, e)
		return 1
	}
	a := &cli.App{Cwd: cwd, In: os.Stdin, Out: os.Stdout, Err: os.Stderr}
	if len(args) > 0 && args[0] == "mcp" && !slices.Contains(args, "--help") {
		if len(args) > 2 || (len(args) == 2 && args[1] != "--json") {
			return a.Run(ctx, []string{"mcp", "--invalid"})
		}
		if e = mcp.Run(ctx, a); e != nil {
			_, p, fix := cli.ErrorDetails(e)
			fmt.Fprintln(os.Stderr, p)
			fmt.Fprintln(os.Stderr, "fix: "+fix)
			return 1
		}
		return 0
	}
	return a.Run(ctx, args)
}

// RegisterMCP adds the task tools to s under prefix (hug mcp uses "task_"),
// working on the repository at the process's current directory.
func RegisterMCP(s *sdk.Server, prefix string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	mcp.Register(s, &cli.App{Cwd: cwd, In: os.Stdin, Out: os.Stderr, Err: os.Stderr}, prefix)
	return nil
}

// MCPInstructions describes the task tools registered under prefix.
func MCPInstructions(prefix string) string { return mcp.PrefixedInstructions(prefix) }
