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
	"sort"
	"strings"
	"time"

	"github.com/veridit/hug/huginn/internal/importer"
	"github.com/veridit/hug/huginn/internal/task"
)

// 3mavpt: hug setup ran the whole backlog dry run on every call (34 s on
// Seid: 2,677 files converted to report "282 to update"). Now the import
// and setup's dry run remember their answer on this machine (in the git
// common dir, never on a branch), keyed on the hug binary and on the
// backlog's listing (folder, name, type, size and mtime of every task file:
// lstat, no contents). While that key holds, setup reads no backlog file:
// it checks that every board task that run matched is still on the board,
// and reports the remembered numbers with their date. Anything else
// (another binary, a backlog file added, edited or removed, a matched task
// gone from the board) runs the dry run again, saying so first. Retiring keeps its own full dry run
// (cleanup.go), so the remembered answer never decides what is deleted.
//
// Board edits are not in the key. After an import by this binary each
// baseline is what this binary generates, so a dry run merges
// base == incoming: local edits are kept and nothing is to update, which
// is what is remembered; a matched task deleted from the board fails the
// board check. Editing a task's origin: or aliases: by hand (re-linking
// it) is not seen until the backlog changes. With baselines from an older binary
// (Seid: written before the state rename), a local edit to a unit the dry
// run would re-render moves that count by one until the backlog or the
// binary changes; the detail says the numbers are remembered, and
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
	fmt.Fprintf(h, "hug %s\n", a.Version)
	if exe, e := os.Executable(); e == nil {
		if fi, e := os.Stat(exe); e == nil {
			fmt.Fprintf(h, "exe %s %d %d\n", exe, fi.Size(), fi.ModTime().UnixNano())
		}
	}
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
	b, e := os.ReadFile(path)
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

func writeBacklogCheck(path string, c backlogCheck) {
	if b, e := json.Marshal(c); e == nil {
		writeAtomic(path, b)
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
func (a *App) rememberBacklog(common, src, key string, at time.Time, r importer.Report) {
	if key == "" {
		return
	}
	if again, ok := a.backlogListing(src); !ok || again != key {
		return
	}
	c := backlogCheck{Source: sourceKey(src), Key: key, At: at.UTC().Format(time.RFC3339), Added: r.Added, Updated: r.Updated, Unchanged: r.Unchanged, Skipped: r.Skipped}
	for _, f := range r.Files {
		if f.ID != "" && f.Action != "added" && f.Action != "skipped" {
			c.Board = append(c.Board, f.ID)
		}
	}
	writeBacklogCheck(backlogCheckFile(common), c)
}

// checkBacklog answers setup's "is the backlog imported". remembered is
// the date of the remembered run it reports, or "" for a fresh dry run.
func (a *App) checkBacklog(ctx context.Context, s *session, src string) (r importer.Report, remembered string, err error) {
	key, ok := a.backlogListing(src)
	if ok {
		if c, hit := readBacklogCheck(backlogCheckFile(s.r.Common), src, key); hit && onBoard(s.s, c.Board) {
			return importer.Report{Added: c.Added, Updated: c.Updated, Unchanged: c.Unchanged, Skipped: c.Skipped}, c.At, nil
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
	r, err = importer.Run(ctx, src, s.s, importer.Options{DryRun: true})
	if err != nil {
		return r, "", err
	}
	a.rememberBacklog(s.r.Common, src, key, start, r)
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
	fmt.Fprintf(h, "hug %s\ncite %s\n", a.Version, cite)
	if exe, e := os.Executable(); e == nil {
		if fi, e := os.Stat(exe); e == nil {
			fmt.Fprintf(h, "exe %s %d %d\n", exe, fi.Size(), fi.ModTime().UnixNano())
		}
	}
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
		if b, e := os.ReadFile(path); e == nil {
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
				writeAtomic(path, b)
			}
		}
	}
	return r, len(r.Unmatched), nil
}

func writeAtomic(path string, b []byte) {
	if os.MkdirAll(filepath.Dir(path), 0o755) != nil {
		return
	}
	tmp := path + ".tmp"
	if os.WriteFile(tmp, b, 0o644) != nil {
		return
	}
	if os.Rename(tmp, path) != nil {
		os.Remove(tmp)
	}
}
