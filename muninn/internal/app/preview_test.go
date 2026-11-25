package app_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/veridit/hug/muninn/internal/app"
	"github.com/veridit/hug/muninn/internal/mcpserver"
	"github.com/veridit/hug/muninn/internal/scopetest"
)

// TestPreviewBindingNamesTheSamePlaces (vrx1ja): hug mcp answers initialize
// from a binding resolved without the database (app.Preview). Its name,
// default scope and places, and so the server instructions, equal those of
// the binding the memory tools use once the store is open, for each kind of
// session: global jcode, a named jcode instance, a plain folder, a client.
func TestPreviewBindingNamesTheSamePlaces(t *testing.T) {
	f := scopetest.New(t)
	pre, err := app.Preview()
	if err != nil {
		t.Fatal(err)
	}
	if pre.DB != nil {
		t.Fatal("Preview opened the database")
	}
	home, _ := os.UserHomeDir()
	plain := filepath.Join(f.Root, "plain-folder")
	os.MkdirAll(plain, 0o700)
	for _, tc := range []struct{ name, jcodeHome, cwd string }{
		{"global in the project", filepath.Join(home, ".jcode"), f.Cwd},
		{"global in a plain folder", filepath.Join(home, ".jcode"), plain},
		{"instance alpha", filepath.Join(f.Root, "alpha", ".jcode"), f.Cwd},
		{"client", filepath.Join(home, ".jcode"), f.ClientCwd},
	} {
		t.Run(tc.name, func(t *testing.T) {
			full := mcpserver.Bind(f.App, f.App.Bound(tc.jcodeHome, tc.cwd))
			quick := pre.Bound(tc.jcodeHome, tc.cwd)
			if full.Err != nil || quick.Err != nil {
				t.Fatalf("errors: full %v, preview %v", full.Err, quick.Err)
			}
			if got, want := mcpserver.InstructionsFor(quick), mcpserver.InstructionsFor(full); got != want {
				t.Errorf("instructions differ\npreview: %s\nfull:    %s", got, want)
			}
			if quick.DefaultScope != full.DefaultScope || quick.Client != full.Client {
				t.Errorf("preview default %q client %q, full %q %q", quick.DefaultScope, quick.Client, full.DefaultScope, full.Client)
			}
		})
	}
}
