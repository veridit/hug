package cli

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/veridit/hug/huginn/internal/importer"
	"github.com/veridit/hug/huginn/internal/task"
	"github.com/veridit/hug/internal/sharefile"
)

// 3mavpt: hug setup ran the whole backlog dry run on every call (34 s on
// Seid: 2,677 files converted to report "282 to update"). Now the import
// and setup's dry run remember their answer on this machine (in the git
// common dir, never on a branch), keyed on importer.ConverterVersion,
// the board layout this hug reads, and the backlog's listing (folder, name, type, size and mtime of every task file:
// lstat, no contents). While that key holds, setup reads no backlog file:
// it checks that every board task that run matched is still on the board,
// and reports the remembered numbers with their date. Anything else
// (another converter, a backlog file added, edited or removed, a matched
// task gone from the board) runs the dry run again, with progress on
// stderr. Retiring keeps its own full dry run (cleanup.go), so the
// remembered answer never decides what is deleted. A hug release that
// leaves the converter alone keeps the answer: TestConverterVersionPinned
// fails when the converter's output changes without a new version.
//
// Board edits are not in the key. After an import by this converter each
// baseline is what it generates, so a dry run merges
// base == incoming: local edits are kept and nothing is to update, which
// is what is remembered; a matched task deleted from the board fails the
// board check. Editing a task's origin: or aliases: by hand (re-linking
// it) is not seen until the backlog changes. With baselines from an older
// converter (Seid: written before the state rename), a local edit to a
// unit the dry run would re-render moves that count by one until the
// backlog or the converter changes; the detail says the numbers are remembered, and
// --cleanup runs its own dry run before anything is deleted.

// backlogCheck is one remembered import or dry run of a backlog folder.
type backlogCheck struct {
	Source    string `json:"source"`
	Key       string `json:"key"`
	At        string `json:"at"` // RFC 3339, when the run started
	Added     int    `json:"added"`
	Updated   int    `json:"updated"`
	Unchanged int    `json:"unchanged"`
	Skipped   int    `json:"skipped"`
	// FieldsOnly and Fields: the updates that only add fields (Report).
	FieldsOnly int            `json:"fields_only,omitempty"`
	Fields     map[string]int `json:"fields_added,omitempty"`
	// Board are the board task ids the run matched backlog files to: each
	// must still be on the board for the answer to hold.
	Board []string `json:"board,omitempty"`
}

func backlogCheckFile(common string) string {
	return filepath.Join(common, "huginn", "setup-backlog.json")
}

// backlogListing fingerprints the backlog's task files without reading
// them. ok is false when a folder or file cannot be listed.
func (a *App) backlogListing(src string) (key string, ok bool) {
	h := sha256.New()
	fmt.Fprintf(h, "converter %d layout %d\n", importer.ConverterVersion, a.layout())
	for _, dir := range importer.BacklogTaskDirs(src) {
		entries, e := os.ReadDir(dir)
		if e != nil {
			return "", false
		}
		fmt.Fprintf(h, "dir %s\n", dir)
		names := []string{}
		for _, d := range entries {
			if strings.HasSuffix(strings.ToLower(d.Name()), ".md") {
				names = append(names, d.Name())
			}
		}
		sort.Strings(names)
		for _, n := range names {
			fi, e := os.Lstat(filepath.Join(dir, n))
			if e != nil {
				return "", false
			}
			// The type, not the permissions: the importer reads what the
			// listing says is a regular file; a chmod changes no content.
			fmt.Fprintf(h, "%q %v %d %d\n", n, fi.Mode().Type(), fi.Size(), fi.ModTime().UnixNano())
		}
	}
	return hex.EncodeToString(h.Sum(nil)), true
}

func readBacklogCheck(path, src, key string) (backlogCheck, bool) {
	b, e := readCache(path)
	if e != nil {
		return backlogCheck{}, false
	}
	var c backlogCheck
	if json.Unmarshal(b, &c) != nil || c.Key != key || c.Source != sourceKey(src) {
		return backlogCheck{}, false
	}
	return c, true
}

// sourceKey names src the same way from setup (git toplevel) and import
// (the folder given, symlinks resolved: macOS /var is /private/var).
func sourceKey(src string) string {
	if abs, e := filepath.Abs(src); e == nil {
		src = abs
	}
	if r, e := filepath.EvalSymlinks(src); e == nil {
		src = r
	}
	return filepath.Clean(src)
}

func (a *App) writeBacklogCheck(ctx context.Context, path string, c backlogCheck) {
	if b, e := json.Marshal(c); e == nil {
		a.remember(ctx, path, b)
	}
}

// importedReport is what a dry run right after a successful import finds:
// the import wrote each result and its baseline (the generated file), so a
// second run merges base == incoming and changes nothing (importer_test.go
// checks the second run is byte-identical).
func importedReport(r importer.Report) importer.Report {
	out := r
	out.Unchanged = r.Added + r.Updated + r.Unchanged
	out.Added, out.Updated = 0, 0
	// Every task the import wrote is on the board now.
	out.Files = append([]importer.FileReport(nil), r.Files...)
	for i := range out.Files {
		if a := out.Files[i].Action; a == "added" || a == "updated" {
			out.Files[i].Action = "unchanged"
		}
	}
	return out
}

// rememberBacklog records report as the answer for src, if the backlog's
// listing is the one taken before the run (key): a file written during the
// run must not be remembered under the old listing.
func (a *App) rememberBacklog(ctx context.Context, common, src, key string, at time.Time, r importer.Report) {
	if key == "" {
		return
	}
	if again, ok := a.backlogListing(src); !ok || again != key {
		return
	}
	c := backlogCheck{Source: sourceKey(src), Key: key, At: at.UTC().Format(time.RFC3339), Added: r.Added, Updated: r.Updated, Unchanged: r.Unchanged, Skipped: r.Skipped, FieldsOnly: r.FieldsOnly, Fields: r.Fields}
	for _, f := range r.Files {
		if f.ID != "" && f.Action != "added" && f.Action != "skipped" {
			c.Board = append(c.Board, f.ID)
		}
	}
	a.writeBacklogCheck(ctx, backlogCheckFile(common), c)
}

// checkBacklog answers setup's "is the backlog imported". remembered is
// the date of the remembered run it reports, or "" for a fresh dry run.
func (a *App) checkBacklog(ctx context.Context, s *session, src string) (r importer.Report, remembered string, err error) {
	key, ok := a.backlogListing(src)
	if ok {
		if c, hit := readBacklogCheck(backlogCheckFile(s.r.Common), src, key); hit && onBoard(s.s, c.Board) {
			return importer.Report{Added: c.Added, Updated: c.Updated, Unchanged: c.Unchanged, Skipped: c.Skipped, FieldsOnly: c.FieldsOnly, Fields: c.Fields}, c.At, nil
		}
	} else {
		key = ""
	}
	var w io.Writer = a.Err
	if w == nil {
		w = io.Discard
	}
	fmt.Fprintf(w, "hug setup: comparing %s/ with the board (a dry run of the import, remembered until the backlog changes)...\n", filepath.Base(src))
	start := time.Now()
	r, err = importer.Run(ctx, src, s.s, importer.Options{DryRun: true, Progress: progressLines(w, "hug setup: ")})
	if err != nil {
		return r, "", err
	}
	a.rememberBacklog(ctx, s.r.Common, src, key, start, r)
	return r, "", nil
}

// onBoard reports whether every id is a task on the board: a task the run
// matched and that is gone since sends setup back to the dry run.
func onBoard(s *task.Store, ids []string) bool {
	ts, e := s.Tasks()
	if e != nil {
		return false
	}
	have := map[string]bool{}
	for _, t := range ts {
		have[t.ID] = true
	}
	for _, id := range ids {
		if !have[id] {
			return false
		}
	}
	return true
}

// filesCheck is setup's remembered dry run of hug task import files: what
// it counts depends on the files in DIR (copied or not) and the board's
// task files (citations to rewrite), so the key is DIR's listing and the
// board's commit, taken only while the board has no uncommitted change.
type filesCheck struct {
	Key       string `json:"key"`
	Copied    int    `json:"copied"`
	Unchanged int    `json:"unchanged"`
	Citations int    `json:"citations"`
	Unmatched int    `json:"unmatched"`
}

func filesCheckFile(common, cite string) string {
	sum := sha256.Sum256([]byte(cite))
	return filepath.Join(common, "huginn", "setup-files-"+hex.EncodeToString(sum[:6])+".json")
}

// filesKey fingerprints dir (every file's path, type, size and mtime) and
// the board in boardDir (its commit; ok is false with uncommitted changes).
func (a *App) filesKey(ctx context.Context, dir, cite, boardDir string) (string, bool) {
	h := sha256.New()
	fmt.Fprintf(h, "converter %d layout %d\ncite %s\n", importer.ConverterVersion, a.layout(), cite)
	e := filepath.WalkDir(dir, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		fi, err := d.Info()
		if err != nil {
			return err
		}
		fmt.Fprintf(h, "%q %v %d %d\n", p, fi.Mode().Type(), fi.Size(), fi.ModTime().UnixNano())
		return nil
	})
	if e != nil {
		return "", false
	}
	out, e := gitIn(ctx, boardDir, "status", "--porcelain=v2", "--branch", "--untracked-files=all")
	if e != nil {
		return "", false
	}
	oid := ""
	for _, l := range strings.Split(strings.TrimRight(out, "\n"), "\n") {
		if v, found := strings.CutPrefix(l, "# branch.oid "); found {
			oid = v
		} else if l != "" && !strings.HasPrefix(l, "# ") {
			return "", false
		}
	}
	if oid == "" || oid == "(initial)" {
		return "", false
	}
	fmt.Fprintf(h, "board %s\n", oid)
	return hex.EncodeToString(h.Sum(nil)), true
}

// checkFiles is setup's "are the task files imported": remembered while
// the key holds, else the dry run of hug task import files.
func (a *App) checkFiles(ctx context.Context, s *session, dir, cite string) (importer.FilesReport, int, error) {
	key, ok := a.filesKey(ctx, dir, cite, s.r.Dir)
	path := filesCheckFile(s.r.Common, cite)
	if ok {
		if b, e := readCache(path); e == nil {
			var c filesCheck
			if json.Unmarshal(b, &c) == nil && c.Key == key {
				return importer.FilesReport{Copied: c.Copied, Unchanged: c.Unchanged, Citations: c.Citations, DryRun: true}, c.Unmatched, nil
			}
		}
	}
	r, e := importer.ImportFiles(dir, cite, s.s, true)
	if e != nil {
		return r, 0, e
	}
	if ok {
		if again, ok2 := a.filesKey(ctx, dir, cite, s.r.Dir); ok2 && again == key {
			if b, e := json.Marshal(filesCheck{Key: key, Copied: r.Copied, Unchanged: r.Unchanged, Citations: r.Citations, Unmatched: len(r.Unmatched)}); e == nil {
				a.remember(ctx, path, b)
			}
		}
	}
	return r, len(r.Unmatched), nil
}

// writeAtomic replaces a remembered check as task files are replaced
// (557s7n): a temp file of its own (two commands may remember at once),
// then sharefile's rename, which on Windows succeeds while a reader has
// the old file open. Before, a fixed path.tmp and os.Rename lost writes
// silently (Windows CI 44172854884: "remembered files check: []").
func writeAtomic(path string, b []byte) error {
	dir := filepath.Dir(path)
	if e := os.MkdirAll(dir, 0o755); e != nil {
		return e
	}
	f, e := os.CreateTemp(dir, "."+filepath.Base(path)+".*.tmp")
	if e != nil {
		return e
	}
	tmp := f.Name()
	_, e = f.Write(b)
	if ce := f.Close(); e == nil {
		e = ce
	}
	if e == nil {
		e = sharefile.Rename(tmp, path)
	}
	if e != nil {
		os.Remove(tmp)
	}
	return e
}

// readCache reads a remembered check without keeping a writer from
// replacing it (sharefile: delete sharing on Windows).
func readCache(path string) ([]byte, error) { return sharefile.ReadFile(path) }

// memoWarning says once per command that a check could not be remembered:
// setup still answers, it is only slower next time, but never silently.
type memoWarning struct {
	w    io.Writer
	once sync.Once
}

type memoWarningKey struct{}

// withMemoWarning gives the command one warning for all its remembered
// checks.
func (a *App) withMemoWarning(ctx context.Context) context.Context {
	var w io.Writer = a.Err
	if w == nil {
		w = io.Discard
	}
	return context.WithValue(ctx, memoWarningKey{}, &memoWarning{w: w})
}

// remember writes a remembered check, and says so once if it cannot.
func (a *App) remember(ctx context.Context, path string, b []byte) {
	e := writeAtomic(path, b)
	if e == nil {
		return
	}
	m, _ := ctx.Value(memoWarningKey{}).(*memoWarning)
	if m == nil {
		m = &memoWarning{w: a.Err}
		if m.w == nil {
			m.w = io.Discard
		}
	}
	m.once.Do(func() {
		// Only this remembered check: removing it is always safe (the next
		// setup computes it again), and it clears a path left in the way.
		fix := "rm -rf " + shellQuote(path) + " && hug task setup"
		if runtime.GOOS == "windows" {
			fix = "del /q " + shellQuote(path) + " && hug task setup"
		}
		fmt.Fprintf(m.w, "hug setup: could not remember this check (%v); the next hug task setup runs it again\nfix: %s\n", e, fix)
	})
}

// shellQuote quotes s for the shell a fix: line is pasted into.
func shellQuote(s string) string {
	if runtime.GOOS == "windows" {
		return "\"" + strings.ReplaceAll(s, "\"", "\"\"") + "\""
	}
	return "'" + strings.ReplaceAll(s, "'", "'\"'\"'") + "'"
}

// layout is the board layout this hug reads (App.Layout, else task.Layout).
func (a *App) layout() int {
	if a.Layout != 0 {
		return a.Layout
	}
	return task.Layout
}

// progressLines prints an importer's progress as a line at most once a
// second, so a slow run is never silent and a fast one prints nothing.
func progressLines(w io.Writer, prefix string) func(phase string, done, total int) {
	last := time.Now()
	return func(phase string, done, total int) {
		if time.Since(last) < time.Second {
			return
		}
		last = time.Now()
		fmt.Fprintf(w, "%s%s %d of %d backlog files\n", prefix, phase, done, total)
	}
}

// backlogUpdates describes n updates for setup and --cleanup: when every
// one only adds fields, which fields, and that re-importing changes
// nothing already on the board.
func backlogUpdates(r importer.Report) string {
	if r.Updated == 0 || r.FieldsOnly != r.Updated || len(r.Fields) == 0 {
		return ""
	}
	names := make([]string, 0, len(r.Fields))
	for f := range r.Fields {
		names = append(names, f)
	}
	sort.Strings(names)
	parts := make([]string, len(names))
	for i, f := range names {
		parts[i] = fmt.Sprintf("%s: on %d", f, r.Fields[f])
	}
	return fmt.Sprintf("every one of the %d updates only adds fields the board copies lack (%s); nothing already on the board changes or goes", r.Updated, strings.Join(parts, ", "))
}
