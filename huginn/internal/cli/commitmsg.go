package cli

import (
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// The commit-msg gate (t709he): every commit in a hug repository names
// what made it in exactly one trailer, Model: <model id>, or Model: human
// when the git author wrote it themselves. There is no Agent: trailer and
// no bypass variable: git commit --no-verify is the emergency exit.
//
// hug never edits a tracked hook. Without core.hooksPath it installs into
// <git-common-dir>/hooks/commit-msg, keeping a hook already there as
// commit-msg.pre-hug and running it first. With core.hooksPath (a
// project's own .githooks) it writes the gate as hooks/hug-commit-msg and
// says which two lines to add to the project's commit-msg; hug setup lists
// that step as open until they are there.

// hookMarker identifies a hook hug wrote.
const hookMarker = "# hug: commit-msg gate (t709he)"

// hookScript is POSIX sh, so it runs with git's own sh on Windows too. It
// costs two git calls and one awk (no other process; TestCommitMsgGate counts
// them). Its fix line keeps an <...> slot
// that the gate itself refuses, so a pasted fix cannot claim a model.
const hookScript = `#!/bin/sh
` + hookMarker + `
# Installed by hug task install commit-msg; hug task uninstall commit-msg
# removes it. Every commit names what made it in one trailer: Model: and the
# model id, or Model: human when the git author wrote it themselves.
# Emergency exit: git commit --no-verify.
msg=$1
pre="${0%/*}/commit-msg.pre-hug"
if [ -x "$pre" ]; then
	"$pre" "$@" || exit $?
fi
# One git call for the git dir and the branch.
set -- "$msg" $(git rev-parse --git-dir --abbrev-ref HEAD 2>/dev/null)
gd=$2 branch=$3
# Merges, cherry-picks, reverts and rebases carry commits already made.
for f in MERGE_HEAD CHERRY_PICK_HEAD REVERT_HEAD rebase-merge rebase-apply; do
	[ -e "$gd/$f" ] && exit 0
done
# hug's own branches: hug adds Model: itself when it knows the model.
case $branch in
hug/tasks | muninn-notes) exit 0 ;;
esac
IFS= read -r subject <"$msg"
case $subject in
"Merge "* | "Revert "* | "fixup!"* | "squash!"* | "amend!"*) exit 0 ;;
esac
problem=$(git interpret-trailers --parse "$msg" | awk '
	tolower($0) ~ /^model:/ { n++; v = $0; sub(/^[^:]*:[ \t]*/, "", v); if (n == 1) first = v }
	END {
		if (n == 0) print "no Model: trailer"
		else if (n > 1) print n " Model: trailers, want exactly one"
		else if (first == "" || first ~ /<.*>/) print "Model: " first " names no model (replace the <...> slot)"
	}')
[ -z "$problem" ] && exit 0
file=$(cd "$(dirname "$msg")" && pwd)/$(basename "$msg")
{
	echo "commit refused: $problem"
	echo "Every commit names what made it, in one trailer line: Model: and the model id you"
	echo "are, or Model: human when you, the git author, wrote it yourself. Replace the slot"
	echo "below with that; it is refused as it stands. Emergency exit: git commit --no-verify"
	echo "fix: git -c trailer.ifexists=replace commit --cleanup=strip -F '$file' --trailer 'Model: <model-id-or-human>'"
} >&2
exit 1
`

// tailLines are what a project's own commit-msg (core.hooksPath) needs to
// run the gate: nothing when hug is not set up in that clone.
const tailLines = `h="$(git rev-parse --git-common-dir)/hooks/hug-commit-msg"
[ ! -x "$h" ] || "$h" "$@" || exit 1`

// hookPlace is where this repository's commit-msg hooks live.
type hookPlace struct {
	common   string // <git-common-dir>/hooks
	hooksDir string // core.hooksPath, absolute; "" when unset
}

func findHooks(ctx context.Context, cwd string) (hookPlace, error) {
	out, e := exec.CommandContext(ctx, "git", "-C", cwd, "rev-parse", "--path-format=absolute", "--git-common-dir", "--show-toplevel").Output()
	if e != nil {
		return hookPlace{}, &OperationError{Problem: "the commit-msg gate needs a Git checkout as its working directory", Fix: "cd /path/to/your/git/repo && hug task install commit-msg"}
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	p := hookPlace{common: filepath.Join(filepath.Clean(lines[0]), "hooks")}
	top := ""
	if len(lines) > 1 {
		top = lines[1]
	}
	if hp, e := exec.CommandContext(ctx, "git", "-C", cwd, "config", "--get", "core.hooksPath").Output(); e == nil {
		if d := strings.TrimSpace(string(hp)); d != "" {
			if strings.HasPrefix(d, "~/") {
				if home, e := os.UserHomeDir(); e == nil {
					d = filepath.Join(home, d[2:])
				}
			}
			if !filepath.IsAbs(d) {
				d = filepath.Join(top, d)
			}
			p.hooksDir = filepath.Clean(d)
		}
	}
	// core.hooksPath pointing at the common hooks folder is no project hook.
	if p.hooksDir != "" && samePath(p.hooksDir, p.common) {
		p.hooksDir = ""
	}
	return p, nil
}

func samePath(a, b string) bool {
	if x, e := filepath.EvalSymlinks(a); e == nil {
		a = x
	}
	if x, e := filepath.EvalSymlinks(b); e == nil {
		b = x
	}
	return a == b
}

func isOurs(path string) bool {
	b, e := os.ReadFile(path)
	return e == nil && bytes.Contains(b, []byte(hookMarker))
}

// gateStatus is the hug setup step: whether every commit here passes the gate.
func gateStatus(ctx context.Context, cwd string) (done bool, detail, do string) {
	p, e := findHooks(ctx, cwd)
	if e != nil {
		return false, e.Error(), "hug task install commit-msg"
	}
	if p.hooksDir == "" {
		path := filepath.Join(p.common, "commit-msg")
		return isOurs(path), "commits without one Model: trailer (a model id, or human) are refused, in " + path, "hug task install commit-msg"
	}
	project := filepath.Join(p.hooksDir, "commit-msg")
	gate := isOurs(filepath.Join(p.common, "hug-commit-msg"))
	b, _ := os.ReadFile(project)
	wired := bytes.Contains(b, []byte("hooks/hug-commit-msg"))
	detail = "core.hooksPath is " + p.hooksDir + ": hug never edits a tracked hook; " + project + " runs the Model: gate"
	switch {
	case !gate:
		return false, detail, "hug task install commit-msg"
	case !wired:
		return false, detail, "add these lines to " + project + ", after its first line (create it with #!/bin/sh first if it is missing, and chmod +x it):\n" + tailLines
	}
	return true, detail, ""
}

// runCommitMsg installs or removes the gate.
func (a *App) runCommitMsg(ctx context.Context, cmd string, dry bool) (Reply, error) {
	cwd := a.Cwd
	if cwd == "" {
		cwd = "."
	}
	p, e := findHooks(ctx, cwd)
	if e != nil {
		return nil, e
	}
	name := "commit-msg"
	if p.hooksDir != "" {
		name = "hug-commit-msg"
	}
	path := filepath.Join(p.common, name)
	pre := filepath.Join(p.common, "commit-msg.pre-hug")
	reply := Reply{"hook": path, "dry_run": dry, "next": "hug task setup"}
	if cmd == "uninstall" {
		if !isOurs(path) {
			reply["message"] = "no hug commit-msg gate at " + path
			return reply, nil
		}
		if !dry {
			if e = os.Remove(path); e != nil {
				return nil, e
			}
			if _, e = os.Stat(pre); e == nil && name == "commit-msg" {
				if e = os.Rename(pre, path); e != nil {
					return nil, e
				}
			}
		}
		reply["message"] = "removed the Model: gate " + path
		return reply, nil
	}
	if dry {
		reply["message"] = "would write the Model: gate to " + path
	} else {
		if e = os.MkdirAll(p.common, 0o755); e != nil {
			return nil, e
		}
		// A hook that is not ours keeps running, first, as commit-msg.pre-hug.
		if _, e = os.Stat(path); e == nil && !isOurs(path) {
			if _, e = os.Stat(pre); e == nil {
				return nil, &OperationError{Problem: path + " is not hug's and " + pre + " exists already: hug will not overwrite either", Fix: "merge them into " + pre + ", remove " + path + ", then hug task install commit-msg"}
			}
			if e = os.Rename(path, pre); e != nil {
				return nil, e
			}
			reply["kept"] = pre
		} else if e != nil && !errors.Is(e, os.ErrNotExist) {
			return nil, e
		}
		if e = os.WriteFile(path, []byte(hookScript), 0o755); e != nil {
			return nil, e
		}
		_ = os.Chmod(path, 0o755)
		reply["message"] = "commits without one Model: trailer are refused now (" + path + ")"
	}
	if p.hooksDir != "" {
		done, _, do := gateStatus(ctx, cwd)
		if !done && !dry {
			reply["message"] = reply["message"].(string) + "; core.hooksPath is " + p.hooksDir + ", so it runs once you " + do
			reply["next"] = "hug task setup"
		}
	}
	return reply, nil
}
