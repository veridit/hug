package tui

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/veridit/hug/internal/sharefile"
)

// Prompt history (xfv9pw, Jørgen 2026-11-18 11:47): the / search (and the
// find in a page, as vim's / serves both) and the : palette each keep the
// entries run in them, like a shell. In an empty prompt Up recalls the
// newest, again goes further back, Down comes forward and past the newest
// is the empty prompt again; Enter runs what is shown. A recalled entry
// that is edited is typed text: Enter adds it as a new entry, and Up and
// Down go back to their other meaning there (rows, matches, completions).
// ctrl-n/p never recall: they keep picking rows and completions.
//
// The entries persist per board in Options.HistoryFile, outside the task
// branch: hug tui puts it in the repository's git dir
// (<git-common-dir>/huginn/tui-history.json), beside the task cache, so it
// is never committed and every worktree of the repository shares it.

// Which list a prompt keeps its entries in.
const (
	searchHistory  = "search"
	commandHistory = "command"
)

// historyCap is how many entries each list keeps, oldest dropped first.
const historyCap = 200

// historyFile is the file's shape: each list oldest first.
type historyFile struct {
	Version int      `json:"version"`
	Search  []string `json:"search"`
	Command []string `json:"command"`
}

// promptHistory is the lists, and where Up and Down stand in one of them.
type promptHistory struct {
	file   string
	lists  map[string][]string // oldest first
	kind   string              // the list being browsed
	at     int                 // 0 the newest; -1 not browsing
	loaded bool
}

// history is the model's prompt history, read from its file on first use.
func (m *model) history() *promptHistory {
	if m.hist == nil {
		m.hist = &promptHistory{file: m.opts.HistoryFile, at: -1}
	}
	if !m.hist.loaded {
		m.hist.lists = readHistory(m.hist.file)
		m.hist.loaded = true
	}
	return m.hist
}

// readHistory reads the lists in path; a missing or broken file is none.
func readHistory(path string) map[string][]string {
	lists := map[string][]string{}
	if path == "" {
		return lists
	}
	b, err := sharefile.ReadFile(path)
	if err != nil {
		return lists
	}
	var f historyFile
	if json.Unmarshal(b, &f) != nil {
		return lists
	}
	lists[searchHistory], lists[commandHistory] = f.Search, f.Command
	return lists
}

// add records entry as the newest in kind: blank entries and a repeat of
// the newest are not kept. With a file, the lists are read again first,
// so two hug tui on one board keep each other's entries, and written back
// in place (a temporary file renamed over it). Saving is best effort: a
// file that cannot be written leaves the entry in this session only.
func (h *promptHistory) add(kind, entry string) {
	h.at = -1
	entry = strings.TrimSpace(entry)
	if entry == "" {
		return
	}
	if h.file != "" {
		h.lists = readHistory(h.file)
	}
	list := h.lists[kind]
	if n := len(list); n > 0 && list[n-1] == entry {
		return
	}
	list = append(list, entry)
	if len(list) > historyCap {
		list = list[len(list)-historyCap:]
	}
	h.lists[kind] = list
	h.save()
}

func (h *promptHistory) save() {
	if h.file == "" {
		return
	}
	b, err := json.MarshalIndent(historyFile{Version: 1, Search: h.lists[searchHistory], Command: h.lists[commandHistory]}, "", "  ")
	if err != nil {
		return
	}
	if os.MkdirAll(filepath.Dir(h.file), 0o755) != nil {
		return
	}
	tmp, err := os.CreateTemp(filepath.Dir(h.file), ".tui-history-*")
	if err != nil {
		return
	}
	_, werr := tmp.Write(append(b, '\n'))
	cerr := tmp.Close()
	if werr != nil || cerr != nil || sharefile.Rename(tmp.Name(), h.file) != nil {
		os.Remove(tmp.Name())
	}
}

// historyKind is the list the open prompt keeps, or "" for none.
func (m model) historyKind() string {
	switch m.prompt {
	case "/", findPrompt:
		return searchHistory
	case ":":
		return commandHistory
	}
	return ""
}

// historyKey handles k in a prompt that keeps a history: Up and Down
// recall (true: k is used); Enter in a find or the palette records what
// runs and lets the prompt run it (false). The / search records when it
// is left (leaveSearch: Enter, Tab or Esc, each keeps the found task).
func (m *model) historyKey(k string) bool {
	kind := m.historyKind()
	if kind == "" {
		return false
	}
	switch {
	case k == "down" && m.prompt == "/":
		// In / search ↓ always picks a match (5cjy88, crab 09:16Z): only
		// ↑ recalls, in the empty prompt, and goes on back.
		return false
	case k == "up" || k == "down":
		return m.recall(kind, k == "up")
	case k == "enter" && m.prompt == findPrompt:
		m.history().add(kind, m.input.Value())
	case k == "enter" && m.prompt == ":" && (m.commandReady() || m.contextReady(m.input.Value())):
		// A command not whole yet is completed by Enter, not run.
		m.history().add(kind, m.input.Value())
	}
	return false
}

// recall shows the entry before (older) or after the one shown. It only
// takes Up in an empty prompt, or Up and Down on a recalled entry as it
// was recalled; anything typed keeps Up and Down for the prompt.
func (m *model) recall(kind string, older bool) bool {
	h := m.history()
	list := h.lists[kind]
	value := m.input.Value()
	browsing := h.kind == kind && h.at >= 0 && h.at < len(list) && list[len(list)-1-h.at] == value
	switch {
	case !browsing && (value != "" || !older || len(list) == 0):
		return false
	case !browsing:
		h.kind, h.at = kind, 0
	case older:
		h.at = min(h.at+1, len(list)-1)
	default:
		h.at--
	}
	shown := ""
	if h.at >= 0 {
		shown = list[len(list)-1-h.at]
	}
	m.input.SetValue(shown)
	m.input.CursorEnd()
	m.completion = 0
	switch m.prompt {
	case "/":
		m.searchTyped()
	case findPrompt:
		m.findTyped()
	}
	return true
}
