package project

import "github.com/veridit/hug/internal/memshare"

// ErrMemoriesLocal: this repository keeps its project memories on this
// machine; sync writes and pushes no branch (34r1ft).
var ErrMemoriesLocal = memshare.ErrLocal

// shareSetting says whether the repository at root lets its project
// memories reach a branch: the one reader every push path asks
// (memshare.Shares). Absent means no: a public repository never pushes real
// memories because someone forgot an opt-out (34r1ft). A variable so tests
// of the push itself can opt in.
var shareSetting = memshare.Shares
