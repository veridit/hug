# internal/fsnotify

A copy of [fsnotify](https://github.com/fsnotify/fsnotify) v1.10.1 (BSD-3-Clause,
see LICENSE here), carried until upstream releases a fix for
[fsnotify#788](https://github.com/fsnotify/fsnotify/issues/788) (mq5ger).

The source is v1.10.1 as released, imports pointed at this package; the tests
are not copied, except `close_test.go`, which is hug's.
