// Package memshare answers one question for every path that could put a
// repository's project memories on its remote: does this repository opt in
// to sharing them? The memory sync (muninn project.SyncBranch) and the
// layout upgrade (hug upgrade, gitbranch.Upgrade) both ask it, so a public
// repository never publishes memories by a route that forgot to check
// (34r1ft, wxsyc5).
package memshare

import "errors"

// ErrLocal: this repository keeps its project memories on this machine.
var ErrLocal = errors.New("this repository keeps project memories local")

// KeptNote is what hug upgrade says when it renamed the memory branch here
// only and left the remote's old branch as it was.
const KeptNote = "remote muninn-notes kept; this repo keeps memories local"

// Shares reports whether the repository at root lets its project memories
// reach a branch on its remote. Absent means no.
//
// STUB until 34r1ft check 1 decides where the setting lives (a committed
// hug.toml marker vs per-machine git config): it always answers local.
func Shares(root string) (bool, error) { return false, nil }
