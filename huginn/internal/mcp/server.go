// Package mcp exposes the same task commands as the CLI over standard I/O.
package mcp

import (
	"context"
	"encoding/json"
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
	filters := func() map[string]any {
		return map[string]any{"type": enum("feature", "bug", "chore", "spike", "docs"), "priority": str("Priority label"), "label": str("Label filter"), "group": str("Exact folder"), "epic": str("Epic id: that epic and its children"), "mine": map[string]any{"type": "boolean"}}
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
	p["status"] = enum("todo", "doing", "done", "dropped")
	add("list", "List tasks in rank order, including done and dropped unless filtered.", p, nil, func(m map[string]any) ([]string, error) {
		return flags([]string{"ls"}, m, "type", "priority", "label", "group", "epic", "mine", "status"), nil
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
	add("new", "Create a task. Describe it with goal (where we aim), purpose (why), principle (what we do and do not do), details (only facts a decision depends on) and acceptance checks. Keep the title under 80 characters.", map[string]any{"title": str("Short task title, under 80 characters"), "goal": str("Where we aim: the end state, one or two sentences"), "purpose": str("Why we aim there: what improves for whom"), "principle": str("The approach: what we do and deliberately do not do"), "details": str("Only facts a decision depends on: numbers, traps, names"), "accept": array("Observable acceptance checks"), "dep": array("Dependency ids, slugs or aliases"), "type": enum("feature", "bug", "chore", "spike", "docs"), "priority": str("Priority label"), "label": array("Labels"), "group": str("Folder"), "top": map[string]any{"type": "boolean"}, "after": str("Task to follow")}, []string{"title", "goal", "accept"}, func(m map[string]any) ([]string, error) {
		return flags([]string{"new", get(m, "title")}, m, "goal", "purpose", "principle", "details", "accept", "dep", "type", "priority", "label", "group", "top", "after"), nil
	})
	add("update", "Change fields (set, parent), dependencies (dep), checks (check ticks item; strike strikes item out as not relevant with reason, undo restores it), promote an item to its own task (promote, item), notes, rank (move), folder (move_group), drop with the reason in text, wait (text: what on) and resume a doing task, or ask a design question on a todo task (design, text; empty text clears it), answer the question a task's stage asks (answer, text: the answer; design, waiting or review; the asker sees it first in show and next --mine), using only the selected action's fields.", map[string]any{"id": str("Task reference"), "action": enum("set", "check", "strike", "promote", "note", "move", "move_group", "mv", "dep", "drop", "wait", "resume", "design", "answer", "rebalance", "attach", "detach"), "name": str("Task file name for attach/detach, e.g. bench/README.md"), "content": str("Text to attach as the file (attach); use instead of paths when the server cannot read your files"), "paths": array("Files or folders to attach (attach), read by the server"), "replace": map[string]any{"type": "boolean"}, "safe_reason": str("Only after a secret warning: why the text is safe"), "status": enum("todo", "doing", "done", "dropped"), "type": enum("feature", "bug", "chore", "spike", "docs"), "priority": str("Priority label"), "title": str("New title"), "label": str("Label"), "assignee": str("Assignee"), "parent": str("Epic this task belongs to (id); none clears. Epic > task > acceptance item"), "item": map[string]any{"type": "integer", "minimum": 1}, "reason": str("Why the check no longer applies (strike, required)"), "undo": map[string]any{"type": "boolean", "description": "strike: restore the struck check"}, "agent": str("Who decides the strike or gives the answer: your agent or session name (optional; default HUG_USER, the harness session, or the server model)"), "model": str("Your model, recorded on a strike when no agent name is given"), "text": str("Note text; the reason for drop; what it waits on for wait; the question for design (empty clears it); the answer for answer"), "for": str("design: the person who must answer (required with a question)"), "on": str("wait: the person it waits on, if any"), "add_deps": array("Dependency references to add, idempotently; use action dep. An epic waits for all its children. A condition is 'when PREDICATE' (date YYYY-MM-DD, load below N, branch NAME merged, release NAME exists, task:ID closed, group NAME closed), optionally 'wakes NAME'; or 'group:NAME'"), "remove_deps": array("Dependency references or conditions ('when ...') to remove; use action dep"), "group": str("Destination folder, requires action move_group and preserves rank and contents"), "before": str("Insert before task"), "after": str("Insert after task"), "top": map[string]any{"type": "boolean"}}, []string{"id"}, func(m map[string]any) ([]string, error) {
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
		allowed := map[string]string{"set": "status type priority title label assignee parent", "check": "item", "strike": "item reason undo agent model", "promote": "item", "note": "text", "move": "before after top", "move_group": "group", "mv": "group", "dep": "add_deps remove_deps", "drop": "text", "wait": "text on", "answer": "text agent model", "resume": "", "design": "text for", "rebalance": "", "attach": "name content paths replace safe_reason text", "detach": "name"}
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
				if empty || (action != "set" && (key == "status" || key == "type" || key == "item")) {
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
			args = flags(args, m, "status", "type", "priority", "title", "label", "assignee", "parent")
		case "check", "promote", "strike":
			n, ok := m["item"].(float64)
			if !ok {
				return nil, &cli.UsageError{Problem: action + " needs item", Fix: "hug task show " + id}
			}
			args = append(args, strconv.Itoa(int(n)))
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
			args = append(args, get(m, "text"))
			if v := get(m, "agent"); v != "" {
				args = append(args, "--as", v)
			}
			if v := get(m, "model"); v != "" {
				args = append(args, "--model", v)
			}
		case "wait":
			args = append(args, get(m, "text"))
			if on := get(m, "on"); on != "" {
				args = append(args, "--on", on)
			}
		case "design":
			if q := get(m, "text"); q != "" {
				args = append(args, q, "--for", get(m, "for"))
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
	add("take", "Claim a task using the remote push lock and set it doing. Say which model you are (model, required: claude-opus-5-5, gpt-6.1-sol, ...; a person: human), the folder you work in (cwd, required: the branch and worktree are read from it and recorded) and, if you have one, your agent or session name (agent). Without agent the assignee is the model.", map[string]any{"id": str("Task reference"), "model": str("The model doing the work, e.g. claude-opus-5-5 or gpt-6.1-sol; a person: human. Required unless HUG_MODEL is set for the server"), "cwd": str("Your working directory (absolute path), e.g. the git worktree you edit in. Required: hug records its branch and worktree on the task"), "agent": str("Your agent or session name, e.g. poodle (optional; default HUG_USER or the harness session)")}, []string{"id"}, func(m map[string]any) ([]string, error) {
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
		add(cmd, map[string]string{"lint": "Report malformed files, dependencies on no task, Depends on lines that are not dependencies, cycles and unchecked done tasks; each finding has a code and a fix command.", "sync": "Commit local edits, rebase and push the task branch only.", "types": "List the five supported task types and their meanings."}[cmd], map[string]any{}, nil, func(map[string]any) ([]string, error) { return []string{cmd}, nil })
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
