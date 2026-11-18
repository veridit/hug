package cli

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/veridit/hug/huginn/gitbranch"
	"github.com/veridit/hug/internal/layout"
	"github.com/veridit/hug/internal/memshare"
)

// memoryShares is memshare.Shares, a variable so tests of the shared
// rename can opt in (34r1ft).
var memoryShares = memshare.Shares

// Project memory is this repository's independent data branch, not the
// machine's native store. It deliberately uses the same branch/worktree
// upgrade mechanism as tasks, never the task document merge resolver.
func upgradeMemoryLayout(cwd string, dry bool) ([]string, error) {
	root, err := gitbranch.MainRoot(cwd)
	if err != nil {
		return nil, nil // outside a repository: only machine work can apply
	}
	oldPath := filepath.Join(root, layout.OldMemoryDir)
	refs, err := exec.Command("git", "-C", root, "for-each-ref", "--format=%(refname)", "refs/heads/", "refs/remotes/").Output()
	if err != nil {
		return nil, &OperationError{Problem: "cannot inspect this repository's memory branch refs", Fix: "git -C " + shellQuote(root) + " show-ref"}
	}
	hasOld, hasNew := false, false
	var remoteRefs []string
	for _, ref := range strings.Fields(string(refs)) {
		hasOld = hasOld || ref == "refs/heads/"+layout.OldMemoryBranch
		hasNew = hasNew || ref == "refs/heads/"+layout.MemoryBranch
		if strings.HasPrefix(ref, "refs/remotes/") && strings.HasSuffix(ref, "/"+layout.OldMemoryBranch) {
			remoteRefs = append(remoteRefs, ref)
		}
	}
	if !hasOld && !hasNew && len(remoteRefs) != 0 {
		ref := remoteRefs[0]
		origin := false
		for _, candidate := range remoteRefs {
			if candidate == "refs/remotes/origin/"+layout.OldMemoryBranch {
				ref = candidate
				origin = true
			}
		}
		if len(remoteRefs) > 1 && !origin {
			return nil, &OperationError{Problem: "project memories have ambiguous remote refs " + strings.Join(remoteRefs, ", ") + "; nothing changed", Fix: "git -C " + shellQuote(root) + " show-ref"}
		}
		for _, target := range []string{oldPath, filepath.Join(root, layout.MemoryDir)} {
			if _, e := os.Lstat(target); e == nil || !os.IsNotExist(e) {
				return nil, &OperationError{Problem: "project memories remain at " + ref + "; checkout path " + target + " is occupied or unreadable; nothing changed", Fix: "git -C " + shellQuote(root) + " worktree list"}
			}
		}
		trees, e := exec.Command("git", "-C", root, "worktree", "list", "--porcelain").Output()
		if e != nil || strings.Contains(strings.ReplaceAll(string(trees), "\r\n", "\n"), "branch refs/heads/"+layout.OldMemoryBranch+"\n") {
			return nil, &OperationError{Problem: "project memories remain at " + ref + "; the old branch's checkout registration needs inspection; nothing changed", Fix: "git -C " + shellQuote(root) + " worktree list"}
		}
		return nil, &OperationError{Problem: "project memories remain at " + ref + "; no empty hug/memory branch created", Fix: "git -C " + shellQuote(root) + " worktree add -b " + layout.OldMemoryBranch + " " + shellQuote(oldPath) + " " + shellQuote(ref) + " && hug upgrade"}
	}
	if _, e := os.Lstat(oldPath); e == nil {
		if _, e := os.Lstat(filepath.Join(oldPath, ".git")); e != nil {
			return nil, &OperationError{Problem: "project memory path " + oldPath + " is not a readable registered Git worktree; all contents kept, nothing changed", Fix: "git -C " + shellQuote(root) + " worktree list"}
		}
	} else if !os.IsNotExist(e) {
		return nil, &OperationError{Problem: "cannot inspect project memory path " + oldPath + "; nothing changed", Fix: "git -C " + shellQuote(root) + " worktree list"}
	}
	if dry {
		_, mountErr := os.Lstat(oldPath)
		if mountErr == nil {
			return []string{"would: " + layout.OldMemoryDir + "/ moves to " + layout.MemoryDir + "/, branch " + layout.OldMemoryBranch + " becomes " + layout.MemoryBranch}, nil
		}
		if hasOld {
			return []string{"would: branch " + layout.OldMemoryBranch + " becomes " + layout.MemoryBranch}, nil
		}
		return nil, nil
	}
	// A repository that keeps its memories local renames here only: the
	// remote keeps muninn-notes untouched (34r1ft, wxsyc5: this rename was
	// how a public repository's memories reached its remote).
	shares, err := memoryShares(root)
	if err != nil {
		return nil, &OperationError{Problem: "cannot read whether this repository shares project memories: " + err.Error() + "; nothing changed", Fix: "hug upgrade"}
	}
	m, err := gitbranch.Upgrade(root, layout.OldMemoryBranch, layout.OldMemoryDir, layout.MemoryBranch, layout.MemoryDir, gitbranch.LocalOnly(!shares))
	if err != nil {
		return nil, err
	}
	var changes []string
	if m.LocalBranch {
		changes = append(changes, "branch "+layout.OldMemoryBranch+" renamed to "+layout.MemoryBranch)
	}
	if m.Worktree {
		changes = append(changes, layout.OldMemoryDir+"/ moved to "+layout.MemoryDir+"/")
	}
	if m.RemoteBranch {
		changes = append(changes, "remote memory branch renamed")
	}
	if m.RemoteKept {
		changes = append(changes, memshare.KeptNote)
	}
	return changes, nil
}
