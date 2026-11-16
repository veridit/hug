package task

import "testing"

func TestSlugShort(t *testing.T) {
	for in, want := range map[string]string{
		"Close bug classes by construction: the compiler proves them impossible": "close-bug-classes-construction",
		"Shape ratchet: counts of bug shapes may only go down":                   "shape-ratchet-counts-bug",
		"HAMT mono-transitivity + dict/hashed SIGSEGV":                           "hamt-mono-transitivity-dict",
		"The":       "the",
		"Fix retry": "fix-retry",
		"Supercalifragilisticexpialidocious-and-even-longer-words here": "supercalifragilisticexpialidocio",
	} {
		if got := Slug(in); got != want {
			t.Errorf("Slug(%q) = %q, want %q", in, got, want)
		}
	}
}
