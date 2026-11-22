package tui

import (
	"slices"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/veridit/hug/huginn/internal/task"
)

// The palette's context groups (05r3pg, Jørgen 2026-11-16: "edit this
// ticket" from inside it, and everything an agent can do to a task over
// MCP, from the TUI): the task you are in or on first, under TASK id (the
// open task or the selected row), its focused check's actions at the top
// under CHECK n, so a short palette shows them; then the
// global tree (palette.go). An action with a key runs what the key runs,
// so confirmation and commits stay the same; the others run the hug task
// command itself through Options.Run, the code the CLI and MCP run, after
// the same confirm line.

// runKey is the confirm that runs a hug task command (pendingArgs).
const runKey = "run"

// ctxItem is one context action. Names are one word, unique across the
// groups, so a typed line names its item by its first word.
type ctxItem struct {
	name, verb, desc string
	check            bool   // in the CHECK group, about the focused check
	keyList, keyTask string // the direct key in a list, in a task ("" none)
	// arg says what follows the name ("" nothing); optional: the action
	// also runs without it (drop asks the reason, wait asks what on).
	arg      string
	optional bool
	avail    func(m model, t task.Task) bool
	// sub is the next level of the tree after "name ".
	sub func(m model, t task.Task, rest string) []completion
	// args is the hug task command for the typed rest, nil while incomplete.
	args func(m model, t task.Task, rest string) []string
	// do runs an action that has a key, as that key does.
	do func(m *model, t task.Task, rest string) tea.Cmd
}

func always(model, task.Task) bool    { return true }
func doing(_ model, t task.Task) bool { return t.State == "doing" }
func open(_ model, t task.Task) bool  { return t.State == "todo" || t.State == "doing" }

// text is the args builder for an action that takes free text after it.
func text(verb string, extra ...string) func(model, task.Task, string) []string {
	return func(_ model, t task.Task, rest string) []string {
		if rest = strings.TrimSpace(rest); rest == "" {
			return nil
		}
		return append([]string{verb, t.ID, rest}, extra...)
	}
}

// contextItems is set in init: its actions reach the screen (capture),
// which lists contextItems.
var contextItems []ctxItem

func init() { contextItems = contextItemList() }

func contextItemList() []ctxItem {
	return []ctxItem{
		{name: "tick", verb: "check", desc: "Tick this check", check: true, keyTask: "Space",
			avail: func(m model, t task.Task) bool { a, ok := m.focusedCheck(t); return ok && !a.Struck && !a.Checked },
			do:    func(m *model, _ task.Task, _ string) tea.Cmd { m.requestAction(" ", ""); return nil }},
		{name: "untick", verb: "check", desc: "Untick this check", check: true, keyTask: "Space",
			avail: func(m model, t task.Task) bool { a, ok := m.focusedCheck(t); return ok && a.Checked },
			do:    func(m *model, _ task.Task, _ string) tea.Cmd { m.requestAction(" ", ""); return nil }},
		{name: "strike", verb: "strike", desc: "Strike it out as not relevant, with why", check: true, keyTask: "-", arg: "reason", optional: true,
			avail: func(m model, t task.Task) bool { a, ok := m.focusedCheck(t); return ok && !a.Struck && !a.Checked },
			do: func(m *model, t task.Task, rest string) tea.Cmd {
				if rest = strings.TrimSpace(rest); rest != "" {
					m.confirmPath = t.Path
					m.strike(rest)
					return nil
				}
				return m.strikeKey()
			}},
		{name: "restore", verb: "strike", desc: "Restore the struck check", check: true, keyTask: "-",
			avail: func(m model, t task.Task) bool { a, ok := m.focusedCheck(t); return ok && a.Struck },
			do:    func(m *model, _ task.Task, _ string) tea.Cmd { m.requestAction("-", ""); return nil }},
		{name: "promote", verb: "promote", desc: "Make this check its own task", check: true,
			avail: func(m model, t task.Task) bool { _, ok := m.focusedCheck(t); return ok },
			args: func(m model, t task.Task, _ string) []string {
				return []string{"promote", t.ID, strconv.Itoa(m.acceptance + 1)}
			}},

		{name: "take", verb: "take", desc: "Take it: doing, yours", keyList: "t", keyTask: "t",
			avail: func(_ model, t task.Task) bool { return t.State == "todo" },
			do:    func(m *model, _ task.Task, _ string) tea.Cmd { return m.startAction("t") }},
		{name: "giveback", verb: "take", desc: "Give it back to todo", keyList: "t", keyTask: "t",
			avail: func(m model, t task.Task) bool { return t.State == "doing" && contains(t.Assignee, m.me()) },
			do:    func(m *model, _ task.Task, _ string) tea.Cmd { return m.startAction("t") }},
		{name: "done", verb: "done", desc: "Mark it done", keyList: "d", keyTask: "d", avail: doing,
			do: func(m *model, _ task.Task, _ string) tea.Cmd { return m.startAction("d") }},
		{name: "drop", verb: "drop", desc: "Drop it, with why", keyList: "x", keyTask: "x", arg: "reason", optional: true, avail: open,
			do: func(m *model, _ task.Task, rest string) tea.Cmd {
				if rest = strings.TrimSpace(rest); rest != "" {
					m.requestAction("x", rest)
					return nil
				}
				return m.startAction("x")
			}},
		{name: "edit", verb: "edit", desc: "Open it in the editor", keyList: "e", keyTask: "e", avail: always,
			do: func(m *model, _ task.Task, _ string) tea.Cmd { m.requestAction("e", ""); return nil }},
		{name: "request", verb: "request", desc: "Write a request to the agents", keyList: "r", keyTask: "r", avail: always,
			do: func(m *model, _ task.Task, _ string) tea.Cmd { return m.requestKey() }},
		{name: "answer", verb: "answer", desc: "Answer what it asks you", keyTask: "a",
			avail: func(m model, t task.Task) bool { return m.forYou(t) },
			do:    func(m *model, _ task.Task, _ string) tea.Cmd { return m.answerKey() }},
		{name: "wait", verb: "wait", desc: "Park it on a thing, e.g. CI", keyTask: "w", arg: "on what", optional: true, avail: doing,
			do: func(m *model, t task.Task, rest string) tea.Cmd {
				if rest = strings.TrimSpace(rest); rest != "" {
					m.confirmPath = t.Path
					m.wait(rest)
					return nil
				}
				return m.waitKey()
			}},
		{name: "resume", verb: "resume", desc: "End the wait or the question",
			avail: func(m model, t task.Task) bool {
				g := m.taskStage(t)
				return t.State == "doing" && (g == "waiting" || g == "asked")
			},
			args: func(_ model, t task.Task, _ string) []string { return []string{"resume", t.ID} }},
		{name: "yank", verb: "move", desc: "Pick it up to paste elsewhere", keyList: "y", keyTask: "y",
			avail: func(m model, _ task.Task) bool { return m.moveRefusal() == "" },
			do:    func(m *model, _ task.Task, _ string) tea.Cmd { m.yank(); return nil }},
		{name: "note", verb: "note", desc: "Add a note", arg: "text", avail: always, args: text("note")},
		{name: "set", verb: "set", desc: "Change a field", arg: "field", avail: always, sub: setSub, args: setArgs},
		{name: "move", verb: "move", desc: "Change its rank", arg: "where", keyList: "m", avail: always, sub: moveSub, args: moveArgs},
		{name: "mv", verb: "mv", desc: "Move it to another group", arg: "group", avail: always,
			sub: func(m model, _ task.Task, rest string) []completion {
				return m.valueRows("mv ", rest, m.groupNames(), nil)
			},
			args: func(_ model, t task.Task, rest string) []string {
				if g := strings.TrimSpace(rest); g != "" {
					return []string{"mv", t.ID, g}
				}
				return nil
			}},
		{name: "dep", verb: "dep", desc: "Add or remove a dependency", arg: "add|remove", avail: always, sub: depSub, args: depArgs},
		{name: "ask", verb: "ask", desc: "Ask a person; it waits for the answer", arg: "who question", avail: doing,
			sub: func(m model, _ task.Task, rest string) []completion {
				if strings.Contains(rest, " ") {
					return nil
				}
				return m.valueRows("ask ", rest, m.peopleNames(), nil)
			},
			args: func(_ model, t task.Task, rest string) []string {
				who, q, _ := strings.Cut(strings.TrimSpace(rest), " ")
				if who == "" || strings.TrimSpace(q) == "" {
					return nil
				}
				return []string{"ask", t.ID, "--to", who, strings.TrimSpace(q)}
			}},
		{name: "review", verb: "review", desc: "Ask a person for a review", arg: "who", avail: doing,
			sub: func(m model, _ task.Task, rest string) []completion {
				return m.valueRows("review ", rest, m.peopleNames(), nil)
			},
			args: func(_ model, t task.Task, rest string) []string {
				if who := strings.TrimSpace(rest); who != "" {
					return []string{"review", t.ID, "--by", who}
				}
				return nil
			}},
		{name: "design", verb: "design", desc: "A question before anyone starts", arg: "who question",
			avail: func(_ model, t task.Task) bool { return t.State == "todo" },
			sub: func(m model, _ task.Task, rest string) []completion {
				if strings.Contains(rest, " ") {
					return nil
				}
				return m.valueRows("design ", rest, append([]string{"clear"}, m.peopleNames()...), nil)
			},
			args: func(_ model, t task.Task, rest string) []string {
				rest = strings.TrimSpace(rest)
				if rest == "clear" {
					return []string{"design", t.ID, "--clear"}
				}
				who, q, _ := strings.Cut(rest, " ")
				if who == "" || strings.TrimSpace(q) == "" {
					return nil
				}
				return []string{"design", t.ID, strings.TrimSpace(q), "--for", who}
			}},
		{name: "resolve", verb: "resolve", desc: "Close a request with what changed", arg: "n text",
			avail: func(_ model, t task.Task) bool { return len(openRequests(t)) > 0 },
			sub: func(_ model, t task.Task, rest string) []completion {
				if strings.Contains(rest, " ") {
					return nil
				}
				var out []completion
				for _, r := range openRequests(t) {
					n := strconv.Itoa(r.N)
					if strings.HasPrefix(n, rest) {
						out = append(out, completion{value: "resolve " + n + " ", description: oneLine(r.Text)})
					}
				}
				return out
			},
			args: func(_ model, t task.Task, rest string) []string {
				n, what, _ := strings.Cut(strings.TrimSpace(rest), " ")
				if _, e := strconv.Atoi(n); e != nil || strings.TrimSpace(what) == "" {
					return nil
				}
				return []string{"resolve", t.ID, n, strings.TrimSpace(what)}
			}},
		{name: "attach", verb: "attach", desc: "Attach a file or folder", arg: "path", avail: always, args: text("attach")},
		{name: "detach", verb: "detach", desc: "Remove one of its files", arg: "name",
			avail: func(m model, t task.Task) bool { return len(m.filesOf(t)) > 0 },
			sub: func(m model, t task.Task, rest string) []completion {
				return m.valueRows("detach ", rest, m.filesOf(t), nil)
			},
			args: text("detach")},
		{name: "files", verb: "edit", desc: "Its files in the editor", keyTask: "E",
			avail: func(m model, t task.Task) bool { return len(m.filesOf(t)) > 0 },
			do:    func(m *model, _ task.Task, _ string) tea.Cmd { return m.openFolder() }},
		{name: "screen", verb: "attach", desc: "Copy the screen, then attach it", keyList: captureKey, keyTask: captureKey, avail: always,
			do: func(m *model, _ task.Task, _ string) tea.Cmd { m.capture(); return nil }},
		{name: "paste", verb: "attach", desc: "Paste the clipboard as a file", keyList: clipKey, keyTask: clipKey, avail: always,
			do: func(m *model, _ task.Task, _ string) tea.Cmd { return m.pasteClipboard() }},
	}
}

// PaletteVerbs are the hug task commands the palette runs: its context
// actions and rebalance (05r3pg parity with MCP task_update).
func PaletteVerbs() []string {
	out := []string{"rebalance"}
	for _, it := range contextItems {
		if !slices.Contains(out, it.verb) {
			out = append(out, it.verb)
		}
	}
	return out
}

// focusedCheck is the check marked > in the open task, when the checks
// have the focus.
func (m model) focusedCheck(t task.Task) (task.Acceptance, bool) {
	if !m.detail || m.glance || m.detailPane != focusAcceptance || len(t.Acceptance) == 0 {
		return task.Acceptance{}, false
	}
	return t.Acceptance[min(m.acceptance, len(t.Acceptance)-1)], true
}

// subject is the task the context groups act on: the open task, else the
// selected row.
func (m model) subject() (task.Task, bool) { return m.selected() }

func (it ctxItem) key(m model) string {
	if m.detail {
		return it.keyTask
	}
	return it.keyList
}

// contextItem is the available item named name.
func (m model) contextItem(name string) (ctxItem, task.Task, bool) {
	t, ok := m.subject()
	if !ok {
		return ctxItem{}, t, false
	}
	for _, it := range contextItems {
		if it.name == name && it.avail(m, t) {
			return it, t, true
		}
	}
	return ctxItem{}, t, false
}

// contextCompletions are the context rows for what is typed: the items
// starting with it, or the level under the item it names.
func (m model) contextCompletions(value string) []completion {
	t, ok := m.subject()
	if !ok {
		return nil
	}
	name, rest, hasArg := strings.Cut(value, " ")
	var out []completion
	for _, group := range []bool{true, false} {
		for _, it := range contextItems {
			if it.check != group || !it.avail(m, t) {
				continue
			}
			label := "TASK"
			if it.check {
				label = "CHECK"
			}
			switch {
			case !hasArg && strings.HasPrefix(it.name, name):
				v := it.name
				if it.arg != "" && !it.optional {
					v += " "
				}
				out = append(out, completion{value: v, description: it.desc, branch: it.sub != nil, group: label, key: it.key(m)})
			case hasArg && it.name == name && it.sub != nil:
				for _, c := range it.sub(m, t, strings.TrimLeft(rest, " ")) {
					c.group = label
					out = append(out, c)
				}
			}
		}
	}
	return out
}

// contextReady says whether the typed line is a whole context action.
func (m model) contextReady(value string) bool {
	name, rest, _ := strings.Cut(strings.TrimLeft(value, " "), " ")
	it, t, ok := m.contextItem(name)
	switch {
	case !ok:
		return false
	case it.do != nil:
		return strings.TrimSpace(rest) == "" || it.arg != ""
	case it.args != nil:
		return it.args(m, t, rest) != nil
	}
	return false
}

// runContext runs the typed context action: an action with a key as the
// key does, any other through the confirm line and Options.Run.
func (m *model) runContext(value string) tea.Cmd {
	name, rest, _ := strings.Cut(strings.TrimLeft(value, " "), " ")
	it, t, _ := m.contextItem(name)
	if it.do != nil {
		return it.do(m, t, rest)
	}
	m.askRun(it.args(*m, t, rest), t.Path)
	return nil
}

// askRun asks to run hug task args, on the task at path ("" for none).
func (m *model) askRun(args []string, path string) {
	m.confirm, m.confirmPath, m.pendingArgs = runKey, path, args
}

// shellLine is args as a shell line: hug task note ID "two words".
func shellLine(args []string) string {
	out := []string{"hug", "task"}
	for _, a := range args {
		if a == "" || strings.ContainsAny(a, " \t\n\"'$`\\") {
			a = strconv.Quote(a)
		}
		out = append(out, a)
	}
	return strings.Join(out, " ")
}

// runArgs runs the confirmed hug task command through Options.Run, the
// code the CLI and MCP run, which takes the board turn and commits
// itself; then the board is read again and the task kept in view.
func (m *model) runArgs(args []string, path string) {
	if m.opts.Run == nil {
		m.message = "not run: this hug tui has no command runner\nfix: " + shellLine(args)
		return
	}
	id := ""
	if len(args) > 1 {
		id = args[1]
	}
	said, err := m.opts.Run(args)
	if err != nil {
		m.refresh(path)
		m.message = err.Error()
		return
	}
	m.message = said
	if m.message == "" {
		m.message = "ran " + shellLine(args[:min(2, len(args))])
	}
	m.refresh(path)
	// mv renames the file: find the task by its id again.
	if _, ok := m.taskAt(path); !ok && id != "" {
		for _, t := range m.all {
			if t.ID == id {
				if m.detail {
					m.detailPath = t.Path
				}
				m.refresh(t.Path)
				break
			}
		}
	}
	m.syncDetail()
}

// valueRows are rows prefix+v for each value v starting with typed.
func (m model) valueRows(prefix, typed string, values []string, desc func(string) string) []completion {
	var out []completion
	for _, v := range values {
		if strings.HasPrefix(strings.ToLower(v), strings.ToLower(typed)) {
			c := completion{value: prefix + v}
			if desc != nil {
				c.description = desc(v)
			}
			out = append(out, c)
		}
	}
	return out
}

func (m model) groupNames() []string {
	var out []string
	for _, g := range m.values("group") {
		if g != "(root)" {
			out = append(out, g)
		}
	}
	return out
}

// peopleNames are who a question can go to: hug.toml [people], then
// every assignee on the board.
func (m model) peopleNames() []string {
	var out []string
	for _, p := range m.people {
		out = append(out, p.Short)
	}
	for _, a := range m.values("assignee") {
		if a != "(unassigned)" && !slices.Contains(out, a) {
			out = append(out, a)
		}
	}
	return out
}

func openRequests(t task.Task) []task.Request {
	var out []task.Request
	for _, r := range t.Requests {
		if r.Resolved == nil {
			out = append(out, r)
		}
	}
	return out
}

// setFields are set's fields: the chosen ones list values, the rest are
// typed text.
var setFields = []struct{ name, desc string }{
	{"title", "New title, typed"}, {"type", "feature, bug, chore, spike, docs"}, {"priority", "critical, high, medium, low"},
	{"tag", "One tag, typed"}, {"assignee", "Who it is assigned to"}, {"parent", "Its epic, or none"},
	{"state", "todo, doing, done, dropped"}, {"goal", "Where we aim, typed"}, {"purpose", "Why, typed"},
	{"principle", "What we do and do not do"}, {"details", "Facts a decision hangs on"}, {"accept", "Add an acceptance check"},
}

func (m model) setValues(field string, t task.Task) ([]string, func(string) string) {
	switch field {
	case "type":
		return task.Types, nil
	case "state":
		return task.StateList, nil
	case "priority":
		vs := []string{"critical", "high", "medium", "low"}
		for _, v := range m.values("priority") {
			if v != "(none)" && !slices.Contains(vs, v) {
				vs = append(vs, v)
			}
		}
		return vs, nil
	case "assignee":
		return m.peopleNames(), nil
	case "tag":
		return m.values("tag"), nil
	case "parent":
		vs := []string{"none"}
		titles := map[string]string{}
		for _, v := range m.values("epic") {
			id, title, _ := strings.Cut(v, " ")
			if !strings.EqualFold(id, t.ID) {
				vs = append(vs, id)
				titles[id] = title
			}
		}
		return vs, func(id string) string { return titles[id] }
	}
	return nil, nil
}

func setSub(m model, t task.Task, rest string) []completion {
	field, typed, hasValue := strings.Cut(rest, " ")
	if !hasValue {
		var out []completion
		for _, f := range setFields {
			if strings.HasPrefix(f.name, field) {
				vs, _ := m.setValues(f.name, t)
				out = append(out, completion{value: "set " + f.name + " ", description: f.desc, branch: vs != nil})
			}
		}
		return out
	}
	vs, desc := m.setValues(field, t)
	return m.valueRows("set "+field+" ", strings.TrimLeft(typed, " "), vs, desc)
}

func setArgs(m model, t task.Task, rest string) []string {
	field, value, _ := strings.Cut(strings.TrimSpace(rest), " ")
	value = strings.TrimSpace(value)
	known := false
	for _, f := range setFields {
		known = known || f.name == field
	}
	if !known || value == "" {
		return nil
	}
	switch field {
	case "type":
		if !slices.Contains(task.Types, value) {
			return nil
		}
	case "state":
		if !slices.Contains(task.StateList, value) {
			return nil
		}
	case "parent":
		value, _, _ = strings.Cut(value, " ")
	}
	return []string{"set", t.ID, "--" + field, value}
}

// moveSub offers top, up, down (in rank order, where the rows are), and
// before or after another task.
func moveSub(m model, t task.Task, rest string) []completion {
	where, typed, hasID := strings.Cut(rest, " ")
	if hasID {
		if where != "before" && where != "after" {
			return nil
		}
		var out []completion
		for _, v := range m.all {
			if v.ID != t.ID && strings.HasPrefix(v.ID, strings.ToLower(strings.TrimSpace(typed))) && v.State != "done" && v.State != "dropped" {
				out = append(out, completion{value: "move " + where + " " + v.ID, description: oneLine(v.Title)})
			}
		}
		return out
	}
	var out []completion
	for _, w := range []struct{ name, desc string }{{"top", "First in its group"}, {"up", "Above the row before it"}, {"down", "Below the row after it"}, {"before", "Before a task you name"}, {"after", "After a task you name"}} {
		if !strings.HasPrefix(w.name, where) {
			continue
		}
		v := "move " + w.name
		if (w.name == "up" || w.name == "down") && moveArgs(m, t, w.name) == nil {
			continue
		}
		branch := w.name == "before" || w.name == "after"
		if branch {
			v += " "
		}
		out = append(out, completion{value: v, description: w.desc, branch: branch})
	}
	return out
}

func moveArgs(m model, t task.Task, rest string) []string {
	where, id, _ := strings.Cut(strings.TrimSpace(rest), " ")
	id = strings.TrimSpace(id)
	switch where {
	case "top":
		return []string{"move", t.ID, "--top"}
	case "before", "after":
		if id == "" {
			return nil
		}
		return []string{"move", t.ID, "--" + where, id}
	case "up", "down":
		// The neighbour in the rows shown, in rank order: not in a sort.
		if len(m.sort) > 0 {
			return nil
		}
		rows := m.indices()
		at := m.rowOf(rows, t.Path)
		if at < 0 {
			return nil
		}
		if where == "up" && at > 0 {
			return []string{"move", t.ID, "--before", m.visible[rows[at-1]].ID}
		}
		if where == "down" && at+1 < len(rows) {
			return []string{"move", t.ID, "--after", m.visible[rows[at+1]].ID}
		}
	}
	return nil
}

func depSub(m model, t task.Task, rest string) []completion {
	how, typed, hasRef := strings.Cut(rest, " ")
	if !hasRef {
		return m.valueRows("dep ", how, []string{"add", "remove"}, func(v string) string {
			if v == "add" {
				return "A task it waits for"
			}
			return "One of its dependencies"
		})
	}
	typed = strings.TrimLeft(typed, " ")
	title := func(id string) string {
		for _, v := range m.all {
			if v.ID == id {
				return oneLine(v.Title)
			}
		}
		return ""
	}
	var ids []string
	if how == "remove" {
		for _, d := range t.Dependencies {
			ids = append(ids, d.Ref)
		}
	} else if how == "add" {
		for _, v := range m.all {
			if v.ID != t.ID && v.State != "done" && v.State != "dropped" {
				ids = append(ids, v.ID)
			}
		}
	}
	return m.valueRows("dep "+how+" ", typed, ids, title)
}

func depArgs(_ model, t task.Task, rest string) []string {
	how, ref, _ := strings.Cut(strings.TrimSpace(rest), " ")
	ref = strings.TrimSpace(ref)
	if ref == "" || (how != "add" && how != "remove") {
		return nil
	}
	return []string{"dep", t.ID, "--" + how, ref}
}

// paletteUp is the palette's line one level up (05r3pg: Esc backs out one
// level, then closes): the level being chosen goes, with what is typed in
// it, so "set type " and "set type fea" are "set ", "set " and "edi"
// are "".
func paletteUp(value string) string {
	words := strings.Fields(value)
	if !strings.HasSuffix(value, " ") && len(words) > 0 {
		words = words[:len(words)-1] // the word being typed
	}
	if len(words) > 0 {
		words = words[:len(words)-1] // the level it is typed in
	}
	if len(words) == 0 {
		return ""
	}
	return strings.Join(words, " ") + " "
}
