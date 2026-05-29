#!/usr/bin/env bash
# Build logify for every supported platform.
#
# Output: dist/logify-<os>-<arch>[.exe]   and   dist/SHA256SUMS
#
# Usage:
#   scripts/build-all.sh            # uses version from VERSION file or "dev"
#   VERSION=v0.2.0 scripts/build-all.sh
set -euo pipefail

cd "$(dirname "$0")/.."
mkdir -p dist
rm -f dist/logify-* dist/SHA256SUMS

VERSION="${VERSION:-$( [ -f VERSION ] && cat VERSION || echo dev )}"
LDFLAGS="-s -w -X main.version=${VERSION}"

PLATFORMS=(
  "linux amd64"
  "linux arm64"
  "darwin amd64"
  "darwin arm64"
  "windows amd64"
  "windows arm64"
)

for combo in "${PLATFORMS[@]}"; do
  read -r OS ARCH <<<"$combo"
  EXT=""
  [ "$OS" = "windows" ] && EXT=".exe"
  OUT="dist/logify-${OS}-${ARCH}${EXT}"
  echo "▸ building $OUT (version=${VERSION})"
  GOOS="$OS" GOARCH="$ARCH" go build -ldflags="${LDFLAGS}" -o "$OUT" ./cmd/logify
done

# checksum file for the release page
( cd dist && shasum -a 256 logify-* > SHA256SUMS )

echo
echo "✓ done"
ls -la dist/
