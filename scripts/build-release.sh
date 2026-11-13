#!/usr/bin/env bash
# Build hug release archives for every platform (pure Go, CGO off).
# Usage: scripts/build-release.sh v0.7.0 -> dist/hug_<ver>_<os>_<arch>.{tar.gz,zip} + SHA256SUMS
# Each archive holds one binary, hug, plus muninn and huginn: the same program,
# which acts as that part when called by that name (hooks and MCP configs
# installed before hug keep working).
set -euo pipefail
cd "$(dirname "$0")/.."
ver="${1:?version}"
rm -rf dist && mkdir -p dist
ld="-s -w -X main.Version=$ver -X github.com/veridit/hug/muninn/run.Version=$ver -X github.com/veridit/hug/huginn/run.Version=$ver"
for t in darwin/arm64 darwin/amd64 linux/amd64 linux/arm64 windows/amd64 windows/arm64; do
  os=${t%/*}; arch=${t#*/}
  name="hug_${ver#v}_${os}_${arch}"
  work="dist/$name"; mkdir -p "$work"
  ext=""; [ "$os" = windows ] && ext=.exe
  CGO_ENABLED=0 GOOS=$os GOARCH=$arch go build -trimpath -ldflags "$ld" -o "$work/hug$ext" ./cmd/hug
  # Copies, not links: zip and Windows have no symlinks. tar.gz keeps them small
  # with hard links.
  if [ "$os" = windows ]; then
    cp "$work/hug$ext" "$work/muninn$ext"; cp "$work/hug$ext" "$work/huginn$ext"
  else
    ln "$work/hug" "$work/muninn"; ln "$work/hug" "$work/huginn"
  fi
  cp LICENSE README.md "$work/"
  if [ "$os" = windows ]; then (cd dist && zip -qr "$name.zip" "$name"); else tar -C dist -czf "dist/$name.tar.gz" "$name"; fi
  rm -rf "$work"
done
(cd dist && shasum -a 256 * > SHA256SUMS)
ls -la dist
