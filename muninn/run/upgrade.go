package run

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/spf13/cobra"

	"github.com/veridit/hug/muninn/internal/app"
	"github.com/veridit/hug/muninn/internal/harness"
	"github.com/veridit/hug/muninn/internal/ingest"
)

// ExitUpgradePending is hug mem upgrade's exit code when the preview found
// changes, the same as hug upgrade's (asr4t2).
const ExitUpgradePending = 5

// upgradeCmd folds the twins older syncs made (5sc5rw): every memory of
// this machine that a project sync had copied back from the notes file
// under a second id. It previews by default; --apply folds, with a backup.
func upgradeCmd() *cobra.Command {
	var apply bool
	c := &cobra.Command{Use: "upgrade", Short: "Fold the copies older syncs made of this machine's own memories (preview; --apply folds)",
		Long: `upgrade finds every memory that a project sync copied back from the
notes file under a second id (before hug 0.7.49 every capture in a repository
project had such a twin, so forgetting one left the other in recall). Each pair
is the same memory: the one written here stays, with its id, short id and
capture details, and its copy goes. A newer forget or edit made through the copy
moves to the memory first. Memories from other clones are not touched.

Without --apply it only says what would change. With --apply it writes a backup
of the store next to it, folds, and reindexes.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			out := cmd.OutOrStdout()
			// jcode instances still on the turn_start-only hook, which put
			// recall into the person's input box (z9q42h).
			hooks, err := migrateJcodeHooks(out, apply)
			if err != nil {
				return err
			}
			path := app.NativePath()
			st, err := ingest.FoldTwins(path, true)
			if err != nil {
				return err
			}
			if st.Folded == 0 {
				if hooks == 0 {
					fmt.Fprintln(out, "memory: nothing to do")
				} else if !apply {
					fmt.Fprintln(out, "fix: hug mem upgrade --apply")
					cmd.SilenceErrors = true
					return pendingErr{}
				}
				return nil
			}
			verb := "would fold"
			if apply {
				verb = "folded"
				backup := path + ".bak-upgrade-" + time.Now().UTC().Format("20060102T150405Z")
				b, err := os.ReadFile(path)
				if err != nil {
					return err
				}
				if err := os.WriteFile(backup, b, 0o600); err != nil {
					return err
				}
				if st, err = ingest.FoldTwins(path, false); err != nil {
					return fmt.Errorf("%w; the store is unchanged or restorable from %s", err, backup)
				}
				defer fmt.Fprintf(out, "memory: backup %s (restore: cp %q %q, then hug mem index)\n", filepath.Base(backup), backup, path)
				if err := withApp(func(ctx context.Context, a *app.App) error {
					_, err := a.Reindex(ctx)
					return err
				}); err != nil {
					return fmt.Errorf("folded, reindex failed: %w; fix: hug mem index", err)
				}
			}
			scopes := make([]string, 0, len(st.Scopes))
			for s := range st.Scopes {
				scopes = append(scopes, s)
			}
			sort.Strings(scopes)
			for _, s := range scopes {
				fmt.Fprintf(out, "memory: %s %d copies of this machine's own memories in %s\n", verb, st.Scopes[s], s)
			}
			if st.Retired > 0 {
				fmt.Fprintf(out, "memory: %d of them carried a newer forget, which their memory takes\n", st.Retired)
			}
			if !apply {
				fmt.Fprintln(out, "fix: hug mem upgrade --apply")
				cmd.SilenceErrors = true
				return pendingErr{}
			}
			return nil
		}}
	c.Flags().BoolVar(&apply, "apply", false, "fold, with a backup of the store")
	return c
}

// pendingErr ends a preview with changes pending: exit 5, nothing printed.
type pendingErr struct{}

func (pendingErr) Error() string { return "" }

// migrateJcodeHooks moves every jcode instance whose config runs muninn's
// recall hook on turn_start only (hug before 0.7.53, which delivered with
// jcode transcript into the person's input box) onto turn_start,
// post_tool_feedback and turn_end, keeping the binary it already runs. It
// returns how many instances need (or got) the change.
func migrateJcodeHooks(out io.Writer, apply bool) (int, error) {
	var homes []harness.JcodeInstance
	if err := withApp(func(_ context.Context, a *app.App) error {
		homes = a.JcodeInstances()
		return nil
	}); err != nil {
		return 0, err
	}
	n := 0
	for _, inst := range homes {
		bin, stale := harness.JcodeRecallNeedsMigration(inst.Home)
		if !stale {
			continue
		}
		n++
		file := filepath.Join(inst.Home, "config.toml")
		if !apply {
			fmt.Fprintf(out, "memory: would deliver jcode[%s] recall in a tool result, not the input box: %s runs %s on turn_start, post_tool_feedback and turn_end\n", inst.Name, file, bin+" hook jcode")
			continue
		}
		r, err := harness.InstallJcodeHome(inst.Home, "jcode["+inst.Name+"]", bin, true, false)
		if err != nil {
			return n, fmt.Errorf("jcode[%s]: %w; fix: hug mem install jcode --jcode-home %s", inst.Name, err, inst.Home)
		}
		fmt.Fprintf(out, "memory: jcode[%s] recall now arrives in a tool result, not the input box: %s (backup %s)\n", inst.Name, file, filepath.Base(r.Backup))
	}
	return n, nil
}
