#!/usr/bin/env bash
# The diff between this copy and upstream fsnotify v1.10.1 (mq5ger).
#   internal/fsnotify/upstream-diff.sh          prints it
#   internal/fsnotify/upstream-diff.sh --check  fails unless upstream.patch, applied to
#     v1.10.1, gives this copy byte for byte (the code, not diff's hunk layout,
#     which differs between GNU and BSD diff)
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
cd "$here"
files="$(find . -type f \( -name '*.go' -o -name LICENSE \) ! -name '*_test.go' | sed 's|^\./||' | LC_ALL=C sort)"
for f in $files; do
  if [ ! -f "$up/$f" ]; then
    echo "upstream-diff: $f is not in fsnotify v1.10.1" >&2
    echo "fix: carry only fsnotify v1.10.1's files in internal/fsnotify (README.md)" >&2
    exit 1
  fi
done
if [ "${1:-}" = --check ]; then
  # upstream.patch on a fresh v1.10.1 tree (its own repository, so git
  # apply reads the paths from there) must give the copy byte for byte.
  tree="$tmp/tree"
  mkdir "$tree"
  for f in $files; do
    mkdir -p "$tree/$(dirname "$f")"
    cp "$up/$f" "$tree/$f"
    chmod u+w "$tree/$f"
  done
  git -C "$tree" init -q
  if ! git -C "$tree" apply --whitespace=nowarn "$here/upstream.patch"; then
    echo "upstream-diff: upstream.patch does not apply to fsnotify v1.10.1" >&2
    echo "fix: internal/fsnotify/upstream-diff.sh > internal/fsnotify/upstream.patch, then review it" >&2
    exit 1
  fi
  differ=""
  for f in $files; do
    cmp -s "$tree/$f" "$f" || differ="$differ $f"
  done
  if [ -n "$differ" ]; then
    echo "upstream-diff: v1.10.1 plus upstream.patch differs from the copy in:$differ" >&2
    for f in $differ; do
      diff -u --label "v1.10.1+patch/$f" --label "copy/$f" "$tree/$f" "$f" >&2 || true
    done
    echo "fix: internal/fsnotify/upstream-diff.sh > internal/fsnotify/upstream.patch, then review it" >&2
    exit 1
  fi
  echo "upstream-diff: v1.10.1 plus upstream.patch is the copy, byte for byte"
else
  for f in $files; do
    diff -u --label "a/$f" --label "b/$f" "$up/$f" "$f" || true
  done
fi
