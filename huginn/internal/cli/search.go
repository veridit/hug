package cli

import (
	"context"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/veridit/hug/huginn/internal/task"
)

// searchLimit is how many results hug task search returns without --limit.
const searchLimit = 20

// search is hug task search WORDS (49n9kw): the tasks matching every word,
// ranked by BM25 over id, aliases, title, goal, description, acceptance and
// notes, each with the field that matched and a snippet. The ls filters
// (--state, --type, --epic, ...) narrow the results.
func (a *App) search(ctx context.Context, s *session, f flags, filter task.Filter) (Reply, error) {
	query := strings.TrimSpace(strings.Join(f.args, " "))
	if query == "" {
		return nil, usage("search needs words to look for", `hug task search "memory limit"`)
	}
	limit := searchLimit
	if v := f.get("limit"); v != "" {
		n, e := strconv.Atoi(v)
		if e != nil || n < 1 {
			return nil, usage("--limit takes a number from 1, got "+v, `hug task search "`+query+`" --limit 20`)
		}
		limit = n
	}
	ts, e := s.s.Tasks() // lists the board: the index is current after this
	if e != nil {
		return nil, e
	}
	byPath := make(map[string]task.Task, len(ts))
	for _, t := range ts {
		byPath[t.Path] = t
	}
	keep := func(path string) bool { t, ok := byPath[path]; return ok && matchesFilter(t, filter, ts) }
	hits, e := s.c.Search(ctx, query, limit, keep)
	if e != nil {
		return nil, e
	}
	out := []Reply{}
	for _, h := range hits {
		t := byPath[h.Path]
		out = append(out, Reply{"id": t.ID, "state": t.State, "type": t.Type, "title": t.Title, "aliases": t.Aliases,
			"field": h.Field, "snippet": h.Snippet, "score": h.Score})
	}
	reply := Reply{}
	reply["query"] = query
	reply["results"] = out
	if len(out) > 0 {
		reply["next"] = "hug task show " + out[0]["id"].(string)
	} else {
		reply["next"] = "hug task ls"
	}
	return reply, nil
}

// matchesFilter is the ls filter rule for one task.
func matchesFilter(t task.Task, filter task.Filter, ts []task.Task) bool {
	return filter.Keeps(t, ts)
}

// writeSearch prints search results: one line per task (id, status, title),
// then the field that matched and its snippet, indented.
func writeSearch(w io.Writer, query string, rs []Reply) {
	if len(rs) == 0 {
		fmt.Fprintf(w, "no task matches %q\n", query)
		return
	}
	for _, r := range rs {
		fmt.Fprintf(w, "%-6s  %-7s  %s\n", r["id"], r["state"], r["title"])
		if r["field"] != "" {
			fmt.Fprintf(w, "        %s: %s\n", r["field"], r["snippet"])
		}
	}
}
