#!/usr/bin/env bash
#
# build.sh: cross-compile kload for all supported platforms
# Produces stripped binaries (-s -w) in the ./dist directory.
#
set -euo pipefail

# Config
APP="kload"
ENTRY="./main.go"
DIST="dist"
VERSION="${1:-dev}"   # pass version as first arg, e.g. ./build.sh 1.0.0

# Strip debug info + embed version
LDFLAGS="-s -w -X main.version=${VERSION}"

# platform list: "GOOS/GOARCH"
PLATFORMS=(
  "linux/amd64"
  "linux/arm64"
  "darwin/amd64"
  "darwin/arm64"
  "windows/amd64"
)

# Build
echo "Building ${APP} ${VERSION}"
rm -rf "${DIST}"
mkdir -p "${DIST}"

for platform in "${PLATFORMS[@]}"; do
  GOOS="${platform%/*}"
  GOARCH="${platform#*/}"

  output="${DIST}/${APP}-${GOOS}-${GOARCH}"
  if [ "${GOOS}" = "windows" ]; then
    output="${output}.exe"
  fi

  echo "  → ${GOOS}/${GOARCH}"
  GOOS="${GOOS}" GOARCH="${GOARCH}" CGO_ENABLED=0 \
    go build -ldflags="${LDFLAGS}" -o "${output}" "${ENTRY}"
done

echo ""
echo "Done. Binaries in ./${DIST}:"
ls -lh "${DIST}"