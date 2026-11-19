package cli

import (
	"context"
	"os"
	"path/filepath"

	"github.com/veridit/hug/huginn/internal/harness"
	"github.com/veridit/hug/internal/hugconf"
)

func (a *App) runInstall(cmd string, f flags) (Reply, error) {
	if err := f.count(1); err != nil {
		return nil, err
	}
	if f.args[0] == "commit-msg" {
		if f.has("all-instances") || f.has("jcode-home") {
			return nil, usage("--all-instances and --jcode-home are for jcode", "hug task "+cmd+" commit-msg")
		}
		return a.runCommitMsg(context.Background(), cmd, f.has("dry-run"))
	}
	if f.args[0] != "jcode" {
		return nil, usage("supported: jcode, commit-msg", "hug task "+cmd+" jcode")
	}
	if f.has("all-instances") && f.has("jcode-home") {
		return nil, usage("--all-instances and --jcode-home are mutually exclusive", "hug task "+cmd+" jcode --help")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	var homes []string
	switch {
	case f.has("all-instances"):
		extras := []string{}
		if h := os.Getenv("JCODE_HOME"); h != "" {
			extras = append(extras, h)
		}
		for _, i := range harness.DiscoverJcode(home, extras) {
			homes = append(homes, i.Home)
		}
		if len(homes) == 0 {
			return nil, usage("no Jcode instances found", "hug task "+cmd+" jcode --jcode-home PATH")
		}
	case f.has("jcode-home"):
		if f.get("jcode-home") == "" {
			return nil, usage("--jcode-home cannot be empty", "hug task "+cmd+" jcode --help")
		}
		h := f.get("jcode-home")
		if !filepath.IsAbs(h) {
			h = filepath.Join(a.Cwd, h)
		}
		homes = []string{h}
	default:
		h := os.Getenv("JCODE_HOME")
		if h == "" {
			h = filepath.Join(home, ".jcode")
		}
		homes = []string{h}
	}
	// The MCP entry runs hug (ynh84t): hug mcp serves the task tools as
	// task_* next to memory, the same entry hug mem install writes.
	bin := hugconf.HugPath()
	// Preflight every instance before modifying any of them.
	for _, h := range homes {
		if _, err := harness.InstallJcodeHome(h, bin, cmd == "install", true); err != nil {
			return nil, err
		}
	}
	results := []harness.InstallResult{}
	for _, h := range homes {
		r, err := harness.InstallJcodeHome(h, bin, cmd == "install", f.has("dry-run"))
		if err != nil {
			return nil, err
		}
		results = append(results, r)
	}
	return Reply{"installations": results, "dry_run": f.has("dry-run"), "next": "hug task next"}, nil
}
