package tui

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/veridit/hug/huginn/internal/task"
)

// The TUI reloads when tasks change on disk (nfv18p). Every hug task command
// commits to the task branch, which rewrites one small file,
// <git-common-dir>/refs/heads/<branch>. The watch asks the OS about that
// file's folder (git replaces the file by rename, so the folder, not the
// file), about packed-refs (git gc can move the ref there) and about the
// tasks folder (hand edits that are not committed). Nothing runs while
// nothing changes. If the OS watch cannot be set up, the ref's modification
// time is checked every pollEvery instead.
var (
	debounce   = 200 * time.Millisecond
	maxDelay   = time.Second // a steady stream of events still reloads
	pollEvery  = 2 * time.Second
	newWatcher = fsnotify.NewWatcher
)

type watch struct {
	C    chan struct{} // one value per settled burst of changes
	mode string        // "watch", "watch+poll" or "poll": for the report
	stop chan struct{}
	done chan struct{}
}

// paths the watch follows for a task worktree at root.
type watchPaths struct {
	root, common, ref string // ref is the loose ref file; empty without git
}

func gitPaths(root string) watchPaths {
	// Events name paths as the OS resolves them (/private/var on macOS for
	// /var): compare resolved paths.
	if r, err := filepath.EvalSymlinks(root); err == nil {
		root = r
	}
	p := watchPaths{root: root}
	out, err := exec.Command("git", "-C", root, "rev-parse", "--git-common-dir").Output()
	if err != nil {
		return p
	}
	common := strings.TrimSpace(string(out))
	if !filepath.IsAbs(common) {
		common = filepath.Join(root, common)
	}
	head, err := exec.Command("git", "-C", root, "symbolic-ref", "-q", "HEAD").Output()
	if err != nil {
		return p
	}
	p.common = filepath.Clean(common)
	if r, err := filepath.EvalSymlinks(p.common); err == nil {
		p.common = r
	}
	p.ref = filepath.Join(p.common, filepath.FromSlash(strings.TrimSpace(string(head))))
	return p
}

func startWatch(root string) *watch {
	return startWatchPaths(gitPaths(root))
}

func startWatchPaths(p watchPaths) *watch {
	w := &watch{C: make(chan struct{}, 1), stop: make(chan struct{}), done: make(chan struct{})}
	fw, err := newWatcher()
	if err != nil {
		fw = nil
	}
	poll := fw == nil
	if fw != nil {
		if p.ref != "" {
			if e := fw.Add(filepath.Dir(p.ref)); e != nil {
				poll = true // e.g. the ref is only in packed-refs: no folder yet
			}
			if e := fw.Add(p.common); e != nil {
				poll = true
			}
		}
		for _, dir := range taskDirs(p.root) {
			if e := fw.Add(dir); e != nil && p.ref == "" {
				poll = true
			}
		}
	}
	switch {
	case fw == nil:
		w.mode = "poll"
	case poll:
		w.mode = "watch+poll"
	default:
		w.mode = "watch"
	}
	go w.run(fw, p, poll)
	return w
}

func (w *watch) close() {
	close(w.stop)
	<-w.done
}

// taskDirs is the tasks folder and its group folders, as Store.Tasks walks
// them: no dot folders, no task files folders.
func taskDirs(root string) []string {
	var out []string
	filepath.WalkDir(root, func(path string, d os.DirEntry, e error) error {
		if e != nil || !d.IsDir() {
			return nil
		}
		if path != root {
			rel, _ := filepath.Rel(root, path)
			if strings.HasPrefix(d.Name(), ".") || task.IsTaskFolder(root, filepath.ToSlash(rel)) {
				return filepath.SkipDir
			}
		}
		out = append(out, path)
		return nil
	})
	return out
}

// relevant says whether an event can change what the TUI shows.
func (p watchPaths) relevant(ev fsnotify.Event) bool {
	if ev.Op == fsnotify.Chmod {
		return false // attributes only; reading a file can do this (atime)
	}
	dir, base := filepath.Dir(ev.Name), filepath.Base(ev.Name)
	if p.ref != "" && ev.Name == p.ref {
		return true
	}
	if p.common != "" && dir == p.common {
		return base == "packed-refs"
	}
	if strings.HasPrefix(base, ".") {
		return false // temporary files (.huginn-*), the worktree's .git file
	}
	rel, err := filepath.Rel(p.root, ev.Name)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return false
	}
	if strings.HasSuffix(base, ".md") {
		return true
	}
	// A folder created, removed or renamed: a group came or went.
	return ev.Has(fsnotify.Create) || ev.Has(fsnotify.Remove) || ev.Has(fsnotify.Rename)
}

type stamp struct {
	mod  time.Time
	size int64
	ok   bool
}

func stat(path string) stamp {
	if path == "" {
		return stamp{}
	}
	fi, err := os.Stat(path)
	if err != nil {
		return stamp{}
	}
	return stamp{fi.ModTime(), fi.Size(), true}
}

func (w *watch) run(fw *fsnotify.Watcher, p watchPaths, poll bool) {
	defer close(w.done)
	var events <-chan fsnotify.Event
	var errs <-chan error
	if fw != nil {
		defer fw.Close()
		events, errs = fw.Events, fw.Errors
	}
	// Polling follows the ref and packed-refs, or the tasks folder itself
	// when there is no git: stat calls only, never a reload unless one moved.
	polled := []string{p.ref, filepath.Join(p.common, "packed-refs")}
	if p.ref == "" {
		polled = []string{p.root}
	}
	last := make([]stamp, len(polled))
	for i, path := range polled {
		last[i] = stat(path)
	}
	var tick <-chan time.Time
	if poll {
		t := time.NewTicker(pollEvery)
		defer t.Stop()
		tick = t.C
	}
	var fire <-chan time.Time
	var first time.Time
	changed := func() {
		now := time.Now()
		if fire == nil {
			first = now
		}
		wait := min(debounce, max(0, first.Add(maxDelay).Sub(now)))
		fire = time.After(wait)
	}
	for {
		select {
		case <-w.stop:
			return
		case ev, ok := <-events:
			if !ok {
				events = nil
				continue
			}
			if !p.relevant(ev) {
				continue
			}
			if ev.Has(fsnotify.Create) {
				if fi, err := os.Stat(ev.Name); err == nil && fi.IsDir() {
					for _, dir := range taskDirs(ev.Name) {
						fw.Add(dir)
					}
				}
			}
			changed()
		case _, ok := <-errs:
			if !ok {
				errs = nil
				continue
			}
			changed() // an overflow may have dropped events: reload to be sure
		case <-tick:
			for i, path := range polled {
				if now := stat(path); now != last[i] {
					last[i] = now
					changed()
				}
			}
		case <-fire:
			fire = nil
			select {
			case w.C <- struct{}{}:
			default: // a reload is already due
			}
		}
	}
}
