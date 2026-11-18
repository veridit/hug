#!/usr/bin/env bash
# The diff between this copy and upstream fsnotify v1.10.1 (mq5ger).
#   internal/fsnotify/upstream-diff.sh          prints it
#   internal/fsnotify/upstream-diff.sh --check  fails unless it equals upstream.patch
# It is the four import lines plus the fsnotify PR 789 hunks (README.md says
# which were rewritten). hug's own files (README.md, the _test.go files, this
# script, upstream.patch) are not upstream's and are left out.
set -euo pipefail
here="$(cd "$(dirname "$0")" && pwd)"
tmp="$(mktemp -d)"
trap 'rm -r "$tmp"' EXIT
# Outside hug's module, so go.mod is not touched; the module cache keeps it.
up="$(cd "$tmp" && GOFLAGS= GO111MODULE=on go mod download -json github.com/fsnotify/fsnotify@v1.10.1 |
  sed -n 's/^[[:space:]]*"Dir": "\(.*\)",$/\1/p')"
[ -d "$up" ] || { echo "upstream-diff: no module dir for fsnotify v1.10.1" >&2; exit 1; }
out="$tmp/diff"
: > "$out"
cd "$here"
for f in $(find . -type f \( -name '*.go' -o -name LICENSE \) ! -name '*_test.go' | sed 's|^\./||' | LC_ALL=C sort); do
  if [ ! -f "$up/$f" ]; then
    echo "upstream-diff: $f is not in fsnotify v1.10.1" >&2
    exit 1
  fi
  diff -u --label "a/$f" --label "b/$f" "$up/$f" "$f" >> "$out" || true
done
if [ "${1:-}" = --check ]; then
  if diff -u upstream.patch "$out"; then
    echo "upstream-diff: matches upstream.patch"
  else
    echo "upstream-diff: the copy differs from upstream.patch" >&2
    echo "fix: internal/fsnotify/upstream-diff.sh > internal/fsnotify/upstream.patch, then review it" >&2
    exit 1
  fi
else
  cat "$out"
fi
