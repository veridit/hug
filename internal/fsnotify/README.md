# internal/fsnotify

A copy of [fsnotify](https://github.com/fsnotify/fsnotify) v1.10.1 (BSD-3-Clause,
see LICENSE here), carried until upstream releases a fix for
[fsnotify#788](https://github.com/fsnotify/fsnotify/issues/788) (mq5ger).

The one change from v1.10.1 is upstream PR
[#789](https://github.com/fsnotify/fsnotify/pull/789), applied by hand to
`backend_kqueue.go` (its `Close` hunk did not apply to v1.10.1 as is). Imports
point at this package; the tests are not copied, except `close_test.go`, which
is hug's.

v1.10.1's kqueue `Close` (macOS, BSD) closed every watch descriptor itself,
while its reader closed the same descriptors on the Remove events of files
being deleted. A descriptor closed twice closed whatever had reused its number
in between: a `t.TempDir` clean-up failed `unlinkat ...: bad file descriptor`
on macOS CI (run 44384875316, `TestChangeBeforeWatchIsUpIsSeen`), and outside
tests any file hug held open could lose its descriptor. With #789 the reader
releases the watches on its way out and `Close` waits for it.

To drop this copy: when a release includes #789 (or another fix for #788),
`go get github.com/fsnotify/fsnotify@vX.Y.Z`, point the imports back, delete
this folder, and keep `close_test.go`'s check passing against the release.
