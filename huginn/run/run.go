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

// Version is set by the release build (-X), the same string as hug's
// main.Version; "dev" like hug version in a plain go build.
var Version = "dev"

// MemUpgrade runs hug mem upgrade for hug upgrade's memory part (5sc5rw);
// cmd/hug sets it, so huginn does not import muninn.
var MemUpgrade func(args []string) (int, string)

// Main runs huginn with args (without the program name) and returns the exit code.
func Main(args []string) int {
	if len(args) == 1 && args[0] == "version" {
		fmt.Println("hug " + Version)
		return 0
	}
	cli.SpawnSelf()
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()
	cwd, e := os.Getwd()
	if e != nil {
		fmt.Fprintln(os.Stderr, e)
		return 1
	}
	a := &cli.App{Cwd: cwd, In: os.Stdin, Out: os.Stdout, Err: os.Stderr, Version: Version, MemUpgrade: MemUpgrade}
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
	cli.SpawnSelf()
	mcp.Register(s, &cli.App{Cwd: cwd, In: os.Stdin, Out: os.Stderr, Err: os.Stderr}, prefix)
	return nil
}

// MCPInstructions describes the task tools registered under prefix.
func MCPInstructions(prefix string) string { return mcp.PrefixedInstructions(prefix) }

// Acknowledge is hug's hook in jcode session sid (JCODE_HOME home)
// acknowledging what hug delivered to it once its transcript holds it
// (gmzd87). It returns what it acknowledged, "ID request N" each.
func Acknowledge(ctx context.Context, home, sid string) ([]string, error) {
	return cli.Acknowledge(ctx, home, sid)
}

// DeliverPending is hug's hook at the start of jcode session sid's turn
// (JCODE_HOME home): what could not reach the coordinator this session is
// named after is handed to it now, once (bbxdd7). It returns what it
// delivered, "ID request N" each.
func DeliverPending(ctx context.Context, home, sid string, getenv func(string) string) ([]string, error) {
	return cli.DeliverPending(ctx, home, sid, getenv)
}

// Complete answers shell completion for hug task: words after "hug task",
// the last the word being completed. It returns the words (a tab before a
// description) and cobra's directive, and never writes (5mmqjk).
func Complete(cwd string, words []string) ([]string, int) {
	return cli.Complete(context.Background(), cwd, words)
}

// NoFileComp is the directive that offers only the words given.
const NoFileComp = cli.NoFileComp
