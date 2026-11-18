package gitbranch

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/veridit/hug/internal/flock"
)

// Background sync (0hkaa5). A write commits locally and nudges one detached
// syncer per branch; reads nudge it when the last pass is stale. Nothing a
// person or an agent runs waits on the network.
//
// One queue, one runner, no lost wakeups: a syncer first takes the queue
// lock (another one queued already? it will see this commit: exit), then
// the run lock (waiting for a running pass), frees the queue, waits
// Debounce to gather a burst of writes, and syncs. A nudge while it runs
// queues the next pass.
//
// The board turn: foreground writes hold it from reading a task to
// committing it, and the syncer only while it rebases, so a background
// rebase never moves files under a command that is editing them. The
// network (fetch, push) runs outside the turn.
type Background struct {
	base string // <git-common-dir>/hug/sync/<branch>
}

// State is what the last background pass found, for ls and lint.
type State struct {
	At       time.Time `json:"at"`
	Fetched  bool      `json:"fetched"`
	Pushed   bool      `json:"pushed"`
	Unpushed int       `json:"unpushed"`
	Error    string    `json:"error,omitempty"`
	Conflict []string  `json:"conflict,omitempty"`
	// Layout is set when the pass stopped because this clone's unpushed
	// commits and the remote are on different board layouts (w40cmk check
	// 9): its fix is hug upgrade, not sync.
	Layout string `json:"layout,omitempty"`
	// Skipped says why a pass pushed nothing without an error, e.g.
	// uncommitted task edits (the commit that ends them nudges again).
	Skipped string `json:"skipped,omitempty"`
}

// Background locates r's background-sync files. It reads the worktree's
// .git file instead of running git, so a nudge costs a few file reads.
func (r *Repository) Background() (Background, error) {
	common, e := commonDir(r.Dir)
	if e != nil {
		out, err := r.git("rev-parse", "--path-format=absolute", "--git-common-dir")
		if err != nil {
			return Background{}, err
		}
		common = out
	}
	return BackgroundAt(common, r.Branch), nil
}

// BackgroundAt is the background-sync files of branch ("hug/tasks") in
// the shared git directory common, for a reader that has no Repository:
// the TUI's watch asks whether the board turn is held (46ek1f).
func BackgroundAt(common, branch string) Background {
	name := strings.NewReplacer("/", "-", "\\", "-", ":", "-").Replace(branch)
	return Background{base: filepath.Join(common, "hug", "sync", name)}
}

// gitDir is the git directory of the worktree at dir, from its .git folder
// or file, without running git; linked is true for a .git file (a linked
// worktree, whose shared directory is named in commondir).
func gitDir(dir string) (gitdir string, linked bool, err error) {
	dot := filepath.Join(dir, ".git")
	st, e := os.Stat(dot)
	if e != nil {
		return "", false, e
	}
	if st.IsDir() {
		return dot, false, nil
	}
	b, e := os.ReadFile(dot)
	if e != nil {
		return "", false, e
	}
	gitdir, ok := strings.CutPrefix(strings.TrimSpace(string(b)), "gitdir: ")
	if !ok {
		return "", false, errors.New("not a gitdir file: " + dot)
	}
	if !filepath.IsAbs(gitdir) {
		gitdir = filepath.Join(dir, gitdir)
	}
	return filepath.Clean(gitdir), true, nil
}

// commonDir is dir's shared git directory, from its .git file or folder.
func commonDir(dir string) (string, error) {
	gitdir, linked, e := gitDir(dir)
	if e != nil || !linked {
		return gitdir, e
	}
	c, e := os.ReadFile(filepath.Join(gitdir, "commondir"))
	if e != nil {
		return gitdir, nil
	}
	common := strings.TrimSpace(string(c))
	if !filepath.IsAbs(common) {
		common = filepath.Join(gitdir, common)
	}
	return filepath.Clean(common), nil
}

// Turn waits for the board turn and returns its release. Writers hold it
// from reading a task to committing it; the background pass while it
// rebases. A rebase also refuses to start over uncommitted edits, so an
// editor left open (hug task edit, the TUI) is never overwritten either.
func (b Background) Turn() (func(), error) {
	if e := os.MkdirAll(filepath.Dir(b.base), 0o700); e != nil {
		return nil, e
	}
	return flock.Acquire(b.base + ".turn")
}

// TurnHeld reports a writer holding the board turn: a try, never a wait.
// An open TUI asks before it reloads (46ek1f): a write that holds it has
// changed a task file and not yet moved the branch.
func (b Background) TurnHeld() bool {
	release, ok, e := flock.TryAcquire(b.base + ".turn")
	if e != nil {
		return false
	}
	if ok {
		release()
	}
	return !ok
}

// Queued reports a syncer waiting to run, or one just started that has not
// taken its place yet (a pending marker younger than pendingFor): a nudge
// then has nothing to do.
func (b Background) Queued() bool {
	if st, e := os.Stat(b.base + ".pending"); e == nil && time.Since(st.ModTime()) < pendingFor {
		return true
	}
	return b.queueHeld()
}

// queueHeld reports a syncer holding the queue. Finding out takes the queue
// lock for a moment, and a syncer whose Run tried it in that moment took the
// probe for a queued syncer and exited without a pass, leaving its commit
// unpushed (mmm8c8, macOS CI 44227534600). So a probe holds the probe lock
// while it holds the queue, and a syncer refused the queue waits for the
// probe lock before it tries again: refused a second time, it was refused by
// a syncer, never by a probe. Lock order: probe, then queue.
func (b Background) queueHeld() bool {
	probe, e := flock.Acquire(b.base + ".probe")
	if e != nil {
		return false
	}
	defer probe()
	release, ok, e := flock.TryAcquire(b.base + ".queue")
	if e != nil || !ok {
		return e == nil
	}
	release()
	return false
}

// takeQueue takes the queue unless a syncer holds it (see queueHeld).
func (b Background) takeQueue() (func(), bool, error) {
	if release, ok, e := flock.TryAcquire(b.base + ".queue"); e != nil || ok {
		return release, ok, e
	}
	probe, e := flock.Acquire(b.base + ".probe")
	if e != nil {
		return nil, false, e
	}
	defer probe()
	return flock.TryAcquire(b.base + ".queue")
}

// Pending reports a syncer Nudge started that has not taken its place in
// the queue yet, however long its start takes: the child removes the
// marker once it holds the queue (or finds another syncer there). Queued
// stops counting it after pendingFor, so a start that died does not hold
// nudges back; a caller waiting for the pass itself (a test) waits on this.
func (b Background) Pending() bool {
	_, e := os.Stat(b.base + ".pending")
	return e == nil
}

// Busy reports a pass queued, starting or running.
func (b Background) Busy() bool {
	if b.Queued() {
		return true
	}
	return b.RunHeld()
}

// RunHeld reports a held run lock, independently of the queue. Busy includes
// the queue, so it cannot distinguish a settling child from a running pass.
func (b Background) RunHeld() bool {
	release, ok, e := flock.TryAcquire(b.base + ".run")
	if e != nil || !ok {
		return e == nil
	}
	release()
	return false
}

// pendingFor bounds how long a started syncer may take to queue; one that
// died before then no longer holds nudges back.
const pendingFor = 5 * time.Second

// Log is the file background passes write to.
func (b Background) Log() string { return b.base + ".log" }

// LogPhase records progress before a potentially long wait, not just when a
// pass finishes. Diagnostics must never change whether a sync can run.
func (b Background) LogPhase(format string, args ...any) {
	if b.base == "" {
		return
	}
	f, err := os.OpenFile(b.Log(), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return
	}
	defer f.Close()
	line := fmt.Sprintf(format, args...)
	fmt.Fprintf(f, "%s pid %d: %s\n", time.Now().UTC().Format(time.RFC3339Nano), os.Getpid(), line)
}

// Stale reports whether the last pass finished more than age ago (or never).
func (b Background) Stale(age time.Duration) bool {
	st, e := os.Stat(b.base + ".json")
	return e != nil || time.Since(st.ModTime()) > age
}

// State is the last pass's result; the zero State when there was none.
func (b Background) State() State {
	var s State
	if raw, e := os.ReadFile(b.base + ".json"); e == nil {
		_ = json.Unmarshal(raw, &s)
	}
	return s
}

// Record stores s as the last pass's result (a foreground hug task sync
// records its outcome too, so ls stops reporting what it fixed).
func (b Background) Record(s State) error {
	s.At = time.Now().UTC()
	raw, e := json.Marshal(s)
	if e != nil {
		return e
	}
	tmp := b.base + ".json." + strconv.Itoa(os.Getpid())
	if e = os.WriteFile(tmp, raw, 0o600); e != nil {
		return e
	}
	return os.Rename(tmp, b.base+".json")
}

// Nudge starts a detached syncer, argv[0] with argv[1:], unless one is
// queued already. It never waits for it: the child gets no pipes of ours
// (a caller reading our output must not wait for the child to exit), its
// own session, and a log file next to the state.
func (b Background) Nudge(dir string, argv []string, env ...string) error {
	if len(argv) == 0 || b.Queued() {
		return nil
	}
	if e := os.MkdirAll(filepath.Dir(b.base), 0o700); e != nil {
		return e
	}
	logPath := b.base + ".log"
	if st, e := os.Stat(logPath); e == nil && st.Size() > 256<<10 {
		_ = os.Rename(logPath, logPath+".old")
	}
	log, e := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if e != nil {
		return e
	}
	defer log.Close()
	if e = os.WriteFile(b.base+".pending", nil, 0o600); e != nil {
		return e
	}
	c := exec.Command(argv[0], argv[1:]...)
	c.Dir = dir
	c.Env = append(os.Environ(), env...)
	c.Stdout, c.Stderr = log, log
	detach(c)
	if e = c.Start(); e != nil {
		_ = os.Remove(b.base + ".pending")
		return e
	}
	// Reap it in long-lived callers (MCP server, TUI); a short command
	// simply exits and the child carries on.
	go func() { _ = c.Wait() }()
	return nil
}

// Settle is how a pass gathers a burst: it starts once Probe (the branch
// head) has not changed for Quiet, or after Max. Quiet 0 starts at once.
type Settle struct {
	Quiet, Max time.Duration
	Probe      func() string
}

func (s Settle) wait() {
	if s.Quiet <= 0 || s.Probe == nil {
		return
	}
	start, step := time.Now(), min(s.Quiet/4, 250*time.Millisecond)
	last, since := s.Probe(), start
	for time.Since(since) < s.Quiet && time.Since(start) < s.Max {
		time.Sleep(step)
		if h := s.Probe(); h != last {
			last, since = h, time.Now()
		}
	}
}

// Run is the child side of Nudge. It holds the queue while the burst
// settles (nudges meanwhile start nothing: this pass will see their
// commits), then waits for the run lock, frees the queue and runs one pass.
// A commit after that nudges the next syncer into the free queue, so no
// commit is left waiting. It returns nil at once when one is queued already.
// The run lock is held for the pass only: a foreground take or sync waits
// for a running pass (Exclusive), never for a burst to settle.
func (b Background) Run(settle Settle, pass func(turn func() (func(), error)) State) error {
	queue, ok, e := b.takeQueue()
	// In the queue (or another syncer is): this start is no longer pending.
	_ = os.Remove(b.base + ".pending")
	if e != nil || !ok {
		return e
	}
	b.LogPhase("queue taken; settling (quiet %s, max %s)", settle.Quiet, settle.Max)
	settle.wait()
	b.LogPhase("settled; waiting for run lock")
	run, e := flock.Acquire(b.base + ".run")
	queue()
	if e != nil {
		return e
	}
	defer run()
	b.LogPhase("run taken; queue released; starting pass")
	defer b.LogPhase("pass finished; releasing run lock")
	return b.Record(pass(b.Turn))
}

// Exclusive waits for a running background pass and keeps the next one
// from starting until release: for foreground network work (take, sync)
// that must not fetch or push at the same time. Lock order: run, then turn.
func (b Background) Exclusive() (func(), error) {
	if e := os.MkdirAll(filepath.Dir(b.base), 0o700); e != nil {
		return nil, e
	}
	return flock.Acquire(b.base + ".run")
}

var errDirty = errors.New("uncommitted task edits")

type dirtyError struct{ status string }

func (d dirtyError) Error() string   { return errDirty.Error() + ": " + d.status }
func (d dirtyError) Is(t error) bool { return t == errDirty }

// SyncPass is one background pass for r: fetch, rebase local commits under
// the turn, push the rebased commit. It never resets or drops a commit: a
// conflict aborts the rebase and is recorded, as is a failed fetch or push;
// the next pass tries again.
func (r *Repository) SyncPass(turn func() (func(), error)) (s State) {
	b, _ := r.Background()
	defer func() { s.Unpushed = r.Unpushed() }()
	for attempt := 0; attempt < 3; attempt++ {
		b.LogPhase("sync attempt %d: fetch starting", attempt+1)
		exists, e := r.fetch()
		b.LogPhase("sync attempt %d: fetch returned (failed %t)", attempt+1, e != nil)
		if e != nil {
			s.Error = firstLine(e)
			return s
		}
		s.Fetched = true
		b.LogPhase("waiting for board turn")
		release, e := turn()
		if e != nil {
			s.Error = e.Error()
			return s
		}
		b.LogPhase("board turn taken; integrating")
		head, e := r.passIntegrate(exists)
		release()
		b.LogPhase("board turn released; integrate returned (failed %t)", e != nil)
		if errors.Is(e, errDirty) {
			s.Skipped = e.Error()
			return s
		}
		var conflict *ConflictError
		if errors.As(e, &conflict) {
			s.Conflict, s.Error = conflict.Files, "rebase conflict in "+strings.Join(conflict.Files, ", ")+"; local commits are kept"
			return s
		}
		var layout *LayoutError
		if errors.As(e, &layout) {
			s.Error = firstLine(e)
			s.Layout = layout.Fix
			return s
		}
		if e != nil {
			s.Error = firstLine(e)
			return s
		}
		remote, _ := r.git("rev-parse", "--verify", "--quiet", r.remoteRef())
		if head == "" || head == remote {
			s.Error = ""
			if s.Skipped = "nothing to push"; head == "" {
				s.Skipped = "no commits"
			}
			return s
		}
		// Push the commit rebased under the turn, not HEAD: a foreground
		// command may be mid-operation by now.
		b.LogPhase("push starting")
		_, e = r.git("push", r.Remote, head+":refs/heads/"+r.Branch)
		b.LogPhase("push returned (failed %t)", e != nil)
		if e == nil {
			s.Pushed, s.Error = true, ""
			return s
		}
		s.Error = firstLine(e)
		if errors.Is(e, ErrNetTimeout) {
			return s
		}
		// Most likely another clone pushed first: fetch and rebase again.
	}
	return s
}

func (r *Repository) passIntegrate(exists bool) (string, error) {
	if e := r.checkIdle(); e != nil {
		return "", e
	}
	// Uncommitted task edits (an editor open, a command about to commit):
	// never rebase or reset under them. The commit nudges the next pass.
	if status, e := r.git("status", "--porcelain"); e != nil {
		return "", e
	} else if status != "" {
		line, _, _ := strings.Cut(strings.TrimSpace(status), "\n")
		return "", dirtyError{line}
	}
	if exists {
		if e := r.integrate(); e != nil {
			return "", e
		}
	}
	return r.Head()
}

// Unpushed counts local commits the remote does not have (one rev-list).
func (r *Repository) Unpushed() int {
	head, e := r.Head()
	if e != nil || head == "" {
		return 0
	}
	spec := r.remoteRef() + "..HEAD"
	if _, e = r.git("rev-parse", "--verify", "--quiet", r.remoteRef()); e != nil {
		spec = "HEAD"
	}
	out, e := r.git("rev-list", "--count", spec)
	if e != nil {
		return 0
	}
	n, _ := strconv.Atoi(strings.TrimSpace(out))
	return n
}

func firstLine(e error) string {
	msg := e.Error()
	var ge *Error
	if errors.As(e, &ge) {
		msg = ge.Op + ": " + ge.Detail
	}
	line, _, _ := strings.Cut(msg, "\n")
	return line
}
