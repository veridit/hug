package cache

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/veridit/hug/huginn/internal/id"
)

// statWorkers bounds the goroutines that stat task files: stats are
// syscalls, so a few overlap well (10,000 files: 21 ms on one, 10 ms on
// four, macOS), and the machine may be shared, so never more.
const statWorkers = 4

// walkMarkers lists every task file below root (relative, slash separated)
// with its size and modification time (44w1ev check 3, (a) and (b)). One
// ReadDir per folder: its names decide which entries are Markdown files
// and which folders are skipped (hidden folders, and a task's files folder,
// named by the id of a task file beside it), so no folder is listed twice.
// Then the Markdown files are statted (DirEntry.Info), on up to
// statWorkers goroutines.
// Same files and same rules as filepath.WalkDir with Info: symlinks are
// skipped, and any read or stat error fails the walk.
func walkMarkers(ctx context.Context, root string, hint int) (map[string]Marker, error) {
	type file struct {
		d   os.DirEntry
		rel string
	}
	files := make([]file, 0, hint)
	var list func(dir, rel string) error
	list = func(dir, rel string) error {
		if e := ctx.Err(); e != nil {
			return e
		}
		entries, e := os.ReadDir(dir)
		if e != nil {
			return e
		}
		owners := taskFileIDs(entries)
		for _, d := range entries {
			name := d.Name()
			if d.IsDir() {
				if strings.HasPrefix(name, ".") || (id.Valid(name) && owners[name]) {
					continue
				}
				if e := list(filepath.Join(dir, name), join(rel, name)); e != nil {
					return e
				}
				continue
			}
			if !strings.HasSuffix(name, ".md") || d.Type()&os.ModeSymlink != 0 {
				continue
			}
			files = append(files, file{d, join(rel, name)})
		}
		return nil
	}
	if e := list(root, ""); e != nil {
		return nil, e
	}
	markers := make([]Marker, len(files))
	errs := make([]error, statWorkers)
	var wg sync.WaitGroup
	for w := 0; w < statWorkers; w++ {
		wg.Add(1)
		go func(w int) {
			defer wg.Done()
			for i := w; i < len(files); i += statWorkers {
				// Info is the lstat on Unix; on Windows the listing
				// already holds it, and no system call is made.
				st, e := files[i].d.Info()
				if e != nil {
					errs[w] = e
					return
				}
				markers[i] = Marker{Path: files[i].rel, MtimeNS: st.ModTime().UnixNano(), Size: st.Size()}
			}
		}(w)
	}
	wg.Wait()
	for _, e := range errs {
		if e != nil {
			return nil, e
		}
	}
	disk := make(map[string]Marker, len(markers))
	for _, m := range markers {
		disk[m.Path] = m
	}
	return disk, nil
}

func join(rel, name string) string {
	if rel == "" {
		return name
	}
	return rel + "/" + name
}

// taskFileIDs are the ids that name a Markdown file in a listing, as
// task.IsTaskFolder reads them: the second dot-part of rank.id.slug.md, or
// the first of id.slug.md. A folder named by one is that task's files folder.
func taskFileIDs(entries []os.DirEntry) map[string]bool {
	ids := map[string]bool{}
	for _, d := range entries {
		n := d.Name()
		if d.IsDir() || !strings.HasSuffix(n, ".md") {
			continue
		}
		p := strings.Split(n, ".")
		// "*.<id>.*.md": any part but the last two can be followed by the id.
		for i := 0; i+2 < len(p); i++ {
			if i >= 1 {
				ids[p[i]] = true
			}
		}
		// "<id>.*.md"
		if len(p) >= 3 {
			ids[p[0]] = true
		}
	}
	return ids
}
