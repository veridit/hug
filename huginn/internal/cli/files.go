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
var secretPattern = regexp.MustCompile(`(?:sk-(?:proj-|ant-|or-)?[A-Za-z0-9_-]{20,}|gh[pousr]_[A-Za-z0-9]{30,}|github_pat_[A-Za-z0-9_]{40,}|AKIA[0-9A-Z]{16}|xox[abprs]-[A-Za-z0-9-]{10,}|AIza[0-9A-Za-z_-]{35}|-----BEGIN [A-Z ]*PRIVATE KEY-----)`)

func secretHint(b []byte) string {
	if secretPattern.Find(b) == nil && namedSecret(string(b)) {
		return "credential-shaped text (a secret-named key with a key-like value)"
	}
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

// namedSecretRe mirrors muninn's check: NAME=VALUE where the name says
// secret and the value looks like a key (12+ characters, letters and
// digits). Plain names (NOPASSWD, *_API_KEY variable names) do not count.
var namedSecretRe = regexp.MustCompile(`(?i)[a-z0-9_]*(?:secret|password|api[_ -]?key|access[_ -]?token|auth[_ -]?token|private[_ -]?key)[a-z0-9_]*\s*[:=]\s*["']?([A-Za-z0-9+/_=.-]{12,})`)

func namedSecret(text string) bool {
	for _, m := range namedSecretRe.FindAllStringSubmatch(text, -1) {
		v := m[1]
		if strings.ContainsAny(v, "0123456789") && strings.IndexFunc(v, func(r rune) bool { return r >= 'A' && r <= 'Z' || r >= 'a' && r <= 'z' }) >= 0 {
			return true
		}
	}
	return false
}
