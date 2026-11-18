package run

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/veridit/hug/internal/memshare"
	"github.com/veridit/hug/muninn/internal/app"
	"github.com/veridit/hug/muninn/internal/project"
)

func projectCmd() *cobra.Command {
	var dir string
	c := &cobra.Command{Use: "project", Short: "Show this directory's project identity (stable across renames, moves, clones)",
		RunE: func(*cobra.Command, []string) error {
			p, err := project.Find(dirOr(dir), false)
			if err != nil {
				return err
			}
			fmt.Printf("id:    %s\nlabel: %s\nroot:  %s\nscope: %s\nfile:  %s\n", p.ID, p.Label, p.Root, p.Scope(), p.File())
			return nil
		}}
	c.Flags().StringVar(&dir, "dir", "", "directory (default: current)")
	return c
}

func syncCmd() *cobra.Command {
	var dir string
	var dry, migrate, background, held bool
	var release string
	c := &cobra.Command{Use: "sync", Short: "Commit, rebase and push project memories on the hug/memory branch",
		Long: `sync merges .hug/memory/memories.jsonl into your local store, writes only
this project's memories, and uses the independent hug/memory branch mounted
at .hug/memory/. The code branch stays clean. Requires Git >= 2.42.

Run hug upgrade to preview an old muninn-notes/.muninn-notes layout move,
then hug upgrade --apply to move this repository's branch and worktree.

Use --migrate to move a legacy .muninn/memories.jsonl after a successful branch
sync. A tracked legacy file is removed in one commit on the current code branch.
Use --migrate --dry-run to inspect the plan without creating a worktree, changing
files or contacting the remote. Existing client/project markers remain intact.

Personal, global, client and other projects' memories never leave your machine.
Credential-like project memories are withheld. Capture audits are never synced.

The placement gate keeps a project memory local when it names a home folder,
another project or client, or a person, or is a preference or correction
(rnpdnm). Its names live in the local config (sync_gate.terms in config.json),
never in the repository; the first sync seeds them from the places this store
holds memories for and hug.toml [people]. --held lists what it keeps local,
read-only; --release ID lets one held memory push (a person's call, after
reading it).`,
		RunE: func(*cobra.Command, []string) error {
			p, err := project.Find(dirOr(dir), !dry && !background)
			if err != nil {
				if background {
					return nil // nothing to sync here
				}
				return err
			}
			if held || release != "" {
				return withApp(func(_ context.Context, a *app.App) error {
					if release != "" {
						return releaseHeld(os.Stdout, a, p, release)
					}
					return listHeld(os.Stdout, a, p)
				})
			}
			syncOnce := func() error {
				return withApp(func(ctx context.Context, a *app.App) error {
					if _, err := a.Bound("", dirOr(dir)).WriteScope(p.Scope()); err != nil {
						return err
					}
					gate, err := syncGate(a, p, os.Stdout, dry)
					if err != nil {
						return err
					}
					st, err := project.SyncBranch(p, app.NativePath(), project.BranchOptions{DryRun: dry, Migrate: migrate, Gate: gate})
					if errors.Is(err, project.ErrMemoriesLocal) {
						// The intended state, not a failure (34r1ft): no fix or
						// next line, so no agent is steered to publish.
						fmt.Printf("%s%s (%s): %s\n", dryTag(dry), p.Label, p.ID, memshare.LocalStatus)
						return nil
					}
					if err != nil {
						return err
					}
					fmt.Printf("%s%s (%s): %s\n", dryTag(dry), p.Label, p.ID, st)
					printHeld(os.Stdout, st.SyncStats)
					return reindexUnlessDry(ctx, a, dry)
				})
			}
			// The detached pass a remember, forget or stale recall started
			// (0hkaa5): queue, let the burst settle, sync once.
			if background {
				fmt.Printf("%s pid %d: background pass\n", time.Now().UTC().Format(time.RFC3339), os.Getpid())
				return project.BackgroundPass(p, app.NativePath(), syncOnce)
			}
			// A foreground sync waits for a running pass, not beside it.
			defer project.Exclusive(p.Root)()
			return syncOnce()
		}}
	c.Flags().BoolVar(&background, strings.TrimPrefix(project.SyncChildArg, "--"), false, "run one background pass (started by hug itself)")
	_ = c.Flags().MarkHidden(strings.TrimPrefix(project.SyncChildArg, "--"))
	c.Flags().StringVar(&dir, "dir", "", "directory (default: current)")
	c.Flags().BoolVar(&migrate, "migrate", false, "move legacy .muninn/memories.jsonl off the code branch")
	c.Flags().BoolVar(&dry, "dry-run", false, "show counts without writing")
	c.Flags().BoolVar(&held, "held", false, "list the project memories the placement gate keeps local (read-only)")
	c.Flags().StringVar(&release, "release", "", "let one held memory push (a person's call, after reading it with --held)")
	return c
}

func dirOr(d string) string {
	if d != "" {
		return d
	}
	wd, _ := os.Getwd()
	return wd
}
