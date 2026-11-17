package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/veridit/hug/huginn/internal/importer"
	"github.com/veridit/hug/huginn/internal/task"
)

// Import is a one-time move (ggydj2, Jørgen 2026-11-17 14:45-14:47): after
// hug task import backlog DIR, --cleanup retires the source. In one commit on
// the code branch it removes DIR with git rm (history keeps it), backlog's
// MCP servers and its BACKLOG.MD block in AGENTS.md / CLAUDE.md; it then
// records the import in hug.toml on the task branch, and a later import of
// DIR is refused. What hug does not own (hooks, agent settings) is printed
// for the agent as "agent: remove ... from FILE". --dry-run shows the same
// diff and changes nothing.

const backlogBlockStart = "<!-- BACKLOG.MD MCP GUIDELINES START -->"
const backlogBlockEnd = "<!-- BACKLOG.MD MCP GUIDELINES END -->"

// mcpConfigs are the project MCP files hug knows, relative to the repository.
var mcpConfigs = []string{".mcp.json", ".cursor/mcp.json", ".vscode/mcp.json", ".jcode/mcp.json"}

// traceEdit is one file cleanup rewrites.
type traceEdit struct {
	Path    string `json:"path"`
	What    string `json:"what"`
	Tracked bool   `json:"tracked"`
	before  []byte
	after   []byte
	mode    os.FileMode
}

// CleanupPlan is what --cleanup removes and what it leaves to the agent.
type CleanupPlan struct {
	Source string      `json:"source"`
	Files  int         `json:"files"`
	Edits  []traceEdit `json:"edits"`
	Agent  []string    `json:"agent"`
	Cited  int         `json:"cited_elsewhere"`
	Commit string      `json:"commit,omitempty"`
	Config string      `json:"recorded_in,omitempty"`
}

func gitIn(ctx context.Context, dir string, args ...string) (string, error) {
	c := exec.CommandContext(ctx, "git", append([]string{"-C", dir}, args...)...)
	var out, errb bytes.Buffer
	c.Stdout, c.Stderr = &out, &errb
	if e := c.Run(); e != nil {
		msg := strings.TrimSpace(errb.String() + " " + out.String())
		if msg == "" {
			msg = e.Error()
		}
		return out.String(), errors.New(msg)
	}
	return out.String(), nil
}

// importSource is DIR relative to the repository's top folder, with slashes,
// and the top folder itself.
func (a *App) importSource(ctx context.Context, dir string) (rel, top string, err error) {
	cwd := a.Cwd
	if cwd == "" {
		cwd = "."
	}
	out, e := gitIn(ctx, cwd, "rev-parse", "--show-toplevel")
	if e != nil {
		return "", "", &OperationError{Problem: "hug task import needs a Git checkout as its working directory", Fix: "cd /path/to/your/git/repo && hug task import backlog DIR"}
	}
	top = filepath.Clean(strings.TrimSpace(out))
	abs := dir
	if !filepath.IsAbs(abs) {
		abs = filepath.Join(cwd, dir)
	}
	abs, _ = filepath.Abs(abs)
	if r, e := filepath.EvalSymlinks(abs); e == nil {
		abs = r
	}
	if r, e := filepath.EvalSymlinks(top); e == nil {
		top = r
	}
	r, e := filepath.Rel(top, abs)
	if e != nil || r == ".." || strings.HasPrefix(r, ".."+string(filepath.Separator)) {
		return "", "", usage(dir+" is outside this repository", "hug task import backlog .backlog")
	}
	return filepath.ToSlash(r), top, nil
}

// refuseRecordedImport stops a new import of a source that was moved and
// retired: its tasks live in hug, and the old copy is in git history.
func refuseRecordedImport(board, rel string) error {
	im, ok, e := task.FindImport(board, rel)
	if e != nil || !ok {
		return e
	}
	return &OperationError{
		Problem: fmt.Sprintf("not importing %s: it was imported and retired in commit %s on %s (recorded in %s); its tasks live in hug now, and importing the old copy again would overwrite work done since", rel, shortHash(im.Commit), im.Date, task.ConfigName),
		Fix:     fmt.Sprintf("git show --stat %s (the old copy: git show %s^:%s); work on the tasks with hug task ls", shortHash(im.Commit), shortHash(im.Commit), rel),
	}
}

func (a *App) runImportCleanup(ctx context.Context, s *session, f flags, dir string) (Reply, error) {
	dry := f.has("dry-run")
	rel, top, e := a.importSource(ctx, dir)
	if e != nil {
		return nil, e
	}
	again := "hug task import backlog " + rel + " --cleanup"
	recorded, retired, e := task.FindImport(s.r.Dir, rel)
	if e != nil {
		return nil, e
	}
	abs := filepath.Join(top, filepath.FromSlash(rel))
	if retired {
		if fi, e := os.Stat(abs); e != nil || !fi.IsDir() {
			return Reply{"message": fmt.Sprintf("%s was already retired in commit %s on %s", rel, shortHash(recorded.Commit), recorded.Date), "next": "hug setup"}, nil
		}
	}
	if fi, e := os.Stat(abs); !retired && (e != nil || !fi.IsDir()) {
		// Removed already (by hand, or a cleanup whose record failed):
		// record the commit that removed it.
		out, _ := gitIn(ctx, top, "log", "-1", "--format=%H", "--diff-filter=D", "--", rel)
		hash := strings.TrimSpace(out)
		if hash == "" {
			return nil, usage(rel+" does not exist and no commit removed it", "hug task import backlog .backlog --cleanup --dry-run")
		}
		plan := &CleanupPlan{Source: rel, Commit: hash}
		if dry {
			return Reply{"cleanup": []string{fmt.Sprintf("%s was removed in commit %s; would record the import in %s", rel, shortHash(hash), task.ConfigName)}, "dry_run": true, "next": again}, nil
		}
		if e := a.recordCleanup(s, plan); e != nil {
			return nil, e
		}
		return Reply{"cleanup": []string{fmt.Sprintf("%s was removed in commit %s; import recorded in %s", rel, shortHash(hash), task.ConfigName)}, "plan": plan, "next": "hug setup"}, nil
	}

	// The move must be complete before its source goes: every task in,
	// every per-task file folder copied. A leftover of a recorded import is
	// not compared: its tasks have moved on in hug since.
	var missing []string
	var fixes []string
	if !retired {
		r, e := importer.Run(ctx, abs, s.s, importer.Options{DryRun: true})
		var none *importer.NoTasksError
		if e != nil && !errors.As(e, &none) {
			return nil, e
		}
		if r.Added+r.Updated > 0 {
			m := fmt.Sprintf("%d tasks to add and %d to update", r.Added, r.Updated)
			if safe := backlogUpdates(r); safe != "" {
				m += " (" + safe + ")"
			}
			missing = append(missing, m)
			fixes = append(fixes, "hug task import backlog "+rel)
		}
		cand := rel + "/evidence"
		d := filepath.Join(top, filepath.FromSlash(cand))
		if fi, e := os.Stat(d); e == nil && fi.IsDir() {
			fr, e := importer.ImportFiles(d, cand, s.s, true)
			if e != nil {
				return nil, e
			}
			if fr.Copied+fr.Citations > 0 {
				missing = append(missing, fmt.Sprintf("%s: %d files to copy and %d citations to rewrite", cand, fr.Copied, fr.Citations))
				fixes = append(fixes, "hug task import files "+cand+" --cite "+cand)
			}
		}
	}
	if len(missing) > 0 {
		fixes = append(fixes, again+" --dry-run")
		return nil, &OperationError{Problem: "not retiring " + rel + ": the import is not complete: " + strings.Join(missing, "; "), Fix: strings.Join(fixes, " && ")}
	}
	if retired {
		return a.removeLeftover(ctx, f, top, rel, recorded)
	}
	if out, _ := gitIn(ctx, top, "status", "--porcelain", "--untracked-files=all", "--", rel); strings.TrimSpace(out) != "" {
		lines := strings.Split(strings.TrimSpace(out), "\n")
		return nil, &OperationError{Problem: fmt.Sprintf("not retiring %s: %d paths in it are not committed (first: %s); git rm would leave them or lose them", rel, len(lines), strings.TrimSpace(lines[0])), Fix: "git -C " + top + " status -- " + rel}
	}
	listed, _ := gitIn(ctx, top, "ls-files", "--", rel)
	files := 0
	if t := strings.TrimSpace(listed); t != "" {
		files = len(strings.Split(t, "\n"))
	}
	if files == 0 {
		return nil, &OperationError{Problem: "not retiring " + rel + ": git does not track it, so git rm cannot keep it in history", Fix: "git -C " + top + " add " + rel + " && git -C " + top + " commit -m 'backlog before the move to hug', then " + again + " --dry-run"}
	}
	plan, e := planCleanup(ctx, top, rel)
	if e != nil {
		return nil, e
	}
	plan.Files = files
	diff := plan.diff()
	if dry {
		return Reply{"cleanup": plan.summary(true), "diff": diff, "agent": plan.Agent, "plan": plan, "dry_run": true,
			"next": "show this diff to the person; once they agree: " + again}, nil
	}
	model := a.takerModel(f)
	if model == "" {
		return nil, usage("--cleanup commits on your code branch and names the model that made the commit (Model: trailer); no model given", again+" --model MODEL")
	}
	if out, _ := gitIn(ctx, top, "diff", "--cached", "--name-only"); strings.TrimSpace(out) != "" {
		return nil, &OperationError{Problem: "not retiring " + rel + ": the index already holds staged changes, which would go into the cleanup commit", Fix: "git -C " + top + " commit (or git -C " + top + " restore --staged .), then " + again}
	}
	if e := plan.commit(ctx, top, model); e != nil {
		return nil, e
	}
	if e := a.recordCleanup(s, plan); e != nil {
		return nil, &OperationError{Problem: fmt.Sprintf("%s was removed in commit %s, but recording the import failed: %v", rel, shortHash(plan.Commit), e), Fix: again}
	}
	return Reply{"cleanup": plan.summary(false), "diff": diff, "agent": plan.Agent, "plan": plan, "next": "hug setup"}, nil
}

// removeLeftover deletes a retired source that came back (restored, or
// checked out from an older commit). Its tasks live in hug and its copy in
// the recorded commit, so nothing in it is kept: tracked files go with git
// rm in one commit, untracked ones are deleted. The record stays.
func (a *App) removeLeftover(ctx context.Context, f flags, top, rel string, recorded task.Import) (Reply, error) {
	abs := filepath.Join(top, filepath.FromSlash(rel))
	did := []string{fmt.Sprintf("%s/ is a leftover of the import retired in commit %s on %s; its tasks live in hug", rel, shortHash(recorded.Commit), recorded.Date)}
	if f.has("dry-run") {
		did = append(did, "would remove "+rel+"/: tracked files with git rm in one commit, the rest deleted")
		return Reply{"cleanup": did, "dry_run": true, "next": "hug task import backlog " + rel + " --cleanup"}, nil
	}
	// Staged but never committed counts as untracked: unstage it first.
	gitIn(ctx, top, "reset", "-q", "--", rel)
	listed, _ := gitIn(ctx, top, "ls-files", "--", rel)
	tracked := 0
	if t := strings.TrimSpace(listed); t != "" {
		tracked = len(strings.Split(t, "\n"))
	}
	if tracked > 0 {
		model := a.takerModel(f)
		if model == "" {
			return nil, usage("--cleanup commits on your code branch and names the model that made the commit (Model: trailer); no model given", "hug task import backlog "+rel+" --cleanup --model MODEL")
		}
		if _, e := gitIn(ctx, top, "rm", "-r", "-q", "-f", "--", rel); e != nil {
			return nil, &OperationError{Problem: "git rm -r " + rel + ": " + e.Error(), Fix: "git -C " + top + " status -- " + rel}
		}
		msg := fmt.Sprintf("tasks live in hug: leftover %s removed (hug task import backlog %s --cleanup)\n\nIt was imported and retired in %s.\n\nModel: %s\n", rel, rel, shortHash(recorded.Commit), model)
		c := exec.CommandContext(ctx, "git", "-C", top, "commit", "-q", "-F", "-", "--", rel)
		c.Stdin = strings.NewReader(msg)
		if out, e := c.CombinedOutput(); e != nil {
			gitIn(ctx, top, "reset", "-q", "--", rel)
			gitIn(ctx, top, "checkout", "-q", "HEAD", "--", rel)
			return nil, &OperationError{Problem: "the commit removing the leftover was refused, nothing changed: " + strings.TrimSpace(string(out)), Fix: "git -C " + top + " status -- " + rel}
		}
		did = append(did, fmt.Sprintf("git rm -r %s: %d tracked files, one commit", rel, tracked))
	}
	if e := os.RemoveAll(abs); e != nil {
		return nil, e
	}
	did = append(did, rel+"/ removed")
	return Reply{"cleanup": did, "next": "hug setup"}, nil
}

func (a *App) recordCleanup(s *session, plan *CleanupPlan) error {
	p, e := task.RecordImport(s.r.Dir, task.Import{Source: plan.Source, Format: "backlog", Commit: plan.Commit, Date: time.Now().Format("2006-01-02")})
	if e != nil {
		return e
	}
	plan.Config = p
	return s.commit("hug task import backlog " + plan.Source + " --cleanup: recorded, removed in " + shortHash(plan.Commit))
}

func (p *CleanupPlan) summary(dry bool) []string {
	verb := map[bool]string{true: "would ", false: ""}[dry]
	out := []string{fmt.Sprintf("%sgit rm -r %s: %d files (git history keeps them)", verb, p.Source, p.Files)}
	for _, e := range p.Edits {
		note := ""
		if !e.Tracked {
			note = " (not tracked by git: changed in place, not in the commit)"
		}
		out = append(out, fmt.Sprintf("%sedit %s: %s%s", verb, e.Path, e.What, note))
	}
	if p.Commit != "" {
		out = append(out, fmt.Sprintf("committed %s; import recorded in %s on the task branch", shortHash(p.Commit), p.Config))
	} else {
		out = append(out, "would commit that as one commit, then record the import in "+task.ConfigName+" on the task branch")
	}
	if p.Cited > 0 {
		out = append(out, fmt.Sprintf("%d documents cite %s/ paths; those links stay readable in git history", p.Cited, p.Source))
	}
	if n := len(p.Agent); n > 0 && p.Commit == "" {
		out = append(out, fmt.Sprintf("the agent: lines below still use backlog or read %s/: hug does not edit them; change them first, or a hook or script may fail once %s/ is gone", p.Source, p.Source))
	}
	return out
}

func (p *CleanupPlan) diff() string {
	var b strings.Builder
	for _, e := range p.Edits {
		b.WriteString("--- " + e.Path + "\n")
		b.WriteString(lineDiff(string(e.before), string(e.after)))
	}
	return b.String()
}

// commit writes the edits, removes the source and commits it all at once.
// A refused commit (a hook) puts everything back.
func (p *CleanupPlan) commit(ctx context.Context, top, model string) error {
	restore := func() {
		gitIn(ctx, top, "reset", "-q")
		gitIn(ctx, top, "checkout", "-q", "HEAD", "--", p.Source)
		for _, e := range p.Edits {
			os.WriteFile(filepath.Join(top, filepath.FromSlash(e.Path)), e.before, e.mode)
		}
	}
	var add []string
	for _, e := range p.Edits {
		if e := os.WriteFile(filepath.Join(top, filepath.FromSlash(e.Path)), e.after, e.mode); e != nil {
			restore()
			return e
		}
		if e.Tracked {
			add = append(add, e.Path)
		}
	}
	if _, e := gitIn(ctx, top, "rm", "-r", "-q", "--", p.Source); e != nil {
		restore()
		return &OperationError{Problem: "git rm -r " + p.Source + ": " + e.Error(), Fix: "git -C " + top + " status"}
	}
	if len(add) > 0 {
		if _, e := gitIn(ctx, top, append([]string{"add", "--"}, add...)...); e != nil {
			restore()
			return &OperationError{Problem: "git add: " + e.Error(), Fix: "git -C " + top + " status"}
		}
	}
	msg := fmt.Sprintf("tasks live in hug: %s retired (hug task import backlog %s --cleanup)\n\nThe import into hug is complete, so the old copy goes: git rm -r %s (%d files; git history keeps them).", p.Source, p.Source, p.Source, p.Files)
	for _, e := range p.Edits {
		if e.Tracked {
			msg += "\n" + e.Path + ": " + e.What + "."
		}
	}
	msg += "\n\nModel: " + model + "\n"
	c := exec.CommandContext(ctx, "git", "-C", top, "commit", "-q", "-F", "-")
	c.Stdin = strings.NewReader(msg)
	out, e := c.CombinedOutput()
	if e != nil {
		restore()
		return &OperationError{Problem: "the cleanup commit was refused, nothing changed: " + strings.TrimSpace(string(out)), Fix: "fix what the commit hook names (see the agent: lines of hug task import backlog " + p.Source + " --cleanup --dry-run), then hug task import backlog " + p.Source + " --cleanup"}
	}
	h, e := gitIn(ctx, top, "rev-parse", "HEAD")
	if e != nil {
		return e
	}
	p.Commit = strings.TrimSpace(h)
	return nil
}

// planCleanup finds backlog's traces: what hug edits itself, what it leaves
// to the agent, and how many other files cite the source.
func planCleanup(ctx context.Context, top, rel string) (*CleanupPlan, error) {
	p := &CleanupPlan{Source: rel}
	tracked := func(path string) bool {
		out, _ := gitIn(ctx, top, "ls-files", "--", path)
		return strings.TrimSpace(out) != ""
	}
	for _, name := range mcpConfigs {
		full := filepath.Join(top, filepath.FromSlash(name))
		raw, e := os.ReadFile(full)
		if e != nil {
			continue
		}
		fi, _ := os.Stat(full)
		after, ok := removeMCPServer(raw, "backlog")
		if ok {
			p.Edits = append(p.Edits, traceEdit{Path: name, What: `"backlog" MCP server removed`, Tracked: tracked(name), before: raw, after: after, mode: fi.Mode().Perm()})
		} else if bytes.Contains(raw, []byte(`"backlog"`)) {
			p.Agent = append(p.Agent, agentLines(name, raw, regexp.MustCompile(`"backlog"`))...)
		}
	}
	for _, name := range []string{"AGENTS.md", "CLAUDE.md"} {
		full := filepath.Join(top, name)
		raw, e := os.ReadFile(full)
		if e != nil {
			continue
		}
		fi, _ := os.Stat(full)
		after, what := removeBacklogBlock(string(raw))
		if what != "" {
			p.Edits = append(p.Edits, traceEdit{Path: name, What: what, Tracked: tracked(name), before: raw, after: []byte(after), mode: fi.Mode().Perm()})
			raw = []byte(after)
		}
		// What is left that still points at backlog: its commands, its MCP
		// tools, its folder (the hug section's alias line is not).
		p.Agent = append(p.Agent, agentLines(name, raw, backlogMention(rel))...)
	}
	// Hooks and agent settings: hug never edits them (a tracked hook is the
	// project's own), so a line that runs backlog, calls its MCP tools or
	// reads the retired folder goes to the agent. The word "backlog" in
	// prose is not a trace.
	any := backlogMention(rel)
	var places []string
	if out, e := gitIn(ctx, top, "config", "--get", "core.hooksPath"); e == nil {
		if d := strings.TrimSpace(out); d != "" {
			places = append(places, d)
		}
	}
	if out, e := gitIn(ctx, top, "rev-parse", "--git-path", "hooks"); e == nil {
		places = append(places, strings.TrimSpace(out))
	}
	places = append(places, ".claude")
	seen := map[string]bool{}
	for _, place := range places {
		dir := place
		if !filepath.IsAbs(dir) {
			dir = filepath.Join(top, dir)
		}
		filepath.WalkDir(dir, func(path string, d os.DirEntry, e error) error {
			if e != nil || d.IsDir() || strings.HasSuffix(path, ".sample") || seen[path] {
				return nil
			}
			seen[path] = true
			raw, e := os.ReadFile(path)
			if e != nil || bytes.IndexByte(raw, 0) >= 0 {
				return nil
			}
			name := path
			if r, e := filepath.Rel(top, path); e == nil && !strings.HasPrefix(r, "..") {
				name = filepath.ToSlash(r)
			}
			p.Agent = append(p.Agent, agentLines(name, raw, any)...)
			return nil
		})
	}
	// Code anywhere in the repository that reads the retired folder breaks
	// when it goes (Seid: tools/evidence.py lists .backlog/tasks from the
	// pre-commit hook). Each such line goes to the agent; documents that
	// cite it are only counted, their links stay readable in history.
	re := `(^|[^A-Za-z0-9_.-])` + regexp.QuoteMeta(rel) + `([/"']|$)`
	docs := map[string]bool{}
	code := map[string][]string{}
	var order []string
	listed := map[string]bool{}
	for _, a := range p.Agent {
		if f, _, ok := strings.Cut(strings.TrimPrefix(a, "remove from "), ":"); ok {
			listed[f] = true
		}
	}
	if out, _ := gitIn(ctx, top, "grep", "-n", "-I", "-E", "-e", re, "--", ".", ":(exclude)"+rel); strings.TrimSpace(out) != "" {
		for _, l := range strings.Split(strings.TrimSpace(out), "\n") {
			f, rest, ok := strings.Cut(l, ":")
			if !ok || listed[f] {
				continue
			}
			switch strings.ToLower(filepath.Ext(f)) {
			case ".md", ".txt", ".log", ".rst", ".html", ".csv", ".json", ".jsonl":
				docs[f] = true
				continue
			}
			n, text, _ := strings.Cut(rest, ":")
			code[f] = append(code[f], fmt.Sprintf("remove from %s:%s: %s", f, n, strings.TrimSpace(text)))
			if len(code[f]) == 1 {
				order = append(order, f)
			}
		}
	}
	// Up to 3 lines per file; the rest by count, so the list stays readable
	// (Seid: 31 lines in one test file).
	for _, f := range order {
		ls := code[f]
		if len(ls) > 3 {
			ls = append(ls[:3:3], fmt.Sprintf("remove from %s: %d more lines that read %s/ (git grep -n '%s' -- %s)", f, len(code[f])-3, rel, rel, f))
		}
		p.Agent = append(p.Agent, ls...)
	}
	p.Cited = len(docs)
	return p, nil
}

// backlogMention matches what still depends on backlog: its CLI commands
// (backlog task ..., backlog board ...), its MCP tools, paths into the
// retired folder, and scripts named after it (remind-backlog-*.sh).
func backlogMention(rel string) *regexp.Regexp {
	return regexp.MustCompile(`mcp__backlog__|(^|[\s"'` + "`" + `(;&|])backlog (task|tasks|board|mcp|search|init|config|draft|doc|decision|milestone|browser|overview|sequence)\b|` + regexp.QuoteMeta(rel+"/") + `|[\w-]*backlog[\w-]*\.(sh|py|js|ts)\b`)
}

func agentLines(name string, raw []byte, re *regexp.Regexp) []string {
	var out []string
	for i, l := range strings.Split(strings.ReplaceAll(string(raw), "\r\n", "\n"), "\n") {
		if re.MatchString(l) {
			out = append(out, fmt.Sprintf("remove from %s:%d: %s", name, i+1, strings.TrimSpace(l)))
		}
	}
	return out
}

// removeBacklogBlock takes backlog.md's guidelines block out of an agent
// instruction file. With no hug section yet, the hug section takes its
// place, as hug setup asks.
func removeBacklogBlock(text string) (string, string) {
	i := strings.Index(text, backlogBlockStart)
	if i < 0 {
		return text, ""
	}
	j := strings.Index(text[i:], backlogBlockEnd)
	if j < 0 {
		return text, ""
	}
	end := i + j + len(backlogBlockEnd)
	for end < len(text) && (text[end] == '\n' || text[end] == '\r') {
		end++
	}
	rest := text[:i] + text[end:]
	if _, ok := taskSection(rest); ok {
		out := strings.TrimRight(text[:i], "\n") + "\n"
		if strings.TrimSpace(text[:i]) == "" {
			out = ""
		}
		if tail := text[end:]; tail != "" {
			if out != "" {
				out += "\n"
			}
			out += tail
		}
		return out, "BACKLOG.MD MCP GUIDELINES block removed"
	}
	return text[:i] + agentSection + "\n\n" + text[end:], "BACKLOG.MD MCP GUIDELINES block replaced by the hug task section"
}

// removeMCPServer deletes one server from "mcpServers" (or "servers") in an
// MCP config, keeping every other byte as it was. ok is false when the
// server is not there or the file is not JSON hug can read.
func removeMCPServer(raw []byte, name string) ([]byte, bool) {
	for _, section := range []string{"mcpServers", "servers"} {
		start, end, ok := memberSpan(raw, section, name)
		if !ok {
			continue
		}
		out := cutMember(raw, start, end)
		if out != nil && json.Valid(out) {
			return out, true
		}
	}
	return raw, false
}

// memberSpan finds "name": value inside the top-level object member
// "section", by offsets into raw: start at the key's quote, end after the
// value.
func memberSpan(raw []byte, section, name string) (int, int, bool) {
	var top map[string]json.RawMessage
	if json.Unmarshal(raw, &top) != nil {
		return 0, 0, false
	}
	if _, ok := top[section]; !ok {
		return 0, 0, false
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	// Walk the top object's keys; skip each value whole except section's.
	if t, e := dec.Token(); e != nil || t != json.Delim('{') {
		return 0, 0, false
	}
	for dec.More() {
		k, e := dec.Token()
		if e != nil {
			return 0, 0, false
		}
		if k != section {
			var skip json.RawMessage
			if dec.Decode(&skip) != nil {
				return 0, 0, false
			}
			continue
		}
		if t, e := dec.Token(); e != nil || t != json.Delim('{') {
			return 0, 0, false
		}
		for dec.More() {
			before := int(dec.InputOffset())
			k, e := dec.Token()
			if e != nil {
				return 0, 0, false
			}
			after := int(dec.InputOffset())
			var skip json.RawMessage
			if dec.Decode(&skip) != nil {
				return 0, 0, false
			}
			if k == name {
				quoted, _ := json.Marshal(name)
				i := bytes.Index(raw[before:after], quoted)
				if i < 0 {
					return 0, 0, false
				}
				return before + i, int(dec.InputOffset()), true
			}
		}
		return 0, 0, false
	}
	return 0, 0, false
}

// cutMember removes raw[start:end] (one "key": value member) with the comma
// that joins it to its neighbour.
func cutMember(raw []byte, start, end int) []byte {
	space := func(b byte) bool { return b == ' ' || b == '\t' || b == '\n' || b == '\r' }
	j := end
	for j < len(raw) && space(raw[j]) {
		j++
	}
	if j < len(raw) && raw[j] == ',' {
		// A member follows: drop this one, its comma and the space up to the
		// next key, keeping this member's own indentation for that key.
		k := j + 1
		for k < len(raw) && space(raw[k]) {
			k++
		}
		return append(append([]byte{}, raw[:start]...), raw[k:]...)
	}
	// The last member: drop the comma before it.
	i := start - 1
	for i >= 0 && space(raw[i]) {
		i--
	}
	if i >= 0 && raw[i] == ',' {
		return append(append([]byte{}, raw[:i]...), raw[end:]...)
	}
	// The only member.
	if i >= 0 && raw[i] == '{' {
		k := end
		for k < len(raw) && space(raw[k]) {
			k++
		}
		return append(append(append([]byte{}, raw[:i+1]...), '}'), raw[k+1:]...)
	}
	return nil
}

// lineDiff shows removed lines as "- " and added ones as "+ ", in order
// (longest common subsequence of lines).
func lineDiff(a, b string) string {
	x := strings.Split(strings.ReplaceAll(a, "\r\n", "\n"), "\n")
	y := strings.Split(strings.ReplaceAll(b, "\r\n", "\n"), "\n")
	n, m := len(x), len(y)
	l := make([][]int, n+1)
	for i := range l {
		l[i] = make([]int, m+1)
	}
	for i := n - 1; i >= 0; i-- {
		for j := m - 1; j >= 0; j-- {
			if x[i] == y[j] {
				l[i][j] = l[i+1][j+1] + 1
			} else if l[i+1][j] >= l[i][j+1] {
				l[i][j] = l[i+1][j]
			} else {
				l[i][j] = l[i][j+1]
			}
		}
	}
	var out strings.Builder
	i, j := 0, 0
	for i < n || j < m {
		switch {
		case i < n && j < m && x[i] == y[j]:
			i++
			j++
		case j < m && (i == n || l[i][j+1] > l[i+1][j]):
			fmt.Fprintf(&out, "+ %s\n", y[j])
			j++
		default:
			fmt.Fprintf(&out, "- %s\n", x[i])
			i++
		}
	}
	return out.String()
}
