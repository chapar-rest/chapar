#!/bin/bash
# Vendor dependencies, then restore the C sources `go mod vendor` drops.
#
# `go mod vendor` copies only directories that hold Go files. The
# tree-sitter grammars keep their C parser in a sibling directory
# (bindings/go includes "../../src/parser.c"), and go-tree-sitter keeps its
# headers and runtime in include/ and src/, so a plain vendor build fails.
# Run this instead of `go mod vendor` (make vendor).
set -euo pipefail

cd "$(dirname "$0")/.."

go mod vendor

for mod in $(awk '/^# github.com\/tree-sitter/ {print $2}' vendor/modules.txt); do
  src=$(GOFLAGS=-mod=mod go list -m -f '{{.Dir}}' "$mod")
  dst="vendor/$mod"
  # Other language bindings (node, python, rust, swift, c) are not needed.
  (cd "$src" && find . -type f \( -name '*.c' -o -name '*.h' \) ! -path './bindings/*' ! -path './test/*') |
    while read -r f; do
      mkdir -p "$dst/$(dirname "$f")"
      cp "$src/$f" "$dst/$f"
      chmod u+w "$dst/$f"
    done
done

# wgpu's lib/vendor.go, which pulls its static libraries into vendor/,
# leaves out linux/arm64.
wgpu=$(GOFLAGS=-mod=mod go list -m -f '{{.Dir}}' github.com/cogentcore/webgpu)
mkdir -p vendor/github.com/cogentcore/webgpu/wgpu/lib/linux/arm64
cp "$wgpu/wgpu/lib/linux/arm64/libwgpu_native.a" vendor/github.com/cogentcore/webgpu/wgpu/lib/linux/arm64/
chmod u+w vendor/github.com/cogentcore/webgpu/wgpu/lib/linux/arm64/libwgpu_native.a

# wgpu-native ships static libraries for every platform. Chapar builds for
# neither Android nor iOS, and their libraries are half the size of vendor/.
# Keep the directories: their vendor.go files are imported.
find vendor/github.com/cogentcore/webgpu/wgpu/lib/android vendor/github.com/cogentcore/webgpu/wgpu/lib/ios -name '*.a' -delete
