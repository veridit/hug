package search

import (
	"context"
	"math"
	"regexp"
	"strings"
	"unicode"
)

// Best-passage scoring (c3hszg, Options.PassageMix). Whole-memory BM25
// divides a match by the memory's full length, so a long memory with one
// paragraph about the query loses to a short memory that names the words in
// passing. Scoring the memory's best passage too lets the focused paragraph
// count. recall4 measured the idea on LongMemEval_S (best passage mixed 0.65
// with the whole round: +0.045 all@5 [+0.016, +0.077] held out); whether it
// helps on real memories is what Options.PassageMix exists to measure.
//
// A passage is scored as FTS5 scores a row (bm25(), k1 1.2, b 0.75): the
// corpus's document count, average length and each word's document
// frequency, the passage's own length and term frequencies. That keeps the
// passage and whole-memory scores on one scale, so they mix without
// normalising either.

// PassageBytes is the size passages are cut to, on paragraph breaks.
const PassageBytes = 600

var paraBreak = regexp.MustCompile(`\n\s*\n`)

// Passages cuts body into ~PassageBytes pieces on paragraph breaks; a
// paragraph longer than that is one passage.
func Passages(body string) []string {
	var out []string
	cur := ""
	for _, para := range paraBreak.Split(body, -1) {
		if cur != "" && len(cur)+len(para) > PassageBytes {
			out = append(out, cur)
			cur = ""
		}
		if cur == "" {
			cur = para
		} else {
			cur += "\n\n" + para
		}
	}
	if cur != "" {
		out = append(out, cur)
	}
	return out
}

// ftsTokens splits text the way FTS5's unicode61 tokenizer does closely
// enough for term counts: runs of letters and digits, lower-cased.
func ftsTokens(text string) []string {
	return strings.FieldsFunc(strings.ToLower(text), func(r rune) bool {
		return !(unicode.IsLetter(r) || unicode.IsNumber(r) || unicode.Is(unicode.Co, r))
	})
}

// phraseCount counts phrase (a token sequence) in toks.
func phraseCount(toks, phrase []string) int {
	n := 0
	for i := 0; i+len(phrase) <= len(toks); i++ {
		match := true
		for j, p := range phrase {
			if toks[i+j] != p {
				match = false
				break
			}
		}
		if match {
			n++
		}
	}
	return n
}

// ftsStats reads units_fts's document count and average row length (all
// columns), as bm25() uses them, from the FTS5 structure record.
func (s *Searcher) ftsStats(ctx context.Context) (n, avg float64, ok bool) {
	var blk []byte
	if err := s.DB.QueryRowContext(ctx, `SELECT block FROM units_fts_data WHERE id=1`).Scan(&blk); err != nil {
		return 0, 0, false
	}
	var vals []uint64
	for i := 0; i < len(blk); {
		var v uint64
		for j := 0; j < 9 && i < len(blk); j++ {
			c := blk[i]
			i++
			if j == 8 {
				v = v<<8 | uint64(c)
				break
			}
			v = v<<7 | uint64(c&0x7f)
			if c&0x80 == 0 {
				break
			}
		}
		vals = append(vals, v)
	}
	if len(vals) < 2 || vals[0] == 0 {
		return 0, 0, false
	}
	var tot uint64
	for _, v := range vals[1:] {
		tot += v
	}
	return float64(vals[0]), float64(tot) / float64(vals[0]), true
}

// idfOf is FTS5's BM25 idf for a term in d of n rows, floored as FTS5 does.
func idfOf(n float64, d int) float64 {
	v := math.Log((n - float64(d) + 0.5) / (float64(d) + 0.5))
	if v <= 0 {
		v = 1e-6
	}
	return v
}

// passageScore is FTS5's bm25() (k1 1.2, b 0.75) for one passage of toks:
// the sum over query words of idf * tf*(k1+1) / (tf + k1*(1-b+b*len/avg)).
func passageScore(toks []string, words map[string][]string, idf map[string]float64, avg float64) float64 {
	const k1, b = 1.2, 0.75
	dl := float64(len(toks))
	sc := 0.0
	for w, ph := range words {
		if tf := float64(phraseCount(toks, ph)); tf > 0 {
			sc += idf[w] * tf * (k1 + 1) / (tf + k1*(1-b+b*dl/avg))
		}
	}
	return sc
}

// bestPassage mixes each long matched memory's best-passage BM25 into its
// keyword score: bm = scale * ((1-mix)*whole + mix*best). A memory of one
// passage keeps its score.
func (s *Searcher) bestPassage(ctx context.Context, words []string, o Options, scores map[int64]*acc) error {
	n, avg, ok := s.ftsStats(ctx)
	if !ok {
		return nil
	}
	df := map[string]int{}
	var ids []int64
	for id, a := range scores {
		if len(a.df) == 0 {
			continue
		}
		ids = append(ids, id)
		for _, w := range a.df {
			df[w]++
		}
	}
	if len(ids) == 0 {
		return nil
	}
	idf := map[string]float64{}
	phrase := map[string][]string{}
	for w, d := range df {
		idf[w] = idfOf(n, d)
		phrase[w] = ftsTokens(w)
	}
	set := make(map[int64]bool, len(ids))
	for _, id := range ids {
		set[id] = true
	}
	rows, err := s.DB.QueryContext(ctx, `SELECT id, body FROM units WHERE length(body) > ? AND active=1`, PassageBytes)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var id int64
		var body string
		if err := rows.Scan(&id, &body); err != nil {
			return err
		}
		if !set[id] {
			continue
		}
		ps := Passages(body)
		if len(ps) < 2 {
			continue
		}
		a := scores[id]
		words := make(map[string][]string, len(a.df))
		for _, w := range a.df {
			words[w] = phrase[w]
		}
		best := 0.0
		for _, p := range ps {
			best = math.Max(best, passageScore(ftsTokens(p), words, idf, avg))
		}
		whole := a.bm / o.BM25Scale
		a.bm = o.BM25Scale * ((1-o.PassageMix)*whole + o.PassageMix*best)
	}
	return rows.Err()
}

// PassageBM25 scores text for the single query word w as bestPassage scores
// a passage (for tests: it must equal FTS5's bm25() on a whole row).
func PassageBM25(ctx context.Context, s *Searcher, w, text string) (float64, error) {
	n, avg, ok := s.ftsStats(ctx)
	if !ok {
		return 0, nil
	}
	var d int
	if err := s.DB.QueryRowContext(ctx, `SELECT count(*) FROM units_fts WHERE units_fts MATCH ?`, fts5Quote(w)).Scan(&d); err != nil {
		return 0, err
	}
	return passageScore(ftsTokens(text), map[string][]string{w: ftsTokens(w)}, map[string]float64{w: idfOf(n, d)}, avg), nil
}
