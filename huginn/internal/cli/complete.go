package cli

import (
	"context"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/veridit/hug/huginn/gitbranch"
	"github.com/veridit/hug/huginn/internal/cache"
	"github.com/veridit/hug/huginn/internal/task"
)

// Shell completion directives, cobra's numbers (the scripts hug completion
// prints read them): FileComp lets the shell complete file names,
// NoFileComp offers only the words given.
const (
	FileComp   = 0
	NoFileComp = 4
)

// idFlags take a task reference; fileFlags a path.
var (
	idFlags   = []string{"parent", "before", "after", "with", "add", "remove", "dep"}
	fileFlags = []string{"show", "file", "jcode-home"}
)

// Complete answers shell completion for hug task (5mmqjk): words are the
// arguments after "hug task", the last one the word being completed (empty
// after a space). Everything comes from hug task's own tree: Commands, each
// command's flags (allowedFlags), the slots of its usage line, task types
// and states, and the open tasks' ids with their titles. It never writes:
// no sync, no cache update, no lock (cache.Peek). Each answer is a word,
// optionally a tab and its description.
func Complete(ctx context.Context, cwd string, words []string) ([]string, int) {
	if len(words) == 0 {
		words = []string{""}
	}
	cur, done := words[len(words)-1], words[:len(words)-1]
	if len(done) == 0 {
		return prefixed(Commands, cur), NoFileComp
	}
	cmd := done[0]
	if !slices.Contains(Commands, cmd) {
		return nil, NoFileComp
	}
	flagNames := strings.Fields(allowedFlags[cmd] + " json help")
	// The value of a flag: --type <TAB> or --type=<TAB>.
	flag, value := "", cur
	if k, v, ok := strings.Cut(cur, "="); ok && strings.HasPrefix(k, "--") {
		flag, value = strings.TrimPrefix(k, "--"), v
	} else if last := done[len(done)-1]; len(done) > 1 && strings.HasPrefix(last, "--") && !strings.Contains(last, "=") && !slices.Contains(boolFlags, strings.TrimPrefix(last, "--")) {
		flag = strings.TrimPrefix(last, "--")
	}
	if flag != "" {
		if !slices.Contains(flagNames, flag) {
			return nil, NoFileComp
		}
		return flagValues(ctx, cwd, flag, value)
	}
	if strings.HasPrefix(cur, "-") {
		var out []string
		for _, f := range flagNames {
			out = append(out, "--"+f)
		}
		return prefixed(out, cur), NoFileComp
	}
	// A positional: which slot of the usage line it fills.
	var args []string
	for i := 1; i < len(done); i++ {
		w := done[i]
		if strings.HasPrefix(w, "--") {
			if !strings.Contains(w, "=") && !slices.Contains(boolFlags, strings.TrimPrefix(w, "--")) {
				i++
			}
			continue
		}
		args = append(args, w)
	}
	return slotValues(ctx, cwd, cmd, args, cur)
}

// flagValues completes the value of --flag.
func flagValues(ctx context.Context, cwd, flag, value string) ([]string, int) {
	switch {
	case flag == "type":
		return prefixed(task.Types, value), NoFileComp
	case flag == "state":
		return prefixed(task.StateList, value), NoFileComp
	case slices.Contains(idFlags, flag):
		return openIDs(ctx, cwd, value), NoFileComp
	case slices.Contains(fileFlags, flag):
		return nil, FileComp
	}
	return nil, NoFileComp
}

// slotValues completes positional number len(args) of cmd from its usage
// line: a literal word (install jcode), ID (open task ids), COMMAND (hug
// task's commands) or FILE/DIR/DIRECTORY (file names). An alternative of
// the usage applies only while the words typed match its literals.
func slotValues(ctx context.Context, cwd, cmd string, args []string, cur string) ([]string, int) {
	line, _, _ := strings.Cut(commandHelp(cmd), "\n")
	line = strings.TrimPrefix(line, "hug task ")
	var words []string
	ids, files := false, false
	for _, alt := range usageAlternatives(cmd, line) {
		n := len(args)
		if n >= len(alt) {
			if len(alt) == 0 || !strings.HasSuffix(alt[len(alt)-1][0], "...") {
				continue
			}
			n = len(alt) - 1
		}
		fits := true
		for i := 0; i < len(args) && i < len(alt); i++ {
			if literal(alt[i][0]) && !slices.Contains(alt[i], args[i]) {
				fits = false
				break
			}
		}
		if !fits {
			continue
		}
		for _, choice := range alt[n] {
			name := strings.TrimSuffix(choice, "...")
			switch {
			case literal(name):
				words = append(words, name)
			case strings.HasPrefix(name, "ID"):
				ids = true
			case name == "COMMAND":
				words = append(words, Commands...)
			case name == "FILE" || name == "DIR" || name == "DIRECTORY":
				files = true
			}
		}
	}
	out := prefixed(words, cur)
	if ids {
		out = append(out, openIDs(ctx, cwd, cur)...)
	}
	if files && len(out) == 0 {
		return nil, FileComp
	}
	return out, NoFileComp
}

// usageAlternatives are the positional slots of each alternative of a
// usage line ("a | b" at bracket depth 0), each slot its choices (memory
// [share | local]: one slot, share or local). Flags and their values are
// left out; brackets and quotes are dropped.
func usageAlternatives(cmd, line string) [][][]string {
	var alts []string
	depth, start := 0, 0
	for i, r := range line {
		switch r {
		case '[', '(':
			depth++
		case ']', ')':
			depth--
		case '|':
			if depth == 0 && i > 0 && line[i-1] == ' ' {
				alts = append(alts, line[start:i])
				start = i + 1
			}
		}
	}
	alts = append(alts, line[start:])
	var out [][][]string
	for _, a := range alts {
		// "the answer" is one placeholder, never the literal words.
		fields := strings.Fields(quoted.ReplaceAllString(a, "TEXT"))
		if len(fields) > 0 && fields[0] == cmd {
			fields = fields[1:]
		}
		var slots [][]string
		join, flagValue := false, false
		for i := 0; i < len(fields); i++ {
			f := strings.Trim(fields[i], "[]()")
			switch {
			case f == "|":
				join = true
				continue
			case strings.HasPrefix(f, "-"):
				join, flagValue = false, false
				if k := strings.TrimLeft(f, "-"); !strings.Contains(k, "=") && !slices.Contains(boolFlags, k) && i+1 < len(fields) {
					i++
					flagValue = true
				}
				continue
			case f == "" || f == "...":
				// [--add REF ...]: more values of the flag, not of a slot.
				if f == "..." && len(slots) > 0 && !flagValue {
					last := slots[len(slots)-1]
					last[0] = strings.TrimSuffix(last[0], "...") + "..."
				}
				continue
			}
			flagValue = false
			if join && len(slots) > 0 {
				slots[len(slots)-1] = append(slots[len(slots)-1], strings.Split(f, "|")...)
			} else {
				slots = append(slots, strings.Split(f, "|"))
			}
			join = false
		}
		out = append(out, slots)
	}
	return out
}

var quoted = regexp.MustCompile(`"[^"]*"`)

// literal is a usage word typed as is (jcode, share), not a placeholder.
func literal(w string) bool {
	if w == "" {
		return false
	}
	for _, r := range w {
		if !(r >= 'a' && r <= 'z' || r == '-') {
			return false
		}
	}
	return true
}

// openIDs are the open (todo, doing) tasks whose id starts with prefix, each
// with its title, in board order. Outside a repository, or on a board that
// cannot be read, there are none.
func openIDs(ctx context.Context, cwd, prefix string) []string {
	loc, e := gitbranch.Locate(cwd)
	if e != nil {
		return nil
	}
	ts, e := cache.Peek(ctx, filepath.Join(loc.Main, TaskDir), cache.File(loc.Common))
	if e != nil {
		return nil
	}
	var out []string
	for _, t := range ts {
		if (t.State == "todo" || t.State == "doing") && strings.HasPrefix(t.ID, strings.ToLower(prefix)) {
			out = append(out, t.ID+"\t"+t.Title)
		}
	}
	return out
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
