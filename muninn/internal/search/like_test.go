package search

import (
	"context"
	"math/rand"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/veridit/hug/muninn/internal/store"
)

// likeDB is a store with the given concept terms, for comparing likeMatch
// and likeConcepts with SQLite's own LIKE on the same rows.
func likeDB(t testing.TB, terms []string) *store.DB {
	t.Helper()
	db, err := store.Open(filepath.Join(t.TempDir(), "m.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	seen := map[string]bool{}
	for _, term := range terms {
		if seen[term] {
			continue
		}
		seen[term] = true
		if _, err := db.Exec(`INSERT INTO concepts(term) VALUES(?)`, term); err != nil {
			t.Fatal(err)
		}
	}
	return db
}

// sqliteLike returns the concept ids SQLite's LIKE matches for pattern.
func sqliteLike(t testing.TB, db *store.DB, pattern string) map[int]bool {
	t.Helper()
	rows, err := db.Query(`SELECT id FROM concepts WHERE term LIKE ?`, pattern)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	out := map[int]bool{}
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			t.Fatal(err)
		}
		out[id] = true
	}
	return out
}

func terms(t testing.TB, db *store.DB) map[int]string {
	t.Helper()
	rows, err := db.Query(`SELECT id, term FROM concepts`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	out := map[int]string{}
	for rows.Next() {
		var id int
		var term string
		if err := rows.Scan(&id, &term); err != nil {
			t.Fatal(err)
		}
		out[id] = term
	}
	return out
}

// compareLike checks likeMatch per row and likeConcepts for all patterns at
// once against SQLite, and returns the number of comparisons made.
func compareLike(t *testing.T, db *store.DB, patterns []string) int {
	t.Helper()
	all := terms(t, db)
	union := map[int]bool{}
	n := 0
	for _, p := range patterns {
		want := sqliteLike(t, db, p)
		for id := range want {
			union[id] = true
		}
		for id, term := range all {
			n++
			if got := likeMatch(p, term); got != want[id] {
				t.Errorf("%q LIKE %q: likeMatch %v, SQLite %v", term, p, got, want[id])
			}
		}
	}
	got := map[int]bool{}
	if err := (&Searcher{DB: db}).likeConcepts(context.Background(), patterns, got); err != nil {
		t.Fatal(err)
	}
	if !sameIDs(got, union) {
		t.Errorf("likeConcepts(%q): %v, SQLite %v", patterns, ids(got), ids(union))
	}
	return n
}

func sameIDs(a, b map[int]bool) bool {
	if len(a) != len(b) {
		return false
	}
	for id := range a {
		if !b[id] {
			return false
		}
	}
	return true
}

func ids(m map[int]bool) []int {
	out := make([]int, 0, len(m))
	for id := range m {
		out = append(out, id)
	}
	sort.Ints(out)
	return out
}

// TestLikeUnderscoreIsAWildcard pins the 4 query words of capture run 8476
// where a plain substring test differs from SQLite's LIKE (b5jks7): in
// '%task_id%' and '%_called%' the '_' matches any one character.
func TestLikeUnderscoreIsAWildcard(t *testing.T) {
	db := likeDB(t, []string{
		"task-id", "task-ids", "task-id-draft-142-promoted-task-447", "task_id", "taskid", "task--id",
		"recalled", "recalled-memory", "uncalled", "auto-recalled", "direct-called", "double-called",
		"fail-if-called", "never-called", "called", "_called", "calling",
	})
	compareLike(t, db, []string{"%task_id%", "%_called%"})
	for _, c := range []struct {
		pattern, term string
		want          bool
	}{
		{"%task_id%", "task-id", true},
		{"%task_id%", "taskid", false},
		{"%_called%", "recalled", true},
		{"%_called%", "called", false},
	} {
		if got := likeMatch(c.pattern, c.term); got != c.want {
			t.Errorf("%q LIKE %q = %v, want %v", c.term, c.pattern, got, c.want)
		}
	}
}

// TestLikeOddWords covers the edges of SQLite's LIKE: '%' and '_' inside the
// word, ASCII-only case folding, non-ASCII and invalid UTF-8, empty word and
// empty term, and NUL (LIKE reads both operands as C strings).
func TestLikeOddWords(t *testing.T) {
	rows := []string{
		"", "a", "A", "_", "%", "%%", "a_b", "a%b", "axb", "ab", "AB", "aB", "Ab",
		"é", "É", "ée", "Éé", "straße", "STRASSE", "ǅ", "ǆ", "ı", "I", "i", "K", "k", "\u212a",
		"日本語", "日本", "x日y", "\xff", "\xc3", "\xc3(", "a\x80b", "\xed\xa0\x80", "\xc0\xaf",
		"\xef\xbf\xbd", "-", "--", "a-b", "100%", "50_50", "tab\tbed", "a\x00b",
	}
	db := likeDB(t, rows)
	patterns := []string{
		"", "%", "%%", "_", "__", "%_%", "%__%", "_%", "%_",
		"%a%", "%A%", "a", "A", "a%", "%b", "a_b", "a%b", "%a_b%", "%a%b%",
		"%é%", "%É%", "%e%", "%straße%", "%strasse%", "%ı%", "%i%", "%k%", "%\u212a%",
		"%日本%", "%日_%", "_本%", "%\xff%", "%\xc3%", "%\x80%", "%\xed\xa0\x80%", "%\ufffd%",
		"%-%", "%--%", "100%", "%0\\%%", "50_50", "%\t%", "%a\x00%", "\x00",
	}
	n := compareLike(t, db, patterns)
	if n != len(patterns)*len(rows) {
		t.Fatalf("compared %d pairs, want %d", n, len(patterns)*len(rows))
	}
}

// TestLikeRandom compares likeMatch with SQLite on random words and terms
// drawn from an alphabet of wildcards, mixed case, multi-byte characters and
// invalid bytes. Deterministic seed: a failure reproduces.
func TestLikeRandom(t *testing.T) {
	alphabet := []string{"a", "A", "b", "B", "_", "%", "-", "é", "É", "ß", "日", "\x80", "\xc3", "\xff", "z"}
	r := rand.New(rand.NewSource(5))
	word := func(max int) string {
		var b strings.Builder
		for i := r.Intn(max + 1); i > 0; i-- {
			b.WriteString(alphabet[r.Intn(len(alphabet))])
		}
		return b.String()
	}
	var rows, patterns []string
	for i := 0; i < 300; i++ {
		rows = append(rows, word(8))
	}
	for i := 0; i < 200; i++ {
		w := word(5)
		if i%2 == 0 {
			w = "%" + w + "%" // the shape Search sends
		}
		patterns = append(patterns, w)
	}
	compareLike(t, likeDB(t, rows), patterns)
}

// FuzzLike: `go test -fuzz FuzzLike ./muninn/internal/search/` explores
// beyond TestLikeRandom; plain `go test` runs the seeds.
func FuzzLike(f *testing.F) {
	for _, s := range [][2]string{{"%task_id%", "task-id"}, {"%_called%", "recalled"}, {"a%b_", "AxxBy"}, {"%é%", "É"}, {"%\xc3%", "\xc3("}, {"", ""}} {
		f.Add(s[0], s[1])
	}
	db, err := store.Open(filepath.Join(f.TempDir(), "m.db"))
	if err != nil {
		f.Fatal(err)
	}
	defer db.Close()
	f.Fuzz(func(t *testing.T, pattern, s string) {
		if len(pattern) > likePatternLimit {
			return
		}
		var want bool
		if err := db.QueryRow(`SELECT ? LIKE ?`, s, pattern).Scan(&want); err != nil {
			t.Fatal(err)
		}
		if got := likeMatch(pattern, s); got != want {
			t.Errorf("%q LIKE %q: likeMatch %v, SQLite %v", s, pattern, got, want)
		}
	})
}
