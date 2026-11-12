package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/veridit/hug/muninn/internal/app"
	"github.com/veridit/hug/muninn/internal/config"
	"github.com/veridit/hug/muninn/internal/tidy"
)

// tidyCmd is registered only on the executable's owner CLI. It never uses a
// session binding for scanning and never reindexes or loads an embedder there.
func tidyCmd() *cobra.Command {
	root := &cobra.Command{Use: "tidy", Short: "Owner-only: review placement or duplicates and deliberately apply a plan"}
	o := tidy.Defaults()
	var out string
	var judge bool
	var maxJudge int
	scan := &cobra.Command{Use: "scan", Short: "Read the whole graph and write a review plan (no memory writes)", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
		if err := o.Validate(); err != nil {
			return err
		}
		if maxJudge < 0 {
			return fmt.Errorf("--max-judge must be nonnegative")
		}
		cfg, err := config.Load()
		if err != nil {
			return err
		}
		graph, err := tidy.ReadGraph(cmd.Context(), config.DBPath())
		if err != nil {
			return err
		}
		home, _ := os.UserHomeDir()
		cwd, _ := os.Getwd()
		bases := []string{cwd}
		if config.Home() == "" {
			bases = append(bases, home)
		}
		roots := tidy.DiscoverRoots(bases)
		plan, err := tidy.Scan(graph, o, roots)
		if err != nil {
			return err
		}
		plan.Database = config.DBPath()
		if out == "" {
			out = filepath.Join(config.DataDir(), "tidy", "plan-"+time.Now().UTC().Format("20060102T150405.000000000Z")+".json")
		}
		if judge {
			if err = tidy.JudgeWithDebug(cmd.Context(), &plan, cfg.Extract, maxJudge, out+".judge-debug.jsonl"); err != nil {
				return err
			}
		}
		if err = saveTidyPlan(out, plan); err != nil {
			return err
		}
		fmt.Fprintf(cmd.OutOrStdout(), "%d active canonical units; %d suspects; %d maybe general\n", plan.Units, len(plan.Entries), len(plan.General))
		fmt.Fprintf(cmd.OutOrStdout(), "Classes: strong=%d group-only=%d vote-only=%d\n", plan.Classes["strong"], plan.Classes["group-only"], plan.Classes["vote-only"])
		fmt.Fprintf(cmd.OutOrStdout(), "Dropped before judging: boundary=%d invalid-target=%d ambiguous-target=%d\n", plan.DroppedBoundary, plan.DroppedInvalidTarget, plan.DroppedAmbiguousTarget)
		for _, t := range plan.Transitions {
			fmt.Fprintf(cmd.OutOrStdout(), "%s -> %s [%s]: %d\n", t.Source, t.Target, t.Class, t.Count)
		}
		for _, g := range plan.Groups() {
			fmt.Fprintf(cmd.OutOrStdout(), "%s -> %s: %d (move=%d stay=%d unsure=%d)\n", g.Source, g.Target, g.Count, g.Decisions["move"], g.Decisions["stay"], g.Decisions["unsure"])
		}
		fmt.Fprintf(cmd.OutOrStdout(), "Plan: %s\nSummary: %s.md\nReview before apply. No memory or index writes performed.\n", out, out)
		return nil
	}}
	scan.Flags().StringVar(&out, "out", "", "new JSON plan path (default data-dir/tidy/plan-<timestamp>.json)")
	scan.Flags().Float64Var(&o.MinSim, "min-sim", o.MinSim, "minimum neighbor similarity")
	scan.Flags().IntVar(&o.Majority, "majority", o.Majority, "minimum votes for another scope")
	scan.Flags().IntVar(&o.OwnMax, "own-max", o.OwnMax, "maximum votes for the current scope")
	scan.Flags().Float64Var(&o.GroupPurity, "group-purity", o.GroupPurity, "minimum share of a topic group's members in its owner scope")
	scan.Flags().IntVar(&o.GroupMin, "group-min", o.GroupMin, "minimum owner members in a client group")
	scan.Flags().Float64Var(&o.GroupFloor, "group-floor", o.GroupFloor, "minimum owner members as a fraction of scope size (0 uses group-min only)")
	scan.Flags().IntVar(&o.GroupScore, "group-score", o.GroupScore, "minimum target client-group concept score (also at least 3x own score)")
	scan.Flags().BoolVar(&judge, "judge", false, "ask the configured capture model to judge proposals")
	scan.Flags().IntVar(&maxJudge, "max-judge", 100, "maximum suspects sent to the judge (strong, group-only, vote-only; 0 disables calls)")
	var dry bool
	var only string
	apply := &cobra.Command{Use: "apply PLAN", Short: "Apply reviewed move decisions, with backup and undo commands", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		p, err := tidy.Load(args[0])
		if err != nil {
			return err
		}
		if p.Database != "" && filepath.Clean(p.Database) != filepath.Clean(config.DBPath()) {
			return fmt.Errorf("plan belongs to a different database")
		}
		if p.Kind == "dupes" {
			if only != "" {
				return fmt.Errorf("--only is for placement plans; set duplicate cluster decision to keep instead")
			}
			if dry {
				g, err := tidy.ReadGraph(cmd.Context(), config.DBPath())
				if err != nil {
					return err
				}
				clusters, err := tidy.SelectedDuplicates(p, g)
				if err != nil {
					return err
				}
				n := 0
				for _, c := range clusters {
					for _, m := range c.Retire {
						fmt.Fprintf(cmd.OutOrStdout(), "would retire %s into %s (strength +%d)\n", m.SID, c.Keeper.SID, m.Strength)
						n++
					}
				}
				fmt.Fprintf(cmd.OutOrStdout(), "Dry run: %d retirements, no writes.\n", n)
				return nil
			}
			return withApp(func(ctx context.Context, a *app.App) error {
				result, err := a.ApplyTidy(ctx, p, "", false)
				if err != nil {
					if result.Backup != "" {
						fmt.Fprintf(cmd.ErrOrStderr(), "Recovery backup: %s\n", result.Backup)
					}
					return err
				}
				for _, r := range result.Retirements {
					fmt.Fprintf(cmd.OutOrStdout(), "retired %s into %s (strength +%d, origin %s)\n", r.SID, r.Keeper, r.Strength, r.Origin)
				}
				fmt.Fprintf(cmd.OutOrStdout(), "Retired %d memories. Backup and receipt: %s\n", len(result.Retirements), result.Backup)
				if result.Backup != "" {
					fmt.Fprintf(cmd.OutOrStdout(), "Undo: muninn tidy undo %s\n", shellQuote(result.Backup))
				}
				return nil
			})
		}
		if dry {
			g, err := tidy.ReadGraph(cmd.Context(), config.DBPath())
			if err != nil {
				return err
			}
			entries, err := tidy.CheckSnapshot(p, g, only)
			if err != nil {
				return err
			}
			for _, e := range entries {
				fmt.Fprintf(cmd.OutOrStdout(), "would move %s: %s -> %s\n", e.SID, e.Source, e.Target)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Dry run: %d moves, no writes.\n", len(entries))
			return nil
		}
		return withApp(func(ctx context.Context, a *app.App) error {
			result, err := a.ApplyTidy(ctx, p, only, false)
			if err != nil {
				if result.Backup != "" {
					fmt.Fprintf(cmd.ErrOrStderr(), "Recovery backup: %s\n", result.Backup)
				}
				return err
			}
			for _, m := range result.Moves {
				fmt.Fprintf(cmd.OutOrStdout(), "moved %s: %s -> %s\nUndo placement: muninn rescope %s --to %s --cross-client\n", m.Ref, m.Source, m.Target, shellQuote(m.Ref), shellQuote(m.Source))
				if m.Upstream != "" {
					fmt.Fprintf(cmd.OutOrStdout(), "Upstream %s remains unchanged and suppressed. Undo moves the native copy back, keeping upstream suppressed to avoid duplicates.\n", m.Upstream)
				}
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Moved %d memories. Backup and receipt: %s\n", len(result.Moves), result.Backup)
			return nil
		})
	}}
	apply.Flags().BoolVar(&dry, "dry-run", false, "validate and print moves without any writes")
	apply.Flags().StringVar(&only, "only", "", "apply only this exact SRC->DST pair (quote it in the shell)")
	undo := &cobra.Command{Use: "undo BACKUP", Short: "Undo a duplicate batch, refusing to overwrite later memory writes", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		return withApp(func(ctx context.Context, a *app.App) error {
			if err := a.UndoTidy(ctx, args[0]); err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), "Duplicate batch undone. Original memory bytes restored.")
			return nil
		})
	}}
	root.AddCommand(scan, apply, placeCmd(), duplicatesCmd(), undo)
	return root
}

func duplicatesCmd() *cobra.Command {
	var out string
	var judge bool
	var maxJudge int
	var duplicateThreshold, reviewThreshold float64
	c := &cobra.Command{Use: "dupes", Short: "Read-only duplicate clusters and reasons for deliberate review", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
		if maxJudge < 0 {
			return fmt.Errorf("--max-judge must be nonnegative")
		}
		cfg, err := config.Load()
		if err != nil {
			return err
		}
		g, err := tidy.ReadGraph(cmd.Context(), config.DBPath())
		if err != nil {
			return err
		}
		automatic, review := cfg.Extract.DedupeThresholds()
		if cmd.Flags().Changed("dedupe-cosine") {
			automatic = duplicateThreshold
		}
		if cmd.Flags().Changed("review-cosine") {
			review = reviewThreshold
		}
		p, err := tidy.ScanDuplicates(g, automatic, review)
		if err != nil {
			return err
		}
		p.Database = config.DBPath()
		if judge {
			if err = tidy.JudgeDuplicates(cmd.Context(), &p, g, cfg.Extract, maxJudge); err != nil {
				return err
			}
		}
		if out == "" {
			out = filepath.Join(config.DataDir(), "tidy", "dupes-"+time.Now().UTC().Format("20060102T150405.000000000Z")+".json")
		}
		if err = saveTidyPlan(out, p); err != nil {
			return err
		}
		autoClusters, judgedClusters, retire, judgedPairs := 0, 0, 0, 0
		for _, c := range p.Duplicates {
			if c.Band == "automatic" {
				autoClusters++
			} else {
				judgedClusters++
			}
			retire += len(c.Retire)
		}
		for _, pair := range p.DuplicatePairs {
			if pair.Judged {
				judgedPairs++
			}
		}
		fmt.Fprintf(cmd.OutOrStdout(), "%d active canonical units; %d duplicate clusters (automatic=%d judged=%d); %d memories would retire; %d pairs judged\nPlan: %s\nSummary: %s.md\nReview before apply. No memory or index writes performed.\n", p.Units, len(p.Duplicates), autoClusters, judgedClusters, retire, judgedPairs, out, out)
		return nil
	}}
	c.Flags().StringVar(&out, "out", "", "new private JSON plan path")
	c.Flags().BoolVar(&judge, "judge", false, "ask the configured cheap extraction model about review-band pairs")
	c.Flags().IntVar(&maxJudge, "max-judge", 100, "maximum review-band pairs to judge (0 disables calls)")
	c.Flags().Float64Var(&duplicateThreshold, "dedupe-cosine", .95, "override configured automatic duplicate threshold for this scan")
	c.Flags().Float64Var(&reviewThreshold, "review-cosine", .88, "override configured same-fact review threshold for this scan")
	return c
}
func shellQuote(s string) string { return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'" }
func saveTidyPlan(path string, p tidy.Plan) error {
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return err
	}
	for _, item := range []struct {
		path string
		data []byte
	}{{path, append(raw, '\n')}, {path + ".md", []byte(p.Markdown())}} {
		f, err := os.OpenFile(item.path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
		if err != nil {
			return err
		}
		_, err = f.Write(item.data)
		closeErr := f.Close()
		if err != nil {
			return err
		}
		if closeErr != nil {
			return closeErr
		}
	}
	return nil
}
