package task

import (
	"path"
	"strings"

	"github.com/veridit/hug/huginn/internal/id"
)

// Layout 3 (w40cmk, Jørgen 2026-11-18 01:27 and 01:32): a task lives at
// C/ID.md for its whole life, C the first character of its id, and its
// files sit in C/ID/. The path follows from the id alone, so a link to a
// task (github.com/veridit/hug/blob/hug/tasks/3/37bryg.md) never breaks.
// Rank and group are front-matter fields: a reorder or a group move edits
// a line and renames nothing. The shard keeps folders near 330 entries at
// 10,000 tasks (GitHub's web listing stops at about 1,000).

// PathFor is where the task with id i lives (slash separated).
func PathFor(i string) string { return i[:1] + "/" + i + ".md" }

// FolderFor is where the files of the task with id i live.
func FolderFor(i string) string { return i[:1] + "/" + i }

// IDPath says whether p (slash separated, relative to the board) is a
// layout-3 task path, and gives its id: C/ID.md with C the id's first
// character.
func IDPath(p string) (string, bool) {
	dir, base := path.Split(p)
	n, ok := strings.CutSuffix(base, ".md")
	if !ok || !id.Valid(n) || dir != n[:1]+"/" {
		return "", false
	}
	return n, true
}

// IDOf is the id a task path names, in either layout: C/ID.md, or
// [group/][rank.]id.slug.md. ok is false for a path that names no task.
func IDOf(p string) (string, bool) {
	p = path.Clean(strings.ReplaceAll(p, "\\", "/"))
	if i, ok := IDPath(p); ok {
		return i, true
	}
	_, i, _, e := ParseName(p)
	return i, e == nil && i != ""
}

// OldName says whether p is named the layout-2 way ([group/]rank.id.slug.md
// or [group/]id.slug.md), which hug upgrade moves to PathFor.
func OldName(p string) bool {
	if _, ok := IDPath(p); ok {
		return false
	}
	_, i, _, e := ParseName(p)
	return e == nil && i != ""
}
