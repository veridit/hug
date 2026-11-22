package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/veridit/hug/huginn/gitbranch"
	"github.com/veridit/hug/huginn/internal/task"
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
  Code branches never see them. Changes reach the other clones in the background,
  within seconds; "hug task sync" syncs now and shows any error.
- Every task has a goal and observable acceptance checks. Work it like this
  (CLI shown; MCP has list, show, new, next, take, done and update):
    hug task next            pick the top task, then hug task take ID --model MODEL
    hug task show ID         read goal, checks, notes, dependencies
    hug task check ID N "evidence"   tick check N with what shows it holds
    hug task note ID "..."   record commands, findings, decisions
    hug task done ID         only when every check is ticked
- New work: hug task new "title" --goal "outcome" --accept "observable check"
- Use --json for structured replies; every reply has a "next" hint.
- Humans browse with "hug task tui".

`

// scratchPrefix is the jcode scratch folder setup offers for forbid_cite.
const scratchPrefix = ".jcode/scratch/"

// runSetup inspects the repository and returns the checklist.
func (a *App) runSetup(ctx context.Context, f flags) (Reply, error) {
	if e := f.count(0); e != nil {
		return nil, e
	}
	// One "could not remember" line for all of setup's checks (557s7n).
	ctx = a.withMemoWarning(ctx)
	cwd := a.Cwd
	if cwd == "" {
		cwd = "."
	}
	out, err := exec.CommandContext(ctx, "git", "-C", cwd, "rev-parse", "--show-toplevel").Output()
	if err != nil {
		if ctx.Err() != nil {
			return nil, interrupted(ctx)
		}
		return nil, &OperationError{Problem: "hug task setup needs a Git checkout as its working directory", Fix: "cd /path/to/your/git/repo && hug task setup"}
	}
	root := strings.TrimSpace(string(out))
	// The task branch is checked out in the main worktree, also when setup
	// runs in a linked one (065wwn).
	board := root
	if m, e := gitbranch.MainRoot(cwd); e == nil {
		board = m
	}
	var steps []SetupStep
	add := func(step string, done bool, detail, do string) {
		if done {
			do = ""
		}
		steps = append(steps, SetupStep{step, done, detail, do})
	}

	if oldLayout(ctx, board, "") {
		add("layout upgrade", false, "tasks are on the old layout (branch huginn at .huginn/); hug keeps them on branch hug/tasks at .hug/tasks/", "hug upgrade")
	}
	_, statErr := os.Stat(filepath.Join(board, TaskDir))
	branch := exec.CommandContext(ctx, "git", "-C", root, "rev-parse", "--verify", "--quiet", "refs/heads/"+TaskBranch).Run() == nil
	add("task branch", branch && statErr == nil, "branch "+TaskBranch+" checked out at "+TaskDir+"/", "hug task init")

	// People (89y0b7, Jørgen 15:31): hug finds a person by git email and
	// names them by the short name in hug.toml [people]; every commit
	// author gets an entry, emails grouped by author name.
	if branch && statErr == nil {
		people, _ := task.People(filepath.Join(board, TaskDir))
		offers := peopleOffers(ctx, root, people)
		detail := fmt.Sprintf("hug.toml [people] names %d people; a person is found by git email and shown by the short name, an agent by its own name", len(people))
		do := ""
		if len(offers) > 0 {
			var names []string
			for _, p := range offers {
				names = append(names, p.Name+" <"+strings.Join(p.Email, ", ")+">")
			}
			detail += "; commit authors without an entry (check the short name each command gives): " + strings.Join(names, "; ")
			do = addCommand(offers[0])
		}
		add("people", len(offers) == 0, detail, do)
	}
	// Coordinator (hbgybn, crab 00:58): who hears a request on a task
	// nobody holds. Unset, every session hears it once.
	if branch && statErr == nil {
		name, _ := task.Coordinator(filepath.Join(board, TaskDir))
		detail := "hug.toml [board] coordinator is " + name + ": hug hands every request and answer a person writes to " + name + "'s live jcode session (gmzd87)"
		if name == "" {
			detail = "no coordinator set: a person's requests and answers are not delivered to any session (they stay pending, not delivered: no coordinator set); name the session that coordinates the agents"
		}
		add("coordinator", name != "", detail, "hug task coordinator NAME")
	}
	// Models (jr55s8): each model's effort levels, for the Model: gate.
	if branch && statErr == nil {
		done, detail, do := modelsStatus(filepath.Join(board, TaskDir))
		add("models", done, detail, do)
	}

	backlogDir := ""
	for _, d := range []string{".backlog", "backlog"} {
		if fi, e := os.Stat(filepath.Join(root, d)); e == nil && fi.IsDir() {
			backlogDir = d
		}
	}
	if backlogDir != "" {
		var recorded task.Import
		retired := false
		if branch && statErr == nil {
			recorded, retired, _ = task.FindImport(filepath.Join(board, TaskDir), backlogDir)
		}
		done, detail := false, backlogDir+"/ exists. Import it; safe to repeat, edits in huginn win, the newest copy of each task wins."
		do := "hug task import backlog " + backlogDir + " --dry-run && hug task import backlog " + backlogDir
		if retired {
			// ggydj2: an import recorded in hug.toml is never offered again;
			// the folder is a leftover copy (restored, or checked out from an
			// older commit).
			detail = fmt.Sprintf("%s/ is a leftover: it was imported and retired in commit %s on %s; its tasks live in hug.", backlogDir, shortHash(recorded.Commit), recorded.Date)
			do = "hug task import backlog " + backlogDir + " --cleanup --dry-run, then show the diff and run it without --dry-run"
			add("retire backlog-md", false, detail, do)
		} else {
			complete := false
			if branch && statErr == nil {
				if s, e := a.open(ctx); e == nil {
					r, at, e := a.checkBacklog(ctx, s, filepath.Join(root, backlogDir))
					s.c.Close()
					if e != nil && ctx.Err() != nil {
						return nil, interrupted(ctx)
					}
					if e == nil {
						complete = r.Added == 0 && r.Updated == 0
						detail = fmt.Sprintf("%s/ vs huginn: %d to add, %d to update, %d in sync, %d skipped (older copies, broken files).", backlogDir, r.Added, r.Updated, r.Unchanged, r.Skipped)
						// 3mavpt: an update that only adds a field this hug now
						// maps (Seid: parent: from parent_task_id on 282) is not
						// in sync, the board lacks it, but importing is safe: say so.
						if safe := backlogUpdates(r); safe != "" {
							detail += " Of these, " + safe + "."
							if r.Added == 0 {
								do = "hug task import backlog " + backlogDir + " (only adds those fields), then hug task setup"
							}
						}
						if at != "" {
							detail += " Remembered from " + at + ": no backlog file changed since, and every task it matched is still on the board."
						}
					}
				}
			}
			add("import backlog-md", complete, detail, do)
			// Once every task is in, the move ends: the old copy goes, so
			// nobody imports it again over work done in hug since.
			if complete {
				add("retire backlog-md", done, backlogDir+"/ is imported; retire it so it is not imported again: git rm in one commit with backlog's MCP server and instructions, recorded in "+task.ConfigName,
					"hug task import backlog "+backlogDir+" --cleanup --dry-run, then show the diff and run it without --dry-run")
			}
		}
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
				r, unmatched, e := a.checkFiles(ctx, s, dir, cand)
				s.c.Close()
				if e == nil {
					if r.Copied+r.Unchanged == 0 {
						continue // not task folders after all
					}
					done = r.Copied == 0 && r.Citations == 0
					detail = fmt.Sprintf("%s/: %d files to copy, %d already in place, %d citations to rewrite, %d folders match no task.", cand, r.Copied, r.Unchanged, r.Citations, unmatched)
				}
			}
		}
		add("import task files", done, detail,
			"hug task import files "+cand+" --cite "+cand+" --dry-run && hug task import files "+cand+" --cite "+cand+
				" && hug task lint; afterwards cite files as file:NAME (this task) or task:ID/NAME, and attach new ones with hug task attach ID FILE")
	}

	// Scratch folders get deleted (cvbpdy): a repo whose agents work in
	// .jcode/scratch/ should have lint refuse citations into it, so evidence
	// is attached to the task instead of cited where it will vanish.
	if fi, e := os.Stat(filepath.Join(root, ".jcode")); e == nil && fi.IsDir() && branch && statErr == nil {
		forbid, _ := task.ForbidCite(filepath.Join(board, TaskDir))
		has := false
		for _, p := range forbid {
			has = has || strings.HasPrefix(scratchPrefix, p)
		}
		// Appending is safe only while hug.toml has no forbid_cite line;
		// with one, the prefix goes into that list by hand.
		do := "printf '\\nforbid_cite = [\"" + scratchPrefix + "\"]\\n' >> " + TaskDir + "/" + task.ConfigName + " && git -C " + TaskDir + " add " + task.ConfigName + " && git -C " + TaskDir + " commit -m 'hug.toml: forbid_cite " + scratchPrefix + "' " + task.ConfigName + " && hug task lint --code forbidden-citation"
		if forbid != nil {
			do = "add \"" + scratchPrefix + "\" to the forbid_cite list in " + TaskDir + "/" + task.ConfigName + ", commit it on branch " + TaskBranch + ", then hug task lint --code forbidden-citation"
		}
		add("scratch citations", has, "hug.toml forbid_cite makes hug task lint report task text that cites "+scratchPrefix+" (a folder that gets deleted); attach the file, or mark the citation (lost)", do)
	}

	instr := []string{}
	backlogMentions := []string{}
	stale := []string{}
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
		if sec, ok := taskSection(text); ok && sec != strings.TrimSpace(agentSection) {
			stale = append(stale, name)
		}
	}
	instrDo := agentInstructionsEdit(backlogMentions)
	if len(backlogMentions) == 0 && len(stale) > 0 {
		instrDo = "in " + strings.Join(stale, " and ") + ", replace the \"## Tasks: hug\" section (up to the next ## heading) with this one, then show the diff:\n" + agentSection
	}
	add("agent instructions", len(instr) > 0 && len(backlogMentions) == 0 && len(stale) == 0,
		"AGENTS.md (and CLAUDE.md if present) tell agents to track work with hug tasks, in this hug's words",
		instrDo)

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
	gate, gateDetail, gateDo := gateStatus(ctx, cwd)
	add("Model: trailer gate", gate, gateDetail, gateDo)

	add("jcode MCP", jcodeDone, "jcode agents get hug's task tools", "hug task install jcode --all-instances")

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
	// 3mavpt: an interrupted setup (^C) reports that, never a checklist
	// whose steps failed only because their git calls were killed (Seid:
	// "the commit-msg gate needs a Git checkout" in its own checkout).
	if ctx.Err() != nil {
		return nil, interrupted(ctx)
	}
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

func interrupted(ctx context.Context) error {
	return &OperationError{Problem: "hug task setup was interrupted (" + ctx.Err().Error() + "); it changed nothing", Fix: "hug task setup"}
}

func agentInstructionsEdit(backlog []string) string {
	do := "add this section to AGENTS.md:\n" + agentSection
	if len(backlog) > 0 {
		do = "in " + strings.Join(backlog, " and ") + ", replace the BACKLOG.MD MCP GUIDELINES block with this section:\n" + agentSection
	}
	return do
}

// taskSection is the "## Tasks: hug" section of text, heading included, up
// to the next "## " heading, trimmed. ok is false when there is none.
func taskSection(text string) (string, bool) {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	i := strings.Index(text, "## Tasks: hug")
	if i < 0 || (i > 0 && text[i-1] != '\n') {
		return "", false
	}
	rest := text[i:]
	if j := strings.Index(rest, "\n## "); j >= 0 {
		rest = rest[:j]
	}
	return strings.TrimSpace(rest), true
}

// agentSection is the text projects paste into AGENTS.md.
const agentSection = `## Tasks: hug
Track all work with hug tasks (on the git branch "hug/tasks", at .hug/tasks/).
Start: hug task next, then hug task take ID --model MODEL (the model you are, e.g.
claude-opus-5-5 or gpt-6.1-sol; a person: --model human; --as NAME adds your agent
or session name, optional), then hug task show ID.
While working: hug task check ID N "evidence" ticks check N and notes what shows it
holds; hug task note ID "text" for anything else.
Two tasks that ship together (one change or release): take the second with the
first, hug task take B --with A --model MODEL; both are doing.
Finish: hug task done ID when every acceptance check is ticked. Done asks whether
the instructions guided you well; answer honestly with --reflect "...": quote the
instruction that misled you or left a gap and say what happened, or "nothing".
Todos are your own next steps on the task you took, nothing else: details,
evidence, waits and dependencies go in tasks (hug task note, hug task dep), never
in todos, and another agent's work is never your todo. When your todos run out,
finish the task and run hug task next: an empty todo list means take the next
task, not blocked.
New work: hug task new "short title" --goal "..." --purpose "..." --principle "..." --details "..." --accept "observable check".
Describe a task with Goal (where we aim), Purpose (why), Principle (what we do
and do not do), Details, then Acceptance. The Goal is one sentence of
at most 140 characters; detail goes in Details. Titles stay under 80 characters.
Epics: hug task set ID --parent EPIC; hug task promote ID N turns item N into
its own task. hug task ls --epic EPIC shows an epic and its children.
MCP tools (hug mcp): task_list, task_show, task_new, task_next, task_take (model
and cwd required: your working folder, its branch is recorded; agent optional),
task_done, task_update (actions set, check, promote, note, move, dep, drop,
request, resolve), task_requests.
Requests: a person's request on a task (hug task requests) comes first in hug
task next. Turn it into changes to the ticket, then hug task resolve ID N
"what changed"; if it is unclear, hug task ask instead.
Questions: one question per ask (hug task ask ID --to NAME "..."), never a
numbered list in one; each is numbered on the task and the person answers each
on its own (hug task answer ID N "..."). A todo task is asked without taking it.
` + DeliveryFlow + `
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
