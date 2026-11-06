package main

import (
	"github.com/veridit/muninn/internal/search"
	"reflect"
	"strings"
	"testing"
)

func TestFocusedPreservesShortQuery(t *testing.T) {
	q := "exact_identifier next"
	if focused(q) != q {
		t.Fatal("short query changed")
	}
}
func TestFocusedUsesIdentifiersAndLastParagraph(t *testing.T) {
	q := strings.Repeat("background text ", 60) + "\n\nInspect internal/search/search.go and exact_identifier now."
	got := focused(q)
	if !strings.Contains(got, "internal/search/search.go") || !strings.Contains(got, "exact_identifier") {
		t.Fatalf("lost identifiers: %q", got)
	}
	if strings.Count(got, "background") > 0 {
		t.Fatalf("promoted ordinary prose as identifier: %q", got)
	}
}
func TestBenchmarkGatesAreIndependent(t *testing.T) {
	rs := []search.Result{{SemanticOn: true, Semantic: .7, Total: 10}, {SemanticOn: true, Semantic: .4, Total: 9}, {SemanticOn: true, Semantic: .8, Total: 4}}
	for _, tc := range []struct {
		v      string
		totals []float64
	}{{"baseline", []float64{10}}, {"no-semantic-gate", []float64{10, 9}}, {"no-relative-gate", []float64{10, 4}}} {
		got := benchmarkGatesFor(rs, tc.v)
		var totals []float64
		for _, r := range got {
			totals = append(totals, r.Total)
		}
		if !reflect.DeepEqual(totals, tc.totals) {
			t.Fatalf("%s: %v", tc.v, totals)
		}
	}
	if got := benchmarkGates(nil); len(got) != 0 {
		t.Fatal("empty input produced hits")
	}
}
func TestBenchmarkUngatedReferenceOptions(t *testing.T) {
	base := benchmarkOptions("baseline")
	if base.MinScore != 6 || base.SemanticFloor != .5 || base.Limit != 10 {
		t.Fatal("baseline options changed")
	}
	dense := benchmarkOptions("semantic-retrieval")
	if dense.MinScore != 0 || dense.SemanticFloor != -1 || !dense.DisableFTS || !dense.DisableConcepts || dense.DisableSemantic {
		t.Fatalf("not a flat dense reference: %+v", dense)
	}
	lexical := benchmarkOptions("fts-retrieval")
	if lexical.MinScore != 0 || !lexical.DisableSemantic || !lexical.DisableConcepts || lexical.DisableFTS {
		t.Fatalf("not a lexical reference: %+v", lexical)
	}
}

func TestKeyphrasesDeterministicAndCapped(t *testing.T) {
	s := "zebra zebra quartz quartz violet deployment database configuration snapshots scopes semantic embedding memories sources passages"
	a, b := keyphrases(s), keyphrases(s)
	if a != b || len(strings.Fields(a)) != 12 {
		t.Fatalf("nondeterministic or uncapped: %q", a)
	}
	if !strings.HasPrefix(a, "quartz zebra") {
		t.Fatalf("frequency tie-break: %q", a)
	}
}
