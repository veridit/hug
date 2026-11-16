package tui

import "unicode"

// shortcut is the key picked for one choice: the letter (lower case) and its
// position in the name, counted in runes, so the letter can be marked in
// place. Pos is -1 when every letter of the name was already taken.
type shortcut struct {
	Key rune
	Pos int
}

// assignShortcuts picks one key per name, in order: each name gets its first
// letter that no earlier name (and no reserved key) has taken. So todo,
// doing, blocked, done, dropped become t, d, b, o, r: done and dropped move
// to their second letter because d is gone by then (2026-11-16, user).
// The header, the palette and the help all take their keys from here, so a
// new choice never needs its letter chosen by hand.
func assignShortcuts(names []string, reserved string) []shortcut {
	taken := map[rune]bool{}
	for _, r := range reserved {
		taken[unicode.ToLower(r)] = true
	}
	out := make([]shortcut, len(names))
	for i, name := range names {
		out[i] = shortcut{Pos: -1}
		pos := 0
		for _, r := range name {
			key := unicode.ToLower(r)
			if (unicode.IsLetter(key) || unicode.IsDigit(key)) && !taken[key] {
				taken[key] = true
				out[i] = shortcut{Key: key, Pos: pos}
				break
			}
			pos++
		}
	}
	return out
}
