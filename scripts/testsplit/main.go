package main

// Per test: git processes, their summed wall time (top-level only: a git
// whose parent is another git is not counted twice), from GIT_TRACE2_EVENT
// files. A process belongs to the test whose TempDir name appears in its argv.
import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

var testRe = regexp.MustCompile(`(Test[A-Za-z0-9_]+?)[0-9]{6,}`)

func main() {
	dir := os.Args[1]
	ents, _ := os.ReadDir(dir)
	type agg struct {
		n    int
		wall float64
	}
	per := map[string]*agg{}
	for _, e := range ents {
		f, err := os.Open(filepath.Join(dir, e.Name()))
		if err != nil {
			continue
		}
		sc := bufio.NewScanner(f)
		sc.Buffer(make([]byte, 1<<20), 1<<24)
		var test string
		var nested bool
		var wall float64
		for sc.Scan() {
			var ev map[string]any
			if json.Unmarshal(sc.Bytes(), &ev) != nil {
				continue
			}
			sid, _ := ev["sid"].(string)
			switch ev["event"] {
			case "start":
				nested = strings.Count(sid, "/") > 0
				b, _ := json.Marshal(ev["argv"])
				if m := testRe.FindStringSubmatch(string(b)); m != nil {
					test = m[1]
				}
			case "def_repo":
				if test == "" {
					b, _ := json.Marshal(ev["worktree"])
					if m := testRe.FindStringSubmatch(string(b)); m != nil {
						test = m[1]
					}
				}
			case "atexit":
				wall, _ = ev["t_abs"].(float64)
			}
		}
		f.Close()
		if nested {
			continue
		}
		if test == "" {
			test = "(none)"
		}
		a := per[test]
		if a == nil {
			a = &agg{}
			per[test] = a
		}
		a.n++
		a.wall += wall
	}
	var keys []string
	for k := range per {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool { return per[keys[i]].wall > per[keys[j]].wall })
	for _, k := range keys {
		fmt.Printf("%s\t%d\t%.2f\n", k, per[k].n, per[k].wall)
	}
}
