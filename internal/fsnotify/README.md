# internal/fsnotify

A copy of [fsnotify](https://github.com/fsnotify/fsnotify) **v1.10.1** (tag
`v1.10.1`, commit `76b01a6e8f502187fecedea8b025e79e5a86085c`, module sum
`h1:b0/UzAf9yR5rhf3RPm9gf3ehBPpf0oZKIjtpKrx59Ho=`), BSD-3-Clause: `LICENSE` here
is upstream's, unchanged. Carried until a fsnotify release fixes
[issue 788](https://github.com/fsnotify/fsnotify/issues/788) (mq5ger).

## Why

v1.10.1's kqueue `Close` (macOS, BSD) closed every watch descriptor itself,
while its reader closed the same descriptors on the Remove events of files
being deleted (`backend_kqueue.go:267` and `:320` in v1.10.1). A descriptor
closed twice closed whatever had reused its number in between: a `t.TempDir`
clean-up failed `unlinkat ...: bad file descriptor` on macOS CI (run
44384875316, `TestChangeBeforeWatchIsUpIsSeen`), and any file hug held open
could lose its descriptor. A `replace` in go.mod would not do: it is ignored
when hug is built as a dependency or with `go install module@version`.

## What differs from v1.10.1

`upstream-diff.sh` prints the whole diff from the module cache's v1.10.1;
`upstream-diff.sh --check` fails unless it equals `upstream.patch`, the reviewed
diff. It holds only:

1. **Imports**: the four `github.com/fsnotify/fsnotify/internal` imports
   (`backend_fen.go`, `backend_inotify.go`, `backend_kqueue.go`,
   `backend_windows.go`) read `github.com/veridit/hug/internal/fsnotify/internal`.
2. **[PR 789](https://github.com/fsnotify/fsnotify/pull/789)** (head
   `7d79a83e4b6a72c45bfdabc56218d4a0a6dcbe34`, open, unreleased) in
   `backend_kqueue.go`: the `opMu` and `exited` fields, `watches.drain`,
   `releaseWatches`, `opMu` taken in `AddWith`, `Remove`, readEvents' remove
   and `sendCreateIfNew`, `Close` waiting on `exited`, readEvents releasing the
   watches on exit, and `addWatch` closing only a descriptor it opened. Applied
   as in the PR, except:
   - **Close**: the PR removes main's `remove2` loop; v1.10.1 has no `remove2`
     and its `Close` is a snapshot loop (#740), so that loop is what is removed.
     The added lines are the PR's.
   - **remove2 comment**: the PR edits `remove2`'s comment; v1.10.1 has no
     `remove2`, so this hunk has no counterpart.
   - **remove's inner call**: v1.10.1's `remove` drops a folder's file watches
     with the public `w.Remove(name)`, which with the PR's lock takes `opMu` a
     second time and hangs (main calls the unlocked `remove2(name, false)`). It
     calls `w.remove(name, false)` instead, as main does in effect.
     `TestRemoveWatchedDirReturns` fails without this (`test timed out`).

The tests are not copied; `close_test.go` is hug's.

## Dropping it

Task 3dzmh5, "Drop internal/fsnotify once fsnotify releases the fix for issue 788":
when a release includes PR 789 or another fix for issue 788,
`go get github.com/fsnotify/fsnotify@vX.Y.Z`, point the three importers back
(`huginn/internal/tui/watch.go`, `watch_test.go`,
`muninn/internal/daemon/watch.go`), move `close_test.go`'s tests next to
them against the module, and delete this folder.
