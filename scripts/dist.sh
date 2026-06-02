#!/bin/bash
set -e

BINARY_NAME="graphify-lens"
VERSION=$(git describe --tags --always --dirty 2>/dev/null || echo "dev")
DIST_DIR="dist/${BINARY_NAME}_${VERSION}"

rm -rf "dist/${BINARY_NAME}_${VERSION}" "dist/${BINARY_NAME}_${VERSION}.tar.gz" "dist/${BINARY_NAME}_${VERSION}.zip"

PLATFORMS=(
  "darwin/amd64"
  "darwin/arm64"
  "linux/amd64"
  "linux/arm64"
  "windows/amd64"
)

for platform in "${PLATFORMS[@]}"; do
  GOOS="${platform%/*}"
  GOARCH="${platform#*/}"
  output_dir="$DIST_DIR/${GOOS}_${GOARCH}"
  mkdir -p "$output_dir"

  binary="$output_dir/${BINARY_NAME}"
  if [ "$GOOS" = "windows" ]; then
    binary="${binary}.exe"
  fi

  echo "building $GOOS/$GOARCH"
  GOOS=$GOOS GOARCH=$GOARCH go build -ldflags="-s -w -X main.version=$VERSION" -o "$binary" ./cmd/graphify-lens/

  cp README.md "$output_dir/" 2>/dev/null || true
  cp config.example.json "$output_dir/" 2>/dev/null || true
done

cd dist
tar czf "${BINARY_NAME}_${VERSION}.tar.gz" "${BINARY_NAME}_${VERSION}"
zip -r "${BINARY_NAME}_${VERSION}.zip" "${BINARY_NAME}_${VERSION}" > /dev/null
cd ..

echo ""
echo "distribution packages:"
ls -lh dist/*.tar.gz dist/*.zip 2>/dev/null
