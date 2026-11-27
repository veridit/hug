// Package cache provides the disposable, incrementally synchronized SQLite mirror.
package cache

import (
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/gob"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/veridit/hug/huginn/internal/task"
	"github.com/veridit/hug/internal/sharefile"
	_ "modernc.org/sqlite"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const SchemaVersion = "34" // 34: Task.Review, the review handed to the reviewer with its receipt, and review N receipt lines (5yh7hr); 33: Task.Approval, a reviewer's approving answer read from History (erv680); 32: a file with both Details headings reads both texts and has an old-layout finding (hxqqdp); 31: Relevant details is Details, ranked before Depends on, its text read under either name (8a0ybb; 6q1d1t: v0.7.101 and v0.7.103 shared cache-30.db and lint served the old section-order finding); 30: file refs are file: and task: links in live text only, not .hug/tasks paths or record text (7j706s); 29: an open question settles shipped-not-done, todo or doing (yf39ya check 7); 28: a file ref's last part may be a glob, and prose after a whole name is cut (ydwamb); 27: Task.Questions and Answer.QuestionN (q2adr8); 26: Outcome.N, Check, Answer and Receipt (41hhgn check 11: an answer's outcome is delivered); 25: a citation marked (lost) anywhere is not a ref, missing-task-ref fix is a note (1phmre); 24: file refs skip a file cited before its "detached file:NAME" note (bqfrda); 23: Request.Receipt, Task.AnswerCount and Task.Answers (gmzd87); 22: Task.Woken and invalid waiting_until findings (z2vmte); 21: shipped-not-done names the open check and is quiet while the stage says why (wdkqzx); 20: Task.Requests (hbgybn); 19: old-layout findings for waiting_for and waiting_show, stage asked (41hhgn); 18: Task JSON says state, not status (6ephqk step 4); 17: question findings from notes (at3qep); 16: file refs skip ' (lost)' and include .hug/tasks/ID/NAME paths (cvbpdy); 15: Task.Answer (3jtfw9); 14: table search, FTS5 (49n9kw); 13: backlog-leftover findings from the parse, set-based fixes for missing Goal and Acceptance (fkpyg3); 12: Task.Waits (16dxsr); 3: Task.Parent and Task.Sections; 4: strict Depends on lines, Finding.Fix (rx74bp); 5: not-a-dependency text (6neax1); 6: Task.Model (et85h1) (2026-11-16); 7: Task.Branch and Task.Worktree (x1t1t1); 8: board blob, board_delta and synced_ns (44w1ev: an older binary must drop them before it writes rows); 9: Task.Texts, notes, description texts and file refs kept from the parse (44w1ev); 10: notes and description texts in table texts, not in the board blob (44w1ev); 11: Task.Stage and Task.Fields (6ephqk)

type Change struct{ Status, OldPath, Path string }
type Git interface {
	Head(context.Context) (string, error)
	Diff(context.Context, string, string) ([]Change, error)
}
type Marker struct {
	Path          string
	MtimeNS, Size int64
	SHA256        string
}
type record struct {
	Marker
	Task     task.Task
	Findings []task.Finding
}
type Stats struct {
	Statted, Hashed, Parsed, Renamed, Deleted int
	Rebuilt                                   bool
	// Path is how the sync learned what changed on disk: PathWalk (every
	// task file listed and statted), PathOwnWrite (a write's own files
	// only) or PathFsmonitor (git's file system monitor named them).
	Path string
}

// The ways a sync learns what changed (44w1ev check 3).
const (
	PathWalk      = "walk"
	PathOwnWrite  = "own-write"
	PathFsmonitor = "fsmonitor"
)

type Cache struct {
	mu      sync.Mutex
	db      *sql.DB
	Root    string
	git     Git
	head    string
	files   map[string]record
	ordered []task.Task
	rebuild bool
	// synced is the wall clock when the last sync began (meta synced_ns).
	// A file modified within racyWindow of it may have changed again in
	// the same mtime tick, so it is hashed even when its marker matches.
	synced int64
	delta  int // rows in board_delta
	// rebuilt: Open found another schema version (kept after the rebuild).
	rebuilt bool
	// read and total count the files a running sync reads, for a screen
	// that waits on it (vnyz8z); atomics, as the sync holds mu.
	read, total atomic.Int64
}

// Progress is how many task files the running sync has read, of how many
// it found; total is 0 before the walk has listed the board.
func (c *Cache) Progress() (read, total int) {
	return int(c.read.Load()), int(c.total.Load())
}

// racyWindow covers coarse mtime clocks (FAT 2 s, some network shares 1 s)
// and writes that land in the same tick as a sync: git's racy-clean rule.
const racyWindow = 2 * time.Second

// Open takes the task worktree root and a database path, normally
// File(mainGitDir). No state is written to task files.
func Open(root, dbPath string, git Git) (*Cache, error) {
	absRoot, e := filepath.Abs(root)
	if e != nil {
		return nil, e
	}
	root = filepath.Clean(absRoot)
	if e := os.MkdirAll(filepath.Dir(dbPath), 0755); e != nil {
		return nil, e
	}
	// Used now: the file's time says so, as a warm open may write nothing.
	// Then caches no hug has used for a week go (gjmqhn).
	now := time.Now()
	os.Chtimes(dbPath, now, now)
	removeStale(filepath.Dir(dbPath), dbPath)
	db, e := sql.Open("sqlite", dsn(dbPath))
	if e != nil {
		return nil, e
	}
	db.SetMaxOpenConns(1)
	c := &Cache{db: db, Root: root, git: git, files: map[string]record{}}
	fail := func(e error) (*Cache, error) { db.Close(); return nil, e }
	if _, e = db.Exec(`CREATE TABLE IF NOT EXISTS meta(key TEXT PRIMARY KEY,value TEXT NOT NULL);`); e != nil {
		return fail(e)
	}
	var version string
	e = db.QueryRow(`SELECT value FROM meta WHERE key='schema_version'`).Scan(&version)
	if e != nil && e != sql.ErrNoRows {
		return fail(e)
	}
	if version != schema {
		c.rebuild, c.rebuilt = true, version != ""
		if _, e = db.Exec(`DROP TABLE IF EXISTS tasks; DROP TABLE IF EXISTS files; DROP TABLE IF EXISTS deps; DROP TABLE IF EXISTS aliases; DROP TABLE IF EXISTS board_delta; DROP TABLE IF EXISTS texts; DROP TABLE IF EXISTS search; DROP TABLE IF EXISTS search_paths; DELETE FROM meta;`); e != nil {
			return fail(e)
		}
	}
	_, e = db.Exec(`CREATE TABLE IF NOT EXISTS tasks(id TEXT NOT NULL,path TEXT PRIMARY KEY,"group" TEXT,rank TEXT,slug TEXT,title TEXT,status TEXT,type TEXT,priority TEXT,assignee TEXT,updated TEXT,payload TEXT NOT NULL,findings TEXT NOT NULL); CREATE INDEX IF NOT EXISTS tasks_id ON tasks(id); CREATE TABLE IF NOT EXISTS files(path TEXT PRIMARY KEY,mtime_ns INTEGER NOT NULL,size INTEGER NOT NULL,sha256 TEXT NOT NULL); CREATE TABLE IF NOT EXISTS deps("from" TEXT NOT NULL,"to" TEXT NOT NULL,line INTEGER NOT NULL,path TEXT NOT NULL,ref TEXT NOT NULL,ref_key TEXT NOT NULL); CREATE INDEX IF NOT EXISTS deps_ref_key ON deps(ref_key); CREATE INDEX IF NOT EXISTS deps_from ON deps("from"); CREATE INDEX IF NOT EXISTS deps_to ON deps("to"); CREATE INDEX IF NOT EXISTS deps_path ON deps(path); CREATE TABLE IF NOT EXISTS aliases(alias TEXT NOT NULL,id TEXT NOT NULL,path TEXT NOT NULL); CREATE INDEX IF NOT EXISTS aliases_alias ON aliases(alias); CREATE INDEX IF NOT EXISTS aliases_path ON aliases(path); CREATE TABLE IF NOT EXISTS board_delta(path TEXT PRIMARY KEY,value BLOB NOT NULL); CREATE TABLE IF NOT EXISTS texts(path TEXT PRIMARY KEY,sha256 TEXT NOT NULL,value BLOB NOT NULL); ` + searchSchema)
	if e != nil {
		return fail(e)
	}
	var synced string
	if e = db.QueryRow(`SELECT value FROM meta WHERE key='synced_ns'`).Scan(&synced); e != nil && e != sql.ErrNoRows {
		return fail(e)
	}
	fmt.Sscan(synced, &c.synced)
	if !c.rebuild && c.loadBoard() {
		e = db.QueryRow(`SELECT value FROM meta WHERE key='head'`).Scan(&c.head)
		if e != nil && e != sql.ErrNoRows {
			return fail(e)
		}
		return c, nil
	}
	rows, e := db.Query(`SELECT f.path,f.mtime_ns,f.size,f.sha256,t.payload,t.findings FROM files f JOIN tasks t ON t.path=f.path`)
	if e != nil {
		return fail(e)
	}
	for rows.Next() {
		var r record
		var payload, findings string
		if e = rows.Scan(&r.Path, &r.MtimeNS, &r.Size, &r.SHA256, &payload, &findings); e != nil {
			rows.Close()
			return fail(e)
		}
		if e = json.Unmarshal([]byte(payload), &r.Task); e != nil {
			rows.Close()
			return fail(e)
		}
		if e = json.Unmarshal([]byte(findings), &r.Findings); e != nil {
			rows.Close()
			return fail(e)
		}
		c.files[r.Path] = r
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		return fail(e)
	}
	e = db.QueryRow(`SELECT value FROM meta WHERE key='head'`).Scan(&c.head)
	if e != nil && e != sql.ErrNoRows {
		return fail(e)
	}
	return c, nil
}

// Concurrent writes wait, they never fail (44w1ev on m8shdj). Every hug
// command opens its own connection and syncs before it holds the board turn,
// so cache writes from parallel commands meet in SQLite. A deferred
// transaction that reads before it writes (sync's storeBoard counts
// board_delta) cannot upgrade once another connection has committed: SQLite
// returns "database is locked (5) (SQLITE_BUSY)" or (517)
// SQLITE_BUSY_SNAPSHOT at once, without waiting on busy_timeout (CI run
// 44049348743: 2 of 20 parallel notes failed). So every transaction begins
// IMMEDIATE, taking the write lock up front and waiting for it, and
// busy_timeout comes before journal_mode: pragmas run in order, and the
// first switch to WAL needs a lock too ((261) SQLITE_BUSY_RECOVERY at open).
const busyTimeout = 30 * time.Second

func dsn(path string) string {
	return fileURI(path) + "?_pragma=busy_timeout(" + strconv.Itoa(int(busyTimeout/time.Millisecond)) + ")&_pragma=journal_mode(WAL)&_pragma=synchronous(NORMAL)&_txlock=immediate"
}

// fileURI is a SQLite file: URI on every OS: forward slashes, a slash before
// a drive letter, and only '%', '?' and '#' escaped.
func fileURI(path string) string {
	p := filepath.ToSlash(path)
	if len(p) >= 2 && p[1] == ':' {
		p = "/" + p
	}
	p = strings.NewReplacer("%", "%25", "?", "%3f", "#", "%23").Replace(p)
	return (&url.URL{Scheme: "file", Opaque: p}).String()
}

// begin starts an IMMEDIATE transaction. busy_timeout is one budget per
// statement: a BEGIN that still finds the lock taken (a long cold rebuild
// elsewhere) tries again until ctx ends or busyTimeout passes once more.
func begin(ctx context.Context, db *sql.DB) (*sql.Tx, error) {
	deadline := time.Now().Add(busyTimeout)
	for {
		tx, e := db.BeginTx(ctx, nil)
		if e == nil || !busy(e) || time.Now().After(deadline) || ctx.Err() != nil {
			return tx, e
		}
		time.Sleep(20 * time.Millisecond)
	}
}
func busy(e error) bool {
	s := e.Error()
	return strings.Contains(s, "SQLITE_BUSY") || strings.Contains(s, "database is locked")
}
func (c *Cache) Close() error { c.mu.Lock(); defer c.mu.Unlock(); return c.db.Close() }

// The board blob (44w1ev): every record in one gob value (meta 'board'),
// plus a delta table of records changed since (board_delta: one row per
// path, an empty value for a path that is gone). Open decodes the blob in
// ~12 ms at 10,000 tasks (decoding the rows' JSON payloads took ~95 ms) and
// applies the delta; a write stores only its own records in the delta, so
// it never rewrites the 6 MB blob. Once the delta holds more than
// deltaLimit rows the next sync folds it into a new blob. The rows stay the
// source for SQL; blob and delta are a cache of the cache and are ignored
// when they do not match the files table's count.
type boardBlob struct {
	Schema  string
	Records []record
}

// deltaLimit: past this many delta rows a sync rewrites the blob. Decoding
// 512 single-record gobs costs about as much as one 10,000-record blob.
const deltaLimit = 512

func (c *Cache) loadBoard() bool {
	var b []byte
	if c.db.QueryRow(`SELECT value FROM meta WHERE key='board'`).Scan(&b) != nil {
		return false
	}
	var blob boardBlob
	if gob.NewDecoder(bytes.NewReader(b)).Decode(&blob) != nil || blob.Schema != schema {
		return false
	}
	files := make(map[string]record, len(blob.Records))
	for _, r := range blob.Records {
		files[r.Path] = r
	}
	rows, e := c.db.Query(`SELECT path,value FROM board_delta`)
	if e != nil {
		return false
	}
	defer rows.Close()
	n := 0
	for rows.Next() {
		var path string
		var v []byte
		if rows.Scan(&path, &v) != nil {
			return false
		}
		n++
		if len(v) == 0 {
			delete(files, path)
			continue
		}
		var r record
		if gob.NewDecoder(bytes.NewReader(v)).Decode(&r) != nil {
			return false
		}
		files[path] = r
	}
	if rows.Err() != nil {
		return false
	}
	var count int
	if c.db.QueryRow(`SELECT count(*) FROM files`).Scan(&count) != nil || count != len(files) {
		return false
	}
	c.files, c.delta = files, n
	return true
}
func encodeBoard(files map[string]record) ([]byte, error) {
	blob := boardBlob{Schema: schema, Records: make([]record, 0, len(files))}
	for _, r := range files {
		blob.Records = append(blob.Records, r)
	}
	var b bytes.Buffer
	e := gob.NewEncoder(&b).Encode(blob)
	return b.Bytes(), e
}
func encodeRecord(r record) ([]byte, error) {
	var b bytes.Buffer
	e := gob.NewEncoder(&b).Encode(r)
	return b.Bytes(), e
}

// storeBoard writes the blob or the delta for a sync that went from the
// records in old to those in next, inside tx.
func (c *Cache) storeBoard(ctx context.Context, tx *sql.Tx, next map[string]record, changed map[string]record, removed []string) (int, error) {
	exec := func(q string, args ...any) error { _, e := tx.ExecContext(ctx, q, args...); return e }
	delta := c.delta + len(changed) + len(removed)
	if c.rebuild || c.files == nil || len(c.files) == 0 || delta > deltaLimit {
		board, e := encodeBoard(next)
		if e != nil {
			return 0, e
		}
		if e = exec(`INSERT OR REPLACE INTO meta(key,value) VALUES('board',?)`, board); e != nil {
			return 0, e
		}
		return 0, exec(`DELETE FROM board_delta`)
	}
	stmt, e := tx.PrepareContext(ctx, `INSERT OR REPLACE INTO board_delta(path,value) VALUES(?,?)`)
	if e != nil {
		return 0, e
	}
	defer stmt.Close()
	for p, r := range changed {
		v, e := encodeRecord(r)
		if e != nil {
			return 0, e
		}
		if _, e = stmt.ExecContext(ctx, p, v); e != nil {
			return 0, e
		}
	}
	for _, p := range removed {
		if _, e = stmt.ExecContext(ctx, p, []byte{}); e != nil {
			return 0, e
		}
	}
	var n int
	e = tx.QueryRowContext(ctx, `SELECT count(*) FROM board_delta`).Scan(&n)
	return n, e
}
func (c *Cache) Sync(ctx context.Context) (Stats, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.sync(ctx, nil)
}

// ListAfterWrite is List after this process wrote the task files at paths
// (relative to Root; a path that is gone was removed or renamed away): it
// stats and hashes those paths only, never walks the board (44w1ev step 8:
// note and check walked 10,000 files three times, the last only to learn
// what they had just written). Every other record is kept as the last sync
// saw it, and the sync time and HEAD are not advanced, so the next full
// sync still checks every other file against the same markers and racy
// window. Call it only for files this process wrote, after a full sync in
// the same command; anything else uses List.
func (c *Cache) ListAfterWrite(ctx context.Context, paths []string) ([]task.Task, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.rebuild || len(c.files) == 0 {
		if _, e := c.sync(ctx, nil); e != nil {
			return nil, e
		}
		return c.snapshot(), nil
	}
	only := make(map[string]bool, len(paths))
	for _, p := range paths {
		only[filepath.ToSlash(filepath.Clean(p))] = true
	}
	if _, e := c.sync(ctx, only); e != nil {
		return nil, e
	}
	return c.snapshot(), nil
}
func (c *Cache) sync(ctx context.Context, only map[string]bool) (Stats, error) {
	var stats Stats
	c.read.Store(0)
	c.total.Store(0)
	stats.Rebuilt = c.rebuild
	started := time.Now().UnixNano()
	racy := c.synced - int64(racyWindow)
	head := c.head
	forced := map[string]bool{}
	renameHints := map[string]string{}
	if only != nil {
		// A write's own paths: no git, no walk. The sync time stays, so the
		// racy window of the last full sync still applies to the rest.
		started = c.synced
		disk := make(map[string]Marker, len(c.files)+len(only))
		for p, r := range c.files {
			if !only[p] {
				disk[p] = r.Marker
			}
		}
		for p := range only {
			st, e := os.Lstat(filepath.Join(c.Root, filepath.FromSlash(p)))
			if os.IsNotExist(e) {
				continue
			}
			if e != nil {
				return stats, e
			}
			if !st.Mode().IsRegular() {
				continue
			}
			stats.Statted++
			forced[p] = true
			disk[p] = Marker{Path: p, MtimeNS: st.ModTime().UnixNano(), Size: st.Size()}
		}
		stats.Path = PathOwnWrite
		return c.apply(ctx, stats, disk, forced, renameHints, head, started, racy, only)
	}
	if c.git != nil {
		var e error
		head, e = c.git.Head(ctx)
		if e != nil {
			return stats, e
		}
		if head != c.head && c.head == "" && !c.rebuild {
			for path := range c.files {
				forced[path] = true
			}
		}
		if head != c.head && c.head != "" && !c.rebuild {
			changes, e := c.git.Diff(ctx, c.head, head)
			if e != nil {
				return stats, e
			}
			for _, ch := range changes {
				forced[filepath.ToSlash(ch.Path)] = true
				if strings.HasPrefix(ch.Status, "R") {
					renameHints[filepath.ToSlash(ch.Path)] = filepath.ToSlash(ch.OldPath)
				}
			}
		}
	}
	disk, err := walkMarkers(ctx, c.Root, len(c.files))
	if err != nil {
		return stats, err
	}
	stats.Statted += len(disk)
	stats.Path = PathWalk
	return c.apply(ctx, stats, disk, forced, renameHints, head, started, racy, nil)
}

// apply brings the records in line with disk (the markers of every task
// file) and stores what changed. only, when set, names the paths a write
// touched: every other path is kept as recorded, never reread.
func (c *Cache) apply(ctx context.Context, stats Stats, disk map[string]Marker, forced map[string]bool, renameHints map[string]string, head string, started, racy int64, only map[string]bool) (Stats, error) {
	if !c.rebuild && len(disk) == len(c.files) && len(forced) == 0 {
		same := true
		for p, m := range disk {
			r, ok := c.files[p]
			if !ok || r.MtimeNS != m.MtimeNS || r.Size != m.Size || m.MtimeNS >= racy {
				same = false
				break
			}
		}
		if same && head == c.head {
			return stats, nil
		}
	}
	// Hash only changed markers (or paths named by HEAD diff). Deleted hashes
	// identify plain filesystem renames without parsing their unchanged contents.
	deletedHashes := map[string][]string{}
	for p, r := range c.files {
		if _, ok := disk[p]; !ok {
			deletedHashes[r.SHA256] = append(deletedHashes[r.SHA256], p)
		}
	}
	next := make(map[string]record, len(disk))
	changed := map[string]record{}
	bodies := map[string]*task.Texts{} // notes and descriptions of parsed files, for table texts
	c.total.Store(int64(len(disk)))
	rechecked := false
	renames := map[string]string{}
	usedOld := map[string]bool{}
	for path, m := range disk {
		old, exists := c.files[path]
		if exists && only != nil && !only[path] {
			next[path] = old
			continue
		}
		if exists && !c.rebuild && !forced[path] && old.MtimeNS == m.MtimeNS && old.Size == m.Size && m.MtimeNS < racy {
			next[path] = old
			continue
		}
		b, e := sharefile.ReadFile(filepath.Join(c.Root, path))
		if e != nil {
			return stats, e
		}
		c.read.Add(1)
		// A sync the caller gave up on (hug tui quit mid-refresh, vnyz8z)
		// stops between files: nothing is stored, the next sync redoes it.
		if e := ctx.Err(); e != nil {
			return stats, e
		}
		sum := sha256.Sum256(b)
		m.SHA256 = hex.EncodeToString(sum[:])
		stats.Hashed++
		if exists && !c.rebuild && old.SHA256 == m.SHA256 {
			if old.Marker == m {
				// A racy recheck that found the same bytes: nothing to store.
				rechecked = true
				next[path] = old
				continue
			}
			old.Marker = m
			next[path] = old
			changed[path] = old
			continue
		}
		oldPath := renameHints[path]
		if oldPath == "" && !exists && !c.rebuild {
			for _, p := range deletedHashes[m.SHA256] {
				if !usedOld[p] {
					oldPath = p
					break
				}
			}
		}
		prior, ok := c.files[oldPath]
		if ok && !usedOld[oldPath] && prior.SHA256 == m.SHA256 && !c.rebuild {
			r, i, slug, e := task.ParseName(path)
			if e == nil && i == prior.Task.ID {
				prior.Marker = m
				prior.Task.Path = path
				prior.Task.Group = filepath.ToSlash(filepath.Dir(path))
				if prior.Task.Group == "." {
					prior.Task.Group = ""
				}
				prior.Task.Rank = r
				prior.Task.Slug = slug
				prior.Findings = append([]task.Finding(nil), prior.Findings...)
				for n := range prior.Findings {
					prior.Findings[n].Path = path
				}
				next[path] = prior
				changed[path] = prior
				renames[oldPath] = path
				usedOld[oldPath] = true
				stats.Renamed++
				continue
			}
		}
		d := task.Parse(path, b)
		stats.Parsed++
		rec := record{Marker: m, Task: d.Task, Findings: d.Findings}
		bodies[path] = d.Task.Texts
		rec.Task.Texts = lean(d.Task.Texts)
		next[path] = rec
		changed[path] = rec
	}
	removed := []string{}
	for p := range c.files {
		if _, ok := next[p]; !ok {
			removed = append(removed, p)
			if _, renamed := renames[p]; !renamed {
				stats.Deleted++
			}
		}
	}
	if len(changed) == 0 && len(removed) == 0 && head == c.head && !c.rebuild {
		// Files that were racy are settled now: remember when we looked, so
		// the next sync trusts their markers.
		if rechecked {
			if _, e := c.db.ExecContext(ctx, `INSERT OR REPLACE INTO meta(key,value) VALUES('synced_ns',?)`, fmt.Sprint(started)); e != nil {
				return stats, e
			}
			c.synced = started
		}
		return stats, nil
	}
	canonical := canonicalizer(next)
	var affected map[string]string
	if len(c.files) > 0 && !c.rebuild {
		affected = affectedReferences(c.files, next, renames)
	}
	tx, e := begin(ctx, c.db)
	if e != nil {
		return stats, e
	}
	defer tx.Rollback()
	exec := func(q string, args ...any) error { _, e := tx.ExecContext(ctx, q, args...); return e }
	// Rename updates existing rows and edges rather than deleting/reinserting them.
	for old, p := range renames {
		r := next[p]
		payload, _ := json.Marshal(r.Task)
		findings, _ := json.Marshal(r.Findings)
		if e = exec(`UPDATE tasks SET path=?,"group"=?,rank=?,slug=?,payload=?,findings=? WHERE path=?`, p, r.Task.Group, r.Task.Rank, r.Task.Slug, string(payload), string(findings), old); e != nil {
			return stats, e
		}
		for _, table := range []string{"files", "deps", "aliases", "texts", "search_paths"} {
			if e = exec(`UPDATE `+table+` SET path=? WHERE path=?`, p, old); e != nil {
				return stats, e
			}
		}
	}
	for _, p := range removed {
		if _, ok := renames[p]; ok {
			continue
		}
		for _, table := range []string{"tasks", "files", "deps", "aliases", "texts"} {
			if e = exec(`DELETE FROM `+table+` WHERE path=?`, p); e != nil {
				return stats, e
			}
		}
		if e = unindexSearch(exec, p); e != nil {
			return stats, e
		}
	}
	taskStmt, e := tx.PrepareContext(ctx, `INSERT OR REPLACE INTO tasks(id,path,"group",rank,slug,title,status,type,priority,assignee,updated,payload,findings) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?)`)
	if e != nil {
		return stats, e
	}
	defer taskStmt.Close()
	fileStmt, e := tx.PrepareContext(ctx, `INSERT INTO files(path,mtime_ns,size,sha256) VALUES(?,?,?,?) ON CONFLICT(path) DO UPDATE SET mtime_ns=excluded.mtime_ns,size=excluded.size,sha256=excluded.sha256`)
	if e != nil {
		return stats, e
	}
	defer fileStmt.Close()
	renamedNew := map[string]bool{}
	for _, p := range renames {
		renamedNew[p] = true
	}
	for p, r := range changed {
		if _, e = fileStmt.ExecContext(ctx, p, r.MtimeNS, r.Size, r.SHA256); e != nil {
			return stats, e
		}
		old, existed := c.files[p]
		if renamedNew[p] || (existed && old.SHA256 == r.SHA256 && !c.rebuild) {
			continue
		}
		payload, e := json.Marshal(r.Task)
		if e != nil {
			return stats, e
		}
		findings, e := json.Marshal(r.Findings)
		if e != nil {
			return stats, e
		}
		t := r.Task
		assignee, _ := json.Marshal(t.Assignee)
		if _, e = taskStmt.ExecContext(ctx, t.ID, p, t.Group, t.Rank, t.Slug, t.Title, t.State, t.Type, t.Priority, string(assignee), t.Updated, string(payload), string(findings)); e != nil {
			return stats, e
		}
		if texts := encodeBodies(bodies[p]); texts != nil {
			e = exec(`INSERT OR REPLACE INTO texts(path,sha256,value) VALUES(?,?,?)`, p, r.SHA256, texts)
		} else {
			e = exec(`DELETE FROM texts WHERE path=?`, p)
		}
		if e != nil {
			return stats, e
		}
		// The search index follows the same rows (49n9kw): one per task file.
		if e = indexSearch(exec, p, t, bodies[p]); e != nil {
			return stats, e
		}
		for _, table := range []string{"deps", "aliases"} {
			if e = exec(`DELETE FROM `+table+` WHERE path=?`, p); e != nil {
				return stats, e
			}
		}
		for _, d := range t.Dependencies {
			to, _ := canonical(d.Ref)
			if e = exec(`INSERT INTO deps("from","to",line,path,ref,ref_key) VALUES(?,?,?,?,?,?)`, t.ID, to, d.Line, p, d.Ref, strings.ToLower(d.Ref)); e != nil {
				return stats, e
			}
		}
		for _, a := range t.Aliases {
			if e = exec(`INSERT INTO aliases(alias,id,path) VALUES(?,?,?)`, a, t.ID, p); e != nil {
				return stats, e
			}
		}
	}
	for key, ref := range affected {
		to, resolved := canonical(ref)
		if resolved {
			e = exec(`UPDATE deps SET "to"=? WHERE ref_key=? AND "to"<>?`, to, key, to)
		} else {
			e = exec(`UPDATE deps SET "to"=ref WHERE ref_key=? AND "to"<>ref`, key)
		}
		if e != nil {
			return stats, e
		}
	}
	delta, e := c.storeBoard(ctx, tx, next, changed, removed)
	if e != nil {
		return stats, e
	}
	if e = exec(`INSERT OR REPLACE INTO meta(key,value) VALUES('schema_version',?),('head',?),('synced_ns',?)`, schema, head, fmt.Sprint(started)); e != nil {
		return stats, e
	}
	if e = tx.Commit(); e != nil {
		return stats, e
	}
	c.files = next
	c.ordered = nil
	c.head = head
	c.rebuild = false
	c.synced = started
	c.delta = delta
	return stats, nil
}

// lean is what a record keeps of a parse's texts: the references lint
// checks. The notes and description bodies go to table texts (44w1ev:
// at 10,000 tasks they were 14 of the board blob's 21.6 MB, decoded by every
// command though only ls prints them).
func lean(x *task.Texts) *task.Texts {
	if x == nil {
		return nil
	}
	return &task.Texts{Refs: x.Refs}
}

// encodeBodies stores a parse's notes and description texts (in
// task.DescriptionSections order) as length-prefixed strings: "N:" then N
// bytes, each. Decoding 10,000 of them is a slice walk; one gob value per
// task cost ls 170 ms at 10,000 tasks in type headers alone (44w1ev).
func encodeBodies(x *task.Texts) []byte {
	if x == nil || !x.Bodies {
		return nil
	}
	var b []byte
	add := func(v string) { b = append(strconv.AppendInt(b, int64(len(v)), 10), ':'); b = append(b, v...) }
	add(x.Notes)
	for _, sec := range task.DescriptionSections {
		add(x.Description[sec])
	}
	return b
}
func decodeBodies(b []byte) (*task.Texts, error) {
	next := func() (string, error) {
		i := bytes.IndexByte(b, ':')
		n, e := strconv.Atoi(string(b[:max(i, 0)]))
		if i < 0 || e != nil || n < 0 || n > len(b)-i-1 {
			return "", fmt.Errorf("cache: bad texts value")
		}
		v := string(b[i+1 : i+1+n])
		b = b[i+1+n:]
		return v, nil
	}
	x := &task.Texts{Bodies: true, Description: make(map[string]string, len(task.DescriptionSections))}
	var e error
	if x.Notes, e = next(); e != nil {
		return nil, e
	}
	for _, sec := range task.DescriptionSections {
		if x.Description[sec], e = next(); e != nil {
			return nil, e
		}
	}
	return x, nil
}

// FillTexts gives each task in ts the notes and description texts of the
// parse the cache holds for its path, in one query (44w1ev: ls prints them
// for every task; other commands read one task file instead). Tasks whose
// path is not cached, or whose file changed since, keep what they have.
func (c *Cache) FillTexts(ctx context.Context, ts []task.Task) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	want := make(map[string]int, len(ts))
	for i, t := range ts {
		if t.Texts == nil || !t.Texts.Bodies {
			want[t.Path] = i
		}
	}
	if len(want) == 0 {
		return nil
	}
	rows, e := c.db.QueryContext(ctx, `SELECT path,sha256,value FROM texts`)
	if e != nil {
		return e
	}
	defer rows.Close()
	for rows.Next() {
		var path, sum string
		var v []byte
		if e = rows.Scan(&path, &sum, &v); e != nil {
			return e
		}
		i, ok := want[path]
		if !ok || c.files[path].SHA256 != sum {
			continue
		}
		x, e := decodeBodies(v)
		if e != nil {
			// A value this binary cannot read: the task reads its file.
			continue
		}
		if old := ts[i].Texts; old != nil {
			x.Refs = old.Refs
		}
		ts[i].Texts = x
	}
	return rows.Err()
}

// List synchronizes first, then returns a detached ranked snapshot.
func (c *Cache) List(ctx context.Context) ([]task.Task, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if _, e := c.sync(ctx, nil); e != nil {
		return nil, e
	}
	return c.snapshot(), nil
}

// Listed is the board as this cache last knew it: as Open loaded it from
// the database, or as the last sync left it. No walk, no git: what a screen
// can draw at once while a sync runs (vnyz8z). Empty when the cache has
// never synced this board.
func (c *Cache) Listed() []task.Task {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.snapshot()
}

func (c *Cache) snapshot() []task.Task {
	if c.ordered == nil {
		c.ordered = make([]task.Task, 0, len(c.files))
		for _, r := range c.files {
			c.ordered = append(c.ordered, r.Task)
		}
		task.Sort(c.ordered)
	}
	out := make([]task.Task, len(c.ordered))
	for i, original := range c.ordered {
		t := original
		t.Assignee = append([]string(nil), t.Assignee...)
		t.Labels = append([]string(nil), t.Labels...)
		t.Aliases = append([]string(nil), t.Aliases...)
		t.Dependencies = append([]task.Dependency(nil), t.Dependencies...)
		t.Waits = append([]task.Wait(nil), t.Waits...)
		t.Acceptance = append([]task.Acceptance(nil), t.Acceptance...)
		out[i] = t
	}
	return out
}
func (c *Cache) Lint(ctx context.Context) ([]task.Finding, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if _, e := c.sync(ctx, nil); e != nil {
		return nil, e
	}
	return c.lint(), nil
}

// LintListed is Lint of the board the last List or Sync saw, without
// walking the files again: for a caller that has just listed (the TUI's
// reload, 44w1ev: a second walk of 10,000 files right after the first).
func (c *Cache) LintListed() []task.Finding {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.lint()
}
func (c *Cache) lint() []task.Finding {
	docs := make([]*task.Document, 0, len(c.files))
	for _, r := range c.files {
		docs = append(docs, &task.Document{Task: r.Task, Findings: r.Findings})
	}
	return task.Lint(docs)
}
func (c *Cache) Markers(ctx context.Context) ([]Marker, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if _, e := c.sync(ctx, nil); e != nil {
		return nil, e
	}
	var out []Marker
	for _, r := range c.files {
		out = append(out, r.Marker)
	}
	return out, nil
}
func (c *Cache) String() string { return fmt.Sprintf("huginn cache (%s)", c.Root) }

// SetSchemaForTest marks the cache as written by another schema version, as
// an older or newer hug binary leaves it; the next Open rebuilds. For tests
// in other packages.
func (c *Cache) SetSchemaForTest(version string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	_, e := c.db.Exec(`UPDATE meta SET value=? WHERE key='schema_version'`, version)
	return e
}

// Rebuilt reports whether Open found another schema version and rebuilt.
func (c *Cache) Rebuilt() bool { c.mu.Lock(); defer c.mu.Unlock(); return c.rebuilt }
