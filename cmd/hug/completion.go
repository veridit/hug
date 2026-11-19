package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	huginn "github.com/veridit/hug/huginn/run"
	muninn "github.com/veridit/hug/muninn/run"
)

const completionHelp = `hug completion bash|zsh|fish: the shell completion script for hug.

One script completes all of hug: its commands, hug task's commands, flags
and open task ids, and hug mem's commands and flags. Install it once:

  bash (4.4 or newer), in ~/.bashrc:   source <(hug completion bash)
  zsh, in ~/.zshrc:                    source <(hug completion zsh)
  fish:                                hug completion fish > ~/.config/fish/completions/hug.fish

Then open a new shell. The script asks hug itself on every TAB, so it
follows upgrades without reinstalling.
`

// topCommands are what hug completes first, with the line hug help gives.
var topCommands = []string{
	"setup\twhat this repository still needs",
	"task\ttasks on the git branch",
	"mem\tmemory: recall, search, remember, forget",
	"tui\ttask board",
	"mcp\tone MCP server: memory_* and task_* tools",
	"apps\tClaude Desktop, Claude Code, ChatGPT/Codex: start hug mcp",
	"upgrade\twhat every migration to the current hug would change",
	"completion\tthe shell completion script for hug",
	"version\tprint the version",
	"help\tthis overview",
}

// runCompletion prints the script for shell (5mmqjk): cobra's own, for a
// command named hug, so the script asks hug __complete.
func runCompletion(args []string) int {
	if len(args) == 1 && (args[0] == "-h" || args[0] == "--help" || args[0] == "help") {
		fmt.Print(completionHelp)
		return 0
	}
	root := &cobra.Command{Use: "hug"}
	var e error
	switch strings.Join(args, " ") {
	case "bash":
		e = root.GenBashCompletionV2(os.Stdout, true)
	case "zsh":
		e = root.GenZshCompletion(os.Stdout)
	case "fish":
		e = root.GenFishCompletion(os.Stdout, true)
	default:
		fmt.Fprintf(os.Stderr, "hug completion: want one shell, bash, zsh or fish; got %q\nfix: hug completion zsh (or bash, fish; see hug completion --help)\n", strings.Join(args, " "))
		return 2
	}
	if e != nil {
		fmt.Fprintf(os.Stderr, "hug completion: %v\nfix: hug completion %s > hug.%s\n", e, args[0], args[0])
		return 1
	}
	return 0
}

// complete answers the scripts' hug __complete WORDS... (and
// __completeNoDesc): one word per line, a tab before its description,
// then :DIRECTIVE, cobra's protocol. hug mem's words come from its own
// cobra tree, hug task's from its command table (cli.Complete); it never
// writes. The script passes the word being completed last, empty after
// a space.
func complete(args []string) int {
	desc := args[0] == cobra.ShellCompRequestCmd
	words := args[1:]
	if len(words) == 0 {
		words = []string{""}
	}
	if len(words) > 1 {
		switch words[0] {
		case "mem", "memory", "muninn":
			return muninn.Main(append([]string{args[0]}, words[1:]...))
		}
	}
	got, directive := completeWords(words)
	for _, w := range got {
		if !desc {
			w, _, _ = strings.Cut(w, "\t")
		}
		fmt.Println(w)
	}
	fmt.Printf(":%d\n", directive)
	return 0
}

func completeWords(words []string) ([]string, int) {
	cur := words[len(words)-1]
	if len(words) == 1 {
		var out []string
		for _, c := range topCommands {
			if strings.HasPrefix(c, cur) {
				out = append(out, c)
			}
		}
		return out, huginn.NoFileComp
	}
	cwd, _ := os.Getwd()
	switch words[0] {
	case "task", "tasks", "huginn":
		return huginn.Complete(cwd, words[1:])
	case "setup", "tui", "upgrade":
		return huginn.Complete(cwd, words)
	case "mcp":
		return prefixed([]string{"--jcode-home", "--help"}, cur), huginn.NoFileComp
	case "apps":
		return prefixed([]string{"install", "--dry-run", "--json", "--help"}, cur), huginn.NoFileComp
	case "completion":
		if len(words) == 2 {
			return prefixed([]string{"bash", "zsh", "fish"}, cur), huginn.NoFileComp
		}
	}
	return nil, huginn.NoFileComp
}

func prefixed(words []string, prefix string) []string {
	var out []string
	for _, w := range words {
		if strings.HasPrefix(w, prefix) {
			out = append(out, w)
		}
	}
	return out
}
