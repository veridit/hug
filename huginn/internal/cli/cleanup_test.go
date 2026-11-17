package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// backlogRepo is a fixture repository moving from backlog-md to hug: one
// task in .backlog/tasks, backlog's MCP server next to hug's in .mcp.json,
// the BACKLOG.MD block in CLAUDE.md, a hook that runs backlog, and the task
// branch initialised. Everything is committed.
func backlogRepo(t *testing.T) (*App, string) {
	t.Helper()
	a := app(t)
	dir := a.Cwd
	write := func(rel, text string) {
		t.Helper()
		p := filepath.Join(dir, filepath.FromSlash(rel))
		if e := os.MkdirAll(filepath.Dir(p), 0o755); e != nil {
			t.Fatal(e)
		}
		if e := os.WriteFile(p, []byte(text), 0o644); e != nil {
			t.Fatal(e)
		}
	}
	write(".backlog/config.yml", "project_name: demo\n")
	write(".backlog/tasks/task-1 - Demo.md", "---\nid: TASK-1\ntitle: Demo task\nstate: To Do\n---\n\n## Description\n<!-- SECTION:DESCRIPTION:BEGIN -->\nDemo goal.\n<!-- SECTION:DESCRIPTION:END -->\n\n## Acceptance Criteria\n<!-- AC:BEGIN -->\n- [ ] #1 It works\n<!-- AC:END -->\n")
	write(".mcp.json", "{\n  \"mcpServers\": {\n    \"backlog\": {\n      \"command\": \"backlog\",\n      \"args\": [\"mcp\", \"start\"]\n    },\n    \"hug\": {\n      \"type\": \"stdio\",\n      \"command\": \"hug\",\n      \"args\": [\"mcp\"]\n    }\n  }\n}\n")
	write("CLAUDE.md", "# Demo\n\nKeep it short.\n\n"+backlogBlockStart+"\nUse backlog task create for new work.\n"+backlogBlockEnd+"\n\n## Build\n\nmake\n")
	write(".githooks/post-commit", "#!/bin/sh\nbacklog board export\n")
	write("docs/notes.md", "See .backlog/tasks/task-1 - Demo.md for the history.\n")
	git(t, dir, "add", "-A")
	git(t, dir, "commit", "-m", "backlog era")
	git(t, dir, "config", "core.hooksPath", ".githooks")
	execute(t, a, "init")
	return a, dir
}

// ggydj2 check 1: --cleanup removes DIR with git rm in one commit, with
// backlog's MCP server and BACKLOG.MD block, shows the diff first
// (--dry-run changes nothing), prints what hug does not own for the agent,
// and records source, commit and date in hug.toml.
func TestImportCleanupRetiresSourceInOneCommit(t *testing.T) {
	t.Parallel()
	a, dir := backlogRepo(t)
	// Not before the import is complete.
	if code, out := runText(a, "import", "backlog", ".backlog", "--cleanup", "--dry-run"); code != 1 ||
		!strings.Contains(out, "not retiring .backlog: the import is not complete: 1 tasks to add and 0 to update") ||
		!strings.Contains(out, "fix: hug task import backlog .backlog && hug task import backlog .backlog --cleanup --dry-run") {
		t.Fatalf("cleanup before import: exit %d\n%s", code, out)
	}
	execute(t, a, "import", "backlog", ".backlog")
	head := git(t, dir, "rev-parse", "HEAD")

	code, out := runText(a, "import", "backlog", ".backlog", "--cleanup", "--dry-run")
	if code != 0 {
		t.Fatalf("dry run: exit %d\n%s", code, out)
	}
	for _, want := range []string{
		"would git rm -r .backlog: 2 files (git history keeps them)\n",
		"would edit .mcp.json: \"backlog\" MCP server removed\n",
		"would edit CLAUDE.md: BACKLOG.MD MCP GUIDELINES block replaced by the hug task section\n",
		"--- .mcp.json\n- " + `    "backlog": {` + "\n",
		"--- CLAUDE.md\n- " + backlogBlockStart + "\n",
		"agent: remove from .githooks/post-commit:2: backlog board export\n",
		"1 documents cite .backlog/ paths; those links stay readable in git history\n",
		"the agent: lines below still use backlog or read .backlog/",
		"next: show this diff to the person; once they agree: hug task import backlog .backlog --cleanup\n",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("dry run lacks %q:\n%s", want, out)
		}
	}
	if git(t, dir, "rev-parse", "HEAD") != head || git(t, dir, "status", "--porcelain") != "" {
		t.Fatal("--dry-run changed the repository")
	}

	code, out = runText(a, "import", "backlog", ".backlog", "--cleanup")
	if code != 0 {
		t.Fatalf("cleanup: exit %d\n%s", code, out)
	}
	if n := git(t, dir, "rev-list", "--count", head+"..HEAD"); n != "1" {
		t.Fatalf("%s commits on the code branch, want 1", n)
	}
	changed := git(t, dir, "show", "--name-status", "--format=", "HEAD")
	for _, want := range []string{"D\t.backlog/config.yml", "M\t.mcp.json", "M\tCLAUDE.md"} {
		if !strings.Contains(changed, want) {
			t.Errorf("cleanup commit lacks %q:\n%s", want, changed)
		}
	}
	if !strings.Contains(git(t, dir, "log", "-1", "--format=%B"), "Model: test-model") {
		t.Error("cleanup commit has no Model: trailer")
	}
	if _, e := os.Stat(filepath.Join(dir, ".backlog")); !os.IsNotExist(e) {
		t.Fatal(".backlog still exists")
	}
	if git(t, dir, "show", "HEAD^:.backlog/config.yml") != "project_name: demo" {
		t.Fatal("git history lost the old copy")
	}
	var mcp map[string]map[string]json.RawMessage
	b, _ := os.ReadFile(filepath.Join(dir, ".mcp.json"))
	if e := json.Unmarshal(b, &mcp); e != nil || mcp["mcpServers"]["backlog"] != nil || mcp["mcpServers"]["hug"] == nil {
		t.Fatalf(".mcp.json after cleanup (%v):\n%s", e, b)
	}
	c, _ := os.ReadFile(filepath.Join(dir, "CLAUDE.md"))
	if strings.Contains(string(c), "BACKLOG.MD") || !strings.Contains(string(c), "## Tasks: hug") || !strings.Contains(string(c), "## Build\n\nmake\n") {
		t.Fatalf("CLAUDE.md after cleanup:\n%s", c)
	}
	// The hook is the project's own: printed for the agent, never edited.
	if h, _ := os.ReadFile(filepath.Join(dir, ".githooks", "post-commit")); !strings.Contains(string(h), "backlog board export") {
		t.Fatal("cleanup edited a tracked hook")
	}
	commit := git(t, dir, "rev-parse", "HEAD")
	toml, e := os.ReadFile(filepath.Join(dir, TaskDir, "hug.toml"))
	if e != nil || !strings.Contains(string(toml), "[[import]]\nsource = \".backlog\"\nformat = \"backlog\"\ncommit = \""+commit+"\"\ndate = \"") {
		t.Fatalf("hug.toml (%v):\n%s", e, toml)
	}
	if st := git(t, filepath.Join(dir, TaskDir), "status", "--porcelain"); st != "" {
		t.Fatalf("the record is not committed on the task branch:\n%s", st)
	}
}

// ggydj2 check 2: once recorded, a new import of the source is refused,
// naming the commit that holds the old copy, with a fix: line. Also when
// the folder comes back (a checkout of an older commit).
func TestImportRefusedAfterCleanup(t *testing.T) {
	t.Parallel()
	a, dir := backlogRepo(t)
	execute(t, a, "import", "backlog", ".backlog")
	execute(t, a, "import", "backlog", ".backlog", "--cleanup")
	commit := git(t, dir, "rev-parse", "HEAD")
	git(t, dir, "checkout", "HEAD^", "--", ".backlog")
	git(t, dir, "reset", "-q")
	for _, args := range [][]string{{"import", "backlog", ".backlog"}, {"import", "backlog", ".backlog", "--dry-run"}, {"import", "backlog", "./.backlog/"}} {
		code, out := runText(a, args...)
		if code != 1 || !strings.Contains(out, "not importing .backlog: it was imported and retired in commit "+commit[:8]) ||
			!strings.Contains(out, "\nfix: git show --stat "+commit[:8]) {
			t.Errorf("%v: exit %d\n%s", args, code, out)
		}
	}
}

// ggydj2 check 3: hug setup offers the import, then the retirement once
// every task is in; after it no import is offered, and a leftover folder is
// flagged with the --cleanup fix line.
func TestSetupStopsOfferingRecordedImport(t *testing.T) {
	t.Parallel()
	a, dir := backlogRepo(t)
	steps := func() map[string]SetupStep {
		t.Helper()
		out := map[string]SetupStep{}
		for _, s := range execute(t, a, "setup")["setup"].([]SetupStep) {
			out[s.Step] = s
		}
		return out
	}
	st := steps()
	if s := st["import backlog-md"]; s.Done || !strings.Contains(s.Do, "hug task import backlog .backlog") {
		t.Fatalf("before import: %+v", s)
	}
	execute(t, a, "import", "backlog", ".backlog")
	st = steps()
	if s := st["retire backlog-md"]; s.Done || !strings.Contains(s.Do, "hug task import backlog .backlog --cleanup --dry-run") {
		t.Fatalf("after import, retire step: %+v", st)
	}
	execute(t, a, "import", "backlog", ".backlog", "--cleanup")
	st = steps()
	if _, ok := st["import backlog-md"]; ok {
		t.Fatalf("setup still offers the import: %+v", st)
	}
	if _, ok := st["retire backlog-md"]; ok {
		t.Fatalf("setup shows retire with no folder: %+v", st)
	}
	// A leftover: the folder is back.
	git(t, dir, "checkout", "HEAD^", "--", ".backlog")
	st = steps()
	s, ok := st["retire backlog-md"]
	if !ok || s.Done || !strings.Contains(s.Detail, ".backlog/ is a leftover: it was imported and retired in commit") ||
		!strings.Contains(s.Do, "hug task import backlog .backlog --cleanup --dry-run") {
		t.Fatalf("leftover: %+v", st)
	}
	if _, ok := st["import backlog-md"]; ok {
		t.Fatal("setup offers to import a leftover")
	}
	// --cleanup removes the leftover without touching the record.
	execute(t, a, "import", "backlog", ".backlog", "--cleanup")
	if _, e := os.Stat(filepath.Join(dir, ".backlog")); !os.IsNotExist(e) {
		t.Fatal("leftover not removed")
	}
	if _, ok := steps()["retire backlog-md"]; ok {
		t.Fatal("setup still flags the leftover")
	}
}

// A hook that refuses the cleanup commit (Seid's pre-commit reads
// .backlog/tasks) leaves everything as it was: the folder, the edited
// files, the index, and no record. The script that reads the folder is
// listed for the agent in the dry run.
func TestImportCleanupRefusedByHookChangesNothing(t *testing.T) {
	t.Parallel()
	a, dir := backlogRepo(t)
	execute(t, a, "import", "backlog", ".backlog")
	hook := "#!/bin/sh\ntest -d .backlog/tasks || { echo 'no .backlog/tasks'; exit 1; }\n"
	if e := os.WriteFile(filepath.Join(dir, ".githooks", "pre-commit"), []byte(hook), 0o755); e != nil {
		t.Fatal(e)
	}
	git(t, dir, "add", ".githooks/pre-commit")
	git(t, dir, "commit", "-m", "hook")
	if code, out := runText(a, "import", "backlog", ".backlog", "--cleanup", "--dry-run"); code != 0 ||
		!strings.Contains(out, "agent: remove from .githooks/pre-commit:2: test -d .backlog/tasks") {
		t.Fatalf("dry run does not list the hook: exit %d\n%s", code, out)
	}
	head := git(t, dir, "rev-parse", "HEAD")
	mcp, _ := os.ReadFile(filepath.Join(dir, ".mcp.json"))
	code, out := runText(a, "import", "backlog", ".backlog", "--cleanup")
	if code != 1 || !strings.Contains(out, "the cleanup commit was refused, nothing changed: no .backlog/tasks") {
		t.Fatalf("exit %d\n%s", code, out)
	}
	if git(t, dir, "rev-parse", "HEAD") != head || git(t, dir, "status", "--porcelain") != "" {
		t.Fatalf("a refused cleanup changed the repository:\n%s", git(t, dir, "status", "--porcelain"))
	}
	if now, _ := os.ReadFile(filepath.Join(dir, ".mcp.json")); string(now) != string(mcp) {
		t.Fatal(".mcp.json not restored")
	}
	// A new board's hug.toml says only its layout (dx9vw2); no import.
	if toml, _ := os.ReadFile(filepath.Join(dir, TaskDir, "hug.toml")); strings.Contains(string(toml), "[[import]]") {
		t.Fatalf("a refused cleanup recorded the import:\n%s", toml)
	}
}

func TestRemoveMCPServerKeepsOtherBytes(t *testing.T) {
	for _, c := range []struct{ in, want string }{
		{`{"mcpServers":{"backlog":{"command":"backlog"},"hug":{"command":"hug"}}}`, `{"mcpServers":{"hug":{"command":"hug"}}}`},
		{`{"mcpServers":{"hug":{"command":"hug"},"backlog":{"command":"backlog"}}}`, `{"mcpServers":{"hug":{"command":"hug"}}}`},
		{`{"mcpServers":{"backlog":{"args":["a",{"b":1}]}}}`, `{"mcpServers":{}}`},
		{"{\n  \"servers\": {\n    \"x\": {},\n    \"backlog\": {\"a\": \"}\"}\n  },\n  \"other\": 1\n}\n", "{\n  \"servers\": {\n    \"x\": {}\n  },\n  \"other\": 1\n}\n"},
	} {
		got, ok := removeMCPServer([]byte(c.in), "backlog")
		if !ok || string(got) != c.want {
			t.Errorf("%s:\n got %s (%v)\nwant %s", c.in, got, ok, c.want)
		}
	}
	if _, ok := removeMCPServer([]byte(`{"mcpServers":{"hug":{}}, "backlog": {}}`), "backlog"); ok {
		t.Error("removed a top-level key that is not a server")
	}
}
