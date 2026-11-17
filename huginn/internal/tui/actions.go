package tui

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/veridit/hug/huginn/internal/editor"
	"github.com/veridit/hug/huginn/internal/task"
)

var actionNames = map[string]string{"t": "take", "d": "done", "x": "drop", "J": "move down", "K": "move up", " ": "toggle acceptance", "e": "edit", "n": "new"}

// dropPrompt asks why a task is dropped: the reason is required (6ephqk).
const dropPrompt = "drop: why never?"

// startAction begins the action for key k on the selected task: x first
// asks the reason, d on a todo task is refused (take before done, Jørgen
// 11:37), the rest go to requestAction's confirm.
func (m *model) startAction(k string) tea.Cmd {
	t, ok := m.selected()
	switch {
	case ok && k == "d" && t.Status == "todo":
		m.message = t.ID + " is todo: t takes it, then d marks it done"
		return nil
	case ok && k == "x" && t.Status != "dropped":
		return m.openInput(dropPrompt, "")
	}
	m.requestAction(k, "")
	return nil
}

func (m *model) requestAction(k, value string) {
	t, ok := m.selected()
	if k != "n" && !ok {
		return
	}
	if k == "J" || k == "K" {
		// J and K move past the row shown below or above, in this section
		// and after filters; a task the reader cannot see is never the
		// target (ksv323: moving past a hidden epic changed nothing visible).
		// One step needs no confirm: the opposite key undoes it (3xcesq).
		if why := m.moveRefusal(); why != "" {
			m.message = why
			return
		}
		target, why := m.moveTarget(t, k)
		if why != "" {
			m.message = why
			return
		}
		o, where := task.MoveOptions{After: target.ID}, "below"
		if k == "K" {
			o, where = task.MoveOptions{Before: target.ID}, "above"
		}
		m.editNotice = ""
		m.moveTo(t.ID, o, where, target.ID)
		return
	}
	m.confirm = k
	m.confirmPath = t.Path
	m.pendingValue = value
	if k == "e" {
		// Name the editor in the prompt; resolved once here, not on every
		// frame (2026-11-16, user: the prompt did not say which editor).
		m.pendingValue = strings.Join(editor.Command(), " ")
	}
}
func (m *model) executeAction() tea.Cmd {
	k, value, path := m.confirm, m.pendingValue, m.confirmPath
	m.confirm = ""
	m.pendingValue = ""
	m.editNotice = ""
	// Resolve again by path so confirmation never applies to another selected task.
	var t task.Task
	if k != "n" {
		found := false
		for _, v := range m.all {
			if v.Path == path {
				t = v
				found = true
				break
			}
		}
		if !found {
			m.message = "task disappeared; refresh and try again"
			return nil
		}
	}
	var r task.Result
	var err error
	action := actionNames[k]
	switch k {
	case "n":
		r, err = m.store.New(task.NewOptions{Title: value, Goal: value, Acceptance: []string{"Verify " + value}})
	case "t":
		if m.opts.Take != nil {
			r, err = m.opts.Take(t.ID)
		} else {
			r, err = m.store.Take(t.ID, m.opts.User)
		}
	case "d":
		r, err = m.store.Done(t.ID)
	case "x":
		r, err = m.store.Drop(t.ID, value)
	case " ":
		if len(t.Acceptance) == 0 {
			m.message = "no acceptance items"
			return nil
		}
		n := min(m.acceptance, len(t.Acceptance)-1) + 1
		if !t.Acceptance[n-1].Checked {
			r, err = m.store.Check(t.ID, n)
			action = "check"
		} else {
			d, e := m.store.Get(t.ID)
			err = e
			if err == nil {
				lines := strings.Split(string(d.Bytes()), "\n")
				line := d.Task.Acceptance[n-1].Line - 1
				if line >= 0 && line < len(lines) {
					lines[line] = strings.Replace(strings.Replace(lines[line], "[x]", "[ ]", 1), "[X]", "[ ]", 1)
					err = task.AtomicWrite(filepath.Join(m.store.Root, d.Task.Path), []byte(strings.Join(lines, "\n")))
				}
			}
			r = task.Result{Task: t, Changed: err == nil}
			action = "uncheck"
		}
	case "e":
		return m.runEditor(t)
	}
	m.finish(r, err, action)
	return nil
}

// runEditor opens the task file in the user's editor; editorMsg comes back
// with the path, so the result is checked for that task (rx74bp).
func (m *model) runEditor(t task.Task) tea.Cmd {
	filename, e := m.editorPath(t)
	if e != nil {
		m.message = e.Error()
		return nil
	}
	args := editor.Command()
	cmd := exec.Command(args[0], append(args[1:], filename)...)
	path := t.Path
	// Remember what was wrong before, so the notice names what this edit
	// introduced, not an old finding.
	m.editBefore = map[string]bool{}
	for _, f := range m.editFindings(path) {
		m.editBefore[f.Problem] = true
	}
	m.editing = true // a change on disk meanwhile waits for the editor to close
	return tea.ExecProcess(cmd, func(e error) tea.Msg { return editorMsg{path: path, err: e} })
}

// editFindings are the lint findings of the task file at path, closed
// task or not: an edit is judged on the file itself.
func (m model) editFindings(path string) []task.Finding {
	var out []task.Finding
	for _, f := range append(append([]task.Finding(nil), m.findings...), m.closedFindings...) {
		if f.Path == path {
			out = append(out, f)
		}
	}
	return out
}

// noticeTask is the task the edit notice is about, with the findings the
// edit introduced, while it still exists and still has one: only then does
// e reopen the editor on it.
func (m model) noticeTask() (task.Task, []task.Finding, bool) {
	if m.editNotice == "" {
		return task.Task{}, nil, false
	}
	var fs []task.Finding
	for _, f := range m.editFindings(m.editNotice) {
		if !m.editBefore[f.Problem] {
			fs = append(fs, f)
		}
	}
	if len(fs) == 0 {
		return task.Task{}, nil, false
	}
	for _, t := range m.all {
		if t.Path == m.editNotice {
			return t, fs, true
		}
	}
	return task.Task{}, nil, false
}
func (m *model) finish(r task.Result, err error, action string) {
	if err != nil {
		path := ""
		if t, ok := m.selected(); ok {
			path = t.Path
		}
		m.refresh(path)
		m.message = err.Error()
		return
	}
	m.message = action + " " + r.Task.ID
	if len(r.Warnings) > 0 {
		m.message += "\n" + strings.Join(r.Warnings, "\n")
	}
	if r.Changed && m.opts.Commit != nil && !(action == "take" && m.opts.Take != nil) {
		if e := m.opts.Commit("hug task " + action + " " + r.Task.ID); e != nil {
			m.message = e.Error()
		}
	}
	if m.detail {
		m.detailPath = r.Task.Path
	}
	m.refresh(r.Task.Path)
}

// Malformed files stay repairable, but neither absolute paths nor symlinks may escape the store.
func (m model) editorPath(t task.Task) (string, error) {
	if t.Path == "" || filepath.IsAbs(t.Path) {
		return "", fmt.Errorf("invalid task path")
	}
	root, err := filepath.EvalSymlinks(m.store.Root)
	if err != nil {
		return "", err
	}
	root, err = filepath.Abs(root)
	if err != nil {
		return "", err
	}
	path := filepath.Join(root, t.Path)
	rel, err := filepath.Rel(root, path)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("task path escapes store")
	}
	actual, err := filepath.EvalSymlinks(path)
	if err != nil {
		return "", err
	}
	if actual != path {
		return "", fmt.Errorf("symlinked task paths cannot be edited")
	}
	info, err := os.Stat(path)
	if err != nil {
		return "", err
	}
	if !info.Mode().IsRegular() {
		return "", fmt.Errorf("task is not a regular file")
	}
	return path, nil
}
