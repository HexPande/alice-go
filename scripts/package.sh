#!/usr/bin/env bash
set -euo pipefail

# Run from the repository root. No C toolchain or ARM runner is required.
mkdir -p dist/linux-arm64 dist/linux-armv7

CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build \
  -trimpath -ldflags='-s -w' -o dist/linux-arm64/alice ./cmd/alice
CGO_ENABLED=0 GOOS=linux GOARCH=arm GOARM=7 go build \
  -trimpath -ldflags='-s -w' -o dist/linux-armv7/alice ./cmd/alice

for target in linux-arm64 linux-armv7; do
  tar -czf "dist/alice-go_${target}.tar.gz" \
    -C "dist/${target}" alice -C ../.. README.md scripts/install.sh
done

(
  cd dist
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum alice-go_linux-arm64.tar.gz alice-go_linux-armv7.tar.gz > SHA256SUMS
  else
    shasum -a 256 alice-go_linux-arm64.tar.gz alice-go_linux-armv7.tar.gz > SHA256SUMS
  fi
)
