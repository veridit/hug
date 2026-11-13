package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/veridit/hug/huginn/internal/importer"
)

// SetupStep is one item of the setup checklist.
type SetupStep struct {
	Step   string `json:"step"`
	Done   bool   `json:"done"`
	Detail string `json:"detail"`
	Do     string `json:"do,omitempty"` // exact command or edit when not done
}

// setupIntro is printed above the checklist. It is written for the agent
// that runs it: everything it needs to switch a repo to huginn itself.
const setupIntro = `huginn setup: switch this repository's task tracking to huginn.
For the agent reading this: work through the checklist top to bottom. Each
open step has the exact command or edit. Run "huginn setup" again after each
step until every step is done. setup itself only reads; it changes nothing.

How huginn works, in short:
- Tasks are Markdown files on the git branch "huginn", checked out at .huginn/.
  Code branches never see them. "huginn sync" shares them through the remote.
- Every task has a goal and observable acceptance checks. Work it like this
  (CLI shown; MCP has list, show, new, next, take, done and update):
    huginn next            pick the top task (or huginn take ID)
    huginn show ID         read goal, checks, notes, dependencies
    huginn check ID N      tick check N when you have evidence
    huginn note ID "..."   record evidence, commands, findings
    huginn done ID         only when every check is ticked
- New work: huginn new "title" --goal "outcome" --accept "observable check"
- Use --json for structured replies; every reply has a "next" hint.
- Humans browse with "huginn tui".

`

// runSetup inspects the repository and returns the checklist.
func (a *App) runSetup(ctx context.Context, f flags) (Reply, error) {
	if e := f.count(0); e != nil {
		return nil, e
	}
	cwd := a.Cwd
	if cwd == "" {
		cwd = "."
	}
	out, err := exec.CommandContext(ctx, "git", "-C", cwd, "rev-parse", "--show-toplevel").Output()
	if err != nil {
		return nil, &OperationError{Problem: "huginn setup needs a Git checkout as its working directory", Fix: "cd /path/to/your/git/repo && huginn setup"}
	}
	root := strings.TrimSpace(string(out))
	var steps []SetupStep
	add := func(step string, done bool, detail, do string) {
		if done {
			do = ""
		}
		steps = append(steps, SetupStep{step, done, detail, do})
	}

	_, statErr := os.Stat(filepath.Join(root, ".huginn"))
	branch := exec.CommandContext(ctx, "git", "-C", root, "rev-parse", "--verify", "--quiet", "refs/heads/huginn").Run() == nil
	add("task branch", branch && statErr == nil, "branch huginn checked out at .huginn/", "huginn init")

	backlogDir := ""
	for _, d := range []string{".backlog", "backlog"} {
		if fi, e := os.Stat(filepath.Join(root, d)); e == nil && fi.IsDir() {
			backlogDir = d
		}
	}
	if backlogDir != "" {
		done, detail := false, backlogDir+"/ exists. Import it; safe to repeat, edits in huginn win, the newest copy of each task wins."
		if branch && statErr == nil {
			if s, e := a.open(ctx); e == nil {
				r, e := importer.Run(ctx, filepath.Join(root, backlogDir), s.s, importer.Options{DryRun: true})
				s.c.Close()
				if e == nil {
					done = r.Added == 0 && r.Updated == 0
					detail = fmt.Sprintf("%s/ vs huginn: %d to add, %d to update, %d in sync, %d skipped (older copies, broken files).", backlogDir, r.Added, r.Updated, r.Unchanged, r.Skipped)
				}
			}
		}
		add("import backlog-md", done, detail,
			"huginn import backlog "+backlogDir+" --dry-run && huginn import backlog "+backlogDir)
	}

	instr := []string{}
	backlogMentions := []string{}
	for _, name := range []string{"AGENTS.md", "CLAUDE.md"} {
		raw, e := os.ReadFile(filepath.Join(root, name))
		if e != nil {
			continue
		}
		text := string(raw)
		if strings.Contains(text, "huginn") {
			instr = append(instr, name)
		}
		if strings.Contains(text, "BACKLOG.MD") || strings.Contains(strings.ToLower(text), "backlog.md mcp") {
			backlogMentions = append(backlogMentions, name)
		}
	}
	add("agent instructions", len(instr) > 0 && len(backlogMentions) == 0,
		"AGENTS.md (and CLAUDE.md if present) tell agents to track work with huginn",
		agentInstructionsEdit(backlogMentions))

	mcpPath := filepath.Join(root, ".mcp.json")
	hasHuginn, hasBacklog := mcpServers(mcpPath)
	if _, e := os.Stat(mcpPath); e == nil || hasBacklog {
		do := `add to "mcpServers" in .mcp.json: "hug": {"type": "stdio", "command": "hug", "args": ["mcp"]}`
		if hasBacklog {
			do += `; remove the "backlog" server once the import is done`
		}
		add("project MCP (.mcp.json)", hasHuginn && !hasBacklog, "Claude Code and other clients that read .mcp.json start hug mcp (memory and tasks) in this repo", do)
	}
	jcodeDone := false
	for _, p := range []string{filepath.Join(root, ".jcode", "mcp.json"), filepath.Join(os.Getenv("HOME"), ".jcode", "mcp.json")} {
		if h, _ := mcpServers(p); h {
			jcodeDone = true
		}
	}
	add("jcode MCP", jcodeDone, "jcode agents get the huginn tools", "huginn install jcode --all-instances")

	open := 0
	next := "huginn next"
	for _, s := range steps {
		if !s.Done {
			open++
		}
	}
	if open > 0 {
		next = "do the first open step above, then run huginn setup again"
	}
	return Reply{"setup": steps, "open": open, "intro": setupIntro, "next": next}, nil
}

func agentInstructionsEdit(backlog []string) string {
	do := "add this section to AGENTS.md:\n" + agentSection
	if len(backlog) > 0 {
		do = "in " + strings.Join(backlog, " and ") + ", replace the BACKLOG.MD MCP GUIDELINES block with this section:\n" + agentSection
	}
	return do
}

// agentSection is the text projects paste into AGENTS.md.
const agentSection = `## Tasks: huginn
Track all work with huginn (tasks on the git branch "huginn", at .huginn/).
Start: huginn next (or huginn take ID), then huginn show ID.
While working: huginn check ID N with evidence, huginn note ID "evidence".
Finish: huginn done ID when every acceptance check is ticked.
New work: huginn new "title" --goal "outcome" --accept "observable check".
MCP tools: list, show, new, next, take, done, and update (action check, note, set, move, dep, drop).
Old backlog ids (TASK-123) work as aliases.
Never edit .huginn/ files by hand while another agent may be writing; use the commands.`

// mcpServers reports whether an MCP config file lists huginn and backlog.
func mcpServers(path string) (huginn, backlog bool) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return false, false
	}
	var cfg map[string]json.RawMessage
	if json.Unmarshal(raw, &cfg) != nil {
		return false, false
	}
	for _, key := range []string{"mcpServers", "servers"} {
		var servers map[string]json.RawMessage
		if json.Unmarshal(cfg[key], &servers) == nil {
			if _, ok := servers["huginn"]; ok {
				huginn = true
			}
			if _, ok := servers["hug"]; ok {
				huginn = true
			}
			if _, ok := servers["backlog"]; ok {
				backlog = true
			}
		}
	}
	return huginn, backlog
}
