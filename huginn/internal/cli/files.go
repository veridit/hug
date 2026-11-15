package cli

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/veridit/hug/huginn/internal/task"
)

type attachIn struct {
	name string
	b    []byte
}

// readAttach reads a file, or every file under a folder (keeping the
// folder's own name and layout), for attach. name renames a single file.
func readAttach(path, name string) ([]attachIn, error) {
	st, e := os.Stat(path)
	if e != nil {
		return nil, &OperationError{Problem: "cannot read " + path + ": " + e.Error(), Fix: "check the path"}
	}
	if !st.IsDir() {
		b, e := os.ReadFile(path)
		if e != nil {
			return nil, e
		}
		if name == "" {
			name = filepath.Base(path)
		}
		return []attachIn{{name, b}}, nil
	}
	base := filepath.Base(filepath.Clean(path))
	if name != "" {
		base = name
	}
	var out []attachIn
	e = filepath.WalkDir(path, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if p != path && strings.HasPrefix(d.Name(), ".") {
				return filepath.SkipDir
			}
			return nil
		}
		if d.Type()&os.ModeSymlink != 0 || strings.HasPrefix(d.Name(), ".") {
			return nil
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(path, p)
		out = append(out, attachIn{filepath.ToSlash(filepath.Join(base, rel)), b})
		return nil
	})
	if e == nil && len(out) == 0 {
		return nil, &OperationError{Problem: path + " has no files to attach", Fix: "check the path"}
	}
	return out, e
}

// secretHint mirrors muninn's write-time check: credential-shaped tokens.
var secretPattern = regexp.MustCompile(`(?:sk-(?:proj-|ant-|or-)?[A-Za-z0-9_-]{20,}|gh[pousr]_[A-Za-z0-9]{30,}|github_pat_[A-Za-z0-9_]{40,}|AKIA[0-9A-Z]{16}|xox[abprs]-[A-Za-z0-9-]{10,}|AIza[0-9A-Za-z_-]{35}|-----BEGIN [A-Z ]*PRIVATE KEY-----|(?i:api[_ -]?key|password|secret|access[_ -]?token)\s*[:=]\s*["']?[^\s"'` + "`" + `<>]{12,})`)

func secretHint(b []byte) string {
	if m := secretPattern.Find(b); m != nil {
		s := string(m)
		if len(s) > 12 {
			s = s[:6] + "..."
		}
		return "credential-shaped text near " + s
	}
	return ""
}

var _ = task.MaxFileBytes
