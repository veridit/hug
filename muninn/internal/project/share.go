package project

import "errors"

// ErrMemoriesLocal: this repository keeps its project memories on this
// machine; sync writes and pushes no branch (34r1ft).
var ErrMemoriesLocal = errors.New("this repository keeps project memories local")

// shareSetting says whether the repository at root lets its project
// memories reach a branch. Absent means no: a public repository never
// pushes real memories because someone forgot an opt-out (34r1ft).
//
// STUB until check 1 decides where the setting lives (committed hug.toml
// marker vs per-machine git config): it always answers local.
var shareSetting = func(root string) (bool, error) { return false, nil }
