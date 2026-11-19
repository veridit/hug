package search

import (
	"context"
	"strings"
	"unicode/utf8"
)

// likePatternLimit is SQLite's default SQLITE_MAX_LIKE_PATTERN_LENGTH. A
// longer pattern is an error in SQLite, so Search sends it to SQLite as
// before instead of matching it here.
const likePatternLimit = 50000

// likeConcepts adds to into every concept whose term is LIKE one of patterns,
// exactly as `SELECT id FROM concepts WHERE term LIKE ?` per pattern would.
// One read of the concepts table per call replaces one full-table LIKE scan
// per word: capture run 8476 spent 53 s of its 104 s of planning in 3,654
// such scans of 84,169 terms (b5jks7). Terms are read on every call, never
// kept: a long-lived Searcher (the MCP server) sees the index change.
func (s *Searcher) likeConcepts(ctx context.Context, patterns []string, into map[int]bool) error {
	ps := make([]likePattern, len(patterns))
	for i, p := range patterns {
		ps[i] = newLikePattern(p)
	}
	rows, err := s.DB.QueryContext(ctx, `SELECT id, term FROM concepts`)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var id int
		var term any
		if err := rows.Scan(&id, &term); err != nil {
			return err
		}
		t, ok := term.(string)
		if !ok {
			continue // LIKE is never true for a BLOB or NULL
		}
		t = beforeNUL(t)
		valid := utf8.ValidString(t)
		folded := ""
		for _, p := range ps {
			if p.fast && valid {
				if folded == "" {
					folded = asciiLower(t)
				}
				if strings.Contains(folded, p.lit) {
					into[id] = true
					break
				}
			} else if likeCompare(likeText{s: p.raw}, likeText{s: t}) == likeMatched {
				into[id] = true
				break
			}
		}
	}
	return rows.Err()
}

// likePattern is a LIKE pattern. fast marks "%lit%" with valid UTF-8 and no
// wildcard in lit: against valid UTF-8 text that is a substring test after
// ASCII case folding, which is what patternCompare computes for it. A byte
// match of valid UTF-8 inside valid UTF-8 always starts on a character
// boundary, and folding bytes below 0x80 folds exactly the ASCII letters.
type likePattern struct {
	raw, lit string
	fast     bool
}

func newLikePattern(p string) likePattern {
	p = beforeNUL(p)
	out := likePattern{raw: p}
	if len(p) >= 2 && p[0] == '%' && p[len(p)-1] == '%' {
		lit := p[1 : len(p)-1]
		if !strings.ContainsAny(lit, "%_") && utf8.ValidString(lit) {
			out.lit, out.fast = asciiLower(lit), true
		}
	}
	return out
}

// likeMatch reports whether `s LIKE pattern` is true in SQLite's built-in
// LIKE without an ESCAPE clause: '%' matches any sequence, '_' any one
// character, and only ASCII letters fold case.
func likeMatch(pattern, s string) bool {
	return likeCompare(likeText{s: beforeNUL(pattern)}, likeText{s: beforeNUL(s)}) == likeMatched
}

// SQLite reads LIKE operands as C strings: both end at the first NUL byte.
func beforeNUL(s string) string {
	if i := strings.IndexByte(s, 0); i >= 0 {
		return s[:i]
	}
	return s
}

func asciiLower(s string) string {
	for i := 0; i < len(s); i++ {
		if 'A' <= s[i] && s[i] <= 'Z' {
			b := []byte(s)
			for j := i; j < len(b); j++ {
				if 'A' <= b[j] && b[j] <= 'Z' {
					b[j] += 'a' - 'A'
				}
			}
			return string(b)
		}
	}
	return s
}

func lowerASCII(c uint32) uint32 {
	if 'A' <= c && c <= 'Z' {
		return c + 'a' - 'A'
	}
	return c
}

func upperASCII(c uint32) uint32 {
	if 'a' <= c && c <= 'z' {
		return c - ('a' - 'A')
	}
	return c
}

// Results of likeCompare, as SQLITE_MATCH, SQLITE_NOMATCH and
// SQLITE_NOWILDCARDMATCH.
const (
	likeMatched = iota
	likeNoMatch
	likeNoWildcardMatch
)

// likeText is a read position in a NUL-free string; reading past the end
// yields 0, the C string terminator.
type likeText struct {
	s string
	i int
}

// sqlite3Utf8Trans1 is SQLite's table for the payload bits of a UTF-8 lead
// byte 0xc0..0xff.
var sqlite3Utf8Trans1 = [64]byte{
	0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07,
	0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f,
	0x10, 0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17,
	0x18, 0x19, 0x1a, 0x1b, 0x1c, 0x1d, 0x1e, 0x1f,
	0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07,
	0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f,
	0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07,
	0x00, 0x01, 0x02, 0x03, 0x00, 0x01, 0x00, 0x00,
}

// read is sqlite3Utf8Read: it decodes leniently (any run of continuation
// bytes, stray continuation bytes as themselves, overlong forms and
// surrogates as U+FFFD), so invalid UTF-8 compares as SQLite compares it.
func (t *likeText) read() uint32 {
	if t.i >= len(t.s) {
		t.i++
		return 0
	}
	c := uint32(t.s[t.i])
	t.i++
	if c >= 0xc0 {
		c = uint32(sqlite3Utf8Trans1[c-0xc0])
		for t.i < len(t.s) && t.s[t.i]&0xc0 == 0x80 {
			c = c<<6 + uint32(0x3f&t.s[t.i])
			t.i++
		}
		if c < 0x80 || c&0xFFFFF800 == 0xD800 {
			c = 0xFFFD
		}
	}
	return c
}

// likeCompare is SQLite's patternCompare (func.c) for LIKE with no ESCAPE:
// matchAll '%', matchOne '_', noCase, and no escape or [...] set.
func likeCompare(p, s likeText) int {
	for {
		c := p.read()
		if c == 0 {
			break
		}
		if c == '%' {
			// Skip "%" and "_" runs; each "_" consumes one input character.
			for {
				c = p.read()
				if c != '%' && c != '_' {
					break
				}
				if c == '_' && s.read() == 0 {
					return likeNoWildcardMatch
				}
			}
			if c == 0 {
				return likeMatched
			}
			if c < 0x80 {
				// strcspn for either case of c, then match the rest from there.
				up, lo := byte(upperASCII(c)), byte(lowerASCII(c))
				for {
					for s.i < len(s.s) && s.s[s.i] != up && s.s[s.i] != lo {
						s.i++
					}
					if s.i >= len(s.s) {
						break
					}
					s.i++
					if r := likeCompare(p, s); r != likeNoMatch {
						return r
					}
				}
			} else {
				for {
					c2 := s.read()
					if c2 == 0 {
						break
					}
					if c2 != c {
						continue
					}
					if r := likeCompare(p, s); r != likeNoMatch {
						return r
					}
				}
			}
			return likeNoWildcardMatch
		}
		c2 := s.read()
		if c == c2 {
			continue
		}
		if c < 0x80 && c2 < 0x80 && lowerASCII(c) == lowerASCII(c2) {
			continue
		}
		if c == '_' && c2 != 0 {
			continue
		}
		return likeNoMatch
	}
	if s.i >= len(s.s) {
		return likeMatched
	}
	return likeNoMatch
}
