package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	huginn "github.com/veridit/hug/huginn/run"
	muninn "github.com/veridit/hug/muninn/run"
)

const mcpHelp = `hug mcp [--jcode-home DIR]: one MCP server over stdio with both parts.

Tools: memory_* (search, remember, forget, recall, related, dream, stats) and
task_* (list, show, new, update, next, take, done, lint, sync, types).
Tasks work on the git repository in the current directory. --jcode-home binds
memory to one jcode instance, as hug mem mcp --jcode-home does.
hug mcp answers at once; memory opens in the background. A memory tool
called before it is open says memory is still opening. If memory cannot
open, the task tools are still served, the memory tools say why, and the
reason is logged to stderr.
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
		case a == "--can-resume":
			// A reloading hug mcp asks this before it execs us (mbdfzs).
			fmt.Println(resumeProbe)
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
	resumed, err := takeResume()
	if err != nil {
		fmt.Fprintln(os.Stderr, "hug mcp: the session handed over by the old hug is lost:", err)
		return 1
	}
	r := newReloader(launchPath(), os.Stdout, os.Stderr)
	go r.pump(os.Stdin, resumed)
	if err := s.Run(ctx, &sdk.IOTransport{Reader: r, Writer: r}); err != nil && ctx.Err() == nil {
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
	s.AddReceivingMiddleware(staleBinaryNotice(), versionMeta(), testHold(os.Getenv("HUG_MCP_TEST_HOLD")))
	return s, closeFn, nil
}

// versionMeta names the hug that served each result in its _meta, so a
// client (and the reload test, mbdfzs) can see which binary answered.
func versionMeta() sdk.Middleware {
	return func(next sdk.MethodHandler) sdk.MethodHandler {
		return func(ctx context.Context, method string, req sdk.Request) (sdk.Result, error) {
			res, err := next(ctx, method, req)
			if r, ok := res.(*sdk.CallToolResult); ok && r != nil {
				m := r.GetMeta()
				if m == nil {
					m = map[string]any{}
				}
				m["hug/version"] = Version
				r.SetMeta(m)
			}
			return res, err
		}
	}
}

// testHold is for the reload test (mbdfzs check 2): with HUG_MCP_TEST_HOLD
// set to a path, a task_types call writes PATH.started and returns only once
// PATH exists, so a test can change the binary while a request is running.
func testHold(path string) sdk.Middleware {
	return func(next sdk.MethodHandler) sdk.MethodHandler {
		if path == "" {
			return next
		}
		return func(ctx context.Context, method string, req sdk.Request) (sdk.Result, error) {
			if p, ok := req.GetParams().(*sdk.CallToolParamsRaw); ok && method == "tools/call" && p.Name == taskPrefix+"types" {
				os.WriteFile(path+".started", nil, 0o644)
				for {
					if _, e := os.Stat(path); e == nil || ctx.Err() != nil {
						break
					}
					time.Sleep(10 * time.Millisecond)
				}
			}
			return next(ctx, method, req)
		}
	}
}

// staleBinaryNotice adds a note to tool results when the hug on disk is newer
// than this running server: an app keeps old servers alive across upgrades
// (2026-11-16: Claude Desktop ran 0.7.2 after 0.7.4 was installed and
// failed on the moved task branch). The note asks for an app restart.
func staleBinaryNotice() sdk.Middleware {
	exe, err := os.Executable()
	if err != nil {
		return func(next sdk.MethodHandler) sdk.MethodHandler { return next }
	}
	exe, _ = filepath.EvalSymlinks(exe)
	// Only installed copies (Homebrew keeps versions side by side) can go
	// stale this way; a dev build or go install is never flagged.
	if !strings.Contains(exe, "/Cellar/hug/") {
		return func(next sdk.MethodHandler) sdk.MethodHandler { return next }
	}
	started := time.Now()
	link := hugBin()
	return func(next sdk.MethodHandler) sdk.MethodHandler {
		return func(ctx context.Context, method string, req sdk.Request) (sdk.Result, error) {
			res, err := next(ctx, method, req)
			if method != "tools/call" || err != nil {
				return res, err
			}
			cur, e := filepath.EvalSymlinks(link)
			if e != nil {
				return res, err
			}
			info, e := os.Stat(cur)
			if (cur != exe || (e == nil && info.ModTime().After(started))) && res != nil {
				if r, ok := res.(*sdk.CallToolResult); ok {
					r.Content = append(r.Content, &sdk.TextContent{Text: "note: hug was upgraded while this server was running (" + filepath.Base(filepath.Dir(filepath.Dir(cur))) + " installed); restart the app to load it"})
				}
			}
			return res, err
		}
	}
}
