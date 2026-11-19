// Package store is muninn's SQLite persistence layer (pure Go, WAL).
package store

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	_ "modernc.org/sqlite"

	"github.com/veridit/hug/internal/flock"
)

// Unit is one recallable piece of text: a memory, or a markdown section.
type Unit struct {
	ID          int64
	SID         string
	Origin      string // stable identity, never the display id
	SourceID    int64
	Key         string // stable key within the source (memory id, or file#line)
	ContentHash string
	Ident       string // hash over text and metadata; unchanged ident = skip entirely
	Kind        string // "memory" | "section"
	Title       string
	Body        string
	Category    string
	Type        string // memory type: rule | preference | fact | record | "" (ingest.NativeMemory.Type)
	Holds       string // condition the memory holds under, "" for none (ingest.NativeMemory.Holds)
	Trigger     string // when-then trigger, canonical JSON (trigger.Encode), "" for none (jrj1vq)
	Tags        []string
	Scope       string // "global" or a project identifier
	Active      bool
	Updated     time.Time
}

// DB wraps the database handle.
type DB struct {
	*sql.DB
	Path string
}

// OldHugView is graph_state since 4vd1mz: an empty view in place of the
// gob table, so a hug from before v0.7.59 cannot write a graph built from
// nothing. An MCP server started before v0.7.54 never reloads itself, so
// such a hug keeps running (mbdfzs check 3). Its index pass reads
// graph_state before it writes it, and SQLite refuses the write (an UPSERT
// on a view) before any trigger could speak, so the read carries the fix:
// state is a json_extract with a bad path, whose error SQLite returns
// verbatim ("bad JSON path: '<text>'"). Nothing current reads state.
const OldHugView = `CREATE VIEW graph_state(id, state) AS SELECT 1, json_extract('{}', '` + OldHugFix + `') WHERE 1`

// OldHugFix is what a hug older than v0.7.59 says when it tries to index.
const OldHugFix = "hug: this hug mcp server is older than v0.7.59 and cannot write this memory store; " +
	"fix: restart it (jcode: mcp action=reload server=hug; Claude/Codex: quit and reopen the app). " +
	"The note was saved: the next current hug that writes or indexes (hug mem index) makes it searchable."

// oldHugView gives a store with the plain graph_state view the one that
// names the fix. A store whose graph_state is still the gob table keeps it
// until its first writer moves it (index.moveGraph).
func oldHugView(db *sql.DB) error {
	var typ, def string
	if err := db.QueryRow(`SELECT type, sql FROM sqlite_master WHERE name='graph_state'`).Scan(&typ, &def); err != nil {
		return err
	}
	if typ != "view" || def == OldHugView {
		return nil
	}
	for _, q := range []string{`DROP VIEW graph_state`, OldHugView} {
		if _, err := db.Exec(q); err != nil {
			return err
		}
	}
	return nil
}

const schema = `
CREATE TABLE IF NOT EXISTS capture_backlog (session TEXT NOT NULL, message TEXT NOT NULL, PRIMARY KEY(session,message));
-- A hook chunk whose reply was still cut at max_tokens after the retry
-- (nmn6wf): seen, so capture goes on, and drained by a later hook at the
-- full cap. harness/home/harness_session say which transcript to read.
CREATE TABLE IF NOT EXISTS capture_deferred (session TEXT NOT NULL, message TEXT NOT NULL, original TEXT NOT NULL DEFAULT '', harness TEXT NOT NULL, home TEXT NOT NULL, harness_session TEXT NOT NULL, at_ns INTEGER NOT NULL, PRIMARY KEY(session,message));
-- Memories whose stated condition (units.holds) has ended: recall skips
-- them, a dream item asks an agent to retire or keep them (ez50rb).
CREATE TABLE IF NOT EXISTS holds_ended (sid TEXT PRIMARY KEY, holds TEXT NOT NULL, reason TEXT NOT NULL, at_ns INTEGER NOT NULL);
-- Memory fixes (rjk44w): one row per forget, and per remember that names
-- an old memory (reinforce, supersedes), with who, whether recall showed
-- it to that session first, and an optional reason. Recall never reads it.
CREATE TABLE IF NOT EXISTS memory_fixes (
  at_ns INTEGER NOT NULL, action TEXT NOT NULL, sid TEXT NOT NULL,
  new_sid TEXT NOT NULL DEFAULT '', session TEXT NOT NULL DEFAULT '',
  harness TEXT NOT NULL DEFAULT '', instance TEXT NOT NULL DEFAULT '',
  surface TEXT NOT NULL DEFAULT '', reason TEXT NOT NULL DEFAULT '',
  shown INTEGER NOT NULL DEFAULT 0, shown_ago_ns INTEGER NOT NULL DEFAULT 0,
  content_hash TEXT NOT NULL DEFAULT '');
CREATE INDEX IF NOT EXISTS memory_fixes_at ON memory_fixes(at_ns);
CREATE INDEX IF NOT EXISTS memory_fixes_sid ON memory_fixes(sid, at_ns);
CREATE TABLE IF NOT EXISTS capture_bindings (session TEXT PRIMARY KEY, scope TEXT NOT NULL);
CREATE TABLE IF NOT EXISTS capture_seen (
 session TEXT NOT NULL, message TEXT NOT NULL, PRIMARY KEY(session,message)
);
CREATE TABLE IF NOT EXISTS capture_state (
 session TEXT PRIMARY KEY, high_water INTEGER NOT NULL DEFAULT 0
);
CREATE TABLE IF NOT EXISTS capture_lock (id INTEGER PRIMARY KEY, token TEXT NOT NULL, expires_ns INTEGER NOT NULL);
CREATE TABLE IF NOT EXISTS capture_reinforcements (
 event TEXT PRIMARY KEY, sid TEXT NOT NULL, at_ns INTEGER NOT NULL
);
CREATE TABLE IF NOT EXISTS meta (key TEXT PRIMARY KEY, value TEXT NOT NULL);
-- Judgement items queued by "hug mem dream" for the driving agent (2026-11-12).
-- kind: dupes (keep which), place (where it belongs), ...; sids are space
-- separated; scope is the effective scope a session must see to get the item.
CREATE TABLE IF NOT EXISTS dream_items (
  id INTEGER PRIMARY KEY, kind TEXT NOT NULL, key TEXT NOT NULL UNIQUE,
  sids TEXT NOT NULL, scope TEXT NOT NULL, instruction TEXT NOT NULL,
  created_ns INTEGER NOT NULL, handed_ns INTEGER NOT NULL DEFAULT 0,
  handed_to TEXT NOT NULL DEFAULT '', resolved_ns INTEGER NOT NULL DEFAULT 0,
  resolution TEXT NOT NULL DEFAULT '');
-- Last user turn per harness session, to notice breaks (start, return after
-- an absence, wrap-up) for handing out dream items.
CREATE TABLE IF NOT EXISTS session_turns (
  session TEXT PRIMARY KEY, last_ns INTEGER NOT NULL, turns INTEGER NOT NULL,
  dream_cued_ns INTEGER NOT NULL DEFAULT 0);
-- The person's turns per clock hour (hour = unix seconds / 3600), for the
-- hour-of-week histogram that places the dream pass (vk9td0). Kept 8 weeks.
CREATE TABLE IF NOT EXISTS activity_hours (hour INTEGER PRIMARY KEY, turns INTEGER NOT NULL);
CREATE TABLE IF NOT EXISTS sources (
  id INTEGER PRIMARY KEY,
  kind TEXT NOT NULL,          -- jcode | markdown | native
  path TEXT NOT NULL UNIQUE,
  mtime_ns INTEGER NOT NULL DEFAULT 0,
  size INTEGER NOT NULL DEFAULT 0,
  file_hash TEXT NOT NULL DEFAULT ''
);
CREATE TABLE IF NOT EXISTS units (
  id INTEGER PRIMARY KEY,
  source_id INTEGER NOT NULL REFERENCES sources(id) ON DELETE CASCADE,
  key TEXT NOT NULL,
  content_hash TEXT NOT NULL,
  ident TEXT NOT NULL DEFAULT '', -- hash of text + metadata: change detection
  kind TEXT NOT NULL,
  title TEXT NOT NULL DEFAULT '',
  body TEXT NOT NULL,
  category TEXT NOT NULL DEFAULT '',
  tags TEXT NOT NULL DEFAULT '',
  scope TEXT NOT NULL DEFAULT 'global',
  active INTEGER NOT NULL DEFAULT 1,
  updated_ns INTEGER NOT NULL DEFAULT 0,
  UNIQUE(source_id, key)
);
CREATE INDEX IF NOT EXISTS units_hash ON units(content_hash);
-- Covering index for scoped candidate scans (search never reads unit bodies
-- until the final few results).
CREATE INDEX IF NOT EXISTS units_scope ON units(scope, active, kind, content_hash);
CREATE VIRTUAL TABLE IF NOT EXISTS units_fts USING fts5(title, body, tags, content='units', content_rowid='id');
CREATE TRIGGER IF NOT EXISTS units_ai AFTER INSERT ON units BEGIN
  INSERT INTO units_fts(rowid, title, body, tags) VALUES (new.id, new.title, new.body, new.tags);
END;
CREATE TRIGGER IF NOT EXISTS units_ad AFTER DELETE ON units BEGIN
  INSERT INTO units_fts(units_fts, rowid, title, body, tags) VALUES ('delete', old.id, old.title, old.body, old.tags);
END;
CREATE TRIGGER IF NOT EXISTS units_au AFTER UPDATE ON units BEGIN
  INSERT INTO units_fts(units_fts, rowid, title, body, tags) VALUES ('delete', old.id, old.title, old.body, old.tags);
  INSERT INTO units_fts(rowid, title, body, tags) VALUES (new.id, new.title, new.body, new.tags);
END;
-- Content-addressed vectors: identical text shares one row; a recipe change
-- (pipeline_hash) invalidates exactly the affected rows.
CREATE TABLE IF NOT EXISTS embeddings (
  content_hash TEXT NOT NULL,
  pipeline_hash TEXT NOT NULL,
  dim INTEGER NOT NULL,
  vec BLOB NOT NULL,      -- normalised float32, little-endian
  qvec BLOB NOT NULL,     -- int8 quantised copy (4x smaller)
  sketch BLOB NOT NULL,   -- 1-bit sign sketch (32x smaller) for Hamming prefilter
  PRIMARY KEY (content_hash, pipeline_hash)
);
-- Narrow copy of the sketches so the Hamming prefilter reads ~100 bytes per
-- unit instead of every 4 KB embeddings row.
CREATE TABLE IF NOT EXISTS sketches (
  content_hash TEXT NOT NULL,
  pipeline_hash TEXT NOT NULL,
  sketch BLOB NOT NULL,
  PRIMARY KEY (content_hash, pipeline_hash)
) WITHOUT ROWID;
-- Narrow copy of the int8 vectors for the neighbour update, which reads
-- every active one (4vd1mz): in embeddings each qvec sits behind the 3 KB
-- float vector on overflow pages, and a cold read of all of them took
-- 635 ms of a 1.6 s forget on a 30,000-memory store (here: 430 ms).
CREATE TABLE IF NOT EXISTS qvecs (
  content_hash TEXT NOT NULL,
  pipeline_hash TEXT NOT NULL,
  qvec BLOB NOT NULL,
  PRIMARY KEY (content_hash, pipeline_hash)
) WITHOUT ROWID;
-- Incremental neighbour graph (k nearest by cosine), maintained per dirty unit.
CREATE TABLE IF NOT EXISTS neighbors (
  unit_id INTEGER NOT NULL,
  nbr_id INTEGER NOT NULL,
  sim REAL NOT NULL,
  PRIMARY KEY (unit_id, nbr_id)
);
CREATE INDEX IF NOT EXISTS neighbors_nbr ON neighbors(nbr_id);
-- Concept graph (seid search): unigram concepts, unit membership, communities.
CREATE TABLE IF NOT EXISTS concepts (id INTEGER PRIMARY KEY, term TEXT NOT NULL UNIQUE);
CREATE TABLE IF NOT EXISTS unit_concepts (
  unit_id INTEGER NOT NULL, concept_id INTEGER NOT NULL,
  PRIMARY KEY (unit_id, concept_id)
);
CREATE INDEX IF NOT EXISTS unit_concepts_c ON unit_concepts(concept_id);
CREATE TABLE IF NOT EXISTS communities (concept_id INTEGER PRIMARY KEY, community_id INTEGER NOT NULL);
-- Per-unit concept counts by community (as in seid search's
-- section_communities): community expansion reads this instead of joining
-- every unit_concepts row of the query communities.
CREATE TABLE IF NOT EXISTS unit_communities (
  community_id INTEGER NOT NULL, unit_id INTEGER NOT NULL, n INTEGER NOT NULL,
  PRIMARY KEY (community_id, unit_id)
) WITHOUT ROWID;
CREATE INDEX IF NOT EXISTS unit_communities_u ON unit_communities(unit_id);
-- The concept graph (4vd1mz): each concept's weighted degree and packed
-- adjacency (uvarint id gap, weight), and each community's total degree;
-- its total weight is meta graph_m. graph_state held the whole graph as one
-- gob before; the first writer moves it here (index.moveGraph) and leaves an
-- empty view in its place.
CREATE TABLE IF NOT EXISTS graph_vertex (concept_id INTEGER PRIMARY KEY, k REAL NOT NULL, adj BLOB NOT NULL);
CREATE TABLE IF NOT EXISTS graph_community (community_id INTEGER PRIMARY KEY, sum_tot REAL NOT NULL);
-- On a new store this is the plain view; Open then makes it the one that
-- tells a hug older than v0.7.59 to restart (OldHugView, mbdfzs).
CREATE VIEW IF NOT EXISTS graph_state(id, state) AS SELECT 1, NULL WHERE 0;
-- Memory types inferred for memories whose source has no type field (jcode
-- memory files, untyped native memories): keyed by content hash, so a type
-- follows the text and survives re-indexing. A type stated on the memory
-- itself always wins. Written by "hug mem types".
CREATE TABLE IF NOT EXISTS inferred_types (
  content_hash TEXT PRIMARY KEY, mtype TEXT NOT NULL, model TEXT NOT NULL, at_ns INTEGER NOT NULL
) WITHOUT ROWID;
-- Anchors a memory names (n7jbmx, package anchor): "path:a/b.go",
-- "file:b.go", "cmd:go test". Tool-call recall matches a call's anchors
-- here exactly. Written at index time, active units only.
CREATE TABLE IF NOT EXISTS unit_anchors (
  unit_id INTEGER NOT NULL, anchor TEXT NOT NULL, PRIMARY KEY (unit_id, anchor)
) WITHOUT ROWID;
CREATE INDEX IF NOT EXISTS unit_anchors_anchor ON unit_anchors(anchor, unit_id);
-- Recall injections, for per-session dedupe and the web/TUI log.
CREATE TABLE IF NOT EXISTS injections (
  session TEXT NOT NULL, unit_id INTEGER NOT NULL, content_hash TEXT NOT NULL,
  harness TEXT NOT NULL, at_ns INTEGER NOT NULL,
  PRIMARY KEY (session, content_hash)
);
`

// Open opens (creating if needed) the database at path.
func Open(path string) (*DB, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, err
	}
	// Memories are private: create the database owner-only before SQLite does
	// (SQLite would apply the umask, typically 0644), and tighten existing files.
	if f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600); err == nil {
		f.Close()
	}
	for _, p := range []string{path, path + "-wal", path + "-shm"} {
		if fi, err := os.Stat(p); err == nil && fi.Mode().Perm()&0o077 != 0 {
			os.Chmod(p, 0o600)
		}
	}
	// mmap: search scans the narrow tables; through the default pager every
	// page was a pread syscall (profiled at ~65% of a warm search).
	// _txlock=immediate: every BeginTx here writes. A deferred transaction
	// that reads first and then writes cannot upgrade after another process
	// commits, and SQLite returns SQLITE_BUSY at once without consulting
	// busy_timeout (2026-11-06: 4 concurrent capture hooks failed this way).
	// BEGIN IMMEDIATE takes the write lock up front and waits instead.
	// busy_timeout comes first: pragmas run in order, and switching a new
	// database to WAL needs a lock (2026-11-10: two first runs at once failed
	// "schema: database is locked" when busy_timeout came second).
	dsn := "file:" + path + "?_pragma=busy_timeout(" + strconv.Itoa(BusyTimeout) + ")&_pragma=journal_mode(WAL)&_pragma=foreign_keys(1)&_pragma=synchronous(NORMAL)" +
		"&_pragma=mmap_size(1073741824)&_pragma=cache_size(-32768)&_txlock=immediate"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1) // single writer; WAL lets other processes read
	// Two processes creating the schema at once got SQLITE_BUSY despite
	// busy_timeout (2026-11-10, two first runs in parallel). Take turns.
	unlockInit, err := flock.Acquire(path + ".init.lock")
	if err != nil {
		db.Close()
		return nil, err
	}
	_, err = db.Exec(schema)
	if err == nil {
		err = oldHugView(db)
	}
	unlockInit()
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("schema: %w", err)
	}
	// One-time backfill for indexes written before the sketches table existed.
	var have, want int
	db.QueryRow(`SELECT EXISTS(SELECT 1 FROM sketches), EXISTS(SELECT 1 FROM embeddings)`).Scan(&have, &want)
	if have == 0 && want == 1 {
		if _, err := db.Exec(`INSERT OR IGNORE INTO sketches(content_hash, pipeline_hash, sketch)
		  SELECT content_hash, pipeline_hash, sketch FROM embeddings`); err != nil {
			db.Close()
			return nil, fmt.Errorf("backfill sketches: %w", err)
		}
	}
	db.QueryRow(`SELECT EXISTS(SELECT 1 FROM qvecs), EXISTS(SELECT 1 FROM embeddings)`).Scan(&have, &want)
	if have == 0 && want == 1 {
		if _, err := db.Exec(`INSERT OR IGNORE INTO qvecs(content_hash, pipeline_hash, qvec)
		  SELECT content_hash, pipeline_hash, qvec FROM embeddings`); err != nil {
			db.Close()
			return nil, fmt.Errorf("backfill qvecs: %w", err)
		}
	}
	db.QueryRow(`SELECT EXISTS(SELECT 1 FROM unit_communities), EXISTS(SELECT 1 FROM communities)`).Scan(&have, &want)
	if have == 0 && want == 1 {
		if _, err := db.Exec(`INSERT INTO unit_communities(community_id, unit_id, n)
		  SELECT c.community_id, uc.unit_id, COUNT(*) FROM unit_concepts uc
		  JOIN communities c ON c.concept_id=uc.concept_id GROUP BY 1, 2`); err != nil {
			db.Close()
			return nil, fmt.Errorf("backfill unit_communities: %w", err)
		}
	}
	returnDB := &DB{DB: db, Path: path}
	if err := returnDB.migrateMemoryType(); err != nil {
		db.Close()
		return nil, err
	}
	if err := returnDB.addUnitColumn("holds"); err != nil {
		db.Close()
		return nil, err
	}
	// mtrigger (jrj1vq): the when-then trigger; "trigger" is an SQL keyword.
	if err := returnDB.addUnitColumn("mtrigger"); err != nil {
		db.Close()
		return nil, err
	}
	if err := returnDB.migrateShortIDs(); err != nil {
		db.Close()
		return nil, fmt.Errorf("short ids: %w", err)
	}
	return returnDB, nil
}

// Source looks up or creates a source row and returns its id and stat.
func (d *DB) Source(kind, path string) (id int64, mtime, size int64, err error) {
	err = d.QueryRow(`SELECT id, mtime_ns, size FROM sources WHERE path=?`, path).Scan(&id, &mtime, &size)
	if err == sql.ErrNoRows {
		res, e := d.Exec(`INSERT INTO sources(kind, path) VALUES(?, ?)`, kind, path)
		if e != nil {
			return 0, 0, 0, e
		}
		id, err = res.LastInsertId()
		return id, -1, -1, err
	}
	return id, mtime, size, err
}

// SetSourceStat records the stat seen at the last successful scan.
func (d *DB) SetSourceStat(id, mtime, size int64) error {
	_, err := d.Exec(`UPDATE sources SET mtime_ns=?, size=? WHERE id=?`, mtime, size, id)
	return err
}

// UnitHashes returns key -> (unit id, ident) for a source.
func (d *DB) UnitHashes(sourceID int64) (map[string][2]string, error) {
	rows, err := d.Query(`SELECT key, id, ident FROM units WHERE source_id=?`, sourceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	m := map[string][2]string{}
	for rows.Next() {
		var k, h string
		var id int64
		if err := rows.Scan(&k, &id, &h); err != nil {
			return nil, err
		}
		m[k] = [2]string{fmt.Sprint(id), h}
	}
	return m, rows.Err()
}

// UpsertUnit inserts or replaces a unit identified by (source, key); returns its id.
func (d *DB) UpsertUnit(u Unit) (int64, error) {
	if err := d.prepareSID(&u); err != nil {
		return 0, err
	}
	active := 0
	if u.Active {
		active = 1
	}
	_, err := d.Exec(`INSERT INTO units(sid, origin, source_id, key, content_hash, ident, kind, title, body, category, mtype, holds, mtrigger, tags, scope, active, updated_ns)
	  VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)
	  ON CONFLICT(source_id, key) DO UPDATE SET sid=excluded.sid, origin=excluded.origin, content_hash=excluded.content_hash, ident=excluded.ident, kind=excluded.kind,
	    title=excluded.title, body=excluded.body, category=excluded.category, mtype=excluded.mtype, holds=excluded.holds, mtrigger=excluded.mtrigger, tags=excluded.tags,
	    scope=excluded.scope, active=excluded.active, updated_ns=excluded.updated_ns`,
		u.SID, u.Origin, u.SourceID, u.Key, u.ContentHash, u.Ident, u.Kind, u.Title, u.Body, u.Category, u.Type, u.Holds, u.Trigger,
		strings.Join(u.Tags, " "), u.Scope, active, u.Updated.UnixNano())
	if err != nil {
		return 0, err
	}
	var id int64
	err = d.QueryRow(`SELECT id FROM units WHERE source_id=? AND key=?`, u.SourceID, u.Key).Scan(&id)
	return id, err
}

// DeleteUnit removes a unit and its concept membership. Neighbour edges are
// left for Indexer.UpdateNeighbors, which needs them to know which other
// lists referenced the unit and must be refilled.
func (d *DB) DeleteUnit(id int64) error {
	for _, q := range []string{
		`DELETE FROM unit_concepts WHERE unit_id=?`,
		`DELETE FROM unit_communities WHERE unit_id=?`,
		`DELETE FROM unit_anchors WHERE unit_id=?`,
		`DELETE FROM units WHERE id=?`,
	} {
		if _, err := d.Exec(q, id); err != nil {
			return err
		}
	}
	return nil
}

// SetUnitAnchors replaces a unit's anchors (nil clears them).
func (d *DB) SetUnitAnchors(id int64, anchors []string) error {
	if _, err := d.Exec(`DELETE FROM unit_anchors WHERE unit_id=?`, id); err != nil {
		return err
	}
	for _, a := range anchors {
		if _, err := d.Exec(`INSERT OR IGNORE INTO unit_anchors(unit_id, anchor) VALUES(?,?)`, id, a); err != nil {
			return err
		}
	}
	return nil
}

// GetUnit loads one unit.
func (d *DB) GetUnit(id int64) (Unit, error) {
	var u Unit
	var tags string
	var active int
	var upd int64
	err := d.QueryRow(`SELECT id, sid, origin, source_id, key, content_hash, kind, title, body, category,
	  COALESCE(NULLIF(mtype,''), (SELECT t.mtype FROM inferred_types t WHERE t.content_hash=units.content_hash), ''), holds, mtrigger, tags, scope, active, updated_ns
	  FROM units WHERE id=?`, id).Scan(&u.ID, &u.SID, &u.Origin, &u.SourceID, &u.Key, &u.ContentHash, &u.Kind, &u.Title, &u.Body,
		&u.Category, &u.Type, &u.Holds, &u.Trigger, &tags, &u.Scope, &active, &upd)
	u.Tags = strings.Fields(tags)
	u.Active = active == 1 && !d.IsSuppressed(u.SID)
	u.Updated = time.Unix(0, upd)
	return u, err
}

// HasEmbedding reports whether a vector exists for (hash, pipeline).
func (d *DB) HasEmbedding(hash, pipeline string) bool {
	var one int
	return d.QueryRow(`SELECT 1 FROM embeddings WHERE content_hash=? AND pipeline_hash=?`, hash, pipeline).Scan(&one) == nil
}

// PutEmbedding stores a vector with its compressed forms.
func (d *DB) PutEmbedding(hash, pipeline string, dim int, vec, qvec, sketch []byte) error {
	_, err := d.Exec(`INSERT OR REPLACE INTO embeddings(content_hash, pipeline_hash, dim, vec, qvec, sketch) VALUES(?,?,?,?,?,?)`,
		hash, pipeline, dim, vec, qvec, sketch)
	if err != nil {
		return err
	}
	_, err = d.Exec(`INSERT OR REPLACE INTO sketches(content_hash, pipeline_hash, sketch) VALUES(?,?,?)`, hash, pipeline, sketch)
	if err != nil {
		return err
	}
	_, err = d.Exec(`INSERT OR REPLACE INTO qvecs(content_hash, pipeline_hash, qvec) VALUES(?,?,?)`, hash, pipeline, qvec)
	return err
}

// GC removes vectors that belong to other pipelines or to no current unit.
func (d *DB) GC(pipeline string) (int64, error) {
	res, err := d.Exec(`DELETE FROM embeddings WHERE pipeline_hash<>? OR content_hash NOT IN (SELECT content_hash FROM units)`, pipeline)
	if err != nil {
		return 0, err
	}
	if _, err := d.Exec(`DELETE FROM sketches WHERE pipeline_hash<>? OR content_hash NOT IN (SELECT content_hash FROM units)`, pipeline); err != nil {
		return 0, err
	}
	if _, err := d.Exec(`DELETE FROM qvecs WHERE pipeline_hash<>? OR content_hash NOT IN (SELECT content_hash FROM units)`, pipeline); err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

// Meta helpers.
func (d *DB) GetMeta(k string) string {
	var v string
	_ = d.QueryRow(`SELECT value FROM meta WHERE key=?`, k).Scan(&v)
	return v
}

func (d *DB) SetMeta(k, v string) error {
	_, err := d.Exec(`INSERT INTO meta(key,value) VALUES(?,?) ON CONFLICT(key) DO UPDATE SET value=excluded.value`, k, v)
	return err
}

// SetInferredType records a model-inferred memory type for a text.
func (d *DB) SetInferredType(contentHash, mtype, model string) error {
	_, err := d.Exec(`INSERT INTO inferred_types(content_hash, mtype, model, at_ns) VALUES(?,?,?,?)
	  ON CONFLICT(content_hash) DO UPDATE SET mtype=excluded.mtype, model=excluded.model, at_ns=excluded.at_ns`,
		contentHash, mtype, model, time.Now().UnixNano())
	return err
}

// UntypedUnits lists active units with neither a stated nor an inferred
// type, oldest id first, at most limit (0 = all).
func (d *DB) UntypedUnits(limit int) ([]Unit, error) {
	q := `SELECT id FROM units WHERE active=1 AND mtype='' AND content_hash NOT IN (SELECT content_hash FROM inferred_types) ORDER BY id`
	if limit > 0 {
		q += fmt.Sprintf(" LIMIT %d", limit)
	}
	rows, err := d.Query(q)
	if err != nil {
		return nil, err
	}
	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return nil, err
		}
		ids = append(ids, id)
	}
	rows.Close()
	out := make([]Unit, 0, len(ids))
	for _, id := range ids {
		u, err := d.GetUnit(id)
		if err != nil {
			return nil, err
		}
		out = append(out, u)
	}
	return out, nil
}

// migrateMemoryType adds units.mtype (memory type, 2026-11-15) to databases
// created before it. Idempotent; runs on every open, independent of the
// shortids schema version (which existing databases have already passed).
func (d *DB) migrateMemoryType() error { return d.addUnitColumn("mtype") }

// addUnitColumn adds a text column (NOT NULL, default empty) to units when a
// database created before it lacks it (mtype 2026-11-15, holds 2026-11-16,
// mtrigger 2026-11-18).
// Idempotent and safe against a concurrent opener.
func (d *DB) addUnitColumn(col string) error {
	rows, err := d.Query(`PRAGMA table_info(units)`)
	if err != nil {
		return err
	}
	has := false
	for rows.Next() {
		var cid, nn, pk int
		var name, typ string
		var def any
		if err := rows.Scan(&cid, &name, &typ, &nn, &def, &pk); err != nil {
			rows.Close()
			return err
		}
		has = has || name == col
	}
	rows.Close()
	if has {
		return nil
	}
	_, err = d.Exec(`ALTER TABLE units ADD COLUMN ` + col + ` TEXT NOT NULL DEFAULT ''`)
	if err != nil && strings.Contains(err.Error(), "duplicate column") {
		return nil // another process added it first
	}
	return err
}
