package store

import (
	"net/url"
	"path/filepath"
	"strings"
)

// FileURI turns a filesystem path into a SQLite file: URI that works on every
// OS. url.URL{Scheme:"file", Path:p} produced "file:C:%5CUsers..." on Windows,
// which SQLite rejects as "invalid uri authority" (Windows CI, 2026-11-11).
// SQLite wants forward slashes, a leading slash before a drive letter, and
// only '?', '#' and '%' escaped in the path.
func FileURI(path string) string {
	p := filepath.ToSlash(path)
	if len(p) >= 2 && p[1] == ':' {
		p = "/" + p
	}
	p = strings.NewReplacer("%", "%25", "?", "%3f", "#", "%23").Replace(p)
	return (&url.URL{Scheme: "file", Opaque: p}).String()
}
