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

func (m *model) requestAction(k, value string) {
	t, ok := m.selected()
	if k != "n" && !ok {
		return
	}
	m.confirm = k
	m.confirmPath = t.Path
	m.pendingValue = value
}
func (m *model) executeAction() tea.Cmd {
	k, value, path := m.confirm, m.pendingValue, m.confirmPath
	m.confirm = ""
	m.pendingValue = ""
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
		r, err = m.store.Drop(t.ID)
	case "J", "K":
		i := 0
		for i < len(m.all) && m.all[i].Path != t.Path {
			i++
		}
		o := task.MoveOptions{}
		if k == "J" && i+1 < len(m.all) {
			o.After = m.all[i+1].ID
		} else if k == "K" && i > 0 {
			o.Before = m.all[i-1].ID
		} else {
			m.message = "already at rank boundary"
			return nil
		}
		r, err = m.store.Move(t.ID, o)
		action = "move"
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
		filename, e := m.editorPath(t)
		if e != nil {
			m.message = e.Error()
			return nil
		}
		args := editor.Command()
		cmd := exec.Command(args[0], append(args[1:], filename)...)
		return tea.ExecProcess(cmd, func(e error) tea.Msg { return editorMsg{e} })
	}
	m.finish(r, err, action)
	return nil
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
		if e := m.opts.Commit("huginn: " + action + " " + r.Task.ID); e != nil {
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
