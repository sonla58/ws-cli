#!/usr/bin/env bash
# Install ws by downloading the matching release archive from GitHub.
#
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/sonla58/ws-cli/main/scripts/install.sh | bash
#
# Environment:
#   WS_REPO     override the GitHub repo (default: sonla58/ws-cli)
#   WS_VERSION  install a specific tag (default: latest release)
#   PREFIX      install prefix (default: $HOME/.local; binary lands in $PREFIX/bin)

set -euo pipefail

REPO="${WS_REPO:-sonla58/ws-cli}"
PREFIX="${PREFIX:-$HOME/.local}"
BINDIR="$PREFIX/bin"

die()  { printf 'error: %s\n' "$*" >&2; exit 1; }
info() { printf '==> %s\n' "$*"; }

command -v curl >/dev/null   || die "curl is required"
command -v tar  >/dev/null   || die "tar is required"
command -v uname >/dev/null  || die "uname is required"

# Detect OS.
os_raw="$(uname -s)"
case "$os_raw" in
  Darwin)  os="darwin"  ;;
  Linux)   os="linux"   ;;
  *)       die "unsupported OS: $os_raw" ;;
esac

# Detect arch.
arch_raw="$(uname -m)"
case "$arch_raw" in
  x86_64|amd64)  arch="x86_64" ;;
  arm64|aarch64) arch="arm64"  ;;
  *)             die "unsupported arch: $arch_raw" ;;
esac

# Resolve version.
if [[ -n "${WS_VERSION:-}" ]]; then
  tag="$WS_VERSION"
else
  info "resolving latest release of $REPO"
  tag="$(curl -fsSL "https://api.github.com/repos/$REPO/releases/latest" \
           | grep -m1 '"tag_name":' \
           | cut -d'"' -f4)"
  [[ -n "$tag" ]] || die "could not resolve latest tag"
fi

version="${tag#v}"
asset="ws_${version}_${os}_${arch}.tar.gz"
url="https://github.com/$REPO/releases/download/$tag/$asset"

info "downloading $url"
tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT
curl -fsSL "$url" -o "$tmp/$asset"

info "extracting"
tar -xzf "$tmp/$asset" -C "$tmp"

mkdir -p "$BINDIR"
install -m 0755 "$tmp/ws" "$BINDIR/ws"
info "installed: $BINDIR/ws"

cat <<EOF

ws $tag installed.

Next steps:
  1. Ensure $BINDIR is on your PATH:
       export PATH="$BINDIR:\$PATH"
  2. Add shell integration to your rc file (pick one):
       zsh:  echo 'eval "\$(ws init zsh)"'  >> ~/.zshrc
       bash: echo 'eval "\$(ws init bash)"' >> ~/.bashrc
       fish: echo 'ws init fish | source'  >> ~/.config/fish/config.fish
  3. Open a new terminal, then:  ws add

EOF
