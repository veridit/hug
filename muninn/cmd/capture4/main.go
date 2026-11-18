// Command capture4 replays real capture calls (ez50rb check 3) through the
// extractor of the tree it is built from: the same prompt, known memories
// and places the hook sent, with the hook's token cap. It writes one JSON
// line per item with the candidates and the agent names each one uses
// (muninn/internal/agents, names from the jcode homes). Offline evaluation
// only; paid calls; nothing is stored.
//
//	go run ./muninn/cmd/capture4 -items items.jsonl -arm new > new.jsonl
package main

import (
	"bufio"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/veridit/hug/muninn/internal/agents"
	"github.com/veridit/hug/muninn/internal/config"
	"github.com/veridit/hug/muninn/internal/extract"
)

type item struct {
	ID         string          `json:"id"`
	Kind       string          `json:"kind"`
	Session    string          `json:"session"`
	Transcript string          `json:"transcript"`
	Known      []extract.Known `json:"known"`
	Places     []extract.Place `json:"places"`
	OffRaw     string          `json:"off_raw,omitempty"`
}

type cand struct {
	extract.Candidate
	Agents []string `json:"agents,omitempty"`
}

type row struct {
	ID         string `json:"id"`
	Kind       string `json:"kind"`
	Arm        string `json:"arm"`
	Candidates []cand `json:"candidates"`
	Rejected   int    `json:"rejected"`
	Tokens     int    `json:"tokens"`
	Error      string `json:"error,omitempty"`
}

func main() {
	items := flag.String("items", "", "items.jsonl from sample.py")
	arm := flag.String("arm", "", "label for this tree's prompt")
	par := flag.Int("p", 2, "parallel calls")
	scan := flag.Bool("scan", false, "no calls: print the agent names each transcript uses")
	rematch := flag.String("rematch", "", "no calls: re-run the agent matcher on this earlier output (ARM.jsonl) with the items' transcripts")
	triggers := flag.Bool("triggers", false, "ask the extractor for when-then triggers (extract.triggers, jrj1vq)")
	reparse := flag.Bool("reparse", false, "no calls: parse each item's recorded off_raw with this tree's parser (jrj1vq arm OFF)")
	flag.Parse()
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	xc := cfg.Extract
	xc.MaxTokens = 1024 // the hook's cap (app/capture.go)
	xc.TimeoutSeconds = 120
	xc.Triggers = *triggers
	home, _ := os.UserHomeDir()
	ss := agents.Sessions([]string{filepath.Join(home, ".jcode")})
	names := make([]string, 0, len(ss))
	for n := range ss {
		names = append(names, n)
	}
	sort.Strings(names)
	m := agents.New(names)

	f, err := os.Open(*items)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	var its []item
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1<<20), 8<<20)
	for sc.Scan() {
		var it item
		if err := json.Unmarshal(sc.Bytes(), &it); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		its = append(its, it)
	}
	if *scan {
		for _, it := range its {
			fmt.Printf("%s\t%s\t%v\n", it.ID, it.Kind, m.Find(it.Transcript))
		}
		return
	}
	if *reparse {
		enc := json.NewEncoder(os.Stdout)
		for _, it := range its {
			r := row{ID: it.ID, Kind: it.Kind, Arm: *arm, Candidates: []cand{}}
			cs, rejected := extract.Parse(it.OffRaw)
			r.Rejected = rejected
			// The same filter Run applies after Parse: a same_as/updates
			// sid must be one the extractor was shown.
			known := map[string]bool{}
			for _, k := range it.Known {
				known[k.SID] = true
			}
			for _, c := range cs {
				if (c.SameAs != "" && !known[c.SameAs]) || (c.Updates != "" && !known[c.Updates]) {
					r.Rejected++
					continue
				}
				r.Candidates = append(r.Candidates, cand{Candidate: c, Agents: m.FindIn(c.Content, it.Transcript)})
			}
			enc.Encode(r)
		}
		return
	}
	if *rematch != "" {
		// Same candidates the judge saw, today's matcher: a matcher change
		// is measured on judged outputs without a new model call (typq1e).
		byID := map[string]item{}
		for _, it := range its {
			byID[it.ID] = it
		}
		prev, err := os.Open(*rematch)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		defer prev.Close()
		ps := bufio.NewScanner(prev)
		ps.Buffer(make([]byte, 1<<20), 8<<20)
		enc := json.NewEncoder(os.Stdout)
		for ps.Scan() {
			var r row
			if err := json.Unmarshal(ps.Bytes(), &r); err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
			for i := range r.Candidates {
				r.Candidates[i].Agents = m.FindIn(r.Candidates[i].Content, byID[r.ID].Transcript)
			}
			enc.Encode(r)
		}
		return
	}
	out := make([]row, len(its))
	var wg sync.WaitGroup
	sem := make(chan bool, *par)
	for i, it := range its {
		wg.Add(1)
		go func() {
			defer wg.Done()
			sem <- true
			defer func() { <-sem }()
			r := row{ID: it.ID, Kind: it.Kind, Arm: *arm, Candidates: []cand{}}
			for try := 0; try < 3; try++ {
				ctx, cancel := context.WithTimeout(context.Background(), 150*time.Second)
				res, err := extract.Run(ctx, xc, it.Transcript, it.Known, it.Places...)
				cancel()
				if err != nil {
					r.Error = err.Error()
					continue
				}
				r.Error, r.Rejected, r.Tokens = "", res.Rejected, res.Usage.TotalTokens
				for _, c := range res.Candidates {
					r.Candidates = append(r.Candidates, cand{Candidate: c, Agents: m.FindIn(c.Content, it.Transcript)})
				}
				break
			}
			out[i] = r
			fmt.Fprintf(os.Stderr, "%s %s %d candidates %s\n", *arm, it.ID, len(r.Candidates), r.Error)
		}()
	}
	wg.Wait()
	enc := json.NewEncoder(os.Stdout)
	for _, r := range out {
		enc.Encode(r)
	}
}
