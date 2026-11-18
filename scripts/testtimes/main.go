// Command testtimes prints the slowest tests of a go test -json stream
// (0n4rnx, measurement only): top-level tests by elapsed time, with the
// package total and the sum of parallel tests' time.
//
//	go test -json ./pkg > out.json; go run ./scripts/testtimes out.json 15
package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
)

type event struct {
	Action  string
	Package string
	Test    string
	Elapsed float64
	Output  string
}

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: testtimes FILE.json [N]\nfix: go run ./scripts/testtimes cli.json 15")
		os.Exit(2)
	}
	n := 15
	if len(os.Args) > 2 {
		n, _ = strconv.Atoi(os.Args[2])
	}
	f, err := os.Open(os.Args[1])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer f.Close()
	type result struct {
		name, action string
		elapsed      float64 // go test's own: a parallel test's clock starts when it resumes
		parallel     bool
	}
	var tests []result
	pkg := map[string]result{}
	running := map[string]bool{}
	parallel := map[string]bool{}
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1<<20), 1<<26)
	for sc.Scan() {
		var e event
		if json.Unmarshal(sc.Bytes(), &e) != nil {
			continue
		}
		switch {
		case e.Test == "" && (e.Action == "pass" || e.Action == "fail"):
			pkg[e.Package] = result{name: e.Package, action: e.Action, elapsed: e.Elapsed}
		case e.Test != "" && e.Action == "run":
			running[e.Test] = true
		case e.Test != "" && e.Action == "cont":
			parallel[e.Test] = true
		case e.Test != "" && !strings.Contains(e.Test, "/") && (e.Action == "pass" || e.Action == "fail" || e.Action == "skip"):
			delete(running, e.Test)
			tests = append(tests, result{name: e.Test, action: e.Action, elapsed: e.Elapsed, parallel: parallel[e.Test]})
		case e.Test != "" && (e.Action == "pass" || e.Action == "fail" || e.Action == "skip"):
			delete(running, e.Test)
		}
	}
	sort.Slice(tests, func(i, j int) bool { return tests[i].elapsed > tests[j].elapsed })
	sum, serial, par := 0.0, 0.0, 0
	for _, t := range tests {
		sum += t.elapsed
		if t.parallel {
			par++
		} else {
			serial += t.elapsed
		}
	}
	for _, p := range pkg {
		fmt.Printf("package %s: %s in %.1f s; %d top-level tests (%d parallel), %.1f s summed, %.1f s in serial tests\n", p.name, p.action, p.elapsed, len(tests), par, sum, serial)
	}
	if len(running) > 0 {
		var names []string
		for t := range running {
			names = append(names, t)
		}
		sort.Strings(names)
		fmt.Printf("still running at the end: %s\n", strings.Join(names, ", "))
	}
	fmt.Println("| # | test | s | parallel | result |\n|---|---|---|---|---|")
	for i, t := range tests {
		if i == n {
			break
		}
		fmt.Printf("| %d | %s | %.2f | %v | %s |\n", i+1, t.name, t.elapsed, t.parallel, t.action)
	}
}
