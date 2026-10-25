#!/usr/bin/env bash
set -euo pipefail

VERSION="${1:-0.1.0}"
COMMIT="$(git rev-parse --short HEAD 2>/dev/null || echo 'unknown')"
DATE="$(date -u +'%Y-%m-%dT%H:%M:%SZ')"
DIST_DIR="dist/releases"

echo "Building Shellix v${VERSION} (${COMMIT}) for all platforms..."
rm -rf "${DIST_DIR}"
mkdir -p "${DIST_DIR}"

TARGETS=(
  "darwin/arm64"
  "darwin/amd64"
  "linux/amd64"
  "linux/arm64"
  "windows/amd64"
  "windows/arm64"
)

LDFLAGS="-s -w -X shellix/internal/core.Version=${VERSION} -X shellix/internal/core.Commit=${COMMIT} -X shellix/internal/core.Date=${DATE}"

for TARGET in "${TARGETS[@]}"; do
  OS="${TARGET%/*}"
  ARCH="${TARGET#*/}"
  EXT=""
  if [ "${OS}" = "windows" ]; then
    EXT=".exe"
  fi

  BIN_NAME="shellix${EXT}"
  BUILD_DIR="dist/build_${OS}_${ARCH}"
  mkdir -p "${BUILD_DIR}"

  echo "  🔨 Compiling ${OS}/${ARCH}..."
  CGO_ENABLED=0 GOOS="${OS}" GOARCH="${ARCH}" go build -ldflags="${LDFLAGS}" -o "${BUILD_DIR}/${BIN_NAME}" ./cmd/shellix

  # Create alias sx symlink / copy
  if [ "${OS}" = "windows" ]; then
    cp "${BUILD_DIR}/${BIN_NAME}" "${BUILD_DIR}/sx.exe"
  else
    cp "${BUILD_DIR}/${BIN_NAME}" "${BUILD_DIR}/sx"
  fi

  # Copy documentation assets
  cp README.md LICENSE "${BUILD_DIR}/"

  # Package archive
  ARCHIVE_NAME="shellix_${VERSION}_${OS}_${ARCH}"
  if [ "${OS}" = "windows" ]; then
    (cd "${BUILD_DIR}" && zip -q -r "../../${DIST_DIR}/${ARCHIVE_NAME}.zip" .)
    echo "  📦 Packaged ${DIST_DIR}/${ARCHIVE_NAME}.zip"
  else
    tar -czf "${DIST_DIR}/${ARCHIVE_NAME}.tar.gz" -C "${BUILD_DIR}" .
    echo "  📦 Packaged ${DIST_DIR}/${ARCHIVE_NAME}.tar.gz"
  fi

  rm -rf "${BUILD_DIR}"
done

echo "  🔒 Generating SHA256 checksums..."
(cd "${DIST_DIR}" && shasum -a 256 shellix_* > checksums.txt)

echo "✨ Release build complete! Assets in ${DIST_DIR}:"
ls -lh "${DIST_DIR}"
