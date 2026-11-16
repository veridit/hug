package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/veridit/hug/huginn/gitbranch"
)

// Background sync of the board (0hkaa5). Every write commits locally and
// nudges one detached syncer, which gathers a burst, then fetches, rebases
// and pushes. Reads nudge it when the last pass is older than FetchAfter,
// so other clones' changes arrive without anyone running sync. take keeps
// its synchronous push: it is the race lock.
var (
	// FetchAfter is how old the last pass may be before a read starts one.
	FetchAfter = 5 * time.Second
	// Settle is how long a burst of writes must be quiet before its push.
	Settle    = 1500 * time.Millisecond
	SettleMax = 10 * time.Second
	// Spawn is the command that runs a background pass, nil: none. The
	// hug binary sets it to itself (SpawnSelf); tests set it to their own
	// binary on purpose, never by default, so a test cannot fork itself.
	Spawn []string
)

// SyncChildArg is the hidden argument a background pass runs with.
const SyncChildArg = "--background-pass"

// SpawnSelf makes background passes run this executable. HUG_SYNC=off
// turns background sync off (sync then happens on hug task sync only).
func SpawnSelf() {
	if strings.EqualFold(os.Getenv("HUG_SYNC"), "off") {
		return
	}
	// A test binary given these arguments would run its whole suite again,
	// which would spawn again: tests opt in through Spawn, never through this.
	if testing.Testing() {
		return
	}
	if exe, e := os.Executable(); e == nil {
		Spawn = []string{exe, "task", "sync", SyncChildArg}
		// A huginn copy (not a link) dispatches without the "task" word.
		if strings.TrimSuffix(filepath.Base(exe), ".exe") == "huginn" {
			Spawn = []string{exe, "sync", SyncChildArg}
		}
	}
}

// background is the board's sync files, or false without a remote.
func (s *session) background() (gitbranch.Background, bool) {
	if s.r.Remote == "" {
		return gitbranch.Background{}, false
	}
	b, e := s.r.Background()
	return b, e == nil
}

// nudge starts a background pass unless one is queued. It costs a lock
// try and, when one starts, a fork: never a git call or a network wait.
// One per command: a second would find the first not yet queued.
func (s *session) nudge() {
	if len(Spawn) == 0 || s.nudged {
		return
	}
	if b, ok := s.background(); ok {
		s.nudged = b.Nudge(s.r.Root, Spawn) == nil
	}
}

// nudgeIfStale is the read side: a pass when the last one is stale.
func (s *session) nudgeIfStale() {
	if len(Spawn) == 0 || s.nudged {
		return
	}
	if b, ok := s.background(); ok && b.Stale(FetchAfter) {
		s.nudged = b.Nudge(s.r.Root, Spawn) == nil
	}
}

// turn holds the board turn for a write: a background rebase waits, so it
// never moves task files between this command reading and committing them.
func (s *session) turn() func() {
	if b, ok := s.background(); ok {
		if release, e := b.Turn(); e == nil {
			return release
		}
	}
	return func() {}
}

// commit commits the board's local edits and nudges the background push.
func (s *session) commit(message string) error {
	if e := s.r.Commit(nil, message); e != nil {
		return e
	}
	s.nudge()
	return nil
}

// syncNow is hug task sync: commit, then fetch, rebase and push in the
// foreground, after any running background pass, and record the outcome
// so ls stops reporting what it fixed.
func (s *session) syncNow() error {
	b, ok := s.background()
	if !ok {
		return s.r.Sync()
	}
	release, e := b.Exclusive()
	if e != nil {
		return s.r.Sync()
	}
	defer release()
	turn := s.turn()
	e = s.r.Sync()
	turn()
	st := gitbranch.State{Fetched: e == nil, Pushed: e == nil, Unpushed: s.r.Unpushed()}
	if e != nil {
		st.Error = strings.SplitN(e.Error(), "\n", 2)[0]
	}
	_ = b.Record(st)
	return e
}

// backgroundPass is the detached child: one pass, then exit. Its output
// goes to the log next to the state file.
func (a *App) backgroundPass(ctx context.Context) (Reply, error) {
	cwd := a.Cwd
	if cwd == "" {
		cwd = "."
	}
	r, e := gitbranch.Open(cwd, TaskBranch, TaskDir)
	if e != nil || r.Remote == "" {
		return Reply{"next": "hug task ls"}, e
	}
	b, e := r.Background()
	if e != nil {
		return nil, e
	}
	probe := func() string { h, _ := r.Head(); return h }
	// Log inside the pass, while it holds the run lock: a test waiting for
	// the pass to finish must not race this write.
	pass := func(turn func() (func(), error)) gitbranch.State {
		st := r.SyncPass(turn)
		fmt.Fprintf(a.Err, "%s pid %d: fetched=%v pushed=%v unpushed=%d error=%q\n", time.Now().UTC().Format(time.RFC3339), os.Getpid(), st.Fetched, st.Pushed, st.Unpushed, st.Error)
		return st
	}
	return Reply{"next": "hug task ls"}, b.Run(gitbranch.Settle{Quiet: Settle, Max: SettleMax, Probe: probe}, pass)
}

// syncStatus reports a sync problem from the last pass: commits not pushed,
// a conflict, or an unreachable remote, with the fix. It reads one small
// file; nothing when the last pass was clean.
func (s *session) syncStatus(reply Reply) {
	b, ok := s.background()
	if !ok {
		return
	}
	st := b.State()
	if st.Error == "" && len(st.Conflict) == 0 {
		return
	}
	// Commits made since that pass count too: one rev-list, and only here,
	// when something is wrong.
	n := s.r.Unpushed()
	if n == 0 && len(st.Conflict) == 0 {
		return // a foreground sync or take pushed everything since
	}
	msg := fmt.Sprintf("%d unpushed: %s", n, st.Error)
	if len(st.Conflict) > 0 {
		msg = fmt.Sprintf("%d unpushed, sync conflict in %s: local commits are kept, nothing was lost", n, strings.Join(st.Conflict, ", "))
	}
	reply["sync"] = Reply{"unpushed": n, "error": st.Error, "conflict": st.Conflict, "at": st.At, "message": msg, "fix": "hug task sync"}
}

// readOnly commands change no task file: they take no board turn.
var readOnly = map[string]bool{"ls": true, "show": true, "next": true, "lint": true, "board": true, "types": true, "help": true, "setup": true, "mcp": true, "upgrade": true}

// exclusive is take's lock order: a running background pass finishes, the
// next waits, then the board turn; the release frees both.
func (s *session) exclusive() func() {
	b, ok := s.background()
	if !ok {
		return func() {}
	}
	run, e := b.Exclusive()
	if e != nil {
		return func() {}
	}
	turn := s.turn()
	return func() { turn(); run() }
}
