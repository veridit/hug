// Package mcp exposes the same task commands as the CLI over standard I/O.
package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/veridit/hug/huginn/internal/task"
	"math"
	"path/filepath"
	"slices"
	"sort"
	"strconv"
	"strings"
	"sync"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/veridit/hug/huginn/gitbranch"
	"github.com/veridit/hug/huginn/internal/cli"
)

const Instructions = "Huginn keeps project tasks on a dedicated Git branch.\nUse list or next, then take (with your model, and your agent name if you have one), update acceptance, and done.\nEvery result includes a next step, and errors include exact fixes."

type Executor interface {
	Execute(context.Context, []string) (cli.Reply, error)
}

// New creates ten flat tools with the CLI's canonical JSON contract and errors.
func New(e Executor) *sdk.Server {
	s := sdk.NewServer(&sdk.Implementation{Name: "huginn", Version: "0.1.1"}, &sdk.ServerOptions{Instructions: Instructions})
	Register(s, e, "")
	return s
}

// PrefixedInstructions describes the task tools when they carry prefix
// (hug mcp registers them as task_list, task_show, ...).
func PrefixedInstructions(prefix string) string {
	return "Tasks (" + prefix + "* tools) live on a dedicated Git branch of the current repository.\nUse " + prefix + "list or " + prefix + "next, then " + prefix + "take (with your model, and your agent name if you have one), " + prefix + "update acceptance, and " + prefix + "done.\nEvery result includes a next step, and errors include exact fixes."
}

// Register adds the task tools to s, each named prefix+name.
func Register(s *sdk.Server, e Executor, prefix string) {
	var mu sync.Mutex
	str := func(desc string) map[string]any { return map[string]any{"type": "string", "description": desc} }
	enum := func(values ...string) map[string]any { return map[string]any{"type": "string", "enum": values} }
	// negatable is a filter's enum with each value's ! form too (7zg9mf):
	// "!done" is every task but the done ones, as in hug task ls.
	negatable := func(values ...string) map[string]any {
		all := append([]string{}, values...)
		for _, v := range values {
			all = append(all, "!"+v)
		}
		return map[string]any{"type": "string", "enum": all, "description": "A value, or !value: every task but those"}
	}
	filters := func() map[string]any {
		return map[string]any{"type": negatable("feature", "bug", "chore", "spike", "docs"), "priority": str("Priority label, or !label: all but those"), "label": str("Label filter, or !label: all but those"), "group": str("Exact folder, or !folder: all but it"), "epic": str("Epic id: that epic and its children"), "mine": map[string]any{"type": "boolean"}}
	}
	add := func(name, desc string, props map[string]any, required []string, build func(map[string]any) ([]string, error)) {
		schema := map[string]any{"type": "object", "properties": props, "additionalProperties": false}
		if len(required) > 0 {
			schema["required"] = required
		}
		// Raw registration lets even schema errors carry the same actionable next hint.
		s.AddTool(&sdk.Tool{Name: prefix + name, Description: desc, InputSchema: schema}, func(ctx context.Context, req *sdk.CallToolRequest) (*sdk.CallToolResult, error) {
			mu.Lock()
			defer mu.Unlock()
			input := map[string]any{}
			var err error
			if len(req.Params.Arguments) > 0 {
				if e := json.Unmarshal(req.Params.Arguments, &input); e != nil {
					err = &cli.UsageError{Problem: "invalid tool arguments: " + e.Error(), Fix: "hug task help"}
				}
			}
			// Flat-tool bridges may materialize omitted optional properties as null.
			// Null means absent, never an instruction to clear a field.
			for key, value := range input {
				_, known := props[key]
				if known && value == nil && !slices.Contains(required, key) {
					delete(input, key)
				}
			}
			if err == nil {
				err = validate(input, props, required)
			}
			var args []string
			if err == nil {
				args, err = build(input)
			}
			var reply cli.Reply
			if err == nil {
				reply, err = e.Execute(ctx, args)
			}
			failed := err != nil
			if failed {
				code, p, fix := cli.ErrorDetails(err)
				reply = cli.Reply{"error": p, "fix": fix, "next": fix, "exit_code": code}
				// Every problem as a list (fkpyg3), beside the one fix.
				var te *task.Error
				if errors.As(err, &te) && len(te.Problems) > 0 {
					reply["findings"] = te.Problems
				}
				if fixes := cli.ErrorFixes(err); fixes != nil {
					reply["fixes"] = fixes
				}
				if waiting := cli.ErrorWaiting(err); waiting != nil {
					reply["waiting"] = waiting
				}
			}
			b, err := json.Marshal(reply)
			if err != nil {
				return nil, err
			}
			return &sdk.CallToolResult{IsError: failed, Content: []sdk.Content{&sdk.TextContent{Text: string(b)}}, StructuredContent: reply}, nil
		})
	}
	get := func(m map[string]any, k string) string { v, _ := m[k].(string); return v }
	flags := func(args []string, m map[string]any, keys ...string) []string {
		for _, k := range keys {
			if v, ok := m[k]; ok {
				switch v := v.(type) {
				case string:
					args = append(args, "--"+k, v)
				case bool:
					if v {
						args = append(args, "--"+k)
					}
				case []any:
					for _, x := range v {
						args = append(args, "--"+k, fmt.Sprint(x))
					}
				}
			}
		}
		return args
	}
	p := filters()
	p["state"] = negatable("todo", "doing", "done", "dropped")
	add("list", "List tasks in rank order, including done and dropped unless filtered.", p, nil, func(m map[string]any) ([]string, error) {
		return flags([]string{"ls"}, m, "type", "priority", "label", "group", "epic", "mine", "state"), nil
	})
	sp := filters()
	// hbgybn: the open requests on the board, oldest first, each with its
	// resolve command; with id, that task's; all includes resolved ones.
	add("requests", "List the open requests people wrote on tasks (hbgybn), oldest first: task id, request number, who, when, the text verbatim, its receipt (request.receipt.state) and the fix that resolves it. With id, that task's; all includes resolved ones with their accounts. Act on a request by changing the ticket, then task_update action resolve. "+cli.DeliveryFlow, map[string]any{"id": str("Task reference (optional)"), "all": map[string]any{"type": "boolean"}}, nil, func(m map[string]any) ([]string, error) {
		args := []string{"requests"}
		if id := get(m, "id"); id != "" {
			args = append(args, id)
		}
		return flags(args, m, "all"), nil
	})
	sp["state"] = negatable("todo", "doing", "done", "dropped")
	sp["query"] = str("Words to find, e.g. memory limit 137 or STATBUS-402; every word must match")
	sp["limit"] = map[string]any{"type": "integer", "minimum": 1}
	add("search", "Find tasks by words, best first (BM25 over id, aliases, title, goal, description, acceptance, notes; id, aliases and title count most). Each result has id, state, title, the field that matched and a snippet. Filters as in list.", sp, []string{"query"}, func(m map[string]any) ([]string, error) {
		args := flags([]string{"search"}, m, "type", "priority", "label", "group", "epic", "mine", "state")
		if n, ok := m["limit"].(float64); ok {
			args = append(args, "--limit", fmt.Sprint(int(n)))
		}
		return append(args, "--", get(m, "query")), nil
	})
	add("show", "Show a task by id, unique prefix, slug or alias with dependencies, acceptance and its files (name, bytes, sha256). With file, returns that file's content.", map[string]any{"id": str("Task reference"), "file": str("A task file name, e.g. bench/README.md, to read its content")}, []string{"id"}, func(m map[string]any) ([]string, error) {
		args := []string{"show", get(m, "id")}
		if f := get(m, "file"); f != "" {
			args = append(args, "--file", f)
		}
		return args, nil
	})
	array := func(desc string) map[string]any {
		return map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "description": desc}
	}
	add("new", "Create a task. Give all of: goal (where we aim), purpose (why), principle (what we do and do not do) and acceptance checks; details (only facts a decision depends on) is optional. A missing part is reported with every other one, as findings plus one fix. The Goal is one sentence of at most 140 characters; detail goes in Details. Keep the title under 80 characters.", map[string]any{"title": str("Short task title, under 80 characters"), "goal": str("Where we aim: one sentence of at most 140 characters; detail goes in details"), "purpose": str("Why we aim there: what improves for whom"), "principle": str("The approach: what we do and deliberately do not do"), "details": str("Only facts a decision depends on: numbers, traps, names"), "accept": array("Observable acceptance checks"), "dep": array("Dependency ids, slugs or aliases"), "type": enum("feature", "bug", "chore", "spike", "docs"), "priority": str("Priority label"), "label": array("Labels"), "group": str("Folder"), "top": map[string]any{"type": "boolean"}, "after": str("Task to follow")}, []string{"title"}, func(m map[string]any) ([]string, error) {
		// Only title is required by the schema: hug task new itself names
		// every missing part at once, with one fix (fkpyg3).
		return flags([]string{"new", get(m, "title")}, m, "goal", "purpose", "principle", "details", "accept", "dep", "type", "priority", "label", "group", "top", "after"), nil
	})
	add("update", "Change fields (set, parent; set also writes goal, purpose, principle, details and appends accept items, so every lint fix runs without an editor), dependencies (dep), checks (check ticks item, with text: the evidence, noted in the same write as 'check N: text'; strike strikes item out as not relevant with reason, undo restores it), promote an item to its own task (promote, item), notes, rank (move), folder (move_group), drop with the reason in text, wait on a thing with no person to answer (text: what on, e.g. CI; until: a date it ends on its own) and resume a doing task, ask a person a question on a doing task, or on a todo task without taking it (it stays todo with its dependencies and shows in the person's ASKED at once) (ask, one question per ask, each numbered and answered on its own; to: who answers, required; ask_after: a date to ask it on, not before; text: the question, kept verbatim with its line breaks; check: the acceptance check it is about; show), ask for a review of a doing task (review, by: the reviewer, required; you keep it), or ask a design question on a todo task (design, text; empty text clears it), write a person's request on a task in their words (request, text, kept verbatim; it does not change the stage; ids: more tasks the same request is for, written once on id with a link on each, every target resolved on its own and the request closed when the last is), close request N with what you changed (resolve, item: N, text), answer the question a task's stage asks (answer, text: the answer, kept verbatim with its line breaks; design, asked or review; item: the question's number when several are open, each answered on its own; the asker sees it first in show and next --mine), using only the selected action's fields. A request or answer reply says its receipt (receipt). "+cli.DeliveryFlow+" A set reply says what it wrote first (wrote: one line per changed field, e.g. 'added check N: ...'), then lists the task's remaining lint findings with one combined fix.", map[string]any{"id": str("Task reference"), "ids": array("request: more tasks the same request is for (one request for a selection)"), "goal": str("New Goal text (set): one sentence of at most 140 characters; detail goes in details"), "purpose": str("New Purpose text (set)"), "principle": str("New Principle text (set)"), "details": str("New Details text (set)"), "accept": array("Acceptance items to append (set)"), "action": enum("set", "check", "strike", "promote", "note", "move", "move_group", "mv", "dep", "drop", "wait", "ask", "resume", "review", "design", "answer", "request", "resolve", "rebalance", "attach", "detach"), "name": str("Task file name for attach/detach, e.g. bench/README.md"), "content": str("Text to attach as the file (attach); use instead of paths when the server cannot read your files"), "paths": array("Files or folders to attach (attach), read by the server"), "replace": map[string]any{"type": "boolean"}, "safe_reason": str("Only after a secret warning: why the text is safe"), "state": enum("todo", "doing", "done", "dropped"), "type": enum("feature", "bug", "chore", "spike", "docs"), "priority": str("Priority label"), "title": str("New title"), "label": str("Label"), "tag": array("Tags to change, one each (set): NAME adds it, OLD=NEW replaces OLD, OLD= removes it; again changes nothing"), "assignee": str("Assignee"), "parent": str("Epic this task belongs to (id); none clears. Epic > task > acceptance item"), "item": map[string]any{"type": "integer", "minimum": 1}, "reason": str("Why the check no longer applies (strike, required)"), "undo": map[string]any{"type": "boolean", "description": "strike: restore the struck check"}, "agent": str("Who decides the strike, asks the question (ask) or gives the answer: your agent or session name (optional; default HUG_USER, the harness session, or the server model)"), "model": str("Your model, recorded on a strike when no agent name is given"), "text": str("Note text; the evidence for check; the request (request); what changed (resolve); the reason for drop; what it waits on for wait; the question for ask; the question for design (empty clears it); the answer for answer"), "for": str("design: the person who must answer (required with a question)"), "until": str("wait: a date YYYY-MM-DD (local) the wait ends on its own: the task comes back to its owner, first in next --mine"), "ask_after": str("ask: a date YYYY-MM-DD (local): the task waits until then and asks the question that day; nothing shows in the person's ASKED before"), "to": str("ask: who must answer (a person's short name in hug.toml [people] or an agent's name), required"), "check": map[string]any{"type": "integer", "description": "ask: the acceptance check the question is about (1, 2, ...)"}, "by": str("review: who reviews it (required)"), "show": str("ask, design or review: the task file the person should look at (attach it first); the detail shows it under the question"), "add_deps": array("Dependency references to add, idempotently; use action dep. An epic waits for all its children. A condition is 'when PREDICATE' (date YYYY-MM-DD, load below N, branch NAME merged, release NAME exists, task:ID closed, group NAME closed), optionally 'wakes NAME'; or 'group:NAME'"), "remove_deps": array("Dependency references or conditions ('when ...') to remove; use action dep"), "group": str("Destination folder, requires action move_group and preserves rank and contents"), "before": str("Insert before task"), "after": str("Insert after task"), "top": map[string]any{"type": "boolean"}}, []string{"id"}, func(m map[string]any) ([]string, error) {
		action, id := get(m, "action"), get(m, "id")
		if action == "" {
			action = "set"
			if deps, _ := m["add_deps"].([]any); len(deps) > 0 {
				action = "dep"
			}
			if deps, _ := m["remove_deps"].([]any); len(deps) > 0 {
				action = "dep"
			}
		}
		allowed := map[string]string{"set": "state type priority title label tag assignee parent goal purpose principle details accept", "check": "item text agent model", "strike": "item reason undo agent model", "promote": "item", "note": "text", "move": "before after top", "move_group": "group", "mv": "group", "dep": "add_deps remove_deps", "drop": "text", "wait": "text until", "ask": "text to check show agent ask_after", "review": "by show", "answer": "item text agent model", "request": "text agent model ids", "resolve": "item text agent model", "resume": "", "design": "text for show", "rebalance": "", "attach": "name content paths replace safe_reason text", "detach": "name"}
		permitted := strings.Fields(allowed[action] + " id action")
		for key := range m {
			if !slices.Contains(permitted, key) {
				// The selected action owns its fields. Flat model bridges often
				// populate unrelated enum and item defaults as well as empty values.
				// Never apply them, but still reject meaningful mixed operations.
				value := m[key]
				empty := value == "" || value == false
				if values, ok := value.([]any); ok {
					empty = len(values) == 0
				}
				if empty || (action != "set" && (key == "state" || key == "type" || key == "item")) {
					delete(m, key)
					continue
				}
				fix := "hug task help"
				if key == "group" {
					fix = "hug task mv " + id + " " + get(m, "group")
				}
				if key == "before" || key == "after" || key == "top" {
					fix = "hug task move " + id + " --" + key
					if key != "top" {
						fix += " " + get(m, key)
					}
					return nil, &cli.UsageError{Problem: "argument " + key + " reorders rank and requires action move, not " + action, Fix: fix}
				}
				return nil, &cli.UsageError{Problem: "argument " + key + " does not apply to action " + action + "; use move_group for folders or dep for dependencies", Fix: fix}
			}
		}
		args := []string{action, id}
		switch action {
		case "attach":
			if get(m, "content") != "" {
				if get(m, "name") == "" {
					return nil, &cli.UsageError{Problem: "attach with content needs name", Fix: "set name, e.g. results.tsv"}
				}
				args = append(args, "--content", get(m, "content"), "--name", get(m, "name"))
			} else {
				ps, _ := m["paths"].([]any)
				if len(ps) == 0 {
					return nil, &cli.UsageError{Problem: "attach needs content (with name) or paths", Fix: "pass content and name"}
				}
				for _, p := range ps {
					args = append(args, fmt.Sprint(p))
				}
				if n := get(m, "name"); n != "" {
					args = append(args, "--name", n)
				}
			}
			if t := get(m, "text"); t != "" {
				args = append(args, "--note", t)
			}
			if r, _ := m["replace"].(bool); r {
				args = append(args, "--replace")
			}
			if sr := get(m, "safe_reason"); sr != "" {
				args = append(args, "--safe-reason", sr)
			}
			return args, nil
		case "detach":
			return []string{"detach", id, get(m, "name")}, nil
		case "set":
			args = flags(args, m, "state", "type", "priority", "title", "label", "tag", "assignee", "parent", "goal", "purpose", "principle", "details", "accept")
		case "check", "promote", "strike":
			n, ok := m["item"].(float64)
			if !ok {
				return nil, &cli.UsageError{Problem: action + " needs item", Fix: "hug task show " + id}
			}
			args = append(args, strconv.Itoa(int(n)))
			// check with text (pw0t19): the evidence, noted in the same
			// write as the tick.
			if action == "check" {
				if v := get(m, "text"); strings.TrimSpace(v) != "" {
					args = append(args, v)
				}
				if v := get(m, "agent"); v != "" {
					args = append(args, "--as", v)
				}
				if v := get(m, "model"); v != "" {
					args = append(args, "--model", v)
				}
			}
			if action == "strike" {
				if u, _ := m["undo"].(bool); u {
					args = append(args, "--undo")
				} else if strings.TrimSpace(get(m, "reason")) == "" {
					return nil, &cli.UsageError{Problem: "strike needs reason: why the check no longer applies", Fix: "task_update with {\"id\": \"" + id + "\", \"action\": \"strike\", \"item\": N, \"reason\": \"why it no longer applies\"}"}
				} else {
					args = append(args, "--reason", get(m, "reason"))
				}
				if v := get(m, "agent"); v != "" {
					args = append(args, "--as", v)
				}
				if v := get(m, "model"); v != "" {
					args = append(args, "--model", v)
				}
			}
		case "note", "drop":
			args = append(args, get(m, "text"))
		case "answer":
			// q2adr8: item is the question's number when several are open.
			if n, ok := m["item"].(float64); ok && n >= 1 {
				args = append(args, strconv.Itoa(int(n)))
			}
			args = append(args, get(m, "text"))
			if v := get(m, "agent"); v != "" {
				args = append(args, "--as", v)
			}
			if v := get(m, "model"); v != "" {
				args = append(args, "--model", v)
			}
		case "request":
			// ec6z6j: ids are the other tasks one request is for; it is
			// written once on id, a link on each of ids.
			if more, _ := m["ids"].([]any); len(more) > 0 {
				for _, x := range more {
					if s, ok := x.(string); ok && strings.TrimSpace(s) != "" {
						args = append(args, "--task", s)
					}
				}
			}
			args = append(args, get(m, "text"))
			if v := get(m, "agent"); v != "" {
				args = append(args, "--as", v)
			}
			if v := get(m, "model"); v != "" {
				args = append(args, "--model", v)
			}
		case "resolve":
			n, ok := m["item"].(float64)
			if !ok || n < 1 {
				return nil, &cli.UsageError{Problem: "resolve needs item: the request number", Fix: "hug task requests " + id}
			}
			args = append(args, strconv.Itoa(int(n)), get(m, "text"))
			if v := get(m, "agent"); v != "" {
				args = append(args, "--as", v)
			}
			if v := get(m, "model"); v != "" {
				args = append(args, "--model", v)
			}
		case "wait":
			args = append(args, get(m, "text"))
			if v := get(m, "until"); v != "" {
				args = append(args, "--until", v)
			}
		case "ask":
			args = append(args, get(m, "text"), "--to", get(m, "to"))
			if v := get(m, "ask_after"); v != "" {
				args = append(args, "--after", v)
			}
			if v, ok := m["check"].(float64); ok && v != 0 {
				args = append(args, "--check", strconv.Itoa(int(v)))
			} else if v := get(m, "check"); v != "" && v != "0" {
				args = append(args, "--check", v)
			}
			if v := get(m, "show"); v != "" {
				args = append(args, "--show", v)
			}
			if v := get(m, "agent"); v != "" {
				args = append(args, "--as", v)
			}
		case "review":
			args = append(args, "--by", get(m, "by"))
			if v := get(m, "show"); v != "" {
				args = append(args, "--show", v)
			}
		case "design":
			if q := get(m, "text"); q != "" {
				args = append(args, q, "--for", get(m, "for"))
				if v := get(m, "show"); v != "" {
					args = append(args, "--show", v)
				}
			} else {
				args = append(args, "--clear")
			}
		case "mv", "move_group":
			if _, ok := m["group"]; !ok {
				return nil, &cli.UsageError{Problem: "move_group needs group", Fix: "hug task show " + id}
			}
			args = []string{"mv", id, get(m, "group")}
		case "dep":
			args = []string{"dep", id}
			for _, pair := range [][2]string{{"add_deps", "add"}, {"remove_deps", "remove"}} {
				if values, ok := m[pair[0]].([]any); ok {
					for _, v := range values {
						args = append(args, "--"+pair[1], v.(string))
					}
				}
			}
		case "move":
			if _, ok := m["group"]; ok {
				return nil, &cli.UsageError{Problem: "action move reorders rank; to change folder use action move_group with group", Fix: "hug task mv " + id + " " + get(m, "group")}
			}
			for _, key := range []string{"before", "after"} {
				if get(m, key) == "" {
					delete(m, key)
				}
			}
			args = flags(args, m, "before", "after", "top")
		}
		return args, nil
	})
	add("take", "Claim a task using the remote push lock and set it doing. Say which model you are (model, required: claude-opus-5-5, gpt-6.1-sol, ...; a person: human), the folder you work in (cwd, required: the branch and worktree are read from it and recorded) and, if you have one, your agent or session name (agent). Without agent the assignee is the model. One ticket in doing per agent: while you have an active doing task, take is refused naming it, with fixes (done, wait, back to todo, a detour, or take it with the active one); a waiting, review or depends task does not count. Tasks that ship together (one change or release) are both doing: take the second with with set to your active task's id.", map[string]any{"id": str("Task reference"), "with": str("Your active task's id, when this one ships together with it (one change or release): both are doing, this one marked with it"), "model": str("The model doing the work, e.g. claude-opus-5-5 or gpt-6.1-sol; a person: human. Required unless HUG_MODEL is set for the server"), "cwd": str("Your working directory (absolute path), e.g. the git worktree you edit in. Required: hug records its branch and worktree on the task"), "agent": str("Your agent or session name, e.g. poodle (optional; default HUG_USER or the harness session)")}, []string{"id"}, func(m map[string]any) ([]string, error) {
		id := get(m, "id")
		// One habit for agents (x1t1t1, Jørgen 2026-11-16): the server's
		// own folder is not where the agent works, so cwd is always given.
		// Checked here, not in the schema, so the error carries the fix.
		cwd := strings.TrimSpace(get(m, "cwd"))
		if cwd == "" {
			return nil, &cli.UsageError{Problem: "no cwd given: say which folder you work in, so the branch and worktree of " + id + " are recorded", Fix: "task_take with cwd set to your working directory (absolute path), e.g. {\"id\": \"" + id + "\", \"model\": \"MODEL\", \"cwd\": \"/path/to/worktree\"}"}
		}
		if !filepath.IsAbs(cwd) {
			return nil, &cli.UsageError{Problem: "cwd must be an absolute path, got " + strconv.Quote(cwd), Fix: "task_take with cwd set to the absolute path of your working directory (pwd)"}
		}
		at, err := gitbranch.Where(cwd)
		if err != nil {
			return nil, &cli.UsageError{Problem: "cwd " + strconv.Quote(cwd) + " is not inside a git checkout", Fix: "task_take with cwd set to the git worktree you work in (git -C DIR rev-parse --show-toplevel)"}
		}
		args := []string{"take", id}
		if v := get(m, "with"); v != "" {
			args = append(args, "--with", v)
		}
		if at.Branch != "" {
			args = append(args, "--branch", at.Branch)
		}
		args = append(args, "--worktree", at.Worktree)
		if v := get(m, "agent"); v != "" {
			args = append(args, "--as", v)
		}
		if v := get(m, "model"); v != "" {
			args = append(args, "--model", v)
		}
		return args, nil
	})
	add("done", "Finish a task after dependencies and acceptance checks are complete. Then answer one question in reflection: "+task.ReflectQuestion+". The answer is stored as a Reflection note on the task.", map[string]any{"id": str("Task reference"), "reflection": str("Optional: did an instruction (AGENTS.md section, tool description, hug message) mislead you or leave a gap? Quote it and say what happened, with evidence. 'nothing' is a fine answer"), "agent": str("Your agent or session name, recorded with the reflection (optional)")}, []string{"id"}, func(m map[string]any) ([]string, error) {
		args := []string{"done", get(m, "id")}
		if v := get(m, "reflection"); v != "" {
			args = append(args, "--reflect", v)
		}
		if v := get(m, "agent"); v != "" {
			args = append(args, "--as", v)
		}
		return args, nil
	})
	add("next", "Find the highest-ranked available task or your doing task with combined filters.", filters(), nil, func(m map[string]any) ([]string, error) {
		return flags([]string{"next"}, m, "type", "priority", "label", "group", "epic", "mine"), nil
	})
	for _, cmd := range []string{"lint", "sync", "types"} {
		add(cmd, map[string]string{"lint": "Report what needs fixing on open tasks (todo, doing): malformed files, dependencies on no task, Depends on lines that are not dependencies, cycles; each finding has a code and a fix command. Done and dropped tasks are not judged.", "sync": "Commit local edits, rebase and push the task branch only.", "types": "List the five supported task types and their meanings."}[cmd], map[string]any{}, nil, func(map[string]any) ([]string, error) { return []string{cmd}, nil })
	}
}
func validate(input map[string]any, props map[string]any, required []string) error {
	bad := func(message string) error { return &cli.UsageError{Problem: message, Fix: "hug task help"} }
	for _, key := range required {
		if _, ok := input[key]; !ok {
			return bad("missing argument " + key)
		}
	}
	keys := []string{}
	for key := range props {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for key, value := range input {
		raw, ok := props[key]
		if !ok {
			e := task.Invalid("argument", key, keys)
			return bad("unknown argument " + key + "; valid: " + strings.Join(keys, ", ") + "; closest: " + e.Closest)
		}
		spec := raw.(map[string]any)
		valid := false
		switch spec["type"] {
		case "string":
			_, valid = value.(string)
		case "boolean":
			_, valid = value.(bool)
		case "integer":
			v, ok := value.(float64)
			valid = ok && math.Trunc(v) == v && v >= 1
		case "array":
			items, ok := value.([]any)
			valid = ok
			for _, v := range items {
				if _, ok := v.(string); !ok {
					valid = false
				}
			}
		}
		if !valid {
			return bad("invalid " + key + ": expected " + fmt.Sprint(spec["type"]))
		}
		if options, ok := spec["enum"].([]string); ok && !slices.Contains(options, value.(string)) {
			e := task.Invalid(key, value.(string), options)
			e.Fix = "hug task help"
			return e
		}
	}
	return nil
}

func Run(ctx context.Context, e Executor) error { return New(e).Run(ctx, &sdk.StdioTransport{}) }
