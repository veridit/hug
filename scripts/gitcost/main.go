// Command gitcost sums the wall time of git processes from a
// GIT_TRACE2_EVENT folder (0n4rnx, measurement only): processes started by
// the test binary (not by another git) grouped by subcommand, with counts,
// so a test's time can be split into git and everything else.
//
//	GIT_TRACE2_EVENT=$PWD/tr go test -run X ./pkg; go run ./scripts/gitcost tr
package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type event struct {
	Event    string   `json:"event"`
	Argv     []string `json:"argv"`
	Ancestry []string `json:"ancestry"`
	TAbs     float64  `json:"t_abs"`
}

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: gitcost TRACEDIR\nfix: go run ./scripts/gitcost tr")
		os.Exit(2)
	}
	files, _ := filepath.Glob(filepath.Join(os.Args[1], "*"))
	type agg struct {
		n   int
		sec float64
	}
	top, nested := map[string]*agg{}, map[string]*agg{}
	total, totalN := 0.0, 0
	for _, f := range files {
		h, err := os.Open(f)
		if err != nil {
			continue
		}
		sc := bufio.NewScanner(h)
		sc.Buffer(make([]byte, 1<<20), 1<<24)
		var argv, anc []string
		end := 0.0
		for sc.Scan() {
			var e event
			if json.Unmarshal(sc.Bytes(), &e) != nil {
				continue
			}
			switch e.Event {
			case "start":
				argv = e.Argv
			case "cmd_ancestry":
				anc = e.Ancestry
			case "exit", "atexit":
				if e.TAbs > end {
					end = e.TAbs
				}
			}
		}
		h.Close()
		verb := subcommand(argv)
		m := top
		if len(anc) > 0 && strings.HasPrefix(strings.ToLower(filepath.Base(anc[0])), "git") {
			m = nested
		} else {
			total += end
			totalN++
		}
		if m[verb] == nil {
			m[verb] = &agg{}
		}
		m[verb].n++
		m[verb].sec += end
	}
	print := func(title string, m map[string]*agg) {
		var ks []string
		for k := range m {
			ks = append(ks, k)
		}
		sort.Slice(ks, func(i, j int) bool { return m[ks[i]].sec > m[ks[j]].sec })
		fmt.Printf("%s\n| command | n | s | ms each |\n|---|---|---|---|\n", title)
		for _, k := range ks {
			fmt.Printf("| %s | %d | %.2f | %.0f |\n", k, m[k].n, m[k].sec, 1000*m[k].sec/float64(m[k].n))
		}
	}
	fmt.Printf("git processes started by the test process: %d, %.1f s wall summed\n", totalN, total)
	print("started by the test process (includes their children)", top)
	print("started by git (inside the above)", nested)
}

// subcommand is git's verb after -C DIR, -c K=V and --git-dir options.
func subcommand(argv []string) string {
	for i := 1; i < len(argv); i++ {
		a := argv[i]
		switch {
		case a == "-C" || a == "-c":
			i++
		case strings.HasPrefix(a, "--git-dir") || strings.HasPrefix(a, "--literal-pathspecs") || strings.HasPrefix(a, "--no-"):
		default:
			if a == "maintenance" || a == "rev-parse" || a == "config" {
				if i+1 < len(argv) {
					return a + " " + argv[i+1]
				}
			}
			return a
		}
	}
	return "?"
}
