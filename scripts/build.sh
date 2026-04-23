#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

cd "${REPO_ROOT}"

VERSION="${VERSION:-$(git describe --tags --always --dirty 2>/dev/null || echo dev)}"
BUILD_TIME="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
LDFLAGS="-X main.version=${VERSION} -X main.buildTime=${BUILD_TIME}"

OUTPUT_DIR="${REPO_ROOT}/dist"
mkdir -p "${OUTPUT_DIR}"

echo "Building video-review ${VERSION}..."

build_target() {
  local os="$1"
  local arch="$2"
  local suffix="${3:-}"
  local output="${OUTPUT_DIR}/video-review-${os}-${arch}${suffix}"

  echo "  ${os}/${arch} -> ${output}"
  GOOS="${os}" GOARCH="${arch}" go build \
    -ldflags "${LDFLAGS}" \
    -o "${output}" \
    ./cmd/video-review/
}

build_target darwin  amd64
build_target darwin  arm64
build_target linux   amd64
build_target linux   arm64
build_target windows amd64 .exe

echo ""
echo "Build complete. Binaries in ${OUTPUT_DIR}/"
ls -lh "${OUTPUT_DIR}/"
