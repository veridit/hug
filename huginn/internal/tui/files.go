package tui

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/veridit/hug/huginn/internal/editor"
	"github.com/veridit/hug/huginn/internal/task"
)

// Task files in the TUI (zgr01t): a task's folder (<group>/<id>/, see
// task.FolderOf) shows as ⎘N on its row, as a Files section inside the task,
// and opens in the editor: Enter on one file, E for the whole folder. What
// the editor changes is committed when it closes, as after e.

// fileMark is the row mark for a task with files: ⎘ and the count.
const fileMark = "⎘"

// taskFiles lists the regular files in t's folder, slash separated and
// sorted; symlinks are left out (lint reports them).
func taskFiles(root string, t task.Task) []string {
	dir := filepath.Join(root, filepath.FromSlash(task.FolderOf(t)))
	var out []string
	filepath.WalkDir(dir, func(p string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || !d.Type().IsRegular() {
			return nil
		}
		rel, err := filepath.Rel(dir, p)
		if err == nil {
			out = append(out, filepath.ToSlash(rel))
		}
		return nil
	})
	sort.Strings(out)
	return out
}

// filesOf is t's file names as of the last refresh.
func (m model) filesOf(t task.Task) []string { return m.files[t.Path] }

// fileCount is the ⎘N mark for t's row, with a trailing space, or "".
func (m model) fileCount(t task.Task) string {
	if n := len(m.filesOf(t)); n > 0 {
		return fmt.Sprintf("%s%d ", fileMark, n)
	}
	return ""
}

// filesLines are the Files section's rows: name and size, > on the one
// Enter opens when the section is focused.
func (m model) filesLines(t task.Task) []string {
	var out []string
	for i, name := range m.filesOf(t) {
		pointer := " "
		if m.detailPane == focusFiles && i == m.file {
			pointer = ">"
		}
		size := ""
		if info, err := os.Stat(filepath.Join(m.store.Root, filepath.FromSlash(task.FolderOf(t)), filepath.FromSlash(name))); err == nil {
			size = m.styles.dim.Render("  " + humanBytes(info.Size()))
		}
		line := m.hangRow(pointer+" "+fileMark+" ", oneLine(name)+size)
		if pointer == ">" {
			line = m.focusRow(m.hangRow(pointer+" "+fileMark+" ", oneLine(name)))
		}
		out = append(out, line)
	}
	return out
}

func humanBytes(n int64) string {
	switch {
	case n < 1024:
		return fmt.Sprintf("%d B", n)
	case n < 1024*1024:
		return fmt.Sprintf("%.1f KiB", float64(n)/1024)
	}
	return fmt.Sprintf("%.1f MiB", float64(n)/(1024*1024))
}

// insideFolder resolves name in t's folder to an absolute path that is a
// regular file (or, for "", the folder itself) and does not leave the store
// through a symlink.
func (m model) insideFolder(t task.Task, name string) (string, error) {
	taskFile, err := m.editorPath(t)
	if err != nil {
		return "", err
	}
	dir := filepath.Join(filepath.Dir(taskFile), t.ID)
	path := dir
	if name != "" {
		path = filepath.Join(dir, filepath.FromSlash(name))
	}
	rel, err := filepath.Rel(dir, path)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("file %s escapes the task folder", name)
	}
	actual, err := filepath.EvalSymlinks(path)
	if err != nil {
		return "", err
	}
	if actual != path {
		return "", fmt.Errorf("symlinked task files cannot be edited")
	}
	return path, nil
}

// openFile opens the file marked > in the Files section in the editor.
func (m *model) openFile() tea.Cmd {
	t, ok := m.selected()
	if !ok {
		return nil
	}
	files := m.filesOf(t)
	if len(files) == 0 {
		m.message = "this task has no files: hug task attach " + t.ID + " FILE"
		return nil
	}
	name := files[min(m.file, len(files)-1)]
	return m.editFiles(t, name, "hug task edit "+t.ID+" file "+name)
}

// openFolder opens t's whole files folder in the editor (E).
func (m *model) openFolder() tea.Cmd {
	t, ok := m.selected()
	if !ok {
		return nil
	}
	if len(m.filesOf(t)) == 0 {
		m.message = "this task has no files: hug task attach " + t.ID + " FILE"
		return nil
	}
	return m.editFiles(t, "", "hug task edit "+t.ID+" files")
}

// editFiles runs the editor on a task file or folder; editorMsg brings the
// task back, re-linted and committed with message.
func (m *model) editFiles(t task.Task, name, message string) tea.Cmd {
	path, err := m.insideFolder(t, name)
	if err != nil {
		m.message = err.Error()
		return nil
	}
	args := editor.Command()
	cmd := exec.Command(args[0], append(args[1:], path)...)
	taskPath := t.Path
	m.editBefore = map[string]bool{}
	for _, f := range m.editFindings(taskPath) {
		m.editBefore[f.Problem] = true
	}
	m.editing = true
	done := editorMsg{path: taskPath, commit: message}
	m.lastExec, m.lastEdit = cmd, done
	return tea.ExecProcess(cmd, func(e error) tea.Msg { done.err = e; return done })
}
