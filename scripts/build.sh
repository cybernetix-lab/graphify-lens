#!/bin/bash
set -e

BINARY_NAME="graphify-lens"
BUILD_DIR="build"
VERSION=$(git describe --tags --always --dirty 2>/dev/null || echo "dev")

PLATFORMS=(
  "darwin/amd64"
  "darwin/arm64"
  "linux/amd64"
  "linux/arm64"
  "windows/amd64"
)

mkdir -p "$BUILD_DIR"

for platform in "${PLATFORMS[@]}"; do
  GOOS="${platform%/*}"
  GOARCH="${platform#*/}"
  output="$BUILD_DIR/${BINARY_NAME}_${VERSION}_${GOOS}_${GOARCH}"

  if [ "$GOOS" = "windows" ]; then
    output="${output}.exe"
  fi

  echo "building $GOOS/$GOARCH -> $output"
  GOOS=$GOOS GOARCH=$GOARCH go build -ldflags="-s -w -X main.version=$VERSION" -o "$output" ./cmd/graphify-lens/
done

echo ""
echo "build complete. binaries in $BUILD_DIR/:"
ls -lh "$BUILD_DIR/"
