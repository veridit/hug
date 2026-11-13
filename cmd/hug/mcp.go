package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	huginn "github.com/veridit/hug/huginn/run"
	muninn "github.com/veridit/hug/muninn/run"
)

const mcpHelp = `hug mcp [--jcode-home DIR]: one MCP server over stdio with both parts.

Tools: memory_* (search, remember, forget, recall, related, dream, stats) and
task_* (list, show, new, update, next, take, done, lint, sync, types).
Tasks work on the git repository in the current directory. --jcode-home binds
memory to one jcode instance, as muninn mcp --jcode-home does.
If memory cannot open, the task tools are still served and the reason is
logged to stderr.
`

// taskPrefix names the task tools in hug mcp (task_list, task_show, ...).
const taskPrefix = "task_"

func runMCP(args []string) int {
	jcodeHome := os.Getenv("JCODE_HOME")
	for i := 0; i < len(args); i++ {
		switch a := args[i]; {
		case a == "--help" || a == "-h" || a == "help":
			fmt.Print(mcpHelp)
			return 0
		case a == "--jcode-home" && i+1 < len(args):
			jcodeHome = args[i+1]
			i++
		case len(a) > len("--jcode-home=") && a[:len("--jcode-home=")] == "--jcode-home=":
			jcodeHome = a[len("--jcode-home="):]
		default:
			fmt.Fprintf(os.Stderr, "hug mcp: unknown argument %q\nfix: hug mcp --help\n", a)
			return 2
		}
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	s, closeFn, err := newMCPServer(ctx, jcodeHome)
	if err != nil {
		fmt.Fprintln(os.Stderr, "hug mcp:", err)
		return 1
	}
	defer closeFn()
	if err := s.Run(ctx, &sdk.StdioTransport{}); err != nil && ctx.Err() == nil {
		fmt.Fprintln(os.Stderr, "hug mcp:", err)
		return 1
	}
	return 0
}

// newMCPServer builds the combined server. Memory failing to open is not
// fatal: tasks still work, and the instructions say why memory is missing.
func newMCPServer(ctx context.Context, jcodeHome string) (*sdk.Server, func(), error) {
	instr := "hug: memory and tasks for this project.\n\n"
	memInstr, register, closeFn, err := muninn.OpenMCP(ctx, jcodeHome)
	if err != nil {
		fmt.Fprintln(os.Stderr, "hug mcp: memory unavailable:", err)
		instr += "Memory tools are unavailable in this session: " + err.Error() + "\n\n"
		closeFn = func() {}
	} else {
		instr += memInstr + "\n\n"
	}
	instr += huginn.MCPInstructions(taskPrefix)
	s := sdk.NewServer(&sdk.Implementation{Name: "hug", Version: Version}, &sdk.ServerOptions{Instructions: instr})
	if register != nil {
		register(s)
	}
	if err := huginn.RegisterMCP(s, taskPrefix); err != nil {
		closeFn()
		return nil, nil, err
	}
	return s, closeFn, nil
}
