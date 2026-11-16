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
const setupIntro = `hug task setup: switch this repository's task tracking to hug.
For the agent reading this: work through the checklist top to bottom. Each
open step has the exact command or edit. Run "hug task setup" again after each
step until every step is done. setup itself only reads; it changes nothing.

How huginn works, in short:
- Tasks are Markdown files on the git branch "hug/tasks", checked out at .hug/tasks/.
  Code branches never see them. "hug task sync" shares them through the remote.
- Every task has a goal and observable acceptance checks. Work it like this
  (CLI shown; MCP has list, show, new, next, take, done and update):
    hug task next            pick the top task (or hug task take ID)
    hug task show ID         read goal, checks, notes, dependencies
    hug task check ID N      tick check N when you have evidence
    hug task note ID "..."   record evidence, commands, findings
    hug task done ID         only when every check is ticked
- New work: hug task new "title" --goal "outcome" --accept "observable check"
- Use --json for structured replies; every reply has a "next" hint.
- Humans browse with "hug task tui".

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
		return nil, &OperationError{Problem: "hug task setup needs a Git checkout as its working directory", Fix: "cd /path/to/your/git/repo && hug task setup"}
	}
	root := strings.TrimSpace(string(out))
	var steps []SetupStep
	add := func(step string, done bool, detail, do string) {
		if done {
			do = ""
		}
		steps = append(steps, SetupStep{step, done, detail, do})
	}

	if oldLayout(ctx, root) {
		add("layout upgrade", false, "tasks are on the old layout (branch huginn at .huginn/); hug keeps them on branch hug/tasks at .hug/tasks/", "hug upgrade")
	}
	_, statErr := os.Stat(filepath.Join(root, TaskDir))
	branch := exec.CommandContext(ctx, "git", "-C", root, "rev-parse", "--verify", "--quiet", "refs/heads/"+TaskBranch).Run() == nil
	add("task branch", branch && statErr == nil, "branch "+TaskBranch+" checked out at "+TaskDir+"/", "hug task init")

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
			"hug task import backlog "+backlogDir+" --dry-run && hug task import backlog "+backlogDir)
	}

	// Per-task file folders kept outside the task branch (any layout: a folder
	// whose subfolders are named by task ids or aliases, such as
	// .backlog/evidence/TASK-1966/). Tasks cite them by path; importing moves
	// them into each task's own folder and rewrites the citations.
	for _, cand := range []string{".backlog/evidence", "evidence", ".evidence"} {
		dir := filepath.Join(root, cand)
		if fi, e := os.Stat(dir); e != nil || !fi.IsDir() {
			continue
		}
		done, detail := false, cand+"/ holds per-task files that tasks cite by path; import them into the tasks' own folders."
		if branch && statErr == nil {
			if s, e := a.open(ctx); e == nil {
				r, e := importer.ImportFiles(dir, cand, s.s, true)
				s.c.Close()
				if e == nil {
					if r.Copied+r.Unchanged == 0 {
						continue // not task folders after all
					}
					done = r.Copied == 0 && r.Citations == 0
					detail = fmt.Sprintf("%s/: %d files to copy, %d already in place, %d citations to rewrite, %d folders match no task.", cand, r.Copied, r.Unchanged, r.Citations, len(r.Unmatched))
				}
			}
		}
		add("import task files", done, detail,
			"hug task import files "+cand+" --cite "+cand+" --dry-run && hug task import files "+cand+" --cite "+cand+
				" && hug task lint; afterwards cite files as file:NAME (this task) or task:ID/NAME, and attach new ones with hug task attach ID FILE")
	}

	instr := []string{}
	backlogMentions := []string{}
	for _, name := range []string{"AGENTS.md", "CLAUDE.md"} {
		raw, e := os.ReadFile(filepath.Join(root, name))
		if e != nil {
			continue
		}
		text := string(raw)
		if strings.Contains(text, "hug task") || strings.Contains(text, "huginn") {
			instr = append(instr, name)
		}
		if strings.Contains(text, "BACKLOG.MD") || strings.Contains(strings.ToLower(text), "backlog.md mcp") {
			backlogMentions = append(backlogMentions, name)
		}
	}
	add("agent instructions", len(instr) > 0 && len(backlogMentions) == 0,
		"AGENTS.md (and CLAUDE.md if present) tell agents to track work with hug tasks",
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
	add("jcode MCP", jcodeDone, "jcode agents get the huginn tools", "hug task install jcode --all-instances")

	// Desktop and CLI apps (per user, not per repo): report the first one
	// whose config exists but lacks a hug server.
	if home, e := os.UserHomeDir(); e == nil {
		apps := []struct{ name, path, marker string }{
			{"Claude Desktop", filepath.Join(home, "Library/Application Support/Claude/claude_desktop_config.json"), `"hug"`},
			{"Claude Code", filepath.Join(home, ".claude.json"), `"hug"`},
			{"ChatGPT / Codex", filepath.Join(home, ".codex", "config.toml"), "[mcp_servers.hug]"},
		}
		var missing []string
		for _, ap := range apps {
			if b, err := os.ReadFile(ap.path); err == nil && !strings.Contains(string(b), ap.marker) {
				missing = append(missing, ap.name)
			}
		}
		detail := "Claude Desktop, Claude Code and ChatGPT/Codex start hug mcp (memory and tasks)"
		if len(missing) > 0 {
			detail += "; missing in " + strings.Join(missing, ", ")
		}
		add("app MCP", len(missing) == 0, detail, "hug apps install --dry-run && hug apps install, then restart those apps")
	}

	open := 0
	next := "hug task next"
	for _, s := range steps {
		if !s.Done {
			open++
		}
	}
	if open > 0 {
		next = "do the first open step above, then run hug task setup again"
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
const agentSection = `## Tasks: hug
Track all work with hug tasks (on the git branch "hug/tasks", at .hug/tasks/).
Start: hug task next (or hug task take ID), then hug task show ID.
While working: hug task check ID N with evidence, hug task note ID "evidence".
Finish: hug task done ID when every acceptance check is ticked.
New work: hug task new "short title" --goal "..." --purpose "..." --principle "..." --details "..." --accept "observable check".
Describe a task with Goal (where we aim), Purpose (why), Principle (what we do
and do not do), Relevant details, then Acceptance. Titles stay under 80
characters; detail goes in the Goal.
Epics: hug task set ID --parent EPIC; hug task promote ID N turns item N into
its own task. hug task ls --epic EPIC shows an epic and its children.
MCP tools (hug mcp): task_list, task_show, task_new, task_next, task_take,
task_done, task_update (actions set, check, promote, note, move, dep, drop).
Old backlog ids (TASK-123) work as aliases.
Never edit .hug/tasks/ files by hand while another agent may be writing; use the commands.`

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
