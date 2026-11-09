package app

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"github.com/veridit/muninn/internal/config"
	"github.com/veridit/muninn/internal/harness"
	"github.com/veridit/muninn/internal/project"
)

// RebuildLegacyFolders is explicit background maintenance, never part of
// Reindex: hooks also call Reindex and must never walk a client tree.
func (a *App) RebuildLegacyFolders(ctx context.Context) error {
	rows, err := a.DB.QueryContext(ctx, `SELECT path,kind FROM sources WHERE kind='jcode'`)
	if err != nil {
		return err
	}
	var sources []project.LegacySource
	for rows.Next() {
		var s project.LegacySource
		if err = rows.Scan(&s.Path, &s.Kind); err != nil {
			rows.Close()
			return err
		}
		sources = append(sources, s)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return err
	}
	cache, err := project.BuildLegacyCache(sources)
	if err != nil {
		return err
	}
	if err = ctx.Err(); err != nil {
		return err
	}
	return cache.Save(config.DataDir())
}

// clientFolderStores computes only cwd ancestors already identified by bound.
// It needs no git command and no recursive filesystem discovery. Root hashes
// remain client-wide, including lexical symlink and literal ~/ spellings.
func clientFolderStores(cwd, physical, root, projectRoot, here, top string, rules []string) map[string]string {
	out := map[string]string{}
	home, _ := os.UserHomeDir()
	add := func(dir, scope string) {
		out[harness.JcodeProjectHash(dir)] = scope
		if rel, err := filepath.Rel(home, dir); home != "" && err == nil && rel != "." && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			out[harness.JcodeProjectHash("~/"+rel)] = scope
		}
	}
	addAlias := func(logical, physical, scope string) {
		if logical != physical {
			if resolved, err := placementDir(logical); err == nil && resolved == physical {
				add(logical, scope)
			}
		}
	}
	logical, err := filepath.Abs(cwd)
	if err != nil {
		logical = physical
	}
	add(physical, here)
	add(logical, here)
	if projectRoot != "" {
		for p, l := physical, logical; ; p, l = filepath.Dir(p), filepath.Dir(l) {
			add(p, here)
			addAlias(l, p, here)
			if p == projectRoot || filepath.Dir(p) == p {
				break
			}
		}
	}
	for _, scope := range rules {
		dir, ok := strings.CutPrefix(scope, top+":folder:")
		if !ok {
			continue
		}
		// Git ancestry stops at this repository. Folder rules above it may
		// own native folder notes, but their hash could be a parent repo's
		// project store and must never be guessed client-wide/ancestral.
		if projectRoot != "" {
			continue
		}
		alias := logical
		if rel, err := filepath.Rel(dir, physical); err == nil && rel != "." {
			for range strings.Split(rel, string(filepath.Separator)) {
				alias = filepath.Dir(alias)
			}
		}
		add(dir, scope)
		addAlias(alias, dir, scope)
	}
	// Ascend the same number of components in the lexical spelling.
	logicalRoot := logical
	if rel, err := filepath.Rel(root, physical); err == nil && rel != "." {
		for range strings.Split(rel, string(filepath.Separator)) {
			logicalRoot = filepath.Dir(logicalRoot)
		}
	}
	add(root, top)
	addAlias(logicalRoot, root, top)
	return out
}
