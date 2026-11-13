package run

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	"github.com/veridit/hug/muninn/internal/config"
	"github.com/veridit/hug/muninn/internal/harness"
	"github.com/veridit/hug/muninn/internal/tidy"
)

func placeCmd() *cobra.Command {
	var selection tidy.PlaceSelection
	var out, sids string
	cmd := &cobra.Command{Use: "place", Short: "Ask the capture model where selected memories belong, writing an owner-review plan only", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
		if out == "" {
			return fmt.Errorf("--out is required")
		}
		// Fail before paid calls if either review artifact already exists.
		for _, path := range []string{out, out + ".md"} {
			if _, err := os.Lstat(path); err == nil {
				return fmt.Errorf("review file already exists: %s", path)
			} else if !os.IsNotExist(err) {
				return err
			}
		}
		if sids != "" {
			ids, err := readPlaceSIDs(sids)
			if err != nil {
				return err
			}
			selection.SIDs = ids
		}
		cfg, err := config.Load()
		if err != nil {
			return err
		}
		home, err := os.UserHomeDir()
		if err != nil {
			return err
		}
		for _, instance := range harness.DiscoverJcode(home, cfg.JcodeHomes) {
			selection.SessionDirs = append(selection.SessionDirs, filepath.Join(instance.Home, "sessions"))
		}
		g, err := tidy.ReadGraph(cmd.Context(), config.DBPath())
		if err != nil {
			return err
		}
		units, err := tidy.SelectPlace(cmd.Context(), g, selection)
		if err != nil {
			return err
		}
		catalog, err := tidy.DiscoverPlaces(cmd.Context(), home, []string{filepath.Join(home, "Projects"), filepath.Join(home, "Veridit", "src")})
		if err != nil {
			return err
		}
		fmt.Fprintf(cmd.ErrOrStderr(), "Placement: %d selected memories, %d candidate places; sequential batches of at most 20.\n", len(units), len(catalog.Candidates))
		plan, placeErr := tidy.Place(cmd.Context(), g, units, catalog, cfg.Extract)
		plan.Database = config.DBPath()
		if err = saveTidyPlan(out, plan); err != nil {
			return err
		}
		decisions := map[string]int{}
		targets := map[string]int{}
		for _, e := range plan.Entries {
			decisions[e.Decision]++
			if e.Target != "" {
				targets[e.Target]++
			}
		}
		fmt.Fprintf(cmd.OutOrStdout(), "Placement: %d selected; move=%d stay=%d unsure=%d\n", len(plan.Entries), decisions["move"], decisions["stay"], decisions["unsure"])
		keys := make([]string, 0, len(targets))
		for target := range targets {
			keys = append(keys, target)
		}
		sort.Strings(keys)
		for _, target := range keys {
			fmt.Fprintf(cmd.OutOrStdout(), "%s: %d\n", target, targets[target])
		}
		r := plan.Placement
		fmt.Fprintf(cmd.OutOrStdout(), "Model: %s; calls=%d failures=%d; tokens prompt=%d completion=%d total=%d; wall_ms=%d\n", r.Model, r.Calls, r.Failures, r.Usage.PromptTokens, r.Usage.CompletionTokens, r.Usage.TotalTokens, r.WallMS)
		fmt.Fprintf(cmd.OutOrStdout(), "Plan: %s\nSummary: %s.md\nReview before apply. No memory or index writes performed.\n", out, out)
		return placeErr
	}}
	cmd.Flags().StringArrayVar(&selection.FromScopes, "from-scope", nil, "exact stored source scope (repeatable, OR within this filter)")
	cmd.Flags().StringArrayVar(&selection.CwdOrigins, "cwd-origin", nil, "exact absolute capture:jcode session working directory (repeatable)")
	cmd.Flags().StringVar(&sids, "sids", "", "SID file, one SID per line (first column of TSV also accepted)")
	cmd.Flags().IntVar(&selection.Max, "max", 100, "maximum matching memories in SID order (0 selects all)")
	cmd.Flags().StringVar(&out, "out", "", "required new JSON plan path; also writes .md with every memory")
	return cmd
}

func readPlaceSIDs(path string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var out []string
	s := bufio.NewScanner(f)
	s.Buffer(make([]byte, 4096), 1<<20)
	for s.Scan() {
		fields := strings.Fields(s.Text())
		if len(fields) > 0 && !strings.HasPrefix(fields[0], "#") {
			out = append(out, fields[0])
		}
	}
	if err := s.Err(); err != nil {
		return nil, err
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("--sids file has no SIDs")
	}
	return out, nil
}
