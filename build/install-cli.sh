#!/bin/sh
# Installs chapar-cli, the command-line runner for Chapar test cases.
#
#   curl -fsSL https://github.com/chapar-rest/chapar/releases/latest/download/install-cli.sh | sh
#   curl -fsSL .../install-cli.sh | sh -s -- -v v0.8.0 -b "$HOME/.local/bin"
#
# Options:
#   -v VERSION  release to install, such as v0.8.0 (default: the latest)
#   -b DIR      directory to install into (default: /usr/local/bin when it
#               is writable, else ~/.local/bin)
#
# Linux and macOS, amd64 and arm64. On Windows, download the .zip from the
# release page instead.
set -eu

repo="chapar-rest/chapar"
base="${CHAPAR_CLI_DOWNLOAD_URL:-https://github.com/$repo/releases}"
version="${CHAPAR_CLI_VERSION:-}"
bindir="${CHAPAR_CLI_BIN_DIR:-}"

usage() {
	cat <<'USAGE'
Installs chapar-cli, the command-line runner for Chapar test cases.

  curl -fsSL https://github.com/chapar-rest/chapar/releases/latest/download/install-cli.sh | sh
  curl -fsSL .../install-cli.sh | sh -s -- -v v0.8.0 -b "$HOME/.local/bin"

Options:
  -v VERSION  release to install, such as v0.8.0 (default: the latest)
  -b DIR      directory to install into (default: /usr/local/bin when it
              is writable, else ~/.local/bin)
USAGE
}

say() { printf 'chapar-cli: %s\n' "$*" >&2; }
fail() {
	say "$*"
	exit 1
}

while getopts "v:b:h" opt; do
	case "$opt" in
	v) version=$OPTARG ;;
	b) bindir=$OPTARG ;;
	h)
		usage
		exit 0
		;;
	*)
		usage >&2
		exit 2
		;;
	esac
done

command -v curl >/dev/null 2>&1 || fail "curl is required"
command -v tar >/dev/null 2>&1 || fail "tar is required"

case "$(uname -s)" in
Linux) os=linux ;;
Darwin) os=macos ;;
*) fail "$(uname -s) is not supported by this script; download chapar-cli from https://github.com/$repo/releases" ;;
esac

case "$(uname -m)" in
x86_64 | amd64) arch=amd64 ;;
arm64 | aarch64) arch=arm64 ;;
*) fail "$(uname -m) is not supported" ;;
esac

if [ -z "$version" ]; then
	# The latest-release page redirects to its tag; reading the redirect
	# needs no API token, so CI runners are not rate limited.
	url=$(curl -fsSLI -o /dev/null -w '%{url_effective}' "$base/latest") ||
		fail "could not find the latest release"
	version=${url##*/}
fi
case "$version" in
v*) ;;
*) version="v$version" ;;
esac

if [ -z "$bindir" ]; then
	if [ -w /usr/local/bin ]; then
		bindir=/usr/local/bin
	else
		bindir="$HOME/.local/bin"
	fi
fi

archive="chapar-cli-$os-$version-$arch.tar.gz"
sums="chapar-cli-$version-checksums.txt"
tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT

say "downloading $archive"
curl -fsSL -o "$tmp/$archive" "$base/download/$version/$archive" ||
	fail "could not download $archive; is $version a release with chapar-cli?"
curl -fsSL -o "$tmp/$sums" "$base/download/$version/$sums" ||
	fail "could not download $sums"

want=$(awk -v f="$archive" '$2 == f || $2 == "*" f { print $1 }' "$tmp/$sums")
[ -n "$want" ] || fail "$sums has no checksum for $archive"
if command -v sha256sum >/dev/null 2>&1; then
	got=$(sha256sum "$tmp/$archive" | awk '{print $1}')
else
	got=$(shasum -a 256 "$tmp/$archive" | awk '{print $1}')
fi
[ "$got" = "$want" ] || fail "checksum mismatch for $archive"

tar -xzf "$tmp/$archive" -C "$tmp" chapar-cli
mkdir -p "$bindir"
if ! mv "$tmp/chapar-cli" "$bindir/chapar-cli" 2>/dev/null; then
	fail "cannot write to $bindir; pick another directory with -b"
fi
chmod +x "$bindir/chapar-cli"

say "installed $("$bindir/chapar-cli" version) to $bindir/chapar-cli"
case ":$PATH:" in
*":$bindir:"*) ;;
*)
	if [ -n "${GITHUB_PATH:-}" ]; then
		# GitHub Actions: later steps of the job find it on PATH.
		echo "$bindir" >>"$GITHUB_PATH"
	else
		say "$bindir is not on your PATH; add it, or run $bindir/chapar-cli"
	fi
	;;
esac
