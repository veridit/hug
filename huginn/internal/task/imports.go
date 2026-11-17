package task

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Import is a finished move from another task system (ggydj2): its source
// folder was imported, then removed with git rm in Commit. hug.toml keeps
// one [[import]] table per move:
//
//	[[import]]
//	source = ".backlog"
//	format = "backlog"
//	commit = "1a2b3c4d..."
//	date = "2026-11-17"
//
// A later import of the same source is refused, naming Commit, where the
// old copy lives in git history.
type Import struct {
	Source string `json:"source"`
	Format string `json:"format"`
	Commit string `json:"commit"`
	Date   string `json:"date"`
}

// Imports reads every recorded import from hug.toml (none when the file is
// missing).
func Imports(root string) ([]Import, error) {
	b, e := os.ReadFile(configPath(root))
	if os.IsNotExist(e) {
		return nil, nil
	}
	if e != nil {
		return nil, e
	}
	var out []Import
	var cur *Import
	for _, line := range strings.Split(string(b), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "[") {
			cur = nil
			if line == "[[import]]" {
				out = append(out, Import{})
				cur = &out[len(out)-1]
			}
			continue
		}
		if cur == nil {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		v := scalar(value)
		switch strings.TrimSpace(key) {
		case "source":
			cur.Source = v
		case "format":
			cur.Format = v
		case "commit":
			cur.Commit = v
		case "date":
			cur.Date = v
		}
	}
	return out, nil
}

// FindImport is the recorded import of source (a path relative to the
// repository root, compared cleaned), if any.
func FindImport(root, source string) (Import, bool, error) {
	all, e := Imports(root)
	if e != nil {
		return Import{}, false, e
	}
	want := cleanSource(source)
	for _, im := range all {
		if cleanSource(im.Source) == want {
			return im, true, nil
		}
	}
	return Import{}, false, nil
}

func cleanSource(s string) string {
	return filepath.ToSlash(filepath.Clean(strings.TrimSpace(s)))
}

// RecordImport appends one [[import]] table to hug.toml, creating the file
// when there is none. Other keys and tables are left as they are. It
// returns the file's path relative to root.
func RecordImport(root string, im Import) (string, error) {
	p := configPath(root)
	b, e := os.ReadFile(p)
	if e != nil && !os.IsNotExist(e) {
		return "", e
	}
	text := string(b)
	if text != "" && !strings.HasSuffix(text, "\n") {
		text += "\n"
	}
	if text != "" {
		text += "\n"
	}
	q := func(s string) string { v, _ := json.Marshal(s); return string(v) }
	text += fmt.Sprintf("[[import]]\nsource = %s\nformat = %s\ncommit = %s\ndate = %s\n",
		q(cleanSource(im.Source)), q(im.Format), q(im.Commit), q(im.Date))
	if e = AtomicWrite(p, []byte(text)); e != nil {
		return "", e
	}
	rel, _ := filepath.Rel(root, p)
	return filepath.ToSlash(rel), nil
}
