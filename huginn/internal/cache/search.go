package cache

import (
	"context"
	"sort"
	"strconv"
	"strings"
	"unicode"

	"github.com/veridit/hug/huginn/internal/task"
)

// Ranked text search over the board (49n9kw): an FTS5 table in the task
// cache, one row per task file, written in the same transaction as the
// task's other rows, so it follows every sync and write without a rebuild.
// BM25 ranks; id, aliases and title weigh most. The tokenizer keeps "-" in
// tokens, so STATBUS-402 and 44w1ev are single words.
// The FTS5 row of a task file has the rowid of its search_paths row:
// path is looked up there by index, never scanned for in the FTS table,
// where a WHERE on an unindexed column reads every row (TestSearchWritesNeverScan).
const searchSchema = `CREATE TABLE IF NOT EXISTS search_paths(rowid INTEGER PRIMARY KEY, path TEXT NOT NULL UNIQUE); CREATE VIRTUAL TABLE IF NOT EXISTS search USING fts5(id, aliases, title, goal, description, acceptance, notes, tokenize="unicode61 tokenchars '-_'");`

// searchColumns are the indexed columns, in table order, with
// their BM25 weights and the name a result gives for a match there.
var searchColumns = []struct {
	name   string
	weight float64
}{{"id", 10}, {"aliases", 10}, {"title", 8}, {"goal", 3}, {"description", 2}, {"acceptance", 2}, {"notes", 1}}

// searchWrites are the statements that change the search index per task
// file, as the code runs them (TestSearchWritesNeverScan reads their plans).
var searchWrites = []string{searchDelete, searchPath, searchInsert, searchForget, searchRename}

const (
	searchDelete = `DELETE FROM search WHERE rowid=(SELECT rowid FROM search_paths WHERE path=?)`
	searchPath   = `INSERT OR IGNORE INTO search_paths(path) VALUES(?)`
	searchInsert = `INSERT INTO search(rowid,id,aliases,title,goal,description,acceptance,notes) VALUES((SELECT rowid FROM search_paths WHERE path=?),?,?,?,?,?,?,?)`
	searchForget = `DELETE FROM search_paths WHERE path=?`
	searchRename = `UPDATE search_paths SET path=? WHERE path=?`
)

// unindexSearch removes path's search row.
func unindexSearch(exec func(string, ...any) error, path string) error {
	if e := exec(searchDelete, path); e != nil {
		return e
	}
	return exec(searchForget, path)
}

// indexSearch writes path's search row from the parsed task and its texts.
func indexSearch(exec func(string, ...any) error, path string, t task.Task, x *task.Texts) error {
	if e := exec(searchDelete, path); e != nil {
		return e
	}
	if e := exec(searchPath, path); e != nil {
		return e
	}
	var desc, notes string
	if x != nil {
		notes = x.Notes
		parts := make([]string, 0, len(task.DescriptionSections))
		for _, sec := range task.DescriptionSections {
			if v := x.Description[sec]; v != "" {
				parts = append(parts, v)
			}
		}
		desc = strings.Join(parts, "\n")
	}
	var acc []string
	for _, a := range t.Acceptance {
		acc = append(acc, a.Text)
	}
	return exec(searchInsert,
		path, t.ID, strings.Join(t.Aliases, " "), t.Title, t.Goal, desc, strings.Join(acc, "\n"), notes)
}

// Hit is one search result: the task file, its BM25 score (higher is
// better), the column the match counts most in and a snippet from it.
type Hit struct {
	Path    string
	Score   float64
	Field   string
	Snippet string
}

// SearchWords splits a query into the words FTS5 matches: runs of letters,
// digits, "-" and "_", lowercased.
func SearchWords(query string) []string {
	var out []string
	for _, w := range strings.FieldsFunc(strings.ToLower(query), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '-' && r != '_'
	}) {
		if w = strings.Trim(w, "-_"); w != "" {
			out = append(out, w)
		}
	}
	return out
}

// Search returns the task files matching every word of query, best first,
// at most limit (0: all). The board must have been listed (List syncs).
func (c *Cache) Search(ctx context.Context, query string, limit int) ([]Hit, error) {
	words := SearchWords(query)
	if len(words) == 0 {
		return nil, nil
	}
	quoted := make([]string, len(words))
	for i, w := range words {
		quoted[i] = `"` + strings.ReplaceAll(w, `"`, `""`) + `"`
	}
	match := strings.Join(quoted, " AND ")
	weights := make([]string, 0, len(searchColumns)+1)
	for _, col := range searchColumns {
		weights = append(weights, strconv.FormatFloat(col.weight, 'f', -1, 64))
	}
	bm := `bm25(search,` + strings.Join(weights, ",") + `)`
	q := `SELECT p.path,-` + bm + `,search.id,search.aliases,search.title,search.goal,search.description,search.acceptance,search.notes FROM search JOIN search_paths p ON p.rowid=search.rowid WHERE search MATCH ? ORDER BY ` + bm
	c.mu.Lock()
	defer c.mu.Unlock()
	rows, e := c.db.QueryContext(ctx, q, match)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	var out []Hit
	for rows.Next() {
		var h Hit
		cols := make([]string, len(searchColumns))
		dest := []any{&h.Path, &h.Score}
		for i := range cols {
			dest = append(dest, &cols[i])
		}
		if e = rows.Scan(dest...); e != nil {
			return nil, e
		}
		h.Field, h.Snippet = bestField(cols, words)
		out = append(out, h)
	}
	if e = rows.Err(); e != nil {
		return nil, e
	}
	// Ties keep path order, so the same board answers the same way.
	sort.SliceStable(out, func(i, j int) bool { return out[i].Score > out[j].Score })
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

// bestField is the highest-weighted column holding a query word, and a
// snippet of about 80 characters around the first such word in it.
func bestField(cols []string, words []string) (string, string) {
	for i, col := range searchColumns {
		text := cols[i]
		lower := strings.ToLower(text)
		for _, w := range words {
			if at := strings.Index(lower, w); at >= 0 {
				return col.name, snippet(text, at, len(w))
			}
		}
	}
	return "", ""
}

// snippet is one line of text around [at, at+n), cut at word boundaries,
// with "…" where it was cut.
func snippet(text string, at, n int) string {
	const around = 36
	start, end := max(0, at-around), min(len(text), at+n+around)
	for start > 0 && start < at && text[start-1] != ' ' && text[start-1] != '\n' {
		start++
	}
	for end < len(text) && end > at+n && text[end] != ' ' && text[end] != '\n' {
		end--
	}
	s := strings.Join(strings.Fields(text[start:end]), " ")
	if start > 0 {
		s = "…" + s
	}
	if end < len(text) {
		s += "…"
	}
	return s
}
