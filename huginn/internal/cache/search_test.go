package cache

import (
	"context"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func taskFile(id, title, aliases, goal, notes string) []byte {
	return []byte("---\nid: " + id + "\ntitle: \"" + title + "\"\nstatus: todo\naliases: [" + aliases + "]\n---\n\n## Goal\n" + goal + "\n\n## Acceptance\n- [ ] works\n\n## Notes\n" + notes + "\n")
}

// 49n9kw checks 1, 2, 4: a title match ranks first and names its field and
// a snippet; an alias ranks above a note that cites it; the index follows a
// rewrite, a rename and a removal with no rebuild.
func TestSearchRanksAndFollowsWrites(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	put(t, root, "a.aaaaaa.oom.md", taskFile("aaaaaa", "Memory limit exit 137 in the worker", "", "Stop the OOM kill", ""))
	put(t, root, "b.bbbbbb.other.md", taskFile("bbbbbb", "Logging", "", "The exit code 137 means the memory limit was hit", ""))
	put(t, root, "c.cccccc.statbus.md", taskFile("cccccc", "Import units", `"STATBUS-402"`, "Units import", ""))
	put(t, root, "d.dddddd.cites.md", taskFile("dddddd", "Follow-up", "", "Later", "Continues STATBUS-402 after the import"))
	c := openTest(t, root, filepath.Join(t.TempDir(), "cache.db"), &fakeGit{head: "h1"})
	defer c.Close()
	if _, e := c.List(ctx); e != nil {
		t.Fatal(e)
	}
	hits, e := c.Search(ctx, "memory limit exit 137", 0)
	if e != nil {
		t.Fatal(e)
	}
	if len(hits) != 2 || hits[0].Path != "a.aaaaaa.oom.md" || hits[0].Field != "title" || !strings.Contains(hits[0].Snippet, "Memory limit exit 137") || hits[1].Field != "goal" {
		t.Fatalf("title first: %+v", hits)
	}
	hits, _ = c.Search(ctx, "statbus-402", 0)
	if len(hits) != 2 || hits[0].Path != "c.cccccc.statbus.md" || hits[0].Field != "aliases" || hits[1].Path != "d.dddddd.cites.md" || hits[1].Field != "notes" {
		t.Fatalf("alias before a note citing it: %+v", hits)
	}
	// A note added is found next time; a removed file and a renamed one follow.
	put(t, root, "b.bbbbbb.other.md", taskFile("bbbbbb", "Logging", "", "Plain", "now mentions zebra"))
	os.Rename(filepath.Join(root, "c.cccccc.statbus.md"), filepath.Join(root, "c.cccccc.moved.md"))
	os.Remove(filepath.Join(root, "d.dddddd.cites.md"))
	if _, e := c.List(ctx); e != nil {
		t.Fatal(e)
	}
	if hits, _ = c.Search(ctx, "zebra", 0); len(hits) != 1 || hits[0].Path != "b.bbbbbb.other.md" || hits[0].Field != "notes" {
		t.Fatalf("note added: %+v", hits)
	}
	if hits, _ = c.Search(ctx, "STATBUS-402", 0); len(hits) != 1 || hits[0].Path != "c.cccccc.moved.md" {
		t.Fatalf("after rename and removal: %+v", hits)
	}
	if hits, _ = c.Search(ctx, "137", 0); len(hits) != 1 || hits[0].Path != "a.aaaaaa.oom.md" {
		t.Fatalf("old goal still indexed: %+v", hits)
	}
	// Operator characters are words, never syntax.
	if _, e := c.Search(ctx, `"NEAR( OR * -x`, 5); e != nil {
		t.Fatalf("query syntax leaked: %v", e)
	}
}

// 49n9kw: a write to the search index looks its row up by an index, never
// by scanning. "DELETE FROM search WHERE path=?" with path an UNINDEXED FTS5
// column read every row, which made a cold ls at 10,000 tasks take 91-94 s
// instead of 3.7-5.1 s. Every statement the index runs per task file must
// use an index: a table SEARCH, or an FTS5 plan with a constraint (the
// "INDEX 0:=" of a rowid lookup; "INDEX 0:" alone is a full scan).
func TestSearchWritesNeverScan(t *testing.T) {
	c := openTest(t, t.TempDir(), filepath.Join(t.TempDir(), "cache.db"), &fakeGit{head: "h1"})
	defer c.Close()
	fullScan := regexp.MustCompile(`^SCAN \S+ VIRTUAL TABLE INDEX \d+:$|^SCAN \S+$`)
	for _, q := range searchWrites {
		args := make([]any, strings.Count(q, "?"))
		for i := range args {
			args[i] = "x"
		}
		rows, e := c.db.Query("EXPLAIN QUERY PLAN "+q, args...)
		if e != nil {
			t.Fatalf("%s: %v", q, e)
		}
		var plan []string
		for rows.Next() {
			var id, parent, unused int
			var detail string
			if e := rows.Scan(&id, &parent, &unused, &detail); e != nil {
				t.Fatal(e)
			}
			plan = append(plan, detail)
		}
		rows.Close()
		for _, d := range plan {
			if fullScan.MatchString(d) {
				t.Errorf("%s scans: %v", q, plan)
			}
		}
	}
}
