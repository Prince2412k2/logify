#!/usr/bin/env bash
# logify installer for Linux + macOS.
#
# Usage:
#   curl -sSL https://raw.githubusercontent.com/Prince2412k2/logify/main/install.sh | bash
#   # or, pinned to a specific release:
#   LOGIFY_VERSION=v0.2.0 curl -sSL .../install.sh | bash
#   # or, from source (requires Go):
#   curl -sSL .../install.sh | INSTALL_MODE=source bash
#
# Env vars:
#   LOGIFY_REPO     — GitHub owner/repo (default: Prince2412k2/logify)
#   LOGIFY_VERSION  — tag to install (default: latest)
#   LOGIFY_PREFIX   — install dir (default: ~/.local/bin, fallback /usr/local/bin)
#   INSTALL_MODE    — "release" (download binary) or "source" (compile)
set -euo pipefail

REPO="${LOGIFY_REPO:-Prince2412k2/logify}"
# VERSION accepts:
#   "latest"  → newest stable tag
#   "nightly" → rolling build of main
#   "vX.Y.Z"  → explicit tag
VERSION="${LOGIFY_VERSION:-latest}"
MODE="${INSTALL_MODE:-release}"

bold()  { printf '\033[1m%s\033[0m' "$1"; }
green() { printf '\033[32m%s\033[0m' "$1"; }
amber() { printf '\033[33m%s\033[0m' "$1"; }
red()   { printf '\033[31m%s\033[0m' "$1"; }
say()   { printf '%s\n' "$@" >&2; }
die()   { say "$(red "✕") $*"; exit 1; }
need()  { command -v "$1" >/dev/null 2>&1 || die "$1 is required but not installed"; }

# ── pick install prefix ────────────────────────────────────────────────
choose_prefix() {
  if [ -n "${LOGIFY_PREFIX:-}" ]; then
    echo "$LOGIFY_PREFIX"; return
  fi
  if [ -w "$HOME/.local/bin" ] 2>/dev/null || mkdir -p "$HOME/.local/bin" 2>/dev/null; then
    echo "$HOME/.local/bin"; return
  fi
  if [ -w /usr/local/bin ] 2>/dev/null; then
    echo /usr/local/bin; return
  fi
  echo "$HOME/.local/bin"
}

# ── detect OS / arch ───────────────────────────────────────────────────
detect_platform() {
  local os arch
  os=$(uname -s | tr '[:upper:]' '[:lower:]')
  case "$os" in
    darwin|linux) ;;
    *) die "unsupported OS: $os (only linux + darwin)" ;;
  esac
  arch=$(uname -m)
  case "$arch" in
    x86_64|amd64) arch=amd64 ;;
    arm64|aarch64) arch=arm64 ;;
    *) die "unsupported arch: $arch" ;;
  esac
  echo "${os}-${arch}"
}

# ── release mode ───────────────────────────────────────────────────────
install_release() {
  need curl
  local platform tag tmp url checksum_url
  platform=$(detect_platform)
  if [ "$VERSION" = "latest" ]; then
    tag=$(curl -fsSL "https://api.github.com/repos/$REPO/releases/latest" \
      | sed -nE 's/.*"tag_name":\s*"([^"]+)".*/\1/p' | head -1)
    [ -n "$tag" ] || die "could not resolve latest release for $REPO (try LOGIFY_VERSION=nightly)"
  else
    tag="$VERSION"
  fi
  url="https://github.com/$REPO/releases/download/${tag}/logify-${platform}"
  say "$(amber "▸") downloading $(bold "$tag") for $(bold "$platform")"
  tmp=$(mktemp)
  curl -fsSL --progress-bar "$url" -o "$tmp" || die "download failed: $url"

  local prefix
  prefix=$(choose_prefix)
  mkdir -p "$prefix"
  chmod +x "$tmp"
  mv "$tmp" "$prefix/logify"
  say "$(green "✓") installed $(bold "$prefix/logify")"
  post_install "$prefix"
}

# ── source mode ────────────────────────────────────────────────────────
install_from_source() {
  need git; need go
  local tmp prefix
  tmp=$(mktemp -d)
  say "$(amber "▸") cloning $REPO"
  git clone --depth 1 "https://github.com/$REPO.git" "$tmp" >/dev/null
  (cd "$tmp" && go build -ldflags="-s -w" -o logify ./cmd/logify) \
    || die "go build failed"
  prefix=$(choose_prefix)
  mkdir -p "$prefix"
  mv "$tmp/logify" "$prefix/logify"
  rm -rf "$tmp"
  say "$(green "✓") built and installed $(bold "$prefix/logify")"
  post_install "$prefix"
}

# ── PATH hint + smoke check ────────────────────────────────────────────
post_install() {
  local prefix="$1"
  # Smoke check: run the freshly-installed binary so we catch broken
  # downloads / arch mismatches immediately.
  local ver
  ver=$("$prefix/logify" version 2>/dev/null || true)
  if [ -z "$ver" ]; then
    say "$(red "✕") installed binary failed to run — try a manual download from"
    say "    https://github.com/$REPO/releases"
    exit 1
  fi
  say "$(green "✓") $ver"
  case ":$PATH:" in
    *":$prefix:"*) ;;
    *)
      say ""
      say "$(amber "⚠")  $(bold "$prefix") is not in your PATH."
      say "    add this line to your shell init file (~/.bashrc, ~/.zshrc):"
      say ""
      say "      export PATH=\"$prefix:\$PATH\""
      say "" ;;
  esac
  say ""
  say "next: run $(bold 'logify login') to point it at your gateway."
}

# ── main ───────────────────────────────────────────────────────────────
case "$MODE" in
  release) install_release ;;
  source)  install_from_source ;;
  *) die "INSTALL_MODE must be 'release' or 'source', got '$MODE'" ;;
esac
