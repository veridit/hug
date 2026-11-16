package project

import (
	"github.com/veridit/hug/internal/env"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/veridit/hug/huginn/gitbranch"
)

// Background sync of project memories (0hkaa5): the board's protocol on the
// muninn-notes branch. A project memory remembered or forgotten nudges one
// detached `hug mem sync --background-pass`; recall and search nudge it when
// the last pass is older than FetchAfter. A pass runs SyncBranch, which
// merges the branch file into the local store and writes it back.
var (
	FetchAfter = 5 * time.Second
	Settle     = 1500 * time.Millisecond
	SettleMax  = 10 * time.Second
	// Spawn is the command that runs a pass; nil (tests, HUG_SYNC=off): none.
	Spawn []string
)

// SyncChildArg is the hidden flag of hug mem sync that runs one pass.
const SyncChildArg = "--background-pass"

// SpawnSelf makes passes run this executable, never from a test binary.
func SpawnSelf() {
	if strings.EqualFold(env.Get(env.Sync), "off") || testing.Testing() {
		return
	}
	exe, e := os.Executable()
	if e != nil {
		return
	}
	Spawn = []string{exe, "mem", "sync", SyncChildArg}
	if strings.TrimSuffix(filepath.Base(exe), ".exe") == "muninn" {
		Spawn = []string{exe, "sync", SyncChildArg}
	}
}

// background is the notes branch's sync files for dir's project, without
// opening the branch: only the shared git folder is read.
func background(dir string) (gitbranch.Background, bool) {
	r := &gitbranch.Repository{Dir: dir, Branch: NotesBranch}
	b, e := r.Background()
	return b, e == nil
}

// Nudge starts a background pass for the project at dir, unless one is
// queued or dir is not a git checkout with a remote. It never waits.
func Nudge(dir string) {
	nudge(dir, false)
}

// NudgeIfStale is Nudge when the last pass is older than FetchAfter: for
// reads (recall, search), which then wait for nothing.
func NudgeIfStale(dir string) {
	nudge(dir, true)
}

// Read reports whether a pass is a read's: nothing remembered since the
// last one, so no burst to wait for.
func readPass(b gitbranch.Background, nativePath string) bool {
	st, e := os.Stat(nativePath)
	last := b.State().At
	return e == nil && !last.IsZero() && st.ModTime().Before(last)
}

func nudge(dir string, stale bool) {
	if len(Spawn) == 0 {
		return
	}
	top, e := git(dir, "rev-parse", "--show-toplevel")
	if e != nil || top == "" {
		return
	}
	if remotes, e := git(top, "remote"); e != nil || strings.TrimSpace(remotes) == "" {
		return
	}
	b, ok := background(top)
	if !ok || (stale && !b.Stale(FetchAfter)) {
		return
	}
	_ = b.Nudge(top, Spawn)
}

// Exclusive keeps background passes for the project at dir out while a
// foreground hug mem sync runs; the release is a no-op outside git.
func Exclusive(dir string) func() {
	if b, ok := background(dir); ok {
		if release, e := b.Exclusive(); e == nil {
			return release
		}
	}
	return func() {}
}

// BackgroundPass runs one queued pass: let the burst settle, then sync. It
// returns at once when a pass is queued already.
func BackgroundPass(p Project, nativePath string, sync func() error) error {
	b, ok := background(p.Root)
	if !ok {
		return sync()
	}
	// A burst is remembers and forgets: they write the local store file.
	head := func() string {
		st, e := os.Stat(nativePath)
		if e != nil {
			return ""
		}
		return st.ModTime().String()
	}
	settle := gitbranch.Settle{Quiet: Settle, Max: SettleMax, Probe: head}
	if readPass(b, nativePath) {
		settle.Quiet = 0
	}
	return b.Run(settle, func(func() (func(), error)) gitbranch.State {
		if e := sync(); e != nil {
			line, _, _ := strings.Cut(e.Error(), "\n")
			return gitbranch.State{Error: line}
		}
		return gitbranch.State{Fetched: true, Pushed: true}
	})
}
