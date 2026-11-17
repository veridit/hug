package cache

import (
	"os"
	"path/filepath"
	"strings"
	"time"
)

// schema is the SchemaVersion this binary writes (a test runs two).
var schema = SchemaVersion

// staleAfter: a cache file no hug has written or opened for this long is
// removed by the next open (gjmqhn).
const staleAfter = 7 * 24 * time.Hour

// File is the cache database for the git common dir common: one file per
// SchemaVersion, so hug binaries of different versions on one machine (a
// TUI left open beside agents on a newer release) each keep their own
// cache warm and never drop the other's tables (gjmqhn: v0.7.41 and
// v0.7.42 rebuilt a shared cache.db on every open, 14,14,12,14 in 10 s).
func File(common string) string {
	return filepath.Join(common, "huginn", "cache-"+schema+".db")
}

// removeStale deletes the cache databases in dir that no hug has used for
// staleAfter: every cache-*.db and the legacy cache.db, with their -wal and
// -shm files, judged by the newest modification time among them (SQLite
// writes the WAL on every sync; Open touches the database). keep, the file
// being opened, is never removed. Errors are ignored: a file another
// process holds open (Windows) stays until a later open.
func removeStale(dir, keep string) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	newest := map[string]time.Time{} // database name -> newest mtime of its files
	for _, d := range entries {
		name := d.Name()
		base := strings.TrimSuffix(strings.TrimSuffix(name, "-wal"), "-shm")
		if base != "cache.db" && !(strings.HasPrefix(base, "cache-") && strings.HasSuffix(base, ".db")) {
			continue
		}
		info, err := d.Info()
		if err != nil {
			continue
		}
		if t := info.ModTime(); t.After(newest[base]) {
			newest[base] = t
		}
	}
	cutoff := time.Now().Add(-staleAfter)
	for base, t := range newest {
		if base == filepath.Base(keep) || t.After(cutoff) {
			continue
		}
		for _, suffix := range []string{"", "-wal", "-shm"} {
			os.Remove(filepath.Join(dir, base+suffix))
		}
	}
}
