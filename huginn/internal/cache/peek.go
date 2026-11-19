package cache

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"sort"

	"github.com/veridit/hug/huginn/internal/rank"
	"github.com/veridit/hug/huginn/internal/task"
	"github.com/veridit/hug/internal/sharefile"
)

// Brief is what a list of task ids needs: shell completion's answer.
type Brief struct{ ID, Title, State, Rank string }

// Peek is every task's Brief as the task files say now, read without
// writing anything (5mmqjk: shell completion asks on every TAB). Every task
// file is statted; a file whose size and time still match the cache's
// files row, outside the racy window of the cache's last sync, takes its
// id, title, state and rank from the tasks row (four columns: no record is
// decoded), and any other file is parsed. With no cache, a cache of
// another schema or one that cannot be read, every file is parsed. In
// board order.
func Peek(ctx context.Context, root, dbPath string) ([]Brief, error) {
	known, synced := peekRows(dbPath)
	disk, e := walkMarkers(ctx, root, len(known))
	if e != nil {
		return nil, e
	}
	racy := synced - int64(racyWindow)
	out := make([]Brief, 0, len(disk))
	for path, m := range disk {
		if r, ok := known[path]; ok && r.mtime == m.MtimeNS && r.size == m.Size && m.MtimeNS < racy {
			out = append(out, r.Brief)
			continue
		}
		b, e := sharefile.ReadFile(filepath.Join(root, filepath.FromSlash(path)))
		if e != nil {
			return nil, e
		}
		t := task.Parse(path, b).Task
		out = append(out, Brief{ID: t.ID, Title: t.Title, State: t.State, Rank: t.Rank})
	}
	sort.Slice(out, func(i, j int) bool { return rank.Less(out[i].Rank, out[i].ID, out[j].Rank, out[j].ID) })
	return out, nil
}

type peekRow struct {
	Brief
	mtime, size int64
}

// peekRows are the cache's rows by path and the time its last sync began;
// nil when there is no readable cache of this schema.
func peekRows(dbPath string) (map[string]peekRow, int64) {
	if _, e := os.Stat(dbPath); e != nil {
		return nil, 0
	}
	// immutable: no lock and no -shm or -wal write, which even a read-only
	// open of a WAL database does (TestCompleteNeverWrites caught both).
	// It reads the database file alone, without the WAL: rows written
	// since the last checkpoint look older or absent, and the marker check
	// in Peek parses those files instead. A read torn by a checkpoint
	// errors, which is the same as no cache.
	db, e := sql.Open("sqlite", fileURI(dbPath)+"?immutable=1")
	if e != nil {
		return nil, 0
	}
	defer db.Close()
	db.SetMaxOpenConns(1)
	var version string
	var synced int64
	if db.QueryRow(`SELECT value FROM meta WHERE key='schema_version'`).Scan(&version) != nil || version != schema {
		return nil, 0
	}
	if db.QueryRow(`SELECT CAST(value AS INTEGER) FROM meta WHERE key='synced_ns'`).Scan(&synced) != nil {
		return nil, 0
	}
	rows, e := db.Query(`SELECT f.path,f.mtime_ns,f.size,t.id,coalesce(t.title,''),coalesce(t.status,''),coalesce(t.rank,'') FROM files f JOIN tasks t ON t.path=f.path`)
	if e != nil {
		return nil, 0
	}
	defer rows.Close()
	out := map[string]peekRow{}
	for rows.Next() {
		var path string
		var r peekRow
		if rows.Scan(&path, &r.mtime, &r.size, &r.ID, &r.Title, &r.State, &r.Rank) != nil {
			return nil, 0
		}
		out[path] = r
	}
	if rows.Err() != nil {
		return nil, 0
	}
	return out, synced
}
