#!/usr/bin/env bash
# Renders the chapar-cli Homebrew formula for a release.
#
#   build/homebrew/render-formula.sh v0.8.0 dist/cli/chapar-cli-v0.8.0-checksums.txt > chapar-cli.rb
#
# The checksums file is `sha256sum` output for the release archives.
set -euo pipefail

tag=$1
sums=$2
version=${tag#v}
dir=$(cd "$(dirname "$0")" && pwd)

sha() {
	local file="chapar-cli-$1-$tag-$2.tar.gz"
	local sum
	sum=$(awk -v f="$file" '$2 == f || $2 == "*" f { print $1 }' "$sums")
	if [ -z "$sum" ]; then
		echo "render-formula: no checksum for $file in $sums" >&2
		exit 1
	fi
	echo "$sum"
}

sed \
	-e "s/{{VERSION}}/$version/g" \
	-e "s/{{SHA_MACOS_ARM64}}/$(sha macos arm64)/" \
	-e "s/{{SHA_MACOS_AMD64}}/$(sha macos amd64)/" \
	-e "s/{{SHA_LINUX_ARM64}}/$(sha linux arm64)/" \
	-e "s/{{SHA_LINUX_AMD64}}/$(sha linux amd64)/" \
	"$dir/chapar-cli.rb.tmpl"
