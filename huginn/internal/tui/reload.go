package tui

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/veridit/hug/internal/env"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// Reload into a new hug (j2vwk0, Jørgen 2026-11-16: "Can the tui reload a
// new version, a bit like how jcode does?"). brew upgrade installs into a
// new Cellar folder and moves the /opt/homebrew/bin/hug symlink; go install
// renames a new file over the old. Either way the path the TUI was started
// from now leads to another file, so the watch follows that path, resolved,
// with the file's mtime and size. When it changes and holds still, the TUI
// asks the new binary for its version. Jørgen's ruling: never by itself.
// A newer hug shows in the version corner, "hug v0.7.35 → v0.7.36 · R
// reloads", and nothing else changes until R. R exists only then, and like
// every key it does nothing while a prompt, the palette, a confirm, help or
// the editor has the keys. R asks the binary again (it may have changed
// since), writes the TUI's place to a small file and execs the same path
// with the same arguments. The new hug reads the file, deletes it and opens
// where the old one was, saying "reloaded hug vX.Y.Z". A binary that does
// not start leaves this one running with a fix: line. State goes only to
// the same or a newer hug (Jørgen: no backwards compatibility), so there
// is one format and no old-format reader.
var (
	binPoll     = 2 * time.Second
	probeBinary = probeVersion
	execBinary  = reexec
)

// handoffEnv names the state file for the hug the TUI execs into.
const handoffEnv = env.TUIState

// binStamp is what the started path leads to.
type binStamp struct {
	target string
	mod    int64
	size   int64
}

func stampOf(path string) (binStamp, bool) {
	target, err := filepath.EvalSymlinks(path)
	if err != nil {
		return binStamp{}, false
	}
	fi, err := os.Stat(target)
	if err != nil || fi.IsDir() {
		return binStamp{}, false
	}
	return binStamp{target, fi.ModTime().UnixNano(), fi.Size()}, true
}

// startedPath is the path this hug was started from, as the shell found it:
// the symlink brew moves, not the Cellar file it pointed at when we started.
func startedPath() string {
	arg := os.Args[0]
	if !strings.ContainsAny(arg, `/\`) {
		if p, err := exec.LookPath(arg); err == nil {
			arg = p
		}
	}
	if p, err := filepath.Abs(arg); err == nil {
		arg = p
	}
	if _, err := os.Stat(arg); err != nil {
		if p, err := os.Executable(); err == nil {
			return p
		}
	}
	return arg
}

// binWatch signals on C when the started path leads to another file, once
// that file has held still for two polls (a copy in progress does not count).
type binWatch struct {
	path       string
	C          chan struct{}
	stop, done chan struct{}
}

func startBinWatch(path string) *binWatch {
	w := &binWatch{path: path, C: make(chan struct{}, 1), stop: make(chan struct{}), done: make(chan struct{})}
	first, _ := stampOf(path)
	go w.run(first)
	return w
}

func (w *binWatch) close() {
	close(w.stop)
	<-w.done
}

func (w *binWatch) run(last binStamp) {
	defer close(w.done)
	t := time.NewTicker(binPoll)
	defer t.Stop()
	var seen binStamp
	for {
		select {
		case <-w.stop:
			return
		case <-t.C:
			now, ok := stampOf(w.path)
			if !ok || now == last {
				seen = binStamp{}
				continue
			}
			if now != seen {
				seen = now // changed: wait one more poll for it to hold still
				continue
			}
			last, seen = now, binStamp{}
			select {
			case w.C <- struct{}{}:
			default: // a reload is already due
			}
		}
	}
}

type binaryMsg struct{}
type probedMsg struct {
	version string
	err     error
	reload  bool // R asked: reload when the answer is good
}

// waitBinary waits for the next new binary at the started path.
func (m model) waitBinary() tea.Cmd {
	if m.bin == nil {
		return nil
	}
	c := m.bin.C
	return func() tea.Msg {
		if _, ok := <-c; !ok {
			return nil
		}
		return binaryMsg{}
	}
}

// reloadKey reloads into the newer hug; bound only while one is installed.
const reloadKey = "R"

// probe asks the binary at the started path for its version, off the UI
// thread; reload says R asked for it.
func (m model) probe(reload bool) tea.Cmd {
	if m.bin == nil {
		return nil
	}
	path := m.bin.path
	return func() tea.Msg {
		v, err := probeBinary(path)
		return probedMsg{v, err, reload}
	}
}

// probed takes the binary's answer: a newer hug goes in the corner (and is
// reloaded into when R asked); a broken or older one is said with a fix:.
func (m *model) probed(msg probedMsg) tea.Cmd {
	current := m.opts.Version
	m.newer = ""
	if msg.err != nil {
		m.message = fmt.Sprintf("still on hug %s: the new hug does not start (%s) at %s\nfix: %s", current, oneLine(msg.err.Error()), m.bin.path, reinstallFix(m.bin.path))
		return nil
	}
	if older(msg.version, current) {
		m.message = fmt.Sprintf("still on hug %s: hug %s, which is older, is now at %s; not reloading\nfix: quit with q, then run hug tui", current, msg.version, m.bin.path)
		return nil
	}
	if msg.version == current && !msg.reload {
		return nil // rebuilt at the same version: nothing to offer
	}
	m.newer = msg.version
	if !msg.reload {
		return nil
	}
	m.reloadTo = msg.version
	return tea.Quit
}

// corner is the version corner: "hug v0.7.35", and while a newer hug is
// installed "hug v0.7.35 → v0.7.36 · R reloads".
func (m model) corner() string {
	if m.opts.Version == "" {
		return ""
	}
	if m.newer != "" {
		return "hug " + m.opts.Version + " → " + m.newer + " · " + reloadKey + " reloads"
	}
	return "hug " + m.opts.Version
}

// reinstallFix is the command that puts a working hug back at path.
func reinstallFix(path string) string {
	if target, err := filepath.EvalSymlinks(path); err == nil && strings.Contains(filepath.ToSlash(target), "/Cellar/") || strings.Contains(filepath.ToSlash(path), "/Cellar/") {
		return "brew reinstall veridit/tap/hug"
	}
	return "go install github.com/veridit/hug/cmd/hug@latest"
}

// probeVersion runs "path version" and returns the version it prints.
func probeVersion(path string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, path, "version")
	cmd.Env = withoutHandoff(os.Environ())
	out, err := cmd.Output()
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok && len(strings.TrimSpace(string(ee.Stderr))) > 0 {
			err = fmt.Errorf("%v: %s", err, strings.TrimSpace(strings.SplitN(string(ee.Stderr), "\n", 2)[0]))
		}
		return "", err
	}
	v, ok := strings.CutPrefix(strings.TrimSpace(string(out)), "hug ")
	if !ok || v == "" || strings.ContainsAny(v, " \t\n") {
		return "", fmt.Errorf("hug version printed %q", oneLine(string(out)))
	}
	return v, nil
}

// older reports whether version a is older than b; versions that are not
// vX.Y.Z (a dev build) are never older.
func older(a, b string) bool {
	pa, ok := semver(a)
	pb, ok2 := semver(b)
	if !ok || !ok2 {
		return false
	}
	for i := range pa {
		if pa[i] != pb[i] {
			return pa[i] < pb[i]
		}
	}
	return false
}

func semver(v string) ([3]int, bool) {
	var out [3]int
	v, ok := strings.CutPrefix(v, "v")
	parts := strings.SplitN(v, ".", 3)
	if !ok || len(parts) != 3 {
		return out, false
	}
	parts[2], _, _ = strings.Cut(parts[2], "-")
	for i, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil {
			return out, false
		}
		out[i] = n
	}
	return out, true
}

func withoutHandoff(env []string) []string {
	out := make([]string, 0, len(env)+1)
	for _, e := range env {
		if !strings.HasPrefix(e, handoffEnv+"=") {
			out = append(out, e)
		}
	}
	return out
}

// handoff is the TUI's place, carried to the hug it execs into.
type handoff struct {
	From       string      `json:"from"` // the hug that wrote it
	View       string      `json:"view"`
	Pane       int         `json:"pane"`
	Row        string      `json:"row,omitempty"`    // the selected row's task id
	Detail     string      `json:"detail,omitempty"` // the open task's id
	DetailPane int         `json:"detail_pane,omitempty"`
	Acceptance int         `json:"acceptance,omitempty"`
	Dependency int         `json:"dependency,omitempty"`
	File       int         `json:"file,omitempty"`
	Offset     int         `json:"offset,omitempty"` // the row list's scroll
	Starts     []int       `json:"starts,omitempty"` // each section's first row shown, overview and board
	Scroll     int         `json:"scroll,omitempty"` // the open task's scroll
	Chips      [][2]string `json:"chips,omitempty"`  // filters, in the order set
	Query      string      `json:"query,omitempty"`
	Results    bool        `json:"results,omitempty"` // Query is Enter's results, not a chip (5cjy88)
	Sort       []sortSpec  `json:"sort,omitempty"`
	Column     string      `json:"column,omitempty"`   // the column the cursor marks (bejd89)
	Grouping   string      `json:"grouping,omitempty"` // "group" or "epic" (kx2ndm)
	Folded     []string    `json:"folded,omitempty"`   // the folded headings, by foldKey
	// NoDefault: the list's default chip was cleared (7zg9mf); the chip
	// itself is not in Chips, the list puts it up again.
	NoDefault bool `json:"no_default,omitempty"`
}

type sortSpec struct {
	Field string `json:"field"`
	Desc  bool   `json:"desc,omitempty"`
}

// rowID is the selected row's task id, also while a task is open.
func (m model) rowID() string {
	for _, i := range m.indices() {
		if i == m.cursor && i < len(m.visible) {
			return m.visible[i].ID
		}
	}
	return ""
}

func (m model) handoff() handoff {
	h := handoff{From: m.opts.Version, View: m.view, Pane: m.pane, Row: m.rowID(), Offset: m.offset, Query: m.query, Results: m.results}
	if v := memoView(m.view); v >= 0 {
		m.remember()
		for p := range maxSections {
			h.Starts = append(h.Starts, m.memo[v][p].start)
		}
	}
	if m.detail {
		if t, ok := m.selected(); ok {
			h.Detail = t.ID
			h.DetailPane, h.Acceptance, h.Dependency, h.File = m.detailPane, m.acceptance, m.dependency, m.file
			h.Scroll = m.viewport.YOffset
		}
	}
	for _, c := range m.chips {
		if m.defaultShown && c.field == "state" {
			continue // the list's default, not the reader's chip (7zg9mf)
		}
		h.Chips = append(h.Chips, [2]string{c.field, c.value})
	}
	h.NoDefault = !m.defaultOn
	h.Grouping = m.grouping
	for k := range m.folded {
		h.Folded = append(h.Folded, k)
	}
	sort.Strings(h.Folded)
	for _, k := range m.sort {
		h.Sort = append(h.Sort, sortSpec{k.field, k.desc})
	}
	if m.column >= 0 {
		h.Column = gridColumns[m.column].field
	}
	return h
}

// resume puts the TUI where h says, as far as the tasks still allow.
func (m *model) resume(h handoff) {
	if h.NoDefault {
		m.retireDefault()
	}
	// The reader's chips first, so a state chip of theirs is there before
	// the list would put its default up (7zg9mf).
	for _, c := range h.Chips {
		m.setFilter(c[0], c[1])
	}
	switch h.View {
	case "overview", "list", "board", "lint":
		m.setView(h.View)
	}
	m.query, m.results = h.Query, h.Results && h.Query != ""
	if slices.Contains(groupings, h.Grouping) {
		m.grouping = h.Grouping
	}
	m.folded = map[string]bool{}
	for _, k := range h.Folded {
		m.folded[k] = true
	}
	m.column = -1
	for i, c := range gridColumns {
		if h.Column != "" && c.field == h.Column {
			m.column = i
		}
	}
	m.sort = nil
	for _, s := range h.Sort {
		if fieldByName(s.Field).name != "" {
			m.sort = append(m.sort, sortKey{s.Field, s.Desc})
		}
	}
	m.applyFilter()
	if (m.view == "overview" || m.view == "board") && h.Pane >= 0 && h.Pane < maxSections {
		m.pane = h.Pane
	}
	for _, i := range m.indices() {
		if m.visible[i].ID == h.Row {
			m.cursor = i
		}
	}
	m.clamp()
	m.offset = h.Offset
	if v := memoView(m.view); v >= 0 {
		for p := 0; p < len(h.Starts) && p < maxSections; p++ {
			m.memo[v][p].start = h.Starts[p]
		}
	}
	if h.Detail != "" {
		for _, t := range m.all {
			if t.ID == h.Detail {
				m.detailPath, m.detail = t.Path, true
				m.detailPane, m.acceptance, m.dependency, m.file = h.DetailPane, h.Acceptance, h.Dependency, h.File
				m.clamp()
				m.syncDetail()
				m.viewport.SetYOffset(h.Scroll)
			}
		}
	}
	m.remember()
}

// takeHandoff reads and deletes the state file the old hug left, if any.
func takeHandoff() *handoff {
	path := os.Getenv(handoffEnv)
	if path == "" {
		return nil
	}
	os.Unsetenv(handoffEnv)
	b, err := os.ReadFile(path)
	os.Remove(path)
	var h handoff
	if err != nil || json.Unmarshal(b, &h) != nil {
		return nil
	}
	return &h
}

// reloadInto writes h and execs path with this process's arguments. It
// returns only when the exec failed; the state file is then removed.
func reloadInto(path string, h handoff) error {
	b, err := json.Marshal(h)
	if err != nil {
		return err
	}
	f, err := os.CreateTemp("", "hug-tui-*.json")
	if err != nil {
		return err
	}
	_, err = f.Write(b)
	if e := f.Close(); err == nil {
		err = e
	}
	if err == nil {
		err = execBinary(path, os.Args, append(withoutHandoff(os.Environ()), handoffEnv+"="+f.Name()))
	}
	os.Remove(f.Name())
	return err
}
