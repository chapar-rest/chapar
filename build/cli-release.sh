#!/usr/bin/env bash
# Builds chapar-cli for every release platform and packages it with the
# license: a .tar.gz per platform (.zip on Windows) and a checksums file.
#
#   build/cli-release.sh v0.8.0 [dist/cli]
#
# The CLI has no cgo and no UI, so every target cross-compiles from one host
# into a static binary.
set -euo pipefail

tag=$1
out=${2:-dist/cli}
targets="linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64 windows/arm64"

rm -rf "$out"
mkdir -p "$out"
out=$(cd "$out" && pwd)
work=$(mktemp -d)
trap 'rm -rf "$work"' EXIT

for target in $targets; do
	os=${target%/*}
	arch=${target#*/}
	platform=$os
	[ "$os" = darwin ] && platform=macos
	bin=chapar-cli
	[ "$os" = windows ] && bin=chapar-cli.exe
	base="chapar-cli-$platform-$tag-$arch"

	dir="$work/$base"
	mkdir -p "$dir"
	CGO_ENABLED=0 GOOS=$os GOARCH=$arch go build -trimpath \
		-ldflags "-s -w -X github.com/chapar-rest/chapar/version.AppVersion=$tag" \
		-o "$dir/$bin" ./cmd/chapar-cli
	cp LICENSE "$dir/"

	if [ "$os" = windows ]; then
		(cd "$dir" && zip -q "$out/$base.zip" "$bin" LICENSE)
	else
		tar -C "$dir" -czf "$out/$base.tar.gz" "$bin" LICENSE
	fi
	echo "built $base"
done

# Only the archives: the checksums file must not list itself.
(cd "$out" && { command -v sha256sum >/dev/null && sha256sum ./*.tar.gz ./*.zip || shasum -a 256 ./*.tar.gz ./*.zip; } |
	sed 's#  \./#  #' >"chapar-cli-$tag-checksums.txt")
