package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// txjkrm check 5: hug upgrade --tags previews the cleanup of a board's
// labels as a diff with counts and writes nothing; --apply writes it in one
// commit, and lint then finds no unlisted tag. Merges (case, - and _, and
// hug.toml [tag_merges]), labels that type/priority/stage already say, and
// tags not on the list are each counted.
func TestUpgradeTagsPreviewThenApply(t *testing.T) {
	t.Parallel()
	a := app(t)
	a.Env["HOME"] = t.TempDir()
	execute(t, a, "init")
	board := filepath.Join(a.Cwd, TaskDir)
	os.WriteFile(filepath.Join(board, "hug.toml"), []byte("tags = [\"compiler\", \"performance\", \"c-backend\", \"design\"]\n\n"+readFile(t, filepath.Join(board, "hug.toml"))+"\n[tag_merges]\nperf = \"performance\"\n"), 0o644)
	git(t, board, "commit", "-qam", "tags")
	mk := func(title string, extra ...string) string {
		args := append([]string{"new", title, "--goal", "g", "--purpose", "p", "--principle", "q", "--accept", "a"}, extra...)
		return execute(t, a, args...)["task"].(Reply)["id"].(string)
	}
	// Variants and a stated merge.
	v := mk("Variants", "--label", "Compiler")
	for _, l := range []string{"perf", "c_backend", "compiler"} {
		execute(t, a, "set", v, "--tag", l)
	}
	// Labels a field already says: type, priority, a done task's stage.
	b := mk("A bug", "--type", "bug", "--priority", "low", "--label", "bug")
	execute(t, a, "set", b, "--tag", "low-pri")
	// A todo task parked by label: deferred, with the label as its reason.
	p := mk("Parked", "--label", "parked")
	// A design label (crab: kept as a listed tag, the only trace someone
	// wanted a design step): stays, with how to make it a stage.
	d := mk("Designish", "--label", "design")
	// Not on the list, not a variant: dropped.
	r := mk("Rare", "--label", "king-ruled")
	execute(t, a, "set", r, "--tag", "compiler")
	// A type label that disagrees with the type: stays, listed with a fix.
	x := mk("Mismatch", "--type", "docs", "--label", "chore")
	// A done task is history: its merges apply, its other labels stay.
	c := mk("Closed", "--type", "bug", "--label", "bug")
	execute(t, a, "set", c, "--tag", "Compiler")
	execute(t, a, "set", c, "--tag", "king-ruled")
	execute(t, a, "drop", c, "history")
	head := strings.TrimSpace(git(t, board, "rev-parse", "HEAD"))

	// Preview: counts and a diff, exit pending, nothing written.
	code, out := runText(a, "upgrade", "--tags")
	if code != UpgradePending {
		t.Fatalf("preview exit %d, want %d (pending):\n%s", code, UpgradePending, out)
	}
	for _, want := range []string{
		"tags: would: 4 labels merge into a listed tag: Compiler -> compiler 2, c_backend -> c-backend 1, perf -> performance 1",
		"tags: would: 2 labels go, the task's type or priority already says it: bug (type bug) 1, low-pri (priority low) 1",
		"tags: would: 1 todo task deferred by its label: parked 1 (" + p + ")",
		"tags: would: 1 label not on the list goes: king-ruled 1",
		"tags: would: 1 label disagrees with the task's field and stays, the only record of it: " + x + " chore (type docs); fix: hug task set ID --type TYPE (or --priority), then hug task set ID --tag LABEL=",
		"tags: would: design: 1 open task keeps the tag; to make one a design stage: hug task design ID \"question\" --for NAME",
		"tags: would: 1 done or dropped task keeps its other labels: closed tasks are history, only merges reach them",
		v + ": labels [Compiler, perf, c_backend, compiler] -> [compiler, performance, c-backend]",
		c + ": labels [bug, Compiler, king-ruled] -> [bug, compiler, king-ruled]",
		"fix: hug upgrade --tags --apply",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("preview lacks %q:\n%s", want, out)
		}
	}
	if now := strings.TrimSpace(git(t, board, "rev-parse", "HEAD")); now != head {
		t.Fatalf("the preview wrote: HEAD %s -> %s", head, now)
	}
	if f := taskFile(t, a, v); !strings.Contains(f, "Compiler") {
		t.Fatalf("the preview changed a task:\n%s", f)
	}
	if _, lint := runText(a, "lint", "--code", "unlisted-tag"); !strings.Contains(lint, "unlisted-tag") {
		t.Fatalf("before the upgrade lint finds no unlisted tag:\n%s", lint)
	}

	// Apply: one commit, lint clean of tag findings.
	ra := execute(t, a, "upgrade", "--tags", "--apply")
	if ra["pending"] == true {
		t.Fatalf("apply still pending: %v", ra)
	}
	if n := strings.TrimSpace(git(t, board, "rev-list", "--count", head+"..HEAD")); n != "1" {
		t.Fatalf("apply made %s commits, want 1", n)
	}
	if msg := strings.TrimSpace(git(t, board, "log", "-1", "--format=%s")); !strings.HasPrefix(msg, "hug upgrade --tags: ") {
		t.Fatalf("commit: %q", msg)
	}
	if f := taskFile(t, a, v); !strings.Contains(f, `labels: ["compiler","performance","c-backend"]`) {
		t.Fatalf("variants not merged:\n%s", f)
	}
	if f := taskFile(t, a, b); !strings.Contains(f, "labels: []") || !strings.Contains(f, "type: bug") {
		t.Fatalf("field labels not gone:\n%s", f)
	}
	if f := taskFile(t, a, p); !strings.Contains(f, `deferred_reason: "label parked (hug upgrade --tags)"`) || !strings.Contains(f, `stage: "deferred"`) || !strings.Contains(f, "labels: []") {
		t.Fatalf("parked not deferred:\n%s", f)
	}
	if f := taskFile(t, a, r); !strings.Contains(f, `labels: ["compiler"]`) {
		t.Fatalf("rare label not dropped:\n%s", f)
	}
	if f := taskFile(t, a, d); !strings.Contains(f, `labels: ["design"]`) {
		t.Fatalf("design tag not kept:\n%s", f)
	}
	if f := taskFile(t, a, x); !strings.Contains(f, `labels: ["chore"]`) || !strings.Contains(f, "type: docs") {
		t.Fatalf("disagreeing label not kept:\n%s", f)
	}
	if f := taskFile(t, a, c); !strings.Contains(f, `labels: ["bug","compiler","king-ruled"]`) {
		t.Fatalf("closed task: merges only:\n%s", f)
	}
	// Lint after: the only tag finding left is the kept disagreement, for
	// a person (crab: its label is the only record of it).
	if _, lint := runText(a, "lint", "--code", "unlisted-tag"); strings.Count(lint, "unlisted tag:") != 1 || !strings.Contains(lint, "unlisted tag: chore") {
		t.Fatalf("after --apply lint finds more than the kept disagreement:\n%s", lint)
	}
	// The person settles it; then lint is clean of tag findings.
	execute(t, a, "set", x, "--type", "chore")
	execute(t, a, "set", x, "--tag", "chore=")
	if _, lint := runText(a, "lint", "--code", "unlisted-tag"); strings.Contains(lint, "unlisted tag") {
		t.Fatalf("after --apply lint still finds unlisted tags:\n%s", lint)
	}
	if code, out := runText(a, "upgrade", "--tags"); code != 0 || !strings.Contains(out, "tags: nothing to do") {
		t.Fatalf("second preview: exit %d\n%s", code, out)
	}
}

// With no list in hug.toml, the preview proposes one: the tags on enough
// tasks (after variant merges), and --apply writes it.
func TestUpgradeTagsProposesAList(t *testing.T) {
	t.Parallel()
	a := app(t)
	a.Env["HOME"] = t.TempDir()
	execute(t, a, "init")
	for i := 0; i < 3; i++ {
		execute(t, a, "new", "Lex "+string(rune('a'+i)), "--goal", "g", "--purpose", "p", "--principle", "q", "--accept", "a", "--label", []string{"parser", "Parser", "parser"}[i])
	}
	execute(t, a, "new", "Once", "--goal", "g", "--purpose", "p", "--principle", "q", "--accept", "a", "--label", "oneoff")
	code, out := runText(a, "upgrade", "--tags")
	// parser, Parser, parser: 3 tasks once case does not count, so on the
	// default 3+ only after merging; oneoff is on 1.
	if code != UpgradePending || !strings.Contains(out, `tags: would: hug.toml gets tags = ["parser"] (1 tag on 3+ tasks after merging)`) {
		t.Fatalf("no list proposed (exit %d):\n%s", code, out)
	}
	if _, out := runText(a, "upgrade", "--tags", "--min", "1"); !strings.Contains(out, `tags: would: hug.toml gets tags = ["parser", "oneoff"] (2 tags on 1+ tasks after merging)`) {
		t.Fatalf("--min 1 not used:\n%s", out)
	}
	execute(t, a, "upgrade", "--tags", "--apply")
	if toml := readFile(t, filepath.Join(a.Cwd, TaskDir, "hug.toml")); !strings.Contains(toml, `tags = ["parser"]`) {
		t.Fatalf("hug.toml:\n%s", toml)
	}
	if _, lint := runText(a, "lint", "--code", "unlisted-tag"); strings.Contains(lint, "unlisted tag") {
		t.Fatalf("lint after apply:\n%s", lint)
	}
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	b, e := os.ReadFile(path)
	if e != nil {
		t.Fatal(e)
	}
	return string(b)
}
