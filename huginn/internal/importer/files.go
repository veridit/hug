package importer

import (
	"bytes"
	"os"
	"path/filepath"
	"regexp"
	"sort"

	"github.com/veridit/hug/huginn/internal/task"
)

// FilesReport reports an import of per-task file folders.
type FilesReport struct {
	Copied    int      `json:"copied"`
	Unchanged int      `json:"unchanged"`
	Refused   []string `json:"refused,omitempty"`   // not text, or over the limit
	Unmatched []string `json:"unmatched,omitempty"` // folder names that are no task
	Citations int      `json:"citations_rewritten"`
	DryRun    bool     `json:"dry_run,omitempty"`
}

// ImportFiles copies a folder of per-task folders into the tasks' own file
// folders: DIR/<task>/<path> becomes the task's <path>, where <task> is an
// id, prefix or alias (TASK-1966). Citations written as CITE/<task>/<path>
// (cite is DIR as tasks write it, e.g. ".backlog/evidence") become
// file:<path> in that task and task:<task>/<path> in others. Any repo's
// layout works; nothing here is specific to Backlog.md. Re-running changes
// nothing.
func ImportFiles(dir, cite string, dst *task.Store, dryRun bool) (FilesReport, error) {
	r := FilesReport{DryRun: dryRun}
	ts, err := dst.Tasks()
	if err != nil {
		return r, err
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return r, err
	}
	for _, ent := range entries {
		if !ent.IsDir() {
			continue
		}
		t, e := task.Resolve(ts, ent.Name())
		if e != nil {
			r.Unmatched = append(r.Unmatched, ent.Name())
			continue
		}
		base := filepath.Join(dir, ent.Name())
		err = filepath.WalkDir(base, func(p string, d os.DirEntry, err error) error {
			if err != nil || d.IsDir() || d.Type()&os.ModeSymlink != 0 {
				return err
			}
			rel, _ := filepath.Rel(base, p)
			b, err := os.ReadFile(p)
			if err != nil {
				return err
			}
			if task.CheckText(rel, b) != nil {
				r.Refused = append(r.Refused, filepath.ToSlash(filepath.Join(ent.Name(), rel)))
				return nil
			}
			target := filepath.Join(dst.Root, filepath.FromSlash(task.FolderOf(t)), rel)
			if old, e := os.ReadFile(target); e == nil && bytes.Equal(old, b) {
				r.Unchanged++
				return nil
			}
			r.Copied++
			if dryRun {
				return nil
			}
			if e := os.MkdirAll(filepath.Dir(target), 0o755); e != nil {
				return e
			}
			return task.AtomicWrite(target, b)
		})
		if err != nil {
			return r, err
		}
	}
	sort.Strings(r.Unmatched)
	if cite == "" {
		return r, nil
	}
	ref := regexp.MustCompile(regexp.QuoteMeta(filepath.ToSlash(filepath.Clean(cite))) + `/([A-Za-z0-9.@_-]+)/([A-Za-z0-9_.+,=@/-]*[A-Za-z0-9_+=@/-])`)
	for _, t := range ts {
		p := filepath.Join(dst.Root, filepath.FromSlash(t.Path))
		b, e := os.ReadFile(p)
		if e != nil || !ref.Match(b) {
			continue
		}
		n := 0
		out := ref.ReplaceAllFunc(b, func(m []byte) []byte {
			sub := ref.FindSubmatch(m)
			owner, e := task.Resolve(ts, string(sub[1]))
			if e != nil {
				return m // a folder that is no task: leave the citation for lint
			}
			n++
			if owner.ID == t.ID {
				return append([]byte("file:"), sub[2]...)
			}
			return append([]byte("task:"+string(sub[1])+"/"), sub[2]...)
		})
		if n == 0 {
			continue
		}
		r.Citations += n
		if !dryRun {
			if e := task.AtomicWrite(p, out); e != nil {
				return r, e
			}
		}
	}
	return r, nil
}
